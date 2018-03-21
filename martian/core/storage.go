// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

//
// Martian runtime storage tracking and recovery.
//

import (
	"os"
	"sort"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

//
// Volatile Disk Recovery
//
type VDRKillReport struct {
	Count     uint     `json:"count"`
	Size      uint64   `json:"size"`
	Timestamp string   `json:"timestamp"`
	Paths     []string `json:"paths"`
	Errors    []string `json:"errors"`
}

type VDRByTimestamp []*VDRKillReport

func (self VDRByTimestamp) Len() int {
	return len(self)
}

func (self VDRByTimestamp) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self VDRByTimestamp) Less(i, j int) bool {
	return self[i].Timestamp < self[j].Timestamp
}

func mergeVDRKillReports(killReports []*VDRKillReport) *VDRKillReport {
	allKillReport := &VDRKillReport{}
	for _, killReport := range killReports {
		allKillReport.Size += killReport.Size
		allKillReport.Count += killReport.Count
		allKillReport.Errors = append(allKillReport.Errors, killReport.Errors...)
		allKillReport.Paths = append(allKillReport.Paths, killReport.Paths...)
	}
	sort.Sort(VDRByTimestamp(killReports))
	if len(killReports) > 0 {
		allKillReport.Timestamp = killReports[len(killReports)-1].Timestamp
	}
	return allKillReport
}

func (self *Fork) vdrKill() *VDRKillReport {
	killReport := &VDRKillReport{}
	if self.node.rt.vdrMode == "disable" {
		return killReport
	}
	if killReport, ok := self.getVdrKillReport(); ok {
		return killReport
	}

	killPaths := []string{}
	// For volatile nodes, kill fork-level files.
	if self.node.rt.overrides.GetOverride(self.node, "force_volatile", self.node.volatile).(bool) {
		if paths, err := self.metadata.enumerateFiles(); err == nil {
			killPaths = append(killPaths, paths...)
		}
		if paths, err := self.split_metadata.enumerateFiles(); err == nil {
			killPaths = append(killPaths, paths...)
		}
		if paths, err := self.join_metadata.enumerateFiles(); err == nil {
			killPaths = append(killPaths, paths...)
		}
	}
	// If the node splits, kill chunk-level files.
	// Must check for split here, otherwise we'll end up deleting
	// output files of non-volatile nodes because single-chunk nodes
	// get their output redirected to the one chunk's files path.
	if self.node.split && self.node.rt.overrides.GetOverride(self.node, "force_volatile", true).(bool) {
		for _, chunk := range self.chunks {
			if paths, err := chunk.metadata.enumerateFiles(); err == nil {
				killPaths = append(killPaths, paths...)
			}
		}
	}
	// Actually delete the paths.
	for _, p := range killPaths {
		util.Walk(p, func(_ string, info os.FileInfo, err error) error {
			if err == nil {
				killReport.Size += uint64(info.Size())
				killReport.Count++
			} else {
				killReport.Errors = append(killReport.Errors, err.Error())
			}
			return nil
		})
		killReport.Paths = append(killReport.Paths, p)
		os.RemoveAll(p)
	}
	// update timestamp to mark actual kill time
	killReport.Timestamp = util.Timestamp()
	self.metadata.Write(VdrKill, killReport)
	return killReport
}

/* Is self or any of its ancestors symlinked? */
func (self *Node) vdrCheckSymlink() (string, error) {

	/* Nope! Got all the way to the top.
	 * (We don't care of the top-level directory is a symlink)
	 */
	if self.parent == nil {
		return "", nil
	}
	statinfo, err := os.Lstat(self.path)

	if err != nil {
		return "", err
	}
	/* Yep! Found a symlink */
	if (statinfo.Mode() & os.ModeSymlink) != 0 {
		return self.path, nil
	}

	return self.parent.getNode().vdrCheckSymlink()
}

func (self *Node) vdrKill() (*VDRKillReport, bool) {

	/*
	 * Refuse to VDR a node if it, or any of its ancestors are symlinked.
	 */
	if symlink, err := self.vdrCheckSymlink(); symlink != "" {
		util.LogInfo("runtime", "Refuse to VDR across a symlink %s: %v", symlink, self.fqname)
		return &VDRKillReport{}, true
	} else if err != nil {
		util.LogError(err, "runtime", "Error reading node directory: %v", self.fqname)
		return &VDRKillReport{}, false
	}

	if self.filePostNodes != nil {
		for _, node := range self.filePostNodes {
			if _, ok := node.(*TopNode); ok {
				// This is a top-level node
				self.volatile = false
			} else if _, ok := node.getNode().parent.(*TopNode); ok {
				// This is a top-level pipestance or stagestance.
				// Don't VDR if the top-level outputs depend on this.
				self.volatile = false
			} else if s := node.getNode().state; s != Complete && s != DisabledState {
				return &VDRKillReport{}, false
			}
		}
	}
	killReports := make([]*VDRKillReport, 0, len(self.forks))
	for _, fork := range self.forks {
		killReports = append(killReports, fork.vdrKill())
	}
	return mergeVDRKillReports(killReports), true
}

type StorageEvent struct {
	Timestamp time.Time
	Delta     int64
	Name      string
}

func NewStorageEvent(timestamp time.Time, delta int64, fqname string) *StorageEvent {
	self := &StorageEvent{}
	self.Timestamp = timestamp
	self.Delta = delta
	self.Name = fqname
	return self
}

type StorageEventByTimestamp []*StorageEvent

func (self StorageEventByTimestamp) Len() int {
	return len(self)
}

func (self StorageEventByTimestamp) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self StorageEventByTimestamp) Less(i, j int) bool {
	return self[i].Timestamp.Before(self[j].Timestamp)
}

// this is due to the fact that the VDR bytes/total bytes
// reported at the fork level is the sum of chunk + split
// + join plus any additional files.  The additional
// files that are unique to the fork cannot be resolved
// unless you sub out chunk/split/join and then child
// stages.
type ForkStorageEvent struct {
	Name          string
	ChildNames    []string
	TotalBytes    uint64
	ChunkBytes    uint64
	ForkBytes     uint64
	TotalVDRBytes uint64
	ForkVDRBytes  uint64
	Timestamp     time.Time
	VDRTimestamp  time.Time
}

func NewForkStorageEvent(timestamp time.Time, totalBytes uint64, vdrBytes uint64, fqname string) *ForkStorageEvent {
	self := &ForkStorageEvent{ChildNames: []string{}}
	self.Name = fqname
	self.TotalBytes = totalBytes // sum total of bytes in fork and children
	self.ForkBytes = self.TotalBytes
	self.TotalVDRBytes = vdrBytes          // sum total of VDR bytes in fork and children
	self.ForkVDRBytes = self.TotalVDRBytes // VDR bytes in forkN/files
	self.Timestamp = timestamp
	return self
}

func (self *Pipestance) VDRKill() *VDRKillReport {
	killReports := []*VDRKillReport{}
	for _, node := range self.node.allNodes() {
		killReport, _ := node.vdrKill()
		killReports = append(killReports, killReport)
	}
	killReport := mergeVDRKillReports(killReports)
	self.metadata.Write(VdrKill, killReport)
	return killReport
}

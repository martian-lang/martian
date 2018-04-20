// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

//
// Martian runtime storage tracking and recovery.
//

import (
	"os"
	"sort"
	"strings"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

type VdrEvent struct {
	Timestamp  time.Time
	DeltaBytes int64
}

type PartialVdrKillReport struct {
	VDRKillReport `json:"report,omitempty"`
	Split         bool `json:"ran_split,omitempty"`
	Chunks        bool `json:"ran_chunks,omitempty"`
	Join          bool `json:"ran_join,omitempty"`
}

//
// Volatile Disk Recovery
//
type VDRKillReport struct {
	Count     uint        `json:"count"`
	Size      uint64      `json:"size"`
	Timestamp string      `json:"timestamp"`
	Paths     []string    `json:"paths"`
	Errors    []string    `json:"errors"`
	Events    []*VdrEvent `json:"events,omitempty"`
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
	var allEvents []*VdrEvent
	if len(killReports) > 0 {
		allEvents = make([]*VdrEvent, 0, len(killReports)+len(killReports[0].Events))
	}
	for _, killReport := range killReports {
		if killReport == nil {
			continue
		}
		allKillReport.Size += killReport.Size
		allKillReport.Count += killReport.Count
		allKillReport.Errors = append(allKillReport.Errors, killReport.Errors...)
		allKillReport.Paths = append(allKillReport.Paths, killReport.Paths...)
		allEvents = append(allEvents, killReport.Events...)
		if allKillReport.Timestamp == "" || allKillReport.Timestamp < killReport.Timestamp {
			allKillReport.Timestamp = killReport.Timestamp
		}
	}
	// Merge events with the same timestamp.
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Timestamp.Before(allEvents[j].Timestamp)
	})
	allKillReport.Events = make([]*VdrEvent, 0, len(allEvents))
	for _, ev := range allEvents {
		last := len(allKillReport.Events) - 1
		if last < 0 ||
			allKillReport.Events[last].Timestamp != ev.Timestamp ||
			(ev.DeltaBytes < 0) != (allKillReport.Events[last].DeltaBytes < 0) {
			allKillReport.Events = append(allKillReport.Events, ev)
		} else {
			allKillReport.Events[last].DeltaBytes += ev.DeltaBytes
		}
	}
	return allKillReport
}

func (self *Fork) partialVdrKill() (*VDRKillReport, bool) {
	self.storageLock.Lock()
	defer self.storageLock.Unlock()
	if state := self.getState(); state.IsFailed() {
		return nil, false
	} else if state == DisabledState {
		return self.vdrKill(nil), true
	} else if rep, ok := self.getVdrKillReport(); ok {
		if self.node.rt.Config.Debug {
			util.LogInfo("storage",
				"%s is already VDRed",
				self.node.GetFQName())
		}
		return rep, ok
	} else {
		partial := self.getPartialKillReport()
		if self.Split() &&
			(partial == nil || !partial.Split) &&
			(state == Complete ||
				state == Complete.Prefixed(SplitPrefix) ||
				state.HasPrefix(ChunksPrefix) ||
				state.HasPrefix(JoinPrefix)) {
			partial = self.cleanSplitTemp(partial)
		}
		if (partial == nil || !partial.Chunks) &&
			(state == Complete ||
				state == Complete.Prefixed(ChunksPrefix) ||
				state.HasPrefix(JoinPrefix)) {
			partial = self.cleanChunkTemp(partial)
		}
		if (partial == nil ||
			!partial.Join) &&
			(state == Complete ||
				state == Complete.Prefixed(JoinPrefix)) {
			partial = self.cleanJoinTemp(partial)
		}
		if state == Complete {
			doneNodes := make([]Nodable, 0, len(self.node.filePostNodes))
			for node := range self.node.filePostNodes {
				if node != nil {
					if st := node.getNode().getState(); st == Complete || st == DisabledState {
						doneNodes = append(doneNodes, node)
					}
				}
			}
			self.node.removeFilePostNodes(doneNodes)
			if len(self.node.filePostNodes) == 0 {
				if self.node.rt.Config.Debug {
					util.LogInfo("storage",
						"Running full vdr on %s",
						self.node.GetFQName())
				}
				return self.vdrKill(partial), true
			} else {
				if self.node.rt.Config.Debug {
					for node, args := range self.node.filePostNodes {
						a := make([]string, 0, len(args))
						for arg := range args {
							a = append(a, arg)
						}
						sort.Strings(a)
						if node != nil {
							util.LogInfo("storage",
								"%s is keeping argument(s) %s of %s alive",
								node.GetFQName(),
								strings.Join(a, ","),
								self.node.GetFQName())
						} else {
							util.LogInfo("storage",
								"Outs %s of %s are being kept alive by the top-level.",
								strings.Join(a, ","),
								self.node.GetFQName())
						}
					}
				}
				if self.node.strictVolatile {
					partial = self.vdrKillSome(partial)
				}
			}
		}
		if partial != nil {
			if self.node.rt.Config.Debug {
				util.LogInfo("storage",
					"Partial vdr of %s in phase %v",
					self.node.GetFQName(), state)
			}
			return &partial.VDRKillReport, false
		} else {
			if self.node.rt.Config.Debug {
				util.LogInfo("storage",
					"No partial vdr of %s in phase %v",
					self.node.GetFQName(), state)
			}
			return nil, false
		}
	}
}

func (self *Fork) vdrKillSome(partial *PartialVdrKillReport) *PartialVdrKillReport {
	// TODO: implement strict-mode volatile logic.
	return partial
}

func (metadata *Metadata) getStartTime() time.Time {
	var jobInfo JobInfo
	if err := metadata.ReadInto(JobInfoFile, &jobInfo); err != nil ||
		jobInfo.WallClockInfo == nil {
		return time.Time{}
	} else {
		t, _ := time.ParseInLocation(util.TIMEFMT, jobInfo.WallClockInfo.Start, time.Local)
		return t
	}
}

func (self *Fork) cleanSplitTemp(partial *PartialVdrKillReport) *PartialVdrKillReport {
	if tempPaths, err := self.split_metadata.enumerateTemp(); err != nil {
		return partial
	} else if filesPaths, err := self.split_metadata.enumerateFiles(); err != nil {
		return partial
	} else {
		if partial == nil {
			partial = new(PartialVdrKillReport)
		}
		partial.Split = true
		var startEvent, cleanupEvent VdrEvent
		startEvent.Timestamp = self.split_metadata.getStartTime()
		for _, p := range tempPaths {
			util.Walk(p, func(tpath string, info os.FileInfo, err error) error {
				if err == nil {
					partial.Size += uint64(info.Size())
					partial.Count++
					startEvent.DeltaBytes += int64(info.Size())
					cleanupEvent.DeltaBytes -= int64(info.Size())
				} else {
					partial.Errors = append(partial.Errors, err.Error())
				}
				return nil
			})
		}
		for _, p := range filesPaths {
			util.Walk(p, func(tpath string, info os.FileInfo, err error) error {
				if err == nil {
					startEvent.DeltaBytes += int64(info.Size())
				} else {
					partial.Errors = append(partial.Errors, err.Error())
				}
				return nil
			})
		}
		if self.node.rt.Config.Debug {
			util.LogInfo("storage",
				"%d bytes of split files for %s",
				startEvent.DeltaBytes, self.node.GetFQName())
		}
		if startEvent.DeltaBytes != 0 {
			partial.Events = append(partial.Events, &startEvent)
			partial.Paths = append(partial.Paths, tempPaths...)
			// Critical section to avoid loosing accounting info.
			util.EnterCriticalSection()
			defer util.ExitCriticalSection()
		}
		if td := self.split_metadata.TempDir(); td != "" {
			if err := os.RemoveAll(self.split_metadata.TempDir()); err != nil {
				partial.Errors = append(partial.Errors, err.Error())
			}
			if cleanupEvent.DeltaBytes != 0 {
				cleanupEvent.Timestamp = time.Now()
				partial.Events = append(partial.Events, &cleanupEvent)
			}
		}

		self.writePartialKill(partial)
		return partial
	}
}

func (self *Fork) cleanChunkTemp(partial *PartialVdrKillReport) *PartialVdrKillReport {
	temps := make([]string, 0, len(self.chunks))
	files := make([]string, 0, len(self.chunks))
	var start time.Time
	for _, chunk := range self.chunks {
		if tempPaths, err := chunk.metadata.enumerateTemp(); err != nil {
			return partial
		} else if filesPaths, err := chunk.metadata.enumerateFiles(); err != nil {
			return partial
		} else {
			if ts := chunk.metadata.getStartTime(); ts.After(start) {
				start = ts
			}
			temps = append(temps, tempPaths...)
			files = append(files, filesPaths...)
		}
	}
	if partial == nil {
		partial = new(PartialVdrKillReport)
	}
	partial.Chunks = true
	var startEvent, cleanupEvent VdrEvent
	if start.IsZero() {
		if len(partial.Events) > 0 {
			startEvent.Timestamp = partial.Events[len(partial.Events)-1].Timestamp
		}
	} else {
		startEvent.Timestamp = start
	}
	for _, p := range temps {
		util.Walk(p, func(tpath string, info os.FileInfo, err error) error {
			if err == nil {
				partial.Size += uint64(info.Size())
				partial.Count++
				startEvent.DeltaBytes += int64(info.Size())
				cleanupEvent.DeltaBytes -= int64(info.Size())
			} else {
				partial.Errors = append(partial.Errors, err.Error())
			}
			return nil
		})
	}
	for _, p := range files {
		util.Walk(p, func(tpath string, info os.FileInfo, err error) error {
			if err == nil {
				startEvent.DeltaBytes += int64(info.Size())
			} else {
				partial.Errors = append(partial.Errors, err.Error())
			}
			return nil
		})
	}
	if self.node.rt.Config.Debug {
		util.LogInfo("storage",
			"%d bytes of chunk files for %s",
			startEvent.DeltaBytes, self.node.GetFQName())
	}
	if startEvent.DeltaBytes != 0 {
		partial.Events = append(partial.Events, &startEvent)
		partial.Paths = append(partial.Paths, temps...)
		// Critical section to avoid loosing accounting info.
		util.EnterCriticalSection()
		defer util.ExitCriticalSection()
	}

	for _, chunk := range self.chunks {
		if td := chunk.metadata.TempDir(); td != "" {
			if err := os.RemoveAll(td); err != nil {
				partial.Errors = append(partial.Errors, err.Error())
			}
		}
	}
	if cleanupEvent.DeltaBytes != 0 {
		cleanupEvent.Timestamp = time.Now()
		partial.Events = append(partial.Events, &cleanupEvent)
	}

	self.writePartialKill(partial)
	return partial
}

func (self *Fork) cleanJoinTemp(partial *PartialVdrKillReport) *PartialVdrKillReport {
	if tempPaths, err := self.join_metadata.enumerateTemp(); err != nil {
		return partial
	} else if filesPaths, err := self.join_metadata.enumerateFiles(); err != nil {
		return partial
	} else {
		if partial == nil {
			partial = new(PartialVdrKillReport)
		}
		partial.Join = true
		var startEvent, cleanupEvent VdrEvent
		if start := self.join_metadata.getStartTime(); start.IsZero() {
			if len(partial.Events) > 0 {
				startEvent.Timestamp = partial.Events[len(partial.Events)-1].Timestamp
			}
		} else {
			startEvent.Timestamp = start
		}

		for _, p := range tempPaths {
			util.Walk(p, func(tpath string, info os.FileInfo, err error) error {
				if err == nil {
					partial.Size += uint64(info.Size())
					partial.Count++
					startEvent.DeltaBytes += int64(info.Size())
					cleanupEvent.DeltaBytes -= int64(info.Size())
				} else {
					partial.Errors = append(partial.Errors, err.Error())
				}
				return nil
			})
		}
		for _, p := range filesPaths {
			util.Walk(p, func(tpath string, info os.FileInfo, err error) error {
				if err == nil {
					startEvent.DeltaBytes += int64(info.Size())
				} else {
					partial.Errors = append(partial.Errors, err.Error())
				}
				return nil
			})
		}
		if self.node.rt.Config.Debug {
			util.LogInfo("storage",
				"%d bytes of join files for %s",
				startEvent.DeltaBytes, self.node.GetFQName())
		}

		if startEvent.DeltaBytes != 0 {
			partial.Events = append(partial.Events, &startEvent)
			partial.Paths = append(partial.Paths, tempPaths...)
			// Critical section to avoid loosing accounting info.
			util.EnterCriticalSection()
			defer util.ExitCriticalSection()
		}
		if td := self.join_metadata.TempDir(); td != "" {
			if err := os.RemoveAll(td); err != nil {
				partial.Errors = append(partial.Errors, err.Error())
			}
			if cleanupEvent.DeltaBytes != 0 {
				cleanupEvent.Timestamp = time.Now()
				partial.Events = append(partial.Events, &cleanupEvent)
			}
		}

		self.writePartialKill(partial)

		return partial
	}
}

// Clean up all files (if volatile) or chunk files (otherwise).  Must be called
// through partialVdrKill in order to ensure accounting information is
// correctly preserved.
func (self *Fork) vdrKill(partialKill *PartialVdrKillReport) *VDRKillReport {
	if self.node.rt.Config.VdrMode == "disable" {
		return nil
	}
	if killReport, ok := self.getVdrKillReport(); ok {
		return killReport
	}

	var killPaths []string
	// For volatile nodes, kill fork-level files.
	if self.node.rt.overrides.GetOverride(self.node, "force_volatile", self.node.volatile).(bool) {
		if paths, err := self.split_metadata.enumerateFiles(); err == nil {
			killPaths = append(killPaths, paths...)
		}
		if paths, err := self.join_metadata.enumerateFiles(); err == nil {
			killPaths = append(killPaths, paths...)
		}
		for _, chunk := range self.chunks {
			if paths, err := chunk.metadata.enumerateFiles(); err == nil {
				killPaths = append(killPaths, paths...)
			}
		}
	} else if self.Split() && self.node.rt.overrides.GetOverride(self.node, "force_volatile", true).(bool) {
		// If the node splits, kill chunk-level files.
		// Must check for split here, otherwise we'll end up deleting
		// output files of non-volatile nodes because single-chunk nodes
		// get their output redirected to the one chunk's files path.
		for _, chunk := range self.chunks {
			if paths, err := chunk.metadata.enumerateFiles(); err == nil {
				killPaths = append(killPaths, paths...)
			}
		}
	}
	killReport := &VDRKillReport{
		Paths: make([]string, 0, len(killPaths)),
	}
	// Sum up the path size.
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
	}
	// Critical section to avoid loosing accounting info.
	util.EnterCriticalSection()
	defer util.ExitCriticalSection()
	// Actually delete the paths.
	for _, p := range killPaths {
		os.RemoveAll(p)
	}
	// update timestamp to mark actual kill time
	killReport.Timestamp = util.Timestamp()
	if killReport.Size > 0 {
		killReport.Events = append(killReport.Events, &VdrEvent{
			Timestamp:  time.Now().Round(time.Second),
			DeltaBytes: -int64(killReport.Size),
		})
	}
	if partialKill != nil {
		if self.node.rt.Config.Debug {
			util.LogInfo("storage",
				"VDR kill on %s with %d storage events in the partial report.",
				self.node.GetFQName(), len(partialKill.Events))
		}
		killReport = mergeVDRKillReports([]*VDRKillReport{killReport, &partialKill.VDRKillReport})
		self.deletePartialKill()
	} else {
		if self.node.rt.Config.Debug {
			util.LogInfo("storage",
				"VDR kill on %s with no partial report.",
				self.node.GetFQName())
		}
	}
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
		return nil, true
	} else if err != nil {
		util.LogError(err, "runtime", "Error reading node directory: %v", self.fqname)
		return nil, false
	}

	allDone := true
	killReports := make([]*VDRKillReport, 0, len(self.forks))
	for _, fork := range self.forks {
		if report, done := fork.partialVdrKill(); !done {
			allDone = false
		} else if report != nil {
			killReports = append(killReports, report)
		}
	}
	return mergeVDRKillReports(killReports), allDone
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

// Combine duplicate storage events.
func (self StorageEventByTimestamp) Collapse() StorageEventByTimestamp {
	if len(self) <= 1 {
		return self
	}
	sort.Sort(self)
	result := make(StorageEventByTimestamp, 1, len(self))
	result[0] = self[0]
	for _, e := range self[1:] {
		last := result[len(result)-1]
		if last.Name == e.Name && e.Timestamp.Sub(last.Timestamp) < time.Second {
			last.Delta += e.Delta
		} else {
			result = append(result, e)
		}
	}
	return result
}

// this is due to the fact that the VDR bytes/total bytes
// reported at the fork level is the sum of chunk + split
// + join plus any additional files.  The additional
// files that are unique to the fork cannot be resolved
// unless you sub out chunk/split/join and then child
// stages.
type ForkStorageEvent struct {
	Name         string
	ChildNames   []string
	TotalBytes   uint64
	ChunkBytes   uint64
	ForkBytes    uint64
	ForkVDRBytes uint64
	Timestamp    time.Time
	VDRTimestamp time.Time
}

func NewForkStorageEvent(timestamp time.Time, totalBytes uint64, vdrBytes uint64, fqname string) *ForkStorageEvent {
	self := &ForkStorageEvent{ChildNames: []string{}}
	self.Name = fqname
	self.TotalBytes = totalBytes // sum total of bytes in fork and children
	self.ForkBytes = self.TotalBytes
	self.ForkVDRBytes = vdrBytes // VDR bytes in forkN/files
	self.Timestamp = timestamp
	return self
}

func (self *Pipestance) VDRKill() *VDRKillReport {
	var killReports []*VDRKillReport
	if nodes := self.node.allNodes(); len(nodes) > 0 {
		killReports = make([]*VDRKillReport, 0, len(nodes))
		for _, node := range self.node.allNodes() {
			if killReport, _ := node.vdrKill(); killReport != nil {
				killReports = append(killReports, killReport)
			}
		}
	}
	killReport := mergeVDRKillReports(killReports)
	self.metadata.Write(VdrKill, killReport)
	return killReport
}

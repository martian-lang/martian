//
// Copyright (c) 201 10X Genomics, Inc. All rights reserved.
//
// Converts system rusage into our structures.
//

package core

import (
	"fmt"
	"io/ioutil"
	"martian/util"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func getRusage(who int) *Rusage {
	var ru syscall.Rusage
	if err := syscall.Getrusage(who, &ru); err == nil {
		return &Rusage{
			MaxRss:       int(ru.Maxrss),
			SharedRss:    int(ru.Ixrss),
			UnsharedRss:  int(ru.Idrss),
			MinorFaults:  int(ru.Minflt),
			MajorFaults:  int(ru.Majflt),
			SwapOuts:     int(ru.Nswap),
			UserTime:     time.Duration(ru.Utime.Nano()).Seconds(),
			SystemTime:   time.Duration(ru.Stime.Nano()).Seconds(),
			InBlocks:     int(ru.Inblock),
			OutBlocks:    int(ru.Oublock),
			MessagesSent: int(ru.Msgsnd),
			MessagesRcvd: int(ru.Msgrcv),
			SignalsRcvd:  int(ru.Nsignals),
		}
	} else {
		return nil
	}
}

func GetRusage() *RusageInfo {
	ru := RusageInfo{
		Self:     getRusage(syscall.RUSAGE_SELF),
		Children: getRusage(syscall.RUSAGE_CHILDREN),
	}
	if ru.Self == nil && ru.Children == nil {
		return nil
	}
	return &ru
}

// Gets the total memory usage for the given process and all of its
// children.  Only errors getting the first process's memory, or the
// set of children for that process, are reported.  includeParent specifies
// whether the top-level pid is included in the total.
func GetProcessTreeMemory(pid int, includeParent bool) (mem ObservedMemory, err error) {
	if includeParent {
		if mem, err = GetRunningMemory(pid); err != nil {
			return mem, err
		}
	} else {
		mem = ObservedMemory{}
	}
	if threads, err := util.Readdirnames(fmt.Sprintf("/proc/%d/task", pid)); err != nil {
		return mem, err
	} else {
		for _, tid := range threads {
			if childrenBytes, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/task/%s/children", pid, tid)); err == nil {
				for _, child := range strings.Fields(string(childrenBytes)) {
					if childPid, err := strconv.Atoi(child); err == nil {
						cmem, _ := GetProcessTreeMemory(childPid, true)
						mem.Add(cmem)
					}
				}
			}
		}
		mem.Procs += len(threads)
	}
	return mem, nil
}

func (self *ObservedMemory) pagesToBytes() {
	pagesize := int64(syscall.Getpagesize())
	self.Rss *= pagesize
	self.Vmem *= pagesize
	self.Shared *= pagesize
	self.Text *= pagesize
	self.Stack *= pagesize
}

// Gets the total vmem and rss memory of a running process by pid.
func GetRunningMemory(pid int) (ObservedMemory, error) {
	if bytes, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/statm", pid)); err != nil {
		return ObservedMemory{}, err
	} else {
		fields := strings.Fields(string(bytes))
		if len(fields) < 3 {
			return ObservedMemory{}, fmt.Errorf(
				"statm: unexpected result %s", string(bytes))
		}
		mem := ObservedMemory{}
		if vmem, err := strconv.Atoi(fields[0]); err != nil {
			return mem, err
		} else {
			mem.Vmem = int64(vmem)
		}
		if rss, err := strconv.Atoi(fields[1]); err != nil {
			return mem, err
		} else {
			mem.Rss = int64(rss)
		}
		if shared, err := strconv.Atoi(fields[2]); err != nil {
			return mem, err
		} else {
			mem.Shared = int64(shared)
		}
		if text, err := strconv.Atoi(fields[3]); err != nil {
			return mem, err
		} else {
			mem.Text = int64(text)
		}
		if stack, err := strconv.Atoi(fields[5]); err != nil {
			return mem, err
		} else {
			mem.Stack = int64(stack)
		}
		mem.pagesToBytes()
		return mem, nil
	}
}

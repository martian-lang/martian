//
// Copyright (c) 201 10X Genomics, Inc. All rights reserved.
//
// Converts system rusage into our structures.
//

package core

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/martian-lang/martian/martian/util"
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

// Get the number of processes (threads) currently running for the current
// user.
func GetUserProcessCount() (int, error) {
	if pids, err := ioutil.ReadDir("/proc"); err != nil {
		return 0, err
	} else {
		uid := uint32(os.Getuid())
		count := 0
		for _, pid := range pids {
			if ufinfo, ok := pid.Sys().(*syscall.Stat_t); !ok {
				return count, fmt.Errorf("Unexpected Stat_t type")
			} else if ufinfo.Uid == uid {
				if _, err := strconv.Atoi(pid.Name()); err == nil {
					if threads, err := util.Readdirnames(path.Join(
						"/proc", pid.Name(), "task")); err == nil {
						count += len(threads)
					}
				}
			}
		}
		return count, nil
	}
}

// Gets the total memory usage for the given process and all of its
// children.  Only errors getting the first process's memory, or the
// set of children for that process, are reported.  includeParent specifies
// whether the top-level pid is included in the total.
func GetProcessTreeMemory(pid int, includeParent bool, io map[int]*IoAmount) (mem ObservedMemory, err error) {
	if includeParent {
		if mem, err = GetRunningMemory(pid); err != nil {
			return mem, err
		}
		if io != nil {
			if ioV, err := GetRunningIo(pid); err == nil {
				io[pid] = ioV
			} else {
				fmt.Println("Error fetching io for", pid, err)
			}
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
						cmem, _ := GetProcessTreeMemory(childPid, true, io)
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

var (
	newline     = []byte{'\n'}
	syscr       = []byte("syscr: ")
	syscw       = []byte("syscw: ")
	read_bytes  = []byte("read_bytes: ")
	write_bytes = []byte("write_bytes: ")
)

// Gets IO statistics for a running process by pid.
func GetRunningIo(pid int) (*IoAmount, error) {
	if b, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/io", pid)); err != nil {
		return nil, err
	} else {
		lines := bytes.Split(b, newline)
		if len(lines) < 6 {
			return nil, fmt.Errorf(
				"io: unexpected result %q", b)
		}
		readTo := func(line, prefix []byte, target *int64) (bool, error) {
			if bytes.HasPrefix(line, prefix) {
				if v, err := strconv.ParseInt(string(bytes.TrimSpace(line[len(prefix):])), 10, 64); err != nil {
					return true, err
				} else {
					*target = v
					return true, nil
				}
			} else {
				return false, nil
			}
		}
		var result IoAmount
		for _, line := range lines {
			if found, err := readTo(line, syscr, &result.Read.Syscalls); found {
				continue
			} else if err != nil {
				return nil, err
			} else if found, err := readTo(line, syscw, &result.Write.Syscalls); found {
				continue
			} else if err != nil {
				return nil, err
			} else if found, err := readTo(line, read_bytes, &result.Read.BlockBytes); found {
				continue
			} else if err != nil {
				return nil, err
			} else if found, err := readTo(line, write_bytes, &result.Read.BlockBytes); found {
				continue
			} else if err != nil {
				return nil, err
			}
		}
		return &result, nil
	}
}

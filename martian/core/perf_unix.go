//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Converts system rusage into our structures.
//
//go:build freebsd || linux || netbsd || openbsd || solaris
// +build freebsd linux netbsd openbsd solaris

package core

import (
	"bytes"
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/martian-lang/martian/martian/util"
	"golang.org/x/sys/unix"
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
			CtxSwitches:  int(ru.Nivcsw),
		}
	} else {
		return nil
	}
}

func (r *Rusage) Add(ru *syscall.Rusage) {
	if rss := int(ru.Maxrss); rss > r.MaxRss {
		r.MaxRss = rss
	}
	if rss := int(ru.Ixrss); rss > r.SharedRss {
		r.SharedRss = rss
	}
	if rss := int(ru.Idrss); rss > r.UnsharedRss {
		r.UnsharedRss = rss
	}
	r.MinorFaults += int(ru.Minflt)
	r.MajorFaults += int(ru.Majflt)
	r.SwapOuts += int(ru.Nswap)
	r.UserTime += time.Duration(ru.Utime.Nano()).Seconds()
	r.SystemTime += time.Duration(ru.Stime.Nano()).Seconds()
	r.InBlocks += int(ru.Inblock)
	r.OutBlocks += int(ru.Oublock)
	r.MessagesSent += int(ru.Msgsnd)
	r.MessagesRcvd += int(ru.Msgrcv)
	r.SignalsRcvd += int(ru.Nsignals)
	r.CtxSwitches += int(ru.Nivcsw)
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
	if proc, err := os.Open("/proc"); err != nil {
		return 0, err
	} else {
		defer proc.Close()
		if pids, err := proc.Readdirnames(-1); err != nil {
			return 0, err
		} else {
			uid := uint32(os.Getuid())
			count := 0
			for _, pid := range pids {
				if _, err := strconv.Atoi(pid); err == nil {
					count += countTasks(int(proc.Fd()), pid, uid)
				}
			}
			return count, nil
		}
	}
}

func countTasks(procFd int, pid string, uid uint32) int {
	if pfd, err := openAt(procFd, pid+"/task",
		os.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW,
		0); err != nil {
		return 0
	} else {
		defer closeHandle(pfd)
		var ufinfo unix.Stat_t
		if err := fstat(pfd, &ufinfo); err != nil ||
			ufinfo.Mode&unix.S_IFDIR == 0 ||
			ufinfo.Uid != uid {
			return 0
		} else {
			c, _ := util.CountDirNames(pfd)
			return c
		}
	}
}

// Gets the total memory usage for the given process and all of its
// children.  Only errors getting the first process's memory, or the
// set of children for that process, are reported.  includeParent specifies
// whether the top-level pid is included in the total.
func GetProcessTreeMemory(pid int, includeParent bool, io map[int]*IoAmount) (mem ObservedMemory, err error) {
	var procFd int
	if procFd, err = openAt(unix.AT_FDCWD, "/proc",
		os.O_RDONLY|unix.O_DIRECTORY|unix.O_PATH,
		0); err != nil {
		return mem, err
	}
	defer closeHandle(procFd)
	if includeParent {
		var buf bytes.Buffer
		return getProcessTreeMemoryAt(procFd, pid, strconv.Itoa(pid), io, &buf)
	} else {
		if tfd, err := openDirForReadAt(procFd, strconv.Itoa(pid)+"/task"); err != nil {
			return mem, err
		} else {
			var buf bytes.Buffer
			err = getChildTreeMemory(procFd, tfd, &mem, io, &buf)
			return mem, err
		}
	}
}

// GetProcessTreeMemoryList returns the memory usage and other stats for all
// processes in the tree rooted with pid.
func GetProcessTreeMemoryList(pid int) (ProcessTree, error) {
	procFd, err := openAt(unix.AT_FDCWD, "/proc",
		os.O_RDONLY|unix.O_DIRECTORY|unix.O_PATH,
		0)
	if err != nil {
		return nil, err
	}
	defer closeHandle(procFd)
	var stats ProcessTree
	var buf bytes.Buffer
	return stats, getProcessTreeMemoryList(procFd, pid,
		strconv.Itoa(pid), 0, &stats, &buf)
}

func openDirAsPathAt(fd int, path string) (int, error) {
	return openAt(fd, path,
		os.O_RDONLY|unix.O_DIRECTORY|unix.O_PATH|unix.O_NOFOLLOW,
		0)
}

func openDirForReadAt(fd int, path string) (int, error) {
	return openAt(fd, path,
		os.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW,
		0)
}

func getProcessTreeMemoryAt(procFd, pid int, spid string,
	io map[int]*IoAmount, buf *bytes.Buffer) (mem ObservedMemory, err error) {
	var fd int
	if fd, err = openDirAsPathAt(procFd, spid); err != nil {
		return mem, err
	}
	closed := false
	defer func() {
		if !closed {
			closeHandle(fd)
		}
	}()
	if mem, err = getRunningMemoryAt(fd, buf); err != nil {
		return mem, err
	}
	if io != nil {
		if ioV, err := getRunningIoAt(fd, buf); err == nil {
			io[pid] = &ioV
		} else {
			errString := err.Error()
			buf.Reset()
			buf.Grow(
				len("Warning: problem fetching io for : ; final statistics may be incomplete\n") +
					len(spid) + len(errString))
			if _, err := buf.WriteString("Warning: problem fetching io for "); err != nil {
				panic(err)
			}
			if _, err := buf.WriteString(spid); err != nil {
				panic(err)
			}
			if _, err := buf.WriteString(": "); err != nil {
				panic(err)
			}
			if _, err := buf.WriteString(errString); err != nil {
				panic(err)
			}
			if _, err := buf.WriteString(
				"; final statistics may be incomplete\n"); err != nil {
				panic(err)
			}
			if _, err := buf.WriteTo(os.Stdout); err != nil {
				util.LogError(err, "runtime",
					"Error writing to standard output.")
			}
		}
	}
	if tfd, err := openDirForReadAt(fd, "task"); err != nil {
		return mem, err
	} else {
		closed = true
		closeHandle(fd)
		err = getChildTreeMemory(procFd, tfd, &mem, io, buf)
		return mem, err
	}
}

func getChildTreeMemory(procFd, tfd int, mem *ObservedMemory, io map[int]*IoAmount, buf *bytes.Buffer) error {
	f := os.NewFile(uintptr(tfd), "task")
	defer f.Close()
	if threads, err := f.Readdirnames(-1); err != nil {
		return err
	} else {
		mem.Procs += len(threads)
		for _, tid := range threads {
			buf.Reset()
			if err := util.ReadFileInto(tfd, tid+"/children", buf); err == nil {
				for _, child := range strings.Fields(string(bytes.TrimSpace(buf.Bytes()))) {
					if ichild, err := strconv.Atoi(child); err == nil {
						cmem, _ := getProcessTreeMemoryAt(procFd, ichild, child, io, buf)
						mem.Add(cmem)
					}
				}
			}
		}
		return nil
	}
}

func getProcessTreeMemoryList(procFd int, pid int, spid string, depth int, stats *ProcessTree,
	buf *bytes.Buffer) error {
	fd, err := openDirAsPathAt(procFd, spid)
	if err != nil {
		return errors.New("error opening process directory: " + err.Error())
	}
	closed := false
	defer func() {
		if !closed {
			closeHandle(fd)
		}
	}()
	mem, err := getRunningMemoryAt(fd, buf)
	if err != nil {
		return err
	}
	stat := ProcessStats{
		Pid:    pid,
		Memory: mem,
		Depth:  depth,
	}
	stat.IO, _ = getRunningIoAt(fd, buf)
	stat.Cmdline = getRunningCommandLine(fd, buf)
	*stats = append(*stats, stat)
	if tfd, err := openDirForReadAt(fd, "task"); err != nil {
		return errors.New("error opening tid directory: " + err.Error())
	} else {
		closed = true
		closeHandle(fd)
		return getChildTreeMemoryList(procFd, tfd, depth+1, stats, buf)
	}
}

func getChildTreeMemoryList(procFd, tfd, depth int, stats *ProcessTree, buf *bytes.Buffer) error {
	f := os.NewFile(uintptr(tfd), "task")
	defer f.Close()
	if threads, err := f.Readdirnames(-1); err != nil {
		return errors.New("error reading tid list: " + err.Error())
	} else {
		(*stats)[len(*stats)-1].Memory.Procs = len(threads)
		for _, tid := range threads {
			buf.Reset()
			if err := util.ReadFileInto(tfd, tid+"/children", buf); err == nil {
				for _, child := range strings.Fields(string(bytes.TrimSpace(buf.Bytes()))) {
					if ichild, err := strconv.Atoi(child); err == nil {
						// Ignore errors.  Usually the issue is that the child
						// has already died.
						_ = getProcessTreeMemoryList(procFd, ichild, child, depth, stats, buf)
					}
				}
			}
		}
		return nil
	}
}

func getRunningCommandLine(fd int, buf *bytes.Buffer) []string {
	buf.Reset()
	if err := util.ReadFileInto(fd, "cmdline", buf); err != nil {
		return nil
	} else {
		args := bytes.Split(buf.Bytes(), []byte{0})
		result := make([]string, len(args))
		for i, a := range args {
			result[i] = string(a)
		}
		return result
	}
}

var sysPagesize = int64(syscall.Getpagesize())

func (self *ObservedMemory) pagesToBytes() {
	self.Rss *= sysPagesize
	self.Vmem *= sysPagesize
	self.Shared *= sysPagesize
	self.Text *= sysPagesize
	self.Stack *= sysPagesize
}

type unexpectedContentError struct {
	fn      string
	content string
}

func (err *unexpectedContentError) Error() string {
	var buf strings.Builder
	buf.Grow(len(": unexpected result:\n") + len(err.fn) + len(err.content))
	buf.WriteString(err.fn)
	buf.WriteString(": unexpected result:\n")
	buf.WriteString(err.content)
	return buf.String()
}

// Gets the total vmem and rss memory of a running process by pid.
func GetRunningMemory(pid int) (ObservedMemory, error) {
	if b, err := os.ReadFile(statmFileName(pid)); err != nil {
		return ObservedMemory{}, err
	} else {
		return parseRunningMemory(b)
	}
}

func getRunningMemoryAt(fd int, buf *bytes.Buffer) (ObservedMemory, error) {
	buf.Reset()
	if err := util.ReadFileInto(fd, "statm", buf); err != nil {
		return ObservedMemory{}, err
	} else {
		return parseRunningMemory(buf.Bytes())
	}
}

func parseRunningMemory(b []byte) (ObservedMemory, error) {
	fields := bytes.Fields(b)
	if len(fields) < 3 {
		return ObservedMemory{}, &unexpectedContentError{
			fn:      "statm",
			content: string(b),
		}
	}
	mem := ObservedMemory{}
	if vmem, err := util.Atoi(fields[0]); err != nil {
		return mem, err
	} else {
		mem.Vmem = vmem
	}
	if rss, err := util.Atoi(fields[1]); err != nil {
		return mem, err
	} else {
		mem.Rss = rss
	}
	if shared, err := util.Atoi(fields[2]); err != nil {
		return mem, err
	} else {
		mem.Shared = shared
	}
	if text, err := util.Atoi(fields[3]); err != nil {
		return mem, err
	} else {
		mem.Text = text
	}
	if stack, err := util.Atoi(fields[5]); err != nil {
		return mem, err
	} else {
		mem.Stack = stack
	}
	mem.pagesToBytes()
	return mem, nil
}

var (
	newline     = []byte{'\n'}
	syscr       = []byte("syscr: ")
	syscw       = []byte("syscw: ")
	read_bytes  = []byte("read_bytes: ")
	write_bytes = []byte("write_bytes: ")
)

func ioFileName(pid int) string {
	var buf strings.Builder
	buf.Grow(len("/proc//io") + 10)
	buf.WriteString("/proc/")
	buf.WriteString(strconv.Itoa(pid))
	buf.WriteString("/io")
	return buf.String()
}

func statmFileName(pid int) string {
	var buf strings.Builder
	buf.Grow(len("/proc//statm") + 10)
	buf.WriteString("/proc/")
	buf.WriteString(strconv.Itoa(pid))
	buf.WriteString("/statm")
	return buf.String()
}

func readTo(line, prefix []byte, target *int64) (bool, error) {
	if bytes.HasPrefix(line, prefix) {
		if v, err := util.Atoi(bytes.TrimSpace(line[len(prefix):])); err != nil {
			return true, err
		} else {
			*target = v
			return true, nil
		}
	} else {
		return false, nil
	}
}

// Gets IO statistics for a running process by pid.
func GetRunningIo(pid int) (*IoAmount, error) {
	if b, err := os.ReadFile(ioFileName(pid)); err != nil {
		return nil, err
	} else {
		r, err := parseRunningIo(b)
		return &r, err
	}
}

// Gets IO statistics for a running process by proc fd.
func getRunningIoAt(fd int, buf *bytes.Buffer) (IoAmount, error) {
	buf.Reset()
	if err := util.ReadFileInto(fd, "io", buf); err != nil {
		return IoAmount{}, err
	} else {
		return parseRunningIo(buf.Bytes())
	}
}

func parseRunningIo(b []byte) (IoAmount, error) {
	var result IoAmount
	lines := bytes.Split(b, newline)
	if len(lines) < 6 {
		return result, &unexpectedContentError{
			fn:      "io",
			content: string(b),
		}
	}
	for _, line := range lines {
		if found, err := readTo(line, syscr, &result.Read.Syscalls); found {
			continue
		} else if err != nil {
			return result, err
		} else if found, err := readTo(line, syscw, &result.Write.Syscalls); found {
			continue
		} else if err != nil {
			return result, err
		} else if found, err := readTo(line, read_bytes, &result.Read.BlockBytes); found {
			continue
		} else if err != nil {
			return result, err
		} else if found, err := readTo(line, write_bytes, &result.Read.BlockBytes); found {
			continue
		} else if err != nil {
			return result, err
		}
	}
	return result, nil
}

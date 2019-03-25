//
// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.
//
// Converts system rusage into our structures.
//
// +build freebsd linux netbsd openbsd solaris

package core

import (
	"bytes"
	"errors"
	"io/ioutil"
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
					if c, err := countTasks(int(proc.Fd()), pid, uid); err != nil {
						return count, err
					} else {
						count += c
					}
				}
			}
			return count, nil
		}
	}
}

func countTasks(procFd int, pid string, uid uint32) (int, error) {
	if pfd, err := unix.Openat(procFd, pid+"/task",
		os.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC,
		0); err != nil {
		return 0, nil
	} else {
		defer unix.Close(pfd)
		var ufinfo unix.Stat_t
		if err := unix.Fstat(pfd, &ufinfo); err != nil ||
			ufinfo.Mode&unix.S_IFDIR == 0 ||
			ufinfo.Uid != uid {
			return 0, nil
		} else {
			c, _ := util.CountDirNames(pfd)
			return c, nil
		}
	}
}

// itob slices from here instead of just a string of digits in order to
// reduce the number of divide/mod ops and also allow use of simd copying.
const smallsString = "00010203040506070809" +
	"10111213141516171819" +
	"20212223242526272829" +
	"30313233343536373839" +
	"40414243444546474849" +
	"50515253545556575859" +
	"60616263646566676869" +
	"70717273747576777879" +
	"80818283848586878889" +
	"90919293949596979899"

func itob(buf *[10]byte, us uint32) []byte {
	i := 10
	for us >= 100 {
		is := us % 100 * 2
		us /= 100
		i -= 2
		buf[i+1] = smallsString[is+1]
		buf[i+0] = smallsString[is+0]
	}
	is := us * 2
	i--
	buf[i] = smallsString[is+1]
	if us >= 10 {
		i--
		buf[i] = smallsString[is]
	}
	return buf[i:]
}

// Gets the total memory usage for the given process and all of its
// children.  Only errors getting the first process's memory, or the
// set of children for that process, are reported.  includeParent specifies
// whether the top-level pid is included in the total.
func GetProcessTreeMemory(pid int, includeParent bool, io map[int]*IoAmount) (mem ObservedMemory, err error) {
	var procFd int
	if procFd, err = unix.Openat(unix.AT_FDCWD, "/proc",
		os.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_PATH,
		0); err != nil {
		return mem, err
	}
	defer unix.Close(procFd)
	if includeParent {
		var buf [10]byte
		return getProcessTreeMemoryAt(procFd, pid, itob(&buf, uint32(pid)), io)
	} else {
		if tfd, err := openDirForReadAt(procFd, strconv.Itoa(pid)+"/task"); err != nil {
			return mem, err
		} else {
			err = getChildTreeMemory(procFd, tfd, &mem, io)
			return mem, err
		}
	}
}

// GetProcessTreeMemoryList returns the memory usage and other stats for all
// processes in the tree rooted with pid.
func GetProcessTreeMemoryList(pid int) (ProcessTree, error) {
	procFd, err := unix.Openat(unix.AT_FDCWD, "/proc",
		os.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_PATH,
		0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(procFd)
	var stats ProcessTree
	var buf [10]byte
	return stats, getProcessTreeMemoryList(procFd, pid,
		itob(&buf, uint32(pid)), 0, &stats)
}

func openDirAsPathAt(fd int, path string) (int, error) {
	return unix.Openat(fd, path,
		os.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_PATH|unix.O_NOFOLLOW,
		0)
}

func openDirForReadAt(fd int, path string) (int, error) {
	return unix.Openat(fd, path,
		os.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0)
}

func getProcessTreeMemoryAt(procFd int, pid int, spid []byte, io map[int]*IoAmount) (mem ObservedMemory, err error) {
	var fd int
	var pidBuf strings.Builder
	pidBuf.Grow(11)
	pidBuf.Write(spid)
	if fd, err = openDirAsPathAt(procFd, pidBuf.String()); err != nil {
		return mem, err
	}
	closed := false
	defer func() {
		if !closed {
			unix.Close(fd)
		}
	}()
	if mem, err = getRunningMemoryAt(fd); err != nil {
		return mem, err
	}
	if io != nil {
		if ioV, err := getRunningIoAt(fd); err == nil {
			io[pid] = &ioV
		} else {
			errString := err.Error()
			var buf bytes.Buffer
			buf.Grow(len("Error fetching io for : ") + len(spid) + len(errString))
			buf.WriteString("Error fetching io for ")
			buf.Write(spid)
			buf.WriteString(": ")
			buf.WriteString(errString)
			buf.WriteTo(os.Stdout)
		}
	}
	if tfd, err := openDirForReadAt(fd, "task"); err != nil {
		return mem, err
	} else {
		closed = true
		unix.Close(fd)
		err = getChildTreeMemory(procFd, tfd, &mem, io)
		return mem, err
	}
}

func getChildTreeMemory(procFd, tfd int, mem *ObservedMemory, io map[int]*IoAmount) error {
	f := os.NewFile(uintptr(tfd), "task")
	defer f.Close()
	if threads, err := f.Readdirnames(-1); err != nil {
		return err
	} else {
		mem.Procs += len(threads)
		for _, tid := range threads {
			if childrenBytes, err := util.ReadFileAt(tfd, tid+"/children"); err == nil {
				for _, child := range bytes.Fields(childrenBytes) {
					if ichild, err := util.Atoi(child); err == nil {
						cmem, _ := getProcessTreeMemoryAt(procFd, int(ichild), child, io)
						mem.Add(cmem)
					}
				}
			}
		}
		return nil
	}
}

func getProcessTreeMemoryList(procFd int, pid int, spid []byte, depth int, stats *ProcessTree) error {
	var pidBuf strings.Builder
	pidBuf.Grow(11)
	pidBuf.Write(spid)
	fd, err := openDirAsPathAt(procFd, pidBuf.String())
	if err != nil {
		return errors.New("error opening process directory: " + err.Error())
	}
	closed := false
	defer func() {
		if !closed {
			unix.Close(fd)
		}
	}()
	mem, err := getRunningMemoryAt(fd)
	if err != nil {
		return err
	}
	stat := ProcessStats{
		Pid:    pid,
		Memory: mem,
		Depth:  depth,
	}
	stat.IO, _ = getRunningIoAt(fd)
	stat.Cmdline = getRunningCommandLine(fd)
	*stats = append(*stats, stat)
	if tfd, err := openDirForReadAt(fd, "task"); err != nil {
		return errors.New("error opening tid directory: " + err.Error())
	} else {
		closed = true
		unix.Close(fd)
		return getChildTreeMemoryList(procFd, tfd, depth+1, stats)
	}
}

func getChildTreeMemoryList(procFd, tfd, depth int, stats *ProcessTree) error {
	f := os.NewFile(uintptr(tfd), "task")
	defer f.Close()
	if threads, err := f.Readdirnames(-1); err != nil {
		return errors.New("error reading tid list: " + err.Error())
	} else {
		(*stats)[len(*stats)-1].Memory.Procs = len(threads)
		for _, tid := range threads {
			if childrenBytes, err := util.ReadFileAt(tfd, tid+"/children"); err == nil {
				for _, child := range bytes.Fields(childrenBytes) {
					if ichild, err := util.Atoi(child); err == nil {
						getProcessTreeMemoryList(procFd, int(ichild), child, depth, stats)
					}
				}
			}
		}
		return nil
	}
}

func getRunningCommandLine(fd int) []string {
	if b, err := util.ReadFileAt(fd, "cmdline"); err != nil {
		return nil
	} else {
		args := bytes.Split(b, []byte{0})
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
	if b, err := ioutil.ReadFile(statmFileName(pid)); err != nil {
		return ObservedMemory{}, err
	} else {
		return parseRunningMemory(b)
	}
}

func getRunningMemoryAt(fd int) (ObservedMemory, error) {
	if b, err := util.ReadFileAt(fd, "statm"); err != nil {
		return ObservedMemory{}, err
	} else {
		return parseRunningMemory(b)
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
	if b, err := ioutil.ReadFile(ioFileName(pid)); err != nil {
		return nil, err
	} else {
		r, err := parseRunningIo(b)
		return &r, err
	}
}

// Gets IO statistics for a running process by proc fd.
func getRunningIoAt(fd int) (IoAmount, error) {
	if b, err := util.ReadFileAt(fd, "io"); err != nil {
		return IoAmount{}, err
	} else {
		return parseRunningIo(b)
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

// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

//go:build !windows
// +build !windows

package main

import (
	"bytes"
	"errors"
	"io/fs"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/martian-lang/martian/martian/util"
	"golang.org/x/sys/unix"
)

// Force the given file to sync.
func syncFile(filename string) {
	if fd, err := syscall.Open(filename, syscall.O_RDONLY, 0); err == nil {
		if err := syscall.Fsync(fd); err != nil {
			util.LogError(err, "mrjob",
				"Error syncing file descriptor for %s", filename)
		}
		syscall.Close(fd)
	}
}

// Log any child processes which are still up.
//
// Returns true if any child processes were found and reported.
func reportChildren() bool {
	myPid := os.Getpid()
	me := strconv.Itoa(myPid)
	procs, err := util.Readdirnames("/proc")
	if err != nil {
		util.LogError(err, "monitor", "Error getting list of processes.")
		return false
	}
	// Reusable buffer for generating paths.  We know this is big enough to hold
	// any PID path, and there may be a lot of processes on the system so
	// allocating for every path could get expensive, and we want this loop to
	// be fast and efficient.
	// Fixed-size buffer allows stack allocation and is sufficient for any
	// PID allowed in 64-bit linux, plus a null terminator.  See
	// https://github.com/torvalds/linux/blob/2d338201/include/linux/threads.h#L32
	const bufSize = len("/proc/1073741824/stat") + 1
	pathBuf := append(make([]byte, 0, bufSize), "/proc/"...)
	var found bool
	for _, proc := range procs {
		if len(proc) == 0 {
			// Shouldn't happen, but just to be safe.
			continue
		}
		// Don't look at directories which aren't PIDs.  Early out to avoid
		// allocation overhead in trying to parse as a number.
		if c := proc[0]; c < '0' || c > '9' {
			continue
		}
		pid, err := strconv.Atoi(proc)
		// Don't look at non-numeric names, as they're either `self` or one of
		// the non-PID entries in /proc, like `sys` or `cpuinfo`.  Also don't
		// look at our own PID.
		if err != nil || pid == 0 || pid == myPid {
			continue
		}
		procPath := append(pathBuf, proc...)
		statBytes, err := ioutil.ReadFile(string(append(procPath, "/stat"...)))
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) &&
				!errors.Is(err, fs.ErrPermission) {
				util.LogInfo("monitor",
					"Orphaned child process %s (unknown: %v).", proc, err)
			}
			continue
		}
		fields := bytes.Fields(statBytes)
		if len(fields) < 4 {
			util.LogInfo("monitor",
				"Orphaned child process %s (unknown: stat %q).", proc, statBytes)
			continue
		}

		if string(fields[3]) == me {
			b, err := ioutil.ReadFile(string(append(procPath, "/cmdline"...)))
			var cmdLine []byte
			if err == nil && len(b) > 0 {
				null := []byte{0}
				cmdLine = bytes.ReplaceAll(bytes.TrimSuffix(b, null), null, []byte{' '})
			} else if len(fields[1]) > 2 {
				cmdLine = fields[1][1 : len(fields[1])-1]
			}
			util.LogInfo("monitor",
				"Orphaned child process %s (%s) is still running (state %s).",
				proc,
				string(cmdLine),
				string(fields[2]))
			found = true
		}
	}
	return found
}

// Wait for any orphaned grandchildren, to collect their rusage.
//
// Returns true if there are any child processes which have not terminated.
func waitChildren() bool {
	var ws syscall.WaitStatus
	wpid, err := syscall.Wait4(-1, &ws, unix.WNOHANG, nil)
	start := time.Now()
	for err == nil && wpid > 0 {
		if ws.Exited() {
			util.LogInfo("monitor",
				"orphaned child process %d terminated with status %d",
				wpid, ws.ExitStatus())
		} else if ws.Signaled() {
			util.LogInfo("monitor",
				"orphaned child process %d got signal %v",
				wpid, ws.Signal())
		} else {
			// This branch should never actually be taken, because
			// wait4 should only return a PID if a process actually exited
			// (which may involve a signal), since we aren't setting any of
			// WUNTRACED, WCONTINUED, or WSTOPPED.  However, just in case,
			// make sure we don't keep spinning forever.
			if time.Since(start) > time.Second {
				break
			} else {
				time.Sleep(time.Millisecond)
			}
		}
		wpid, err = syscall.Wait4(-1, &ws, syscall.WNOHANG, nil)
	}
	return err == nil
}

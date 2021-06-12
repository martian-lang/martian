// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package main

import (
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

// Wait for any orphaned grandchildren, to collect their rusage.
func waitChildren() {
	var ws syscall.WaitStatus
	wpid, err := syscall.Wait4(-1, &ws, unix.WNOHANG, nil)
	start := time.Now()
	for err == nil && wpid > 0 {
		if ws.Exited() {
			util.LogInfo("monitor",
				"orphaned child process %d terminated with status %d",
				wpid, ws.ExitStatus())
		} else {
			if ws.Signaled() {
				util.LogInfo("monitor",
					"orphaned child process %d got signal %v",
					wpid, ws.Signal())
			}
			if time.Since(start) > time.Second {
				// Don't keep waiting around forever.
				return
			}
		}
		wpid, err = syscall.Wait4(-1, &ws, syscall.WNOHANG, nil)
	}
}

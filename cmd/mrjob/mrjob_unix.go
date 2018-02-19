// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package main

import (
	"github.com/martian-lang/martian/martian/util"
	"syscall"
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

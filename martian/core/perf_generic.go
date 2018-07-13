//
// Copyright (c) 201 10X Genomics, Inc. All rights reserved.
//
// Converts system rusage into our structures.
//
// +build !freebsd,!linux,!netbsd,!openbsd,!solaris

package core

import (
	"bytes"
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
	return nil
}

func GetRusage() *RusageInfo {
	return nil
}

// Get the number of processes (threads) currently running for the current
// user.
func GetUserProcessCount() (int, error) {
	return 0, nil
}

// Gets the total memory usage for the given process and all of its
// children.  Only errors getting the first process's memory, or the
// set of children for that process, are reported.  includeParent specifies
// whether the top-level pid is included in the total.
func GetProcessTreeMemory(pid int, includeParent bool, io map[int]*IoAmount) (mem ObservedMemory, err error) {
	return ObservedMemory{}, nil
}

// Gets the total vmem and rss memory of a running process by pid.
func GetRunningMemory(pid int) (ObservedMemory, error) {
	return ObservedMemory{}, nil
}

// Gets IO statistics for a running process by pid.
func GetRunningIo(pid int) (*IoAmount, error) {
	return nil, err
}

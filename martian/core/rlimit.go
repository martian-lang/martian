// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

//go:build !windows
// +build !windows

// Get and set rlimit

package core

import (
	"fmt"

	"github.com/martian-lang/martian/martian/util"
	// syscall package lacks RLIMIT_NPROC, so we use sys/unix instead.
	"golang.org/x/sys/unix"
)

// Gets the current (soft) and maximum (hard) rlimit for number of processes.
//
// See `man getrlimit`.
func GetMaxProcs() (*unix.Rlimit, error) {
	var rlim unix.Rlimit
	return &rlim, unix.Getrlimit(unix.RLIMIT_NPROC, &rlim)
}

// Sets the current (soft) and maximum (hard) rlimit for number of processes.
//
// See `man setrlimit`.
func SetMaxProcs(rlim *unix.Rlimit) error {
	return unix.Setrlimit(unix.RLIMIT_NPROC, rlim)
}

func rlimMax(rlim *unix.Rlimit) int64 {
	return int64(rlim.Max)
}

func rlimCur(rlim *unix.Rlimit) int64 {
	return int64(rlim.Cur)
}

// Sets the soft rlimit for maximum processes equal to the hard limit.
//
// See `man setrlimit`.
func MaximizeMaxProcs() error {
	var rlim unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NPROC, &rlim); err != nil {
		return err
	}
	rlim.Cur = rlim.Max
	return unix.Setrlimit(unix.RLIMIT_NPROC, &rlim)
}

// Gets the current (soft) and maximum (hard) rlimit for number of open files.
//
// See `man getrlimit`.
func GetMaxFiles() (*unix.Rlimit, error) {
	var rlim unix.Rlimit
	return &rlim, unix.Getrlimit(unix.RLIMIT_NOFILE, &rlim)
}

// Set the current (soft) and maximum (hard) rlimit for number of open files.
//
// See `man setrlimit`.
func SetMaxFiles(rlim *unix.Rlimit) error {
	return unix.Setrlimit(unix.RLIMIT_NOFILE, rlim)
}

// Sets the soft rlimit for maximum open files equal to the hard limit.
//
// See `man setrlimit`.
func MaximizeMaxFiles() error {
	var rlim unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &rlim); err != nil {
		return err
	}
	rlim.Cur = rlim.Max
	return unix.Setrlimit(unix.RLIMIT_NOFILE, &rlim)
}

func CheckMaxVmem(amount uint64) uint64 {
	var min uint64
	var rlim unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_AS, &rlim); err != nil {
		util.LogError(err, "jobmngr", "Could not get address space rlimit")
	} else {
		if rlim.Cur != unix.RLIM_INFINITY {
			min = rlim.Cur
			util.LogInfo("jobmngr", "ulimit -v: %dMB (minimum required %dMB)",
				rlim.Cur/(1024*1024), amount/(1024*1024))
			if rlim.Cur < amount {
				util.PrintInfo("jobmngr",
					"WARNING: The current virtual address space size"+
						"\n                              "+
						"limit is too low.\n\t"+
						"Limiting virtual address space size interferes with "+
						"the operation of many\n\tcommon libraries "+
						"and programs, and is not recommended.\n\t"+
						"Contact your system administrator to "+
						"remove this limit.")
			}
		}
	}
	if err := unix.Getrlimit(unix.RLIMIT_DATA, &rlim); err != nil {
		util.LogError(err, "jobmngr", "Could not get data segment size rlimit")
	} else {
		if rlim.Cur != unix.RLIM_INFINITY {
			if rlim.Cur < min {
				min = rlim.Cur
			}
			util.LogInfo("jobmngr", "ulimit -d: %dMB (minimum required %dMB)",
				rlim.Cur/(1024*1024), amount/(1024*1024))
			if rlim.Cur < amount {
				util.PrintInfo("jobmngr",
					"WARNING: The current data segment virtual size"+
						"\n                              "+
						"limit is too low.\n\t"+
						"Limiting virtual memory size interferes with "+
						"the operation of many\n\tcommon libraries "+
						"and programs, and is not recommended.\n\t"+
						"Contact your system administrator to "+
						"remove this limit.")
			}
		}
	}
	return min
}

func SetVMemRLimit(amount uint64) (uint64, error) {
	var rlim unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_AS, &rlim); err != nil {
		return rlim.Cur, err
	} else if rlim.Max != unix.RLIM_INFINITY && rlim.Max < amount {
		return rlim.Cur, fmt.Errorf("could not set RLIMIT_AS %d > %d",
			amount, rlim.Max)
	} else if rlim.Cur == amount {
		// Nothing to do.
		return amount, nil
	}
	oldAmount := rlim.Cur
	rlim.Cur = amount
	return oldAmount, unix.Setrlimit(unix.RLIMIT_AS, &rlim)
}

// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

// Get and set rlimit

package core

import (
	// syscall package lacks RLIMIT_NPROC
	"golang.org/x/sys/unix"
)

// Gets the current (soft) and maximum (hard) rlimit for number of processes.
//
// See `man getrlimit`
func GetMaxProcs() (*unix.Rlimit, error) {
	var rlim unix.Rlimit
	return &rlim, unix.Getrlimit(unix.RLIMIT_NPROC, &rlim)
}

// Sets the current (soft) and maximum (hard) rlimit for number of processes.
//
// See `man setrlimit`
func SetMaxProcs(rlim *unix.Rlimit) error {
	return unix.Setrlimit(unix.RLIMIT_NPROC, rlim)
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
// See `man getrlimit`
func GetMaxFiles() (*unix.Rlimit, error) {
	var rlim unix.Rlimit
	return &rlim, unix.Getrlimit(unix.RLIMIT_NOFILE, &rlim)
}

// Set the current (soft) and maximum (hard) rlimit for number of open files.
//
// See `man setrlimit`
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

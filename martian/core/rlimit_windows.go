// Copyright (c) 2022 10X Genomics, Inc. All rights reserved.

// Stubs to permit compilation on windows.  Just no-ops.

package core

import (
	"errors"
)

// Gets the current (soft) and maximum (hard) rlimit for number of processes.
//
// See `man getrlimit`.
func GetMaxProcs() (interface{}, error) {
	return nil, errors.New("not supported on windows")
}

func CheckMaxVmem(uint64) uint64 { return 0 }

func rlimMax(rlim interface{}) int64 {
	return 0
}

func rlimCur(rlim interface{}) int64 {
	return 0
}

func SetVMemRLimit(uint64) error { return nil }

func MaximizeMaxFiles() error { return nil }

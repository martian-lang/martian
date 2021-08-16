// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// Generic directory utilities.

//go:build !linux
// +build !linux

package util

import (
	"os"
)

// CountDirNames returns the number of files in the directory opened with the
// given file descriptor.
func CountDirNames(fd int) (int, error) {
	s, err := os.NewFile(uintptr(fd), "task").Readdirnames(-1)
	return len(s), err
}

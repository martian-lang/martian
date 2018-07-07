// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// Generic directory utilities.
// +build !linux

package util

// CountDirNames returns the number of files in the directory opened with the
// given file descriptor.
func CountDirNames(fd int) (int, error) {
	s, err := os.NewFile(fd, "task").Readdirnames()
	return len(s), err
}

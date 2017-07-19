//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//
// File system query utility.
//
package core

import (
	"fmt"
	"syscall"
)

func GetAvailableSpace(path string) (bytes, inodes uint64, err error) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return 0, 0, err
	}
	return fs.Bavail * uint64(fs.Bsize), fs.Ffree, nil
}

// The minimum number of inodes available in the pipestance directory
// below which the pipestance will not run.
const PIPESTANCE_MIN_INODES uint64 = 500

// The minimum amount of available disk space for a pipestance directory.
// If the available space falls below this at any time during the run, the
// the pipestance is killed.
const PIPESTANCE_MIN_DISK uint64 = 50 * 1024 * 1024

type DiskSpaceError struct {
	Bytes   uint64
	Inodes  uint64
	Message string
}

func (self *DiskSpaceError) Error() string {
	return self.Message
}

func CheckMinimalSpace(path string) error {
	bytes, inodes, err := GetAvailableSpace(path)
	if err != nil {
		return err
	}
	if bytes < PIPESTANCE_MIN_DISK {
		return &DiskSpaceError{bytes, inodes, fmt.Sprintf(
			"%s has only %dkB remaining space available.",
			path, bytes/1024)}
	}
	if inodes < PIPESTANCE_MIN_INODES {
		return &DiskSpaceError{bytes, inodes, fmt.Sprintf(
			"%s has only %d free inodes remaining.",
			path, inodes)}
	}
	return nil
}

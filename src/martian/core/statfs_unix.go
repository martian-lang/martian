// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package core

//
// File system query utility.
//

import (
	"fmt"
	"os"
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

var disableDiskSpaceCheck = (os.Getenv("MRO_DISK_SPACE_CHECK") == "disable")

// Returns an error if the current available space on the disk drive is
// very low.
func CheckMinimalSpace(path string) error {
	if disableDiskSpaceCheck {
		return nil
	}
	bytes, inodes, err := GetAvailableSpace(path)
	if err != nil {
		return err
	}
	// Allow zero, as if we haven't already failed to write a file it's
	// likely that the filesystem is just lying to us.
	if bytes < PIPESTANCE_MIN_DISK && bytes != 0 {
		return &DiskSpaceError{bytes, inodes, fmt.Sprintf(
			"%s has only %dkB remaining space available.\n"+
				"To ignore this error, set MRO_DISK_SPACE_CHECK=disable in your environment.",
			path, bytes/1024)}
	}
	if inodes < PIPESTANCE_MIN_INODES && inodes != 0 {
		return &DiskSpaceError{bytes, inodes, fmt.Sprintf(
			"%s has only %d free inodes remaining.\n"+
				"To ignore this error, set MRO_DISK_SPACE_CHECK=disable in your environment.",
			path, inodes)}
	}
	return nil
}

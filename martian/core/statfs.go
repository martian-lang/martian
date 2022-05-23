// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Type definitions for statfs results on all OS.

package core

type DiskSpaceError struct {
	Message string
	Bytes   uint64
	Inodes  uint64
}

func (self *DiskSpaceError) Error() string {
	return self.Message
}

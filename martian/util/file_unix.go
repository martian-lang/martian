// Copyright (c) 2022 10X Genomics, Inc. All rights reserved.

//go:build !windows
// +build !windows

package util

import (
	"io/fs"
	"syscall"
)

// GetFileOwner returns the uid and gid for a FileInfo object.
//
// Returns false if the info object does not contain this information.
func GetFileOwner(info fs.FileInfo) (uint32, uint32, bool) {
	if info == nil {
		return 0, 0, false
	}
	if sysInfo, ok := info.Sys().(*syscall.Stat_t); ok {
		return sysInfo.Uid, sysInfo.Gid, true
	}
	return 0, 0, false
}

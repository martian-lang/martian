// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// Utility methods for linux file stuff.

package util

import (
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func FileCreateTime(info os.FileInfo) time.Time {
	switch sysInfo := info.Sys().(type) {
	case *syscall.Stat_t:
		s, ns := sysInfo.Ctim.Unix()
		return time.Unix(s, ns).Truncate(time.Second)
	case *unix.Stat_t:
		s, ns := sysInfo.Ctim.Unix()
		return time.Unix(s, ns).Truncate(time.Second)
	default:
		return info.ModTime()
	}
}

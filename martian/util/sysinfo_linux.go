// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Method to get the linux kernel release version.

package util

import (
	"bytes"

	"golang.org/x/sys/unix"
)

func getKernelVersion() string {
	var name unix.Utsname
	if err := unix.Uname(&name); err != nil {
		LogError(err, "sysinfo", "Error getting kernel version.")
		return ""
	} else if i := bytes.IndexByte(name.Release[:], 0); i >= 0 {
		return string(name.Release[:i])
	} else {
		return string(name.Release[:])
	}
}

// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Utilities for logging system configuration info.

package util

import "runtime"

func LogSysInfo() {
	LogInfo("sysinfo", "%s %s", runtime.GOOS, runtime.GOARCH)
	if ver := getKernelVersion(); ver != "" {
		LogInfo("sysinfo", "Linux kernel release version: %s", ver)
	}
	if libc := getLibcVersion(); libc != "" {
		LogInfo("sysinfo", "glibc version: %s", libc)
	}
}

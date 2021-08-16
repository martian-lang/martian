// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

//go:build !linux
// +build !linux

// Stubs for sysinfo utilities on non-linux systems.

package util

func getKernelVersion() string {
	return ""
}

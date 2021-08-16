// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

//go:build !linux
// +build !linux

// Stubs for cgroup utilities on non-linux systems.

package util

// Returns zero.
func GetCgroupMemoryLimit() (limit, softLimit, usage int64) {
	return 0, 0, 0
}

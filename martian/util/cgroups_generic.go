// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

// +build !linux

// Stubs for cgroup utilities on non-linux systems.

package util

// Returns zero.
func GetCgroupMemoryLimit() (limit, softLimit, usage int64) {
	return 0, 0, 0
}

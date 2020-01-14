// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// +build !linux

// Stubs for unsupported OS

package util

import (
	"os"
	"time"
)

func FileCreateTime(os.FileInfo) time.Time {
	return time.Time{}
}

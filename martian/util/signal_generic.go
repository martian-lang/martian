// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
// +build !linux

package util

import (
	"syscall"
)

// On non-unix systems this will always return false.
func SignalIsIgnored(syscall.Signal) bool {
	return false
}

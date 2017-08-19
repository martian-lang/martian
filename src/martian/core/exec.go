// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
// +build !linux

package core

// Platform-specific code for launching processes.  This file is
// a stub to allow non-linux systems to build with reduced functionality.

import (
	"syscall"
)

// Add pdeathsig to a SysProcAttr structure, if the operating system supports
// it, and return the object.  On other platforms, do nothing.
func Pdeathsig(attr *syscall.SysProcAttr, sig syscall.Signal) *syscall.SysProcAttr {
	return attr
}

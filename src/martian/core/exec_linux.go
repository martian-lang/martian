//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Platform-specific code for launching processes.
//

package core

import (
	"syscall"
)

// Add pdeathsig to a SysProcAttr structure, if the operating system supports
// it, and return the object.  On other platforms, do nothing.
func Pdeathsig(attr *syscall.SysProcAttr, sig syscall.Signal) *syscall.SysProcAttr {
	attr.Pdeathsig = sig
	return attr
}

// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

// This file implements querying whether a given signal is being ignored.
// This could be done easily with CGO and <signal.h>, but that would introduce
// a dependency on the C compiler and, more importantly, runtime libraries.
// Instead, we implement the syscall in assembly, mimicking from the go source.
// See src/runtime/os_linux.go and os_linux_generic.go in the go source.

// +build !mips
// +build !mipsle
// +build !mips64
// +build !mips64le
// +build !s390x
// +build !ppc64
// +build linux

package util

import (
	"syscall"
	"unsafe"
)

// Test whether the given signal is currently ignored.
func SignalIsIgnored(sig syscall.Signal) bool {
	return isIgnored(uint32(sig))
}

// From Go's runtime/defs_linux_arm64.go
type sigactiont struct {
	sa_handler  uintptr
	sa_flags    uint64
	sa_restorer uintptr
	sa_mask     uint64
}

// From Go's runtime/signal_unix.go
const _SIG_IGN uintptr = 1

func isIgnored(sig uint32) bool {
	var sa sigactiont
	if errno := rt_sigaction(uintptr(sig), nil, &sa, unsafe.Sizeof(sa.sa_mask)); errno != 0 {
		LogInfo("runtime",
			"Could not determine whether signal %d is being ignored (errno: %v)",
			sig, errno)
		return false
	}
	return sa.sa_handler == _SIG_IGN
}

// rt_sigaction calls the rt_sigaction system call. It is implemented in assembly.
//go:noescape
func rt_sigaction(sig uintptr, new, old *sigactiont, size uintptr) int32

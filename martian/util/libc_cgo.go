// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

//go:build linux && cgo
// +build linux,cgo

// Get the libc version.

package util

// #include <gnu/libc-version.h>
import "C"

func getLibcVersion() string {
	return C.GoString(C.gnu_get_libc_version())
}

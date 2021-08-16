// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

//go:build !linux || !cgo
// +build !linux !cgo

// Get the libc version.

package util

func getLibcVersion() string {
	return ""
}

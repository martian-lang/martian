// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

// +build !freebsd,!linux,!netbsd,!openbsd,!solaris

package core

//
// File system query utility - stubs for unsupported OS.
//

import (
	"errors"
)

// Does nothing on non-unix systems.
func FsTypeString(int64) string {
	return ""
}

// Not supported.
func GetAvailableSpace(string) (uint64, uint64, string, error) {
	return 0, 0, "", errors.New("unsupported OS")
}

// Does nothing on unsupported OS.
func CheckMinimalSpace(path string) error {
	return nil
}

// Not supported
func GetMountOptions(string) (string, string, error) {
	return "", "", errors.New("unsupported OS")
}

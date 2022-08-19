// Copyright (c) 2022 10X Genomics, Inc. All rights reserved.

//go:build !linux
// +build !linux

package core

import "os"

func writeAtomic(target string, data []byte) error {
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return wrapAtomicError("could not write temp file", err)
	}
	if err := os.Rename(tmp, target); err == nil || os.IsNotExist(err) {
		return nil
	} else {
		_ = os.Remove(tmp) // Attempt cleanup.
		return wrapAtomicError("could not rename temp file", err)
	}
}

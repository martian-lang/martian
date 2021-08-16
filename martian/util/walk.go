// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

//go:build !linux
// +build !linux

// Generic implementation forwarding to filepath.Walk

package util

import (
	"path/filepath"
)

// Fowarads to path/filepath.Walk
func Walk(root string, walkFn filepath.WalkFunc) error {
	return filepath.Walk(root, walkFn)
}

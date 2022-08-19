// Copyright (c) 2022 10X Genomics, Inc. All rights reserved.

//go:build !go1.19
// +build !go1.19

package util

import (
	"os/exec"
)

// lookPath is an alias for exec.LookPath in versions of go before go1.19.
func lookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// Copyright (c) 2022 10X Genomics, Inc. All rights reserved.

//go:build go1.19
// +build go1.19

package util

import (
	"errors"
	"os/exec"
)

// lookPath finds an executable in the PATH, including in PATHs which are
// specified relative to the current working directory.  This should be avoided
// in general, because it is insecure.  However, in this case we're using it
// only for finding the location of the current executable.  Do not use this
// in other situations.
func lookPath(file string) (string, error) {
	p, err := exec.LookPath(file)
	if errors.Is(err, exec.ErrDot) {
		return p, nil
	}
	return p, err
}

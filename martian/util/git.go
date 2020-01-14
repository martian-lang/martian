// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Run git with the given args, setting pdeathsig in case the parent dies.

// +build !linux

package util

import (
	"bytes"
	"os/exec"
)

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(bytes.TrimSpace(out)), err
}

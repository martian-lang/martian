// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

// Run git with the given args, setting pdeathsig in case the parent dies.

package util

import (
	"bytes"
	"os/exec"
	"syscall"
)

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	out, err := cmd.Output()
	return string(bytes.TrimSpace(out)), err
}

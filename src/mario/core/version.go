//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Mario versioning facility.
//
package core

import (
	"os"
	"os/exec"
	"strings"
)

var __VERSION__ string = "<version not embedded>"

func GetVersion() string {
	return __VERSION__
}

func runGit(dir string, args ...string) (string, error) {
	oldCwd, _ := os.Getwd()
	os.Chdir(dir)
	out, err := exec.Command("git", args...).Output()
	os.Chdir(oldCwd)
	return strings.TrimSpace(string(out)), err
}

func GetGitTag(dir string) string {
	if out, err := runGit(dir, "describe", "--tags", "--dirty", "--always"); err == nil {
		return out
	}
	return "noversion"
}

func GetGitBranch(dir string) string {
	if out, err := runGit(dir, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		return out
	}
	return "nobranch"
}

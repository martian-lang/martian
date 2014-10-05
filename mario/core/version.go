//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
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

func GetGitTag(dir string) string {
	oldCwd, _ := os.Getwd()
	os.Chdir(dir)
	out, err := exec.Command("git", "describe", "--tags", "--dirty", "--always").Output()
	os.Chdir(oldCwd)
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return "noversion"
}

//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian versioning facility.
//
package core

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

var __VERSION__ string = "<version not embedded>"

func GetVersion() string {
	return __VERSION__
}

func GetMroVersion(dir string) string {
	versionPath := path.Join(dir, "..", ".version")
	if data, err := ioutil.ReadFile(versionPath); err == nil {
		return string(data)
	}
	return GetGitTag(dir)
}

func runGit(dir string, args ...string) (string, error) {
	oldCwd, _ := os.Getwd()
	os.Chdir(dir)
	out, err := exec.Command("git", args...).Output()
	os.Chdir(oldCwd)
	return strings.TrimSpace(string(out)), err
}

func GetGitTag(dir string) string {
	out, err := runGit(dir, "describe", "--tags", "--dirty", "--always")
	if err == nil {
		return out
	}
	return "noversion"
}

func GetGitBranch(dir string) string {
	out, err := runGit(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		return out
	}
	return "nobranch"
}

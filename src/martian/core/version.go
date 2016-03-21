//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian versioning facility.
//
package core

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

var __VERSION__ string = "<version not embedded>"
var __RELEASE__ string = "false"

func GetVersion() string {
	return __VERSION__
}

func IsRelease() bool {
	out, _ := strconv.ParseBool(__RELEASE__)
	return out
}

func GetMroVersion(dirs []string) (string, error) {
	errs := []string{}
	for _, dir := range dirs {
		if version, err := GetSakeVersion(dir); err == nil {
			return version, nil
		} else {
			errs = append(errs, err.Error())
		}
	}
	for _, dir := range dirs {
		if version, err := GetGitTag(dir); err == nil {
			return version, nil
		} else {
			errs = append(errs, err.Error())
		}
	}
	return "noversion", errors.New(fmt.Sprintf("Failed to get MRO version with errors: %s", strings.Join(errs, ", ")))
}

func GetSakeVersion(dir string) (string, error) {
	versionPath := path.Join(dir, "..", ".version")
	data, err := ioutil.ReadFile(versionPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func GetGitTag(dir string) (string, error) {
	return runGit(dir, "describe", "--tags", "--dirty", "--always")
}

func GetGitBranch(dir string) (string, error) {
	return runGit(dir, "rev-parse", "--abbrev-ref", "HEAD")
}

//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian versioning facility.
//

package util

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

var __VERSION__ string = "<version not embedded>"
var __RELEASE__ string = "false"

// Get the build version for this binary, as embedded by the build system.
func GetVersion() string {
	return __VERSION__
}

// IsRelease returns true if the build system specified that this was a
// release version.
func IsRelease() bool {
	out, _ := strconv.ParseBool(__RELEASE__)
	return out
}

// Searches for a file named .version in the parent directory of any of
// the given directories and returns its content if available.  If none is
// found, attempts to run 'git describe --tags --dirty --always' in those
// directories to get a version.
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
	return "noversion", fmt.Errorf("Failed to get MRO version with errors: %s", strings.Join(errs, ", "))
}

// Searches for a file name .version in the parent of the given directory name
// and returns its content.
func GetSakeVersion(dir string) (string, error) {
	versionPath := path.Join(dir, "..", ".version")
	data, err := ioutil.ReadFile(versionPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// Returns the output of running 'git describe --tags --dirty --always'
// in the given directory, e.g. 'v2.3.0-rc3-10-gf615588-dirty'
func GetGitTag(dir string) (string, error) {
	return runGit(dir, "describe", "--tags", "--dirty", "--always")
}

// Returns the output of running 'git rev-parse --abbrev-ref HEAD' in
// the given directory, e.g. 'master'.
func GetGitBranch(dir string) (string, error) {
	return runGit(dir, "rev-parse", "--abbrev-ref", "HEAD")
}

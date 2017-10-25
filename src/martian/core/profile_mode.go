//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//
// Martian profile modes specify runtime performance profiling.
//

package core

import (
	"martian/util"
	"os"
	"strings"
)

//=============================================================================
// Profile mode
//=============================================================================

// Defines available profiling modes for stage code.
type ProfileMode string

const (
	DisableProfile    ProfileMode = "disable"
	CpuProfile        ProfileMode = "cpu"
	MemProfile        ProfileMode = "mem"
	LineProfile       ProfileMode = "line"
	PyflameProfile    ProfileMode = "pyflame"
	PerfRecordProfile ProfileMode = "perf"
)

var validProfileModes = []ProfileMode{
	DisableProfile,
	CpuProfile,
	MemProfile,
	LineProfile,
	PyflameProfile,
	PerfRecordProfile,
}

func allProfileModes() string {
	profileModeStrings := make([]string, len(validProfileModes))
	for i, validMode := range validProfileModes {
		profileModeStrings[i] = string(validMode)
	}
	return strings.Join(profileModeStrings, ", ")
}

func VerifyProfileMode(profileMode ProfileMode) {
	for _, validMode := range validProfileModes {
		if validMode == profileMode {
			return
		}
	}
	util.PrintInfo("runtime", "Invalid profile mode: %s. Valid profile modes: %s",
		profileMode, allProfileModes())
	os.Exit(1)
}

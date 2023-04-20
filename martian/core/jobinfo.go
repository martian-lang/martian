// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package core

import (
	"fmt"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

// Shared job information structures.

type JobInfo struct {
	Name          string            `json:"name"`
	Type          string            `json:"type,omitempty"`
	Cwd           string            `json:"cwd,omitempty"`
	PythonInfo    *PythonInfo       `json:"python,omitempty"`
	RusageInfo    *RusageInfo       `json:"rusage,omitempty"`
	MemoryUsage   *ObservedMemory   `json:"used_bytes,omitempty"`
	IoStats       *IoStats          `json:"io,omitempty"`
	WallClockInfo *WallClockInfo    `json:"wallclock,omitempty"`
	ProfileConfig *ProfileConfig    `json:"profile_config,omitempty"`
	ProfileMode   ProfileMode       `json:"profile_mode,omitempty"`
	Stackvars     string            `json:"stackvars_flag,omitempty"`
	Monitor       string            `json:"monitor_flag,omitempty"`
	Invocation    *InvocationData   `json:"invocation,omitempty"`
	Version       *VersionInfo      `json:"version,omitempty"`
	ClusterEnv    map[string]string `json:"sge,omitempty"`
	Host          string            `json:"host,omitempty"`
	Pid           int               `json:"pid,omitempty"`
	Threads       float64           `json:"threads,omitempty"`
	MemGB         float64           `json:"memGB,omitempty"`
	VMemGB        float64           `json:"vmemGB,omitempty"`
	SkipPreflight bool              `json:"skip_preflight,omitempty"`
}

type PythonInfo struct {
	BinPath string `json:"binpath"`
	Version string `json:"version"`
}

// WallClockTime is a time value which can be parsed from json in either
// RFC 3339 format or the "legacy" format, "2006-01-02 15:04:05", which lacks
// time zone information.
type WallClockTime time.Time

func (wt WallClockTime) MarshalJSON() ([]byte, error) {
	return time.Time(wt).MarshalJSON()
}

func (wt WallClockTime) MarshalText() ([]byte, error) {
	return time.Time(wt).MarshalText()
}

func (wt *WallClockTime) UnmarshalJSON(b []byte) error {
	var t time.Time
	if err := t.UnmarshalJSON(b); err == nil {
		*wt = WallClockTime(t)
		return nil
	}
	if len(b) == 2 && b[1] == '"' && b[0] == '"' {
		*wt = WallClockTime(t)
		return nil
	}
	t, err := time.ParseInLocation(`"`+util.TIMEFMT+`"`, string(b), time.Local)
	if err != nil {
		return fmt.Errorf("could not parse %q as timestamp: %w", b, err)
	}
	*wt = WallClockTime(t)
	return nil
}

func (wt *WallClockTime) UnmarshalText(b []byte) error {
	var t time.Time
	err := t.UnmarshalText(b)
	*wt = WallClockTime(t)
	return err
}

func (wt WallClockTime) String() string {
	return time.Time(wt).String()
}

func (wt WallClockTime) GoString() string {
	return time.Time(wt).GoString()
}

func (wt WallClockTime) IsZero() bool {
	return time.Time(wt).IsZero()
}

func (wt WallClockTime) Before(u WallClockTime) bool {
	return time.Time(wt).Before(time.Time(u))
}

func (wt WallClockTime) Sub(u WallClockTime) time.Duration {
	return time.Time(wt).Sub(time.Time(u))
}

type WallClockInfo struct {
	Start    WallClockTime `json:"start"`
	End      WallClockTime `json:"end,omitempty"`
	Duration float64       `json:"duration_seconds,omitempty"`
}

type InvocationData struct {
	Call      string          `json:"call"`
	Args      LazyArgumentMap `json:"args"`
	Include   string          `json:"mro_file,omitempty"`
	SweepArgs []string        `json:"sweepargs,omitempty"`
	SplitArgs []string        `json:"splitargs,omitempty"`
}

type VersionInfo struct {
	Martian   string `json:"martian"`
	Pipelines string `json:"pipelines"`
}

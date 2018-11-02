// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

// Shared job information structures.

type JobInfo struct {
	Name          string            `json:"name"`
	Pid           int               `json:"pid,omitempty"`
	Host          string            `json:"host,omitempty"`
	Type          string            `json:"type,omitempty"`
	Cwd           string            `json:"cwd,omitempty"`
	PythonInfo    *PythonInfo       `json:"python,omitempty"`
	RusageInfo    *RusageInfo       `json:"rusage,omitempty"`
	MemoryUsage   *ObservedMemory   `json:"used_bytes,omitempty"`
	IoStats       *IoStats          `json:"io,omitempty"`
	WallClockInfo *WallClockInfo    `json:"wallclock,omitempty"`
	Threads       int               `json:"threads,omitempty"`
	MemGB         int               `json:"memGB,omitempty"`
	ProfileConfig *ProfileConfig    `json:"profile_config,omitempty"`
	ProfileMode   ProfileMode       `json:"profile_mode,omitempty"`
	Stackvars     string            `json:"stackvars_flag,omitempty"`
	Monitor       string            `json:"monitor_flag,omitempty"`
	Invocation    *InvocationData   `json:"invocation,omitempty"`
	Version       *VersionInfo      `json:"version,omitempty"`
	ClusterEnv    map[string]string `json:"sge,omitempty"`
}

type PythonInfo struct {
	BinPath string `json:"binpath"`
	Version string `json:"version"`
}

type WallClockInfo struct {
	Start    string  `json:"start"`
	End      string  `json:"end,omitempty"`
	Duration float64 `json:"duration_seconds,omitempty"`
}

type InvocationData struct {
	Call      string          `json:"call"`
	Args      LazyArgumentMap `json:"args"`
	SweepArgs []string        `json:"sweepargs"`
}

type VersionInfo struct {
	Martian   string `json:"martian"`
	Pipelines string `json:"pipelines"`
}

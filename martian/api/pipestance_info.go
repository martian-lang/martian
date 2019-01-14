//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//

// Data structures used to exchange information and requests over the http
// interface to mrp and similar tools.
package api

import (
	"net/url"
	"path/filepath"
	"strconv"

	"github.com/martian-lang/martian/martian/core"
)

// Stores information about a pipestance which might be interesting to
// a user, or useful for a tool to query.
type PipestanceInfo struct {
	// The name of the host where MRP is running.
	Hostname string `json:"hostname"`

	// The username of the user running MRP.
	Username string `json:"username"`

	// MRP's working directory.
	Cwd string `json:"cwd"`

	// The path to the mrp executable.
	Binpath string `json:"binpath"`

	// The command line used to execute mrp.
	Cmdline string `json:"cmdline"`

	// The PID of the MRP instance.
	Pid int `json:"pid"`

	// The time when the pipestance was first started.
	Start string `json:"start"`

	// The martian version for this mrp.
	Version      string             `json:"version"`
	Pname        string             `json:"pname"`
	PsId         string             `json:"psid"`
	State        core.MetadataState `json:"state"`
	JobMode      string             `json:"jobmode"`
	MaxCores     int                `json:"maxcores"`
	MaxMemGB     int                `json:"maxmemgb"`
	InvokePath   string             `json:"invokepath"`
	InvokeSource string             `json:"invokesrc,omitempty"`
	MroPath      string             `json:"mropath"`
	ProfileMode  core.ProfileMode   `json:"mroprofile"`
	Port         string             `json:"mroport"`
	MroVersion   string             `json:"mroversion"`
	Uuid         string             `json:"uuid"`
	PsPath       string             `json:"pipestance_path,omitempty"`

	// The reason for the most recent pipestance failure, if any.
	LastErrorMessage string `json:"err_msg,omitempty"`
}

// The full state information for a pipestance, including the status of every
// node in the pipeline graph.
type PipestanceState struct {
	Nodes []*core.NodeInfo `json:"nodes"`
	Info  *PipestanceInfo  `json:"info"`
}

// All of the performance information for a pipestance.
type PerfInfo struct {
	Nodes []*core.NodePerfInfo `json:"nodes"`
}

// Gets a shallow copy of this object with the InvokeSource omitted.
func (self *PipestanceInfo) StripMro() *PipestanceInfo {
	return &PipestanceInfo{
		Hostname:   self.Hostname,
		Username:   self.Username,
		Cwd:        self.Cwd,
		Binpath:    self.Binpath,
		Cmdline:    self.Cmdline,
		Pid:        self.Pid,
		Start:      self.Start,
		Version:    self.Version,
		Pname:      self.Pname,
		PsId:       self.PsId,
		State:      self.State,
		JobMode:    self.JobMode,
		MaxCores:   self.MaxCores,
		MaxMemGB:   self.MaxMemGB,
		InvokePath: self.InvokePath,
		// omitted source
		MroPath:          self.MroPath,
		ProfileMode:      self.ProfileMode,
		Port:             self.Port,
		MroVersion:       self.MroVersion,
		Uuid:             self.Uuid,
		PsPath:           self.PsPath,
		LastErrorMessage: self.LastErrorMessage,
	}
}

// Get the absolute path to the pipestance directory
func (self *PipestanceInfo) FullPipestancePath() string {
	if self.PsPath != "" {
		if filepath.IsAbs(self.PsPath) {
			return self.PsPath
		} else {
			return filepath.Join(self.Cwd, self.PsPath)
		}
	} else {
		return filepath.Join(self.Cwd, self.PsId)
	}
}

// Convert url form fields to a PipestanceInfo.
func ParsePipestanceInfoForm(form url.Values) (PipestanceInfo, error) {
	info := PipestanceInfo{
		Hostname:         form.Get("hostname"),
		Username:         form.Get("username"),
		Cwd:              form.Get("cwd"),
		Binpath:          form.Get("binpath"),
		Cmdline:          form.Get("cmdline"),
		Start:            form.Get("start"),
		Version:          form.Get("version"),
		Pname:            form.Get("pname"),
		PsId:             form.Get("psid"),
		State:            core.MetadataState(form.Get("state")),
		JobMode:          form.Get("jobmode"),
		InvokePath:       form.Get("invokepath"),
		InvokeSource:     form.Get("invokesrc"),
		MroPath:          form.Get("mropath"),
		ProfileMode:      core.ProfileMode(form.Get("mroprofile")),
		Port:             form.Get("mroport"),
		MroVersion:       form.Get("mroversion"),
		Uuid:             form.Get("uuid"),
		PsPath:           form.Get("pipestance_path"),
		LastErrorMessage: form.Get("err_msg"),
	}
	var err, lastErr error
	if info.Pid, err = strconv.Atoi(form.Get("pid")); err != nil {
		lastErr = err
	}
	if info.MaxCores, err = strconv.Atoi(form.Get("maxcores")); err != nil {
		lastErr = err
	}
	if info.MaxMemGB, err = strconv.Atoi(form.Get("maxmemgb")); err != nil {
		lastErr = err
	}
	return info, lastErr
}

// Serialize this object as a url form.
func (self *PipestanceInfo) AsForm() url.Values {
	form := url.Values{}
	form.Add("hostname", self.Hostname)
	form.Add("username", self.Username)
	form.Add("cwd", self.Cwd)
	form.Add("binpath", self.Binpath)
	form.Add("cmdline", self.Cmdline)
	form.Add("pid", strconv.Itoa(self.Pid))
	form.Add("start", self.Start)
	form.Add("version", self.Version)
	form.Add("pname", self.Pname)
	form.Add("psid", self.PsId)
	form.Add("state", string(self.State))
	form.Add("jobmode", self.JobMode)
	form.Add("maxcores", strconv.Itoa(self.MaxCores))
	form.Add("maxmemgb", strconv.Itoa(self.MaxMemGB))
	form.Add("invokepath", self.InvokePath)
	if self.InvokeSource != "" {
		form.Add("invokesrc", self.InvokeSource)
	}
	form.Add("mropath", self.MroPath)
	form.Add("mroprofile", string(self.ProfileMode))
	form.Add("mroport", self.Port)
	form.Add("mroversion", self.MroVersion)
	form.Add("uuid", self.Uuid)
	if self.PsPath != "" {
		form.Add("pipestance_path", self.PsPath)
	}
	if self.LastErrorMessage != "" {
		form.Add("err_msg", self.LastErrorMessage)
	}
	return form
}

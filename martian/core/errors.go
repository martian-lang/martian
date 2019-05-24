// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

//
// Martian error types.
//

import (
	"fmt"
)

// RuntimeError is returned to indicate a failure to instantiate or
// advance a pipeline at runtime.
type RuntimeError struct {
	Msg string
}

func (self *RuntimeError) Error() string {
	return fmt.Sprintf("RuntimeError: %s.", self.Msg)
}

// PipestanceInvocationError is returned when attempting to reattach to a
// pipestance directory if the mro source code has changed.
type PipestanceInvocationError struct {
	Psid           string
	InvocationPath string
}

func (self *PipestanceInvocationError) Error() string {
	return fmt.Sprintf(
		"RuntimeError: pipestance '%s' already exists with different "+
			"invocation file %s",
		self.Psid, self.InvocationPath)
}

// PipestancePathError is returned when attempting to reattach to a pipestance
// if the target directory does not appear to be a pipestance.
type PipestancePathError struct {
	Path string
}

func (self *PipestancePathError) Error() string {
	return fmt.Sprintf(
		"RuntimeError: %s is not a pipestance directory",
		self.Path)
}

// PipestanceJobModeError is returned when attempting to reattach to a
// pipestance which was started with a different job mode.
type PipestanceJobModeError struct {
	Psid    string
	JobMode string
}

func (self *PipestanceJobModeError) Error() string {
	return fmt.Sprintf(
		"RuntimeError: pipestance '%s' was originally started "+
			"in job mode '%s'. Please try running again in job mode '%s'.",
		self.Psid, self.JobMode, self.JobMode)
}

// PipestanceLockedError is returned when attempting to Lock a
// pipestance which is already locked.
type PipestanceLockedError struct {
	Psid           string
	PipestancePath string
}

func (self *PipestanceLockedError) Error() string {
	return fmt.Sprintf(
		"RuntimeError: pipestance '%s' already exists and is locked by "+
			"another Martian instance. If you are sure no other Martian "+
			"instance is running, delete the _lock file in %s and start "+
			"Martian again.",
		self.Psid, self.PipestancePath)
}

// PipestanceExistsError is returned by the runtime when attempting to instantiate
// a pipestance in a directory which already has a pipestance.
type PipestanceExistsError struct {
	Psid string
}

func (self *PipestanceExistsError) Error() string {
	return fmt.Sprintf("RuntimeError: pipestance '%s' already exists.", self.Psid)
}

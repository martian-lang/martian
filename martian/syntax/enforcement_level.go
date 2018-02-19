// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

// Methods for controlling how strict the parser and runtime are about
// enforcing semantics.

package syntax

import (
	"fmt"
)

// Specifies how strictly new language features should be.
//
// The intent is that when the compiler or runtime become more strict about
// some existing semantics, in a way which may break older pipelines, the
// level of backwards compatibility for the compiler and runtime is specified
// according to these levels.  Such backwards compatibility measures should be
// considered temporary measures - pipelines affected by this flag are
// depending on deprecated functionality.
type LanguageEnforceLevel int

const (
	// Enforcement of new constraints is disabled.
	EnforceDisable LanguageEnforceLevel = iota

	// Violation of new constraints is logged as a warning.
	EnforceLog

	// Violation of new constraints triggers an alarm in pipelines.  The
	// result of this is that in addition to logging the warning as it
	// occurs, the alarm will be printed with the pipeline's final outputs.
	EnforceAlarm

	// Backwards compatibility is turned off.  Violation of constraints will
	// cause pipelines to fail.
	EnforceError
)

var currentEnforceLevel LanguageEnforceLevel

// Get the current language enforcement level.
func GetEnforcementLevel() LanguageEnforceLevel {
	return currentEnforceLevel
}

// Set the global language enforcement level.
func SetEnforcementLevel(level LanguageEnforceLevel) {
	if level < EnforceDisable || level > EnforceError {
		panic("Invalid level")
	}
	currentEnforceLevel = level
}

func (self LanguageEnforceLevel) String() string {
	switch self {
	case EnforceLog:
		return "log"
	case EnforceAlarm:
		return "alarm"
	case EnforceError:
		return "error"
	default:
		return "disable"
	}
}

// Convert from a string representation.
func ParseEnforcementLevel(level string) LanguageEnforceLevel {
	switch level {
	case "log":
		return EnforceLog
	case "alarm":
		return EnforceAlarm
	case "error":
		return EnforceError
	default:
		return EnforceDisable
	}
}

func (self LanguageEnforceLevel) MarshalText() (text []byte, err error) {
	if self < EnforceDisable || self > EnforceError {
		return nil, fmt.Errorf("Invalid level %d", int(self))
	}
	return []byte(self.String()), nil
}

func (self *LanguageEnforceLevel) UnmarshalText(text []byte) error {
	*self = ParseEnforcementLevel(string(text))
	return nil
}

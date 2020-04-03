//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//
// Martian profile modes specify runtime performance profiling.
//

package core

import (
	"fmt"
	"os"
	"sort"
	"strconv"
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
	LineProfile       ProfileMode = "line"
	MemProfile        ProfileMode = "mem"
	PerfRecordProfile ProfileMode = "perf"
)

func allProfileModes(validProfileModes map[ProfileMode]*ProfileConfig) string {
	profileModeStrings := make([]string, 0, len(validProfileModes))
	for validMode := range validProfileModes {
		profileModeStrings = append(profileModeStrings, string(validMode))
	}
	sort.Strings(profileModeStrings)
	return strings.Join(profileModeStrings, ", ")
}

// ProfileConfig defines a profiling mode.
type ProfileConfig struct {
	// If present, specifies a separate command to run in order to profile
	// the stage code.  The command will be started immediately after the
	// stage code starts up, and must shut down within 15 seconds of the
	// stage code process completing.
	Command string `json:"cmd,omitempty"`

	// The arguments to pass to the specified command.
	// Ignored if Command is not set.
	//
	// Arguments matching $VAR or ${VAR} will be expanded
	// based on the environment variables available at the
	// time the stage code runs, or an empty string if they
	// are not defined, with three exceptions.
	// ${PROFILE_DEST} will expand to the expected output
	// location for human-readable profile data (e.g.
	// /path/to/stage/fork0/split-u12345/_profile.out).
	// ${RAW_PERF_DEST} will expand to the expected output
	// location for binary profiling data (e.g. .../_perf.data).
	// ${STAGE_PID} will expand to the pid of the running stage
	// code process.
	Args []string `json:"args,omitempty"`

	// When expanding environment variables for args, these
	// values are used for empty or missing variables.
	Defaults map[string]string `json:"defaults,omitempty"`

	// Sets these environment variables for the stage code
	// processes.  Specified values are expaned based on the
	// preexisting environment in the same way as for Args,
	// except ${STAGE_PID} is of course not yet available.
	Env map[string]string `json:"env,omitempty"`

	// The profile mode to pass to the language-specific adapter.
	//
	// Adapters are free to define their own profiling modes,
	// and must ignore unrecognised modes.  By convention,
	// "cpu" is used for function-level cpu profiling, "line"
	// for line-level profiling, and "mem" is used for memory
	// profiling.
	Adapter ProfileMode `json:"adapter,omitempty"`
}

type envExpander struct {
	special *strings.Replacer
	whole   map[string]string
}

func (env *envExpander) Replace(s string) string {
	if env.whole != nil {
		if r, ok := env.whole[s]; ok {
			return r
		}
	}
	return os.ExpandEnv(env.special.Replace(s))
}

func (pc *ProfileConfig) newEnvExpander(rawDest, profDest string,
	pid int) (env *envExpander) {
	env = new(envExpander)
	replacements := make([]string, 0, 6+2*len(pc.Defaults))
	env.whole = make(map[string]string, 3+len(pc.Defaults))
	if dflt := pc.Defaults; len(dflt) > 0 {
		for key, val := range dflt {
			if os.Getenv(key) == "" {
				env.whole["$"+key] = val
				replacements = append(replacements, fmt.Sprintf("${%s}", key))
				replacements = append(replacements, val)
			}
		}
	} else {
		env.whole = make(map[string]string, 3)
		replacements = make([]string, 0, 3)
	}
	replacements = append(replacements, "${PROFILE_DEST}")
	replacements = append(replacements, profDest)
	env.whole["$PROFILE_DEST"] = profDest
	replacements = append(replacements, "${RAW_PERF_DEST}")
	replacements = append(replacements, rawDest)
	env.whole["$RAW_PERF_DEST"] = rawDest
	if pid != 0 {
		pidString := strconv.Itoa(pid)
		replacements = append(replacements, "${STAGE_PID}")
		replacements = append(replacements, pidString)
		env.whole["$STAGE_PID"] = rawDest
	}
	env.special = strings.NewReplacer(replacements...)
	return
}

// ExpandedArgs returns the list of arguments to give to the
// Command, with environment variables expanded as specified
// in the documentation for Args.
func (pc *ProfileConfig) ExpandedArgs(rawDest, profDest string, pid int) []string {
	fixedArgs := make([]string, len(pc.Args))
	r := pc.newEnvExpander(rawDest, profDest, pid)
	for i, arg := range pc.Args {
		fixedArgs[i] = r.Replace(arg)
	}
	return fixedArgs
}

// MakeEnv returns an environment variable list by adding
// configured environment variables (with expansion) to the
// existing environment.  Existing environment variables will
// not be overwritten.  Returns nil if the environment is not
// altered.
func (pc *ProfileConfig) MakeEnv(rawDest, profDest string) []string {
	if len(pc.Env) == 0 {
		return nil
	}
	environ := os.Environ()
	seenKeys := make(map[string]struct{}, len(pc.Env))
	for _, pair := range environ {
		if i := strings.IndexByte(pair, '='); i != -1 {
			if _, ok := pc.Env[pair[:i]]; ok {
				seenKeys[pair[:i]] = struct{}{}
			}
		}
	}
	if len(seenKeys) < len(pc.Env) {
		r := pc.newEnvExpander(rawDest, profDest, 0)
		for k, v := range pc.Env {
			if _, ok := seenKeys[k]; !ok {
				environ = append(environ, fmt.Sprintf("%s=%s",
					k, r.Replace(v)))
			}
		}
		return environ
	} else {
		return nil
	}
}

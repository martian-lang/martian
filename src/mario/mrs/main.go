//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Mario stage runner.
//
package main

import (
	"fmt"
	"github.com/docopt/docopt.go"
	"io/ioutil"
	"mario/core"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	core.SetupSignalHandlers()

	//=========================================================================
	// Commandline argument and environment variables.
	//=========================================================================
	// Parse commandline.
	doc := `Mario Stage Runner.

Usage: 
    mrs <call.mro> <stagestance_name> [options]
    mrs -h | --help | --version

Options:
    --sched=<name>   Run jobs on custom scheduler instead of locally.
    --profile        Enable stage performance profiling.
    --debug          Enable debug logging for local scheduler. 
    -h --help        Show this message.
    --version        Show version.`
	marioVersion := core.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, marioVersion, false)
	core.LogInfo("*", "Mario Run Stage")
	core.LogInfo("version", marioVersion)
	core.LogInfo("cmdline", strings.Join(os.Args, " "))

	jobMode := "local"
	if value := os.Getenv("MROSCHED"); len(value) > 0 {
                jobMode = value
        }
	if value := opts["--sched"]; value != nil {
		jobMode = value.(string)
	}
	core.LogInfo("environ", "MROSCHED = %s", jobMode)
	core.VerifyScheduler(jobMode)

	// Compute MRO path.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPath := cwd
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPath = value
	}
	mroVersion := core.GetGitTag(mroPath)
	core.LogInfo("version", "MRO_STAGES = %s", mroVersion)

	// Compute profiling flag.
	profile := opts["--profile"].(bool)
	if value := os.Getenv("MROPROFILE"); len(value) > 0 {
		profile = true
	}

	// Setup invocation-specific values.
	invocationPath := opts["<call.mro>"].(string)
	ssid := opts["<stagestance_name>"].(string)
	stagestancePath := path.Join(cwd, ssid)
	stepSecs := 1
	debug := opts["--debug"].(bool)

	// Validate psid.
	core.DieIf(core.ValidateID(ssid))

	//=========================================================================
	// Configure Mario runtime.
	//=========================================================================
	rt := core.NewRuntime(jobMode, mroPath, marioVersion, mroVersion, profile, debug)

	// Invoke stagestance.
	data, err := ioutil.ReadFile(invocationPath)
	core.DieIf(err)
	stagestance, err := rt.InvokeStage(string(data), invocationPath, ssid, stagestancePath)
	core.DieIf(err)

	//=========================================================================
	// Start run loop.
	//=========================================================================
	go func() {
		for {
			// Refresh metadata on the node.
			stagestance.RefreshMetadata()

			// Check for completion states.
			state := stagestance.GetState()
			if state == "complete" {
				core.LogInfo("runtime", "Stage completed, exiting.")
				os.Exit(0)
			}
			if state == "failed" {
				_, errpath, _, err := stagestance.GetFatalError()
				fmt.Printf("\nStage failed, errors written to:\n%s\n\n%s\n",
					errpath, err)
				core.LogInfo("runtime", "Stage failed, exiting.")
				os.Exit(1)
			}

			// Step the node.
			stagestance.Step()

			// Wait for a bit.
			time.Sleep(time.Second * time.Duration(stepSecs))
		}
	}()

	// Let the daemons take over.
	done := make(chan bool)
	<-done
}

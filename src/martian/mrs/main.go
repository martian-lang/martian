//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian stage runner.
//
package main

import (
	"io/ioutil"
	"martian/core"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/docopt/docopt.go"
	"github.com/dustin/go-humanize"
)

func main() {
	core.SetupSignalHandlers()

	//=========================================================================
	// Commandline argument and environment variables.
	//=========================================================================
	// Parse commandline.
	doc := `Martian Stage Runner.

Usage: 
    mrs <call.mro> <stagestance_name> [options]
    mrs -h | --help | --version

Options:
    --jobmode=<name>   Run jobs on custom or local job manager.
                         Valid job managers are local, sge or .template file
                         Defaults to local.
    --vdrmode=<name>   Enables Volatile Data Removal.
                         Valid options are rolling, post and disable.
                         Defaults to rolling.
    --profile          Enable stage performance profiling.
    --localvars        Print local variables in stage code stack trace.
    --debug            Enable debug logging for local job manager.
    -h --help          Show this message.
    --version          Show version.`
	martianVersion := core.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, martianVersion, false)
	core.LogInfo("*", "Martian Run Stage")
	core.LogInfo("version", martianVersion)
	core.LogInfo("cmdline", strings.Join(os.Args, " "))

	martianFlags := ""
	if martianFlags = os.Getenv("MROFLAGS"); len(martianFlags) > 0 {
		martianOptions := strings.Split(martianFlags, " ")
		core.ParseMroFlags(opts, doc, martianOptions, []string{"call.mro", "stagestance"})
	}

	// Compute MRO path.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPath := cwd
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPath = value
	}
	mroVersion := core.GetGitTag(mroPath)
	core.LogInfo("version", "MRO_STAGES = %s", mroVersion)

	// Compute job manager.
	jobMode := "local"
	if value := opts["--jobmode"]; value != nil {
		jobMode = value.(string)
	}
	core.LogInfo("environ", "job mode = %s", jobMode)
	core.VerifyJobManager(jobMode)

	// Compute vdrMode.
	vdrMode := "rolling"
	if value := opts["--vdrmode"]; value != nil {
		vdrMode = value.(string)
	}
	core.LogInfo("environ", "vdrmode = %s", vdrMode)
	core.VerifyVDRMode(vdrMode)

	// Compute profiling flag.
	profile := opts["--profile"].(bool)

	// Compute localvars flag.
	localVars := opts["--localvars"].(bool)

	// Setup invocation-specific values.
	invocationPath := opts["<call.mro>"].(string)
	ssid := opts["<stagestance_name>"].(string)
	stagestancePath := path.Join(cwd, ssid)
	stepSecs := 1
	debug := opts["--debug"].(bool)

	// Validate psid.
	core.DieIf(core.ValidateID(ssid))

	//=========================================================================
	// Configure Martian runtime.
	//=========================================================================
	rt := core.NewRuntime(jobMode, vdrMode, mroPath, martianVersion, mroVersion, profile, localVars, debug)

	// Invoke stagestance.
	data, err := ioutil.ReadFile(invocationPath)
	core.DieIf(err)
	stagestance, err := rt.InvokeStage(string(data), invocationPath, ssid, stagestancePath)
	core.DieIf(err)

	//=========================================================================
	// Start run loop.
	//=========================================================================
	go func() {
		// Initialize state from metadata
		stagestance.LoadMetadata()
		for {
			// Refresh state on the node.
			stagestance.RefreshState()

			// Check for completion states.
			state := stagestance.GetState()
			if state == "complete" {
				stagestance.Cleanup()
				if warnings, ok := stagestance.GetWarnings(); ok {
					core.Log(warnings)
				}
				if vdrMode == "disable" {
					core.LogInfo("runtime", "VDR disabled. No files killed.")
				} else {
					core.LogInfo("runtime", "Starting VDR kill...")
					killReport := stagestance.GenerateVDRKillReport()
					core.LogInfo("runtime", "VDR killed %d files, %s.",
						killReport.Count, humanize.Bytes(killReport.Size))
				}
				core.LogInfo("runtime", "Stage completed, exiting.")
				os.Exit(0)
			}
			if state == "failed" {
				if warnings, ok := stagestance.GetWarnings(); ok {
					core.Log(warnings)
				}
				if _, errpath, log, kind, err := stagestance.GetFatalError(); kind == "assert" {
					core.Log("\n%s\n", log)
				} else {
					core.Log("\nStage failed, errors written to:\n%s\n\n%s\n",
						errpath, err)
					core.LogInfo("runtime", "Stage failed, exiting.")
				}
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

//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian stage runner.
//
package main

import (
	"io/ioutil"
	"martian/core"
	"martian/util"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/martian-lang/docopt.go"
)

func main() {
	util.SetupSignalHandlers()

	//=========================================================================
	// Commandline argument and environment variables.
	//=========================================================================
	// Parse commandline.
	doc := `Martian Stage Runner.

Usage:
    mrs <call.mro> <stagestance_name> [options]
    mrs -h | --help | --version

Options:
    --jobmode=MODE      Job manager to use. Valid options:
                            local (default), sge, lsf, or a .template file
    --localcores=NUM    Set max cores the pipeline may request at one time.
                            Only applies when --jobmode=local.
    --localmem=NUM      Set max GB the pipeline may request at one time.
                            Only applies when --jobmode=local.
    --mempercore=NUM    Specify min GB per core on your cluster.
                            Only applies in cluster jobmodes.
    --maxjobs=NUM       Set max jobs submitted to cluster at one time.
                            Only applies in cluster jobmodes.
    --jobinterval=NUM   Set delay between submitting jobs to cluster, in ms.
                            Only applies in cluster jobmodes.

    --profile=MODE      Enables stage performance profiling. Valid options:
                            disable (default), cpu, mem, or line
    --stackvars         Print local variables in stage code stack trace.
    --monitor           Kill jobs that exceed requested memory resources.
    --debug             Enable debug logging for local job manager.

    -h --help           Show this message.
    --version           Show version.`
	martianVersion := util.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, martianVersion, false)
	util.Println("Martian Single-Stage Runtime - %s", martianVersion)
	util.LogInfo("cmdline", strings.Join(os.Args, " "))

	martianFlags := ""
	if martianFlags = os.Getenv("MROFLAGS"); len(martianFlags) > 0 {
		martianOptions := strings.Split(martianFlags, " ")
		util.ParseMroFlags(opts, doc, martianOptions, []string{"call.mro", "stagestance"})
		util.LogInfo("environ", "MROFLAGS=%s", martianFlags)
	}

	// Requested cores and memory.
	reqCores := -1
	if value := opts["--localcores"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			reqCores = value
			util.LogInfo("options", "--localcores=%d", reqCores)
		}
	}
	reqMem := -1
	if value := opts["--localmem"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			reqMem = value
			util.LogInfo("options", "--localmem=%d", reqMem)
		}
	}
	reqMemPerCore := -1
	if value := opts["--mempercore"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			reqMemPerCore = value
			util.LogInfo("options", "--mempercore=%d", reqMemPerCore)
		}
	}

	// Max parallel jobs.
	maxJobs := -1
	if value := opts["--maxjobs"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			maxJobs = value
			util.LogInfo("options", "--maxjobs=%d", maxJobs)
		}
	}
	// frequency (in milliseconds) that jobs will be sent to the queue
	// (this is a minimum bound, as it may take longer to emit jobs)
	jobFreqMillis := -1
	if value := opts["--jobinterval"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			jobFreqMillis = value
			util.LogInfo("options", "--jobinterval=%d", jobFreqMillis)
		}
	}

	// Flag for full stage reset, default is chunk-granular
	fullStageReset := false
	if value := os.Getenv("MRO_FULLSTAGERESET"); len(value) > 0 {
		fullStageReset = true
		util.LogInfo("options", "MRO_FULLSTAGERESET=%v", fullStageReset)
	}

	// Special to resources mappings
	jobResources := ""
	if value := os.Getenv("MRO_JOBRESOURCES"); len(value) > 0 {
		jobResources = value
		util.LogInfo("options", "MRO_JOBRESOURCES=%s", jobResources)
	}

	// Compute MRO path.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPaths := util.ParseMroPath(cwd)
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPaths = util.ParseMroPath(value)
	}
	mroVersion, _ := util.GetMroVersion(mroPaths)
	util.LogInfo("environ", "MROPATH=%s", util.FormatMroPath(mroPaths))
	util.LogInfo("version", "MRO Version=%s", mroVersion)

	// Compute job manager.
	jobMode := "local"
	if value := opts["--jobmode"]; value != nil {
		jobMode = value.(string)
	}
	util.LogInfo("options", "--jobmode=%s", jobMode)

	// Compute profiling mode.
	profileMode := core.DisableProfile
	if value := opts["--profile"]; value != nil {
		profileMode = core.ProfileMode(value.(string))
	}
	util.LogInfo("options", "--profile=%s", profileMode)
	core.VerifyProfileMode(profileMode)

	// Compute stackvars flag.
	stackVars := opts["--stackvars"].(bool)
	util.LogInfo("options", "--stackvars=%v", stackVars)

	// Setup invocation-specific values.
	invocationPath := opts["<call.mro>"].(string)
	ssid := opts["<stagestance_name>"].(string)
	stagestancePath := path.Join(cwd, ssid)
	stepSecs := 1
	vdrMode := "disable"
	zip := false
	skipPreflight := false
	enableMonitor := opts["--monitor"].(bool)
	debug := opts["--debug"].(bool)
	envs := map[string]string{}

	// Validate psid.
	util.DieIf(util.ValidateID(ssid))

	//=========================================================================
	// Configure Martian runtime.
	//=========================================================================
	rt := core.NewRuntimeWithCores(jobMode, vdrMode, profileMode, martianVersion,
		reqCores, reqMem, reqMemPerCore, maxJobs, jobFreqMillis, jobResources,
		fullStageReset, stackVars, zip, skipPreflight, enableMonitor,
		debug, false, "", nil, false)
	rt.MroCache.CacheMros(mroPaths)

	// Invoke stagestance.
	data, err := ioutil.ReadFile(invocationPath)
	util.DieIf(err)
	stagestance, err := rt.InvokeStage(string(data), invocationPath, ssid,
		stagestancePath, mroPaths, mroVersion, envs)
	util.DieIf(err)

	// Start writing (including cached entries) to log file.
	util.LogTee(path.Join(stagestancePath, "_log"))

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
				stagestance.PostProcess()
				util.Println("Stage completed, exiting.")
				os.Exit(0)
			}
			if state == "failed" {
				if _, _, errpath, log, kind, err := stagestance.GetFatalError(); kind == "assert" {
					util.Println("\n%s\n", log)
				} else {
					util.Println("\nStage failed, errors written to:\n%s\n\n%s\n",
						errpath, err)
					util.Println("Stage failed, exiting.")
				}
				os.Exit(1)
			}

			// Check job heartbeats.
			stagestance.CheckHeartbeats()

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

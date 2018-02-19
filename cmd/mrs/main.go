//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian stage runner.
//
package main

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/martian-lang/martian/martian/core"
	"github.com/martian-lang/martian/martian/util"

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
	config := core.DefaultRuntimeOptions()
	opts, _ := docopt.Parse(doc, nil, true, config.MartianVersion, false)
	util.Println("Martian Single-Stage Runtime - %s", config.MartianVersion)
	util.LogInfo("cmdline", strings.Join(os.Args, " "))

	martianFlags := ""
	if martianFlags = os.Getenv("MROFLAGS"); len(martianFlags) > 0 {
		martianOptions := strings.Split(martianFlags, " ")
		util.ParseMroFlags(opts, doc, martianOptions, []string{"call.mro", "stagestance"})
		util.LogInfo("environ", "MROFLAGS=%s", martianFlags)
	}

	// Requested cores and memory.
	if value := opts["--localcores"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			config.LocalCores = value
			util.LogInfo("options", "--localcores=%d", config.LocalCores)
		}
	}
	if value := opts["--localmem"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			config.LocalMem = value
			util.LogInfo("options", "--localmem=%d", config.LocalMem)
		}
	}
	if value := opts["--mempercore"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			config.MemPerCore = value
			util.LogInfo("options", "--mempercore=%d", config.MemPerCore)
		}
	}

	// Max parallel jobs.
	if value := opts["--maxjobs"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			config.MaxJobs = value
			util.LogInfo("options", "--maxjobs=%d", config.MaxJobs)
		}
	}
	// frequency (in milliseconds) that jobs will be sent to the queue
	// (this is a minimum bound, as it may take longer to emit jobs)
	if value := opts["--jobinterval"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			config.JobFreqMillis = value
			util.LogInfo("options", "--jobinterval=%d", config.JobFreqMillis)
		}
	}

	// Flag for full stage reset, default is chunk-granular
	if value := os.Getenv("MRO_FULLSTAGERESET"); len(value) > 0 {
		config.FullStageReset = true
		util.LogInfo("options", "MRO_FULLSTAGERESET=true")
	}

	// Special to resources mappings
	if value := os.Getenv("MRO_JOBRESOURCES"); len(value) > 0 {
		config.ResourceSpecial = value
		util.LogInfo("options", "MRO_JOBRESOURCES=%s", config.ResourceSpecial)
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
	if value := opts["--jobmode"]; value != nil {
		config.JobMode = value.(string)
	}
	util.LogInfo("options", "--jobmode=%s", config.JobMode)

	// Compute profiling mode.
	if value := opts["--profile"]; value != nil {
		config.ProfileMode = core.ProfileMode(value.(string))
	}
	util.LogInfo("options", "--profile=%s", config.ProfileMode)
	core.VerifyProfileMode(config.ProfileMode)

	// Compute stackvars flag.
	config.StackVars = opts["--stackvars"].(bool)
	util.LogInfo("options", "--stackvars=%v", config.StackVars)

	// Setup invocation-specific values.
	invocationPath := opts["<call.mro>"].(string)
	ssid := opts["<stagestance_name>"].(string)
	stagestancePath := path.Join(cwd, ssid)
	stepSecs := 1
	config.VdrMode = "disable"
	config.Zip = false
	config.SkipPreflight = false
	config.Monitor = opts["--monitor"].(bool)
	config.Debug = opts["--debug"].(bool)
	envs := map[string]string{}

	// Validate psid.
	util.DieIf(util.ValidateID(ssid))

	//=========================================================================
	// Configure Martian runtime.
	//=========================================================================
	rt := config.NewRuntime()
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
			if state == core.Complete {
				stagestance.PostProcess()
				util.Println("Stage completed, exiting.")
				os.Exit(0)
			}
			if state == core.Failed {
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
			if !stagestance.Step() {
				// Wait for a bit.
				time.Sleep(time.Second * time.Duration(stepSecs))
			}
		}
	}()

	// Let the daemons take over.
	done := make(chan bool)
	<-done
}

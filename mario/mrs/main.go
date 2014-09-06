//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario stage runner.
//
package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"io/ioutil"
	"mario/core"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var __VERSION__ string = "<version not embedded>"

func main() {
	runtime.GOMAXPROCS(2)

	//=========================================================================
	// Commandline argument and environment variables.
	//=========================================================================
	// Parse commandline.
	doc := `Mario Stage Runner.

Usage: 
    mrs <call.mro> [<stagestance_name>] [--sge]
    mrs -h | --help | --version

Options:
    --sge         Run jobs on Sun Grid Engine instead of locally.
    --profile     Enable stage performance profiling.
    -h --help     Show this message.
    --version     Show version.`
	opts, _ := docopt.Parse(doc, nil, true, __VERSION__, false)
	core.LogInfo("*", "Mario Run Stage")
	core.LogInfo("cmdline", strings.Join(os.Args, " "))

	// Required job mode and SGE environment variables.
	jobMode := "local"
	if opts["--sge"].(bool) {
		jobMode = "sge"
		core.EnvRequire([][]string{
			{"SGE_ROOT", "path/to/sge/root"},
			{"SGE_CLUSTER_NAME", "SGE cluster name"},
			{"SGE_CELL", "usually 'default'"},
		}, true)
	}

	// Compute MRO path.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPath := cwd
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPath = value
	}

	// Compute profiling flag.
	profile := opts["--profile"].(bool)
	if value := os.Getenv("MROPROFILE"); len(value) > 0 {
		profile = true
	}

	// Setup invocation-specific values.
	invocationPath := opts["<call.mro>"].(string)
	stagestancePath := cwd
	if sid, ok := opts["<stagestance_name>"]; ok {
		stagestancePath = path.Join(stagestancePath, sid.(string))
	}
	stepSecs := 1

	//=========================================================================
	// Configure Mario runtime.
	//=========================================================================
	rt := core.NewRuntime(jobMode, mroPath, __VERSION__, profile)
	_, err := rt.CompileAll()
	core.DieIf(err)

	// Create the stagestance path.
	if _, err := os.Stat(stagestancePath); err == nil {
		core.DieIf(&core.MarioError{fmt.Sprintf("StagestanceExistsError: '%s'", stagestancePath)})
	}
	err = os.MkdirAll(stagestancePath, 0700)
	core.DieIf(err)

	// Invoke stagestance.
	callSrc, _ := ioutil.ReadFile(invocationPath)
	stagestance, err := rt.InstantiateStage(string(callSrc), stagestancePath)
	core.DieIf(err)

	//=========================================================================
	// Start run loop.
	//=========================================================================
	go func() {
		for {
			// Refresh metadata on the node.
			stagestance.Node().RefreshMetadata()

			// Check for completion states.
			state := stagestance.Node().GetState()
			if state == "complete" || state == "failed" {
				core.LogInfo("runtime", "Stage is complete, exiting.")
				os.Exit(0)
			}

			// Step the node.
			stagestance.Node().Step()

			// Wait for a bit.
			time.Sleep(time.Second * time.Duration(stepSecs))
		}
	}()

	// Let the daemons take over.
	done := make(chan bool)
	<-done
}

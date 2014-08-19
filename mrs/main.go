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
	"margo/core"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

func main() {
	runtime.GOMAXPROCS(2)

	//=========================================================================
	// Commandline argument and environment variables.
	//=========================================================================
	// Parse commandline.
	doc :=
		`Usage: 
    mrs <invocation_mro> [<unique_stagestance_id>] [--sge]
    mrs -h | --help | --version`
	opts, _ := docopt.Parse(doc, nil, true, "mrs", false)

	// Required Mario environment variables.
	env := core.EnvRequire([][]string{
		{"MARIO_PIPELINES_PATH", "path/to/pipelines"},
	}, true)

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

	// Prepare configuration variables.
	invocationPath := opts["<invocation_mro>"].(string)
	stagestancePath, _ := filepath.Abs(path.Dir(os.Args[0]))
	if sid, ok := opts["<unique_stagestance_id>"]; ok {
		stagestancePath = path.Join(stagestancePath, sid.(string))
	}
	stepSecs := 1

	//=========================================================================
	// Configure Mario runtime.
	//=========================================================================
	rt := core.NewRuntime(jobMode, env["MARIO_PIPELINES_PATH"])
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
			// Concurrently run metadata refreshes.
			var wg sync.WaitGroup
			stagestance.Node().RefreshMetadata(&wg)
			wg.Wait()

			// Check for completion states.
			state := stagestance.Node().GetState()
			if state == "complete" || state == "failed" {
				core.LogInfo("RUNTIME", "Stage is complete, exiting.")
				os.Exit(0)
			}

			// Step all nodes.
			stagestance.Node().Step()

			// Wait for a bit.
			time.Sleep(time.Second * time.Duration(stepSecs))
		}
	}()

	// Let the daemons take over.
	done := make(chan bool)
	<-done
}

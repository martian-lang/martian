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
	"time"
)

func main() {
	runtime.GOMAXPROCS(2)

	// Command-line arguments.
	doc :=
		`Usage: 
    mrs <invocation_mro> [<unique_stagestance_id>] [--sge]
    mrs -h | --help | --version`
	opts, _ := docopt.Parse(doc, nil, true, "mrs", false)

	// Mario environment variables.
	env := core.EnvRequire([][]string{
		{"MARIO_PIPELINES_PATH", "path/to/pipelines"},
	}, true)

	// Job mode and SGE environment variables.
	jobMode := "local"
	if opts["--sge"].(bool) {
		jobMode = "sge"
		core.EnvRequire([][]string{
			{"SGE_ROOT", "path/to/sge/root"},
			{"SGE_CLUSTER_NAME", "SGE cluster name"},
			{"SGE_CELL", "usually 'default'"},
		}, true)
	}

	// Compile MRO files.
	rt := core.NewRuntime(jobMode, env["MARIO_PIPELINES_PATH"])
	_, err := rt.CompileAll()
	core.DieIf(err)

	// sid, invocation file, and pipestance.
	invocationPath := opts["<invocation_mro>"].(string)
	stagestancePath, _ := filepath.Abs(path.Dir(os.Args[0]))
	if sid, ok := opts["<unique_stagestance_id>"]; ok {
		stagestancePath = path.Join(stagestancePath, sid.(string))
	}
	STEP_SECS := 1

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

	// Start the runner loop.
	done := make(chan bool)
	go func() {
		for {
			// Concurrently run metadata refreshes.
			rdone := make(chan bool)
			count := stagestance.Node().RefreshMetadata(rdone)
			for i := 0; i < count; i++ {
				<-rdone
			}

			// Check for completion states.
			state := stagestance.Node().GetState()
			if state == "complete" || state == "failed" {
				fmt.Println("[RUNTIME]", core.Timestamp(), "Stage is complete, exiting.")
				done <- true
			}

			// Step all nodes.
			stagestance.Node().Step()

			// Wait for a bit.
			time.Sleep(time.Second * time.Duration(STEP_SECS))
		}
	}()
	<-done
}

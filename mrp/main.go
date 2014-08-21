//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario pipeline runner.
//
package main

import (
	"github.com/docopt/docopt-go"
	"io/ioutil"
	"margo/core"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

//=============================================================================
// Pipestance runner.
//=============================================================================
func runLoop(pipestance *core.Pipestance, stepSecs int) {
	nodes := pipestance.Node().AllNodes()
	for {
		// Concurrently run metadata refreshes.
		//fmt.Println("===============================================================")
		//start := time.Now()
		var wg sync.WaitGroup
		wg.Add(len(nodes))
		for _, node := range nodes {
			node.RefreshMetadata(&wg)
		}
		wg.Wait()
		//fmt.Println(time.Since(start))

		// Check for completion states.
		if pipestance.GetOverallState() == "complete" {
			core.LogInfo("runtime", "Starting VDR kill...")
			killReport := pipestance.VDRKill()
			core.LogInfo("runtime", "VDR killed %d files, %d bytes.", killReport.Count, killReport.Size)
			// Give time for web ui client to get last update.
			time.Sleep(time.Second * 10)
			core.LogInfo("runtime", "Pipestance is complete, exiting.")
			os.Exit(0)
		}

		// Step all nodes.
		for _, node := range nodes {
			node.Step()
		}

		// Wait for a bit.
		time.Sleep(time.Second * time.Duration(stepSecs))
	}
}

func main() {
	runtime.GOMAXPROCS(2)

	//=========================================================================
	// Commandline argument and environment variables.
	//=========================================================================
	// Parse commandline.
	doc :=
		`Usage: 
    mrp <invocation_mro> <unique_pipestance_id> [--sge]
    mrp -h | --help | --version`
	opts, _ := docopt.Parse(doc, nil, true, "mrp", false)
	core.LogInfo("*", "Mario Run Pipeline")
	core.LogInfo("cmdline", strings.Join(os.Args, " "))

	// Required Mario environment variables.
	env := core.EnvRequire([][]string{
		{"MARIO_PORT", ">2000"},
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
	uiport := env["MARIO_PORT"]
	psid := opts["<unique_pipestance_id>"].(string)
	invocationPath := opts["<invocation_mro>"].(string)
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	pipestancePath := path.Join(cwd, psid)
	callSrc, _ := ioutil.ReadFile(invocationPath)
	stepSecs := 3

	//=========================================================================
	// Configure Mario runtime.
	//=========================================================================
	rt := core.NewRuntime(jobMode, env["MARIO_PIPELINES_PATH"])
	_, err := rt.CompileAll()
	core.DieIf(err)

	//=========================================================================
	// Invoke pipestance or Reattach if exists.
	//=========================================================================
	pipestance, err := rt.InvokeWithSource(psid, string(callSrc), pipestancePath)
	if err != nil {
		// If it already exists, try to reattach to it.
		pipestance, err = rt.Reattach(psid, pipestancePath)
		core.DieIf(err)
	}

	//=========================================================================
	// Start run loop.
	//=========================================================================
	go runLoop(pipestance, stepSecs)

	//=========================================================================
	// Start web server.
	//=========================================================================
	go runWebServer(uiport, rt, pipestance)

	// Let daemons take over.
	done := make(chan bool)
	<-done
}

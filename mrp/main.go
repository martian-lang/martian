//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario pipeline runner.
//
package main

import (
	"github.com/docopt/docopt-go"
	"github.com/dustin/go-humanize"
	"io/ioutil"
	"mario/core"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var __VERSION__ string = "<version not embedded>"

//=============================================================================
// Pipestance runner.
//=============================================================================
func runLoop(pipestance *core.Pipestance, stepSecs int, disableVDR bool) {
	nodes := pipestance.Node().AllNodes()
	for {
		// Concurrently run metadata refreshes.
		var wg sync.WaitGroup
		for _, node := range nodes {
			wg.Add(1)
			go func(node *core.Node) {
				node.RefreshMetadata()
				wg.Done()
			}(node)
		}
		wg.Wait()

		// Check for completion states.
		if pipestance.GetOverallState() == "complete" {
			if disableVDR {
				core.LogInfo("runtime", "VDR disabled by --novdr option. No files killed.")
			} else {
				core.LogInfo("runtime", "Starting VDR kill...")
				killReport := pipestance.VDRKill()
				core.LogInfo("runtime", "VDR killed %d files, %s.", killReport.Count, humanize.Bytes(killReport.Size))
			}
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
	doc := `Mario Pipeline Runner.

Usage: 
    mrp <call.mro> <pipestance_name> [options]
    mrp -h | --help | --version

Options:
    --port=<num>  Serve UI at http://localhost:<num>
                    Overrides $MROPORT environment variable.
                    Defaults to 3600 if not otherwise specified.
    --cores=<num> Maximum number of cores to use in local mode.
    --noui        Disable UI.
    --novdr       Disable Volatile Data Removal.
    --sge         Run jobs on Sun Grid Engine instead of locally.
    -h --help     Show this message.
    --version     Show version.`
	opts, _ := docopt.Parse(doc, nil, true, __VERSION__, false)
	core.LogInfo("*", "Mario Run Pipeline")
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

	// Compute UI port.
	uiport := "3600"
	if value := os.Getenv("MROPORT"); len(value) > 0 {
		core.LogInfo("environ", "MROPORT = %s", value)
		uiport = value
	}
	if value := opts["--port"]; value != nil {
		uiport = value.(string)
	}
	if opts["--noui"].(bool) {
		uiport = ""
	}

	// Requested cores.
	reqCores := 1 << 16
	if value := opts["--cores"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			reqCores = value
		}
	}

	// Compute MRO path.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPath := cwd
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPath = value
	}
	core.LogInfo("environ", "MROPATH = %s", mroPath)

	// Setup invocation-specific values.
	disableVDR := opts["--novdr"].(bool)
	psid := opts["<pipestance_name>"].(string)
	invocationPath := opts["<call.mro>"].(string)
	pipestancePath := path.Join(cwd, psid)
	callSrc, _ := ioutil.ReadFile(invocationPath)
	stepSecs := 3

	//=========================================================================
	// Configure Mario runtime.
	//=========================================================================
	rt := core.NewRuntimeWithCores(jobMode, mroPath, reqCores)
	_, err := rt.CompileAll()
	core.DieIf(err)

	//=========================================================================
	// Invoke pipestance or Reattach if exists.
	//=========================================================================
	pipestance, err := rt.InvokeWithSource(psid, string(callSrc), pipestancePath)
	if err != nil {
		if _, ok := err.(*core.PipestanceExistsError); ok {
			// If it already exists, try to reattach to it.
			pipestance, err = rt.Reattach(psid, pipestancePath)
			core.DieIf(err)
		}
		core.DieIf(err)
	}

	//=========================================================================
	// Start run loop.
	//=========================================================================
	go runLoop(pipestance, stepSecs, disableVDR)

	//=========================================================================
	// Start web server.
	//=========================================================================
	if len(uiport) > 0 {
		go runWebServer(uiport, rt, pipestance)
	} else {
		core.LogInfo("webserv", "UI disabled by --noui option.")
	}

	// Let daemons take over.
	done := make(chan bool)
	<-done
}

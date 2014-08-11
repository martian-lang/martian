//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Margo
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
	"time"
)

func main() {
	// Command-line arguments.
	doc :=
		`Usage: 
    mrp <invocation_mro> <unique_pipestance_id> [--sge]
    mrp -h | --help | --version`
	opts, _ := docopt.Parse(doc, nil, true, "mrp", false)

	// Mario environment variables.
	env := core.EnvRequire([][]string{
		{"MARIO_PORT", ">2000"},
		{"MARIO_PIPELINES_PATH", "path/to/pipelines"},
	})

	// Job mode and SGE environment variables.
	JOBMODE := "local"
	if opts["--sge"].(bool) {
		JOBMODE = "sge"
		core.EnvRequire([][]string{
			{"SGE_ROOT", "path/to/sge/root"},
			{"SGE_CLUSTER_NAME", "SGE cluster name"},
			{"SGE_CELL", "usually 'default'"},
		})
	}

	// Compile MRO files.
	rt := core.NewRuntime(JOBMODE, env["MARIO_PIPELINES_PATH"])
	_, err := rt.CompileAll()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// psid, invocation file, and pipestance.
	psid := opts["<unique_pipestance_id>"].(string)
	INVOCATION_PATH := opts["<invocation_mro>"].(string)
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	PIPESTANCE_PATH := path.Join(cwd, psid)
	callSrc, _ := ioutil.ReadFile(INVOCATION_PATH)
	os.MkdirAll(PIPESTANCE_PATH, 0700)

	// Invoke pipestance.
	pipestance, err := rt.InvokeWithSource(psid, string(callSrc), PIPESTANCE_PATH)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Start the runner loop.
	nodes := pipestance.Node().AllNodes()
	for {
		done := make(chan bool)
		count := 0
		for _, node := range nodes {
			count += node.RefreshMetadata(done)
		}
		for i := 0; i < count; i++ {
			<-done
		}
		switch pipestance.GetOverallState() {
		case "complete":
			fmt.Println("[RUNTIME]", core.Timestamp(), "Pipestance is complete, exiting.")
			os.Exit(0)
		case "failed":
			fmt.Println("[RUNTIME]", core.Timestamp(), "Pipestance failed, exiting.")
			os.Exit(1)
		}
		for _, node := range nodes {
			node.Step()
		}
		time.Sleep(time.Second * 1)
	}
}

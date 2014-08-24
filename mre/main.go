//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario MRO editor.
//
package main

import (
	"github.com/docopt/docopt-go"
	"margo/core"
	"os"
	"runtime"
	"strings"
)

func main() {
	runtime.GOMAXPROCS(2)

	//=========================================================================
	// Commandline argument and environment variables.
	//=========================================================================
	// Parse commandline.
	doc :=
		`Usage: 
    mre 
    mre -h | --help | --version`
	opts, _ := docopt.Parse(doc, nil, true, "mre", false)
	_ = opts
	core.LogInfo("*", "Mario MRO Editor")
	core.LogInfo("cmdline", strings.Join(os.Args, " "))

	// Required Mario environment variables.
	env := core.EnvRequire([][]string{
		{"MARIO_PORT", ">2000"},
		{"MARIO_PIPELINES_PATH", "path/to/pipelines"},
	}, true)

	// Prepare configuration variables.
	uiport := env["MARIO_PORT"]

	//=========================================================================
	// Configure Mario runtime.
	//=========================================================================
	rt := core.NewRuntime("local", env["MARIO_PIPELINES_PATH"])

	//=========================================================================
	// Start web server.
	//=========================================================================
	go runWebServer(uiport, rt)

	// Let daemons take over.
	done := make(chan bool)
	<-done
}

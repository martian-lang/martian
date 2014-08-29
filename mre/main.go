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
		{"MROPORT", ">2000"},
		{"MROPATH", "path/to/mros"},
	}, true)

	// Prepare configuration variables.
	uiport := env["MROPORT"]

	//=========================================================================
	// Configure Mario runtime.
	//=========================================================================
	rt := core.NewRuntime("local", env["MROPATH"])

	//=========================================================================
	// Start web server.
	//=========================================================================
	go runWebServer(uiport, rt)

	// Let daemons take over.
	done := make(chan bool)
	<-done
}

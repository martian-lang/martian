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
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

var __VERSION__ string

func main() {
	runtime.GOMAXPROCS(2)

	//=========================================================================
	// Commandline argument and environment variables.
	//=========================================================================
	// Parse commandline.
	doc := `Mario MRO Editor.

Usage:
    mre [--port=<num>]
    mre -h | --help | --version

Options:
    --port=<num>  Serve UI at http://localhost:<num>
                    Overrides $MROPORT_EDITOR environment variable.
                    Defaults to 3601 if not otherwise specified.
    --sge         Run jobs on Sun Grid Engine instead of locally.
    -h --help     Show this message.
    --version     Show version.`
	opts, _ := docopt.Parse(doc, nil, true, __VERSION__, false)
	core.LogInfo("*", "Mario MRO Editor")
	core.LogInfo("cmdline", strings.Join(os.Args, " "))

	// Compute UI port.
	uiport := "3601"
	if value := os.Getenv("MROPORT_EDITOR"); len(value) > 0 {
		core.LogInfo("environ", "MROPORT_EDITOR = %s", value)
		uiport = value
	}
	if value := opts["--port"]; value != nil {
		uiport = value.(string)
	}

	// Compute MRO path.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPath := cwd
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPath = value
	}
	core.LogInfo("environ", "MROPATH = %s", mroPath)

	//=========================================================================
	// Configure Mario runtime.
	//=========================================================================
	rt := core.NewRuntime("local", mroPath)

	//=========================================================================
	// Start web server.
	//=========================================================================
	go runWebServer(uiport, rt)

	// Let daemons take over.
	done := make(chan bool)
	<-done
}

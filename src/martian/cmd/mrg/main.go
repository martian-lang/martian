//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian command-line invocation generator.
//
package main

import (
	"encoding/json"
	"fmt"
	"github.com/martian-lang/docopt.go"
	"martian/core"
	"martian/util"
	"os"
	"path"
	"path/filepath"
)

func main() {
	util.SetupSignalHandlers()
	// Command-line arguments.
	doc := `Martian Invocation Generator.

Usage:
    mrg
    mrg -h | --help | --version

Options:
    -h --help       Show this message.
    --version       Show version.`
	martianVersion := util.GetVersion()
	docopt.Parse(doc, nil, true, martianVersion, false)

	util.ENABLE_LOGGING = false

	// Martian environment variables.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPaths := util.ParseMroPath(cwd)
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPaths = util.ParseMroPath(value)
	}

	// Setup runtime with MRO path.
	rt := core.NewRuntime("local", "disable", "disable", martianVersion)
	rt.MroCache.CacheMros(mroPaths)

	// Read and parse JSON from stdin.
	dec := json.NewDecoder(os.Stdin)
	dec.UseNumber()
	var input map[string]interface{}
	if err := dec.Decode(&input); err == nil {
		incpaths := []string{}
		if ilist, ok := input["incpaths"].([]interface{}); ok {
			incpaths = util.ArrayToString(ilist)
		}
		name, ok := input["call"].(string)
		if !ok {
			fmt.Println("No pipeline or stage specified.")
			os.Exit(1)
		}

		args, ok := input["args"].(map[string]interface{})
		if !ok {
			fmt.Println("No args given.")
			os.Exit(1)
		}

		sweepargs := []string{}
		if sweeplist, ok := input["sweepargs"].([]interface{}); ok {
			sweepargs = util.ArrayToString(sweeplist)
		}

		src, bldErr := rt.BuildCallSource(incpaths, name, args, sweepargs, mroPaths)

		if bldErr == nil {
			fmt.Print(src)
			os.Exit(0)
		} else {
			fmt.Println(bldErr)
			os.Exit(1)
		}
	}
	os.Exit(1)
}

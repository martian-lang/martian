//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario command-line compiler. Primarily used for unit testing.
//
package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"margo/core"
)

func main() {
	// Command-line arguments.
	doc :=
		`Usage: 
    mrc <mro_name>... | --all
    mrc -h | --help | --version`
	opts, _ := docopt.Parse(doc, nil, true, "mrc", false)

	// Mario environment variables.
	env := core.EnvRequire([][]string{
		{"MROPATH", "path/to/mros"},
	}, false)

	// Setup runtime with pipelines path.
	rt := core.NewRuntime("local", env["MROPATH"])

	count := 0
	if opts["--all"].(bool) {
		// Compile all MRO files in pipelines path.
		num, err := rt.CompileAll()
		core.DieIf(err)
		count += num
	} else {
		// Compile just the specified MRO files in pipeliens path.
		for _, name := range opts["<mro_name>"].([]string) {
			_, err := rt.Compile(name + ".mro")
			core.DieIf(err)
			count++
		}
	}
	fmt.Println("Successfully compiled", count, "mro files.")
}

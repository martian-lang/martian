//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Margo
//
package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"margo/core"
	"os"
)

func main() {
	// Command-line arguments.
	doc :=
		`Usage: 
    mrc <mro_name>... | --all
    mrc -h | --help | --version`
	opts, _ := docopt.Parse(doc, nil, true, "mrp", false)

	// Mario environment variables.
	env := core.EnvRequire([][]string{
		{"MARIO_PIPELINES_PATH", "path/to/pipelines"},
	})

	// Compile MRO files.
	rt := core.NewRuntime("local", env["MARIO_PIPELINES_PATH"])
	count := 0
	if opts["--all"].(bool) {
		num, err := rt.CompileAll()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		count += num
	} else {
		for _, name := range opts["<mro_name>"].([]string) {
			_, err := rt.Compile(name + ".mro")
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			count += 1
		}
	}
	fmt.Println("Successfully compiled", count, "mro files.")
}

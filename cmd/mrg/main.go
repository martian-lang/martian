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
	"github.com/martian-lang/martian/martian/core"
	"github.com/martian-lang/martian/martian/util"
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

	// Read and parse JSON from stdin.
	dec := json.NewDecoder(os.Stdin)
	dec.UseNumber()
	var input core.InvocationData
	if err := dec.Decode(&input); err == nil {
		if input.Call == "" {
			fmt.Println("No pipeline or stage specified.")
			os.Exit(1)
		}
		callable, err := core.GetCallable(mroPaths, input.Call)
		if err != nil {
			fmt.Printf("Could not find %s: %v\n", input.Call, err)
		}

		if input.Args == nil {
			fmt.Println("No args given.")
			os.Exit(1)
		}

		src, bldErr := core.BuildCallSource(
			input.Call, input.Args, input.SweepArgs,
			callable)

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

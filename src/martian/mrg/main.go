//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian command-line invocation generator.
//
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/docopt/docopt.go"
	"martian/core"
	"os"
	"path"
	"path/filepath"
)

func main() {
	// Command-line arguments.
	doc := `Martian Invocation Generator.

Usage:
    mrg 
    mrg -h | --help | --version

Options:
    -h --help       Show this message.
    --version       Show version.`
	martianVersion := core.GetVersion()
	docopt.Parse(doc, nil, true, martianVersion, false)

	core.ENABLE_LOGGING = false

	// Martian environment variables.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPath := cwd
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPath = value
	}
	mroVersion, _ := core.GetGitTag(mroPath)

	// Setup runtime with MRO path.
	rt := core.NewRuntime("local", mroPath, martianVersion, mroVersion, false, false, false)

	// Read and parse JSON from stdin.
	bio := bufio.NewReader(os.Stdin)
	if line, _, err := bio.ReadLine(); err == nil {
		var input map[string]interface{}
		if err := json.Unmarshal(line, &input); err == nil {
			incpaths := []string{}
			if ilist, ok := input["incpaths"].([]interface{}); ok {
				for _, i := range ilist {
					if incpath, ok := i.(string); ok {
						incpaths = append(incpaths, incpath)
					}
				}
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

			src, bldErr := rt.BuildCallSource(incpaths, name, args)

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
	os.Exit(1)
}

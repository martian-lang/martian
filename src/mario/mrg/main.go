//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Mario command-line invocation generator.
//
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/docopt/docopt.go"
	"mario/core"
	"os"
	"path"
	"path/filepath"
)

func main() {
	// Command-line arguments.
	doc := `Mario Invocation Generator.

Usage:
    mrg 
    mrg -h | --help | --version

Options:
    -h --help       Show this message.
    --version       Show version.`
	marioVersion := core.GetVersion()
	docopt.Parse(doc, nil, true, marioVersion, false)

	core.ENABLE_LOGGING = false

	// Mario environment variables.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPath := cwd
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPath = value
	}
	mroVersion := core.GetGitTag(mroPath)

	// Setup runtime with MRO path.
	rt := core.NewRuntime("local", mroPath, marioVersion, mroVersion, false, false)

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
			pipeline, ok := input["pipeline"].(string)
			if !ok {
				fmt.Println("No pipeline specified.")
				return
			}
			args, ok := input["args"].(map[string]interface{})
			if !ok {
				fmt.Println("No args given.")
				return
			}

            src, bldErr := rt.BuildCallSource(incpaths, pipeline, args)

            if bldErr == nil {
                fmt.Print(src)
                os.Exit(0)
            } else {
                fmt.Println(bldErr)
                os.Exit(1)
            }
		}
	}
    os.Exit(1)
}

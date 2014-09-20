//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario command-line formatter. Enforces the one true style.
//
package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"mario/core"
	"os"
	"path"
	"path/filepath"
)

var __VERSION__ string = "<version not embedded>"

func main() {
	// Command-line arguments.
	doc := `Mario Formatter.

Usage:
    mrf <file.mro>... | --all
    mrf -h | --help | --version

Options:
    --all         Format all files in $MROPATH.
    -h --help     Show this message.
    --version     Show version.`
	opts, _ := docopt.Parse(doc, nil, true, __VERSION__, false)

	// Mario environment variables.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPath := cwd
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPath = value
	}

	count := 0
	if opts["--all"].(bool) {
		// Format all MRO files in MRO path.
		paths, err := filepath.Glob(mroPath + "/*.mro")
		core.DieIf(err)
		for _, p := range paths {
			_ = p
			//core.DieIf(err)
			count += 1
		}
	} else {
		// Format just the specified MRO files.
		for _, fname := range opts["<file.mro>"].([]string) {
			fsrc, err := core.FormatFile(fname)
			core.DieIf(err)
			fmt.Println(fsrc)
			count++
		}
	}
	fmt.Println("Successfully formatted", count, "mro files.")
}

//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Mario command-line compiler. Primarily used for unit testing.
//
package main

import (
	"fmt"
	"mario/core"
	"os"
	"path"
	"path/filepath"

	"github.com/docopt/docopt-go"
)

func main() {
	// Command-line arguments.
	doc := `Mario Compiler.

Usage:
    mrc <file.mro>... [--checksrcpath]
    mrc --all [--checksrcpath]
    mrc -h | --help | --version

Options:
    --all           Compile all files in $MROPATH.
    --checksrcpath  Check that stage source paths exist.
    -h --help       Show this message.
    --version       Show version.`
	marioVersion := core.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, marioVersion, false)

	core.ENABLE_LOGGING = false

	// Mario environment variables.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPath := cwd
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPath = value
	}
	checkSrcPath := opts["--checksrcpath"].(bool)
	mroVersion := core.GetGitTag(mroPath)

	// Setup runtime with MRO path.
	rt := core.NewRuntime("local", mroPath, marioVersion, mroVersion, false, false)

	count := 0
	if opts["--all"].(bool) {
		// Compile all MRO files in MRO path.
		num, err := rt.CompileAll(checkSrcPath)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		count += num
	} else {
		// Compile just the specified MRO files.
		for _, fname := range opts["<file.mro>"].([]string) {
			_, _, err := rt.Compile(path.Join(cwd, fname), checkSrcPath)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			count++
		}
	}
	fmt.Println("Successfully compiled", count, "mro files.")
}

//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian command-line compiler. Primarily used for unit testing.
//
package main

import (
	"fmt"
	"martian/core"
	"os"
	"path"
	"path/filepath"

	"github.com/10XDev/docopt.go"
)

func main() {
	core.SetupSignalHandlers()
	// Command-line arguments.
	doc := `Martian Compiler.

Usage:
    mrc <file.mro>...
    mrc --all
    mrc -h | --help | --version

Options:
    --all           Compile all files in $MROPATH.
    -h --help       Show this message.
    --version       Show version.`
	martianVersion := core.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, martianVersion, false)

	core.ENABLE_LOGGING = false

	// Martian environment variables.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPaths := core.ParseMroPath(cwd)
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPaths = core.ParseMroPath(value)
	}
	checkSrcPath := true

	// Setup runtime with MRO path.
	rt := core.NewRuntime("local", "disable", "disable", martianVersion)

	count := 0
	if opts["--all"].(bool) {
		// Compile all MRO files in MRO path.
		num, err := rt.CompileAll(mroPaths, checkSrcPath)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		count += num
	} else {
		// Compile just the specified MRO files.
		for _, fname := range opts["<file.mro>"].([]string) {
			_, _, _, err := rt.Compile(path.Join(cwd, fname), mroPaths, checkSrcPath)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			count++
		}
	}
	fmt.Println("Successfully compiled", count, "mro files.")
}

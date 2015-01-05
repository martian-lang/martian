//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian command-line formatter. Enforces the one true style.
//
package main

import (
	"fmt"
	"github.com/docopt/docopt.go"
	"io/ioutil"
	"martian/core"
	"os"
	"path"
	"path/filepath"
)

func main() {
	// Command-line arguments.
	doc := `Martian Formatter.

Usage:
    mrf <file.mro>... [--rewrite] 
    mrf --all
    mrf -h | --help | --version

Options:
    --rewrite     Rewrite the specified file(s) in place in addition to 
                  printing reformatted source to stdout.
    --all         Rewrite all files in MROPATH.
    -h --help     Show this message.
    --version     Show version.`
	martianVersion := core.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, martianVersion, false)

	// Martian environment variables.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPath := cwd
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPath = value
	}

	if opts["--all"].(bool) {
		// Format all MRO files in MRO path.
		fnames, err := filepath.Glob(mroPath + "/*.mro")
		core.DieIf(err)
		for _, fname := range fnames {
			fsrc, err := core.FormatFile(fname)
			core.DieIf(err)
			ioutil.WriteFile(fname, []byte(fsrc), 0644)
		}
		fmt.Printf("Successfully reformatted %d files.\n", len(fnames))
	} else {
		// Format just the specified MRO files.
		for _, fname := range opts["<file.mro>"].([]string) {
			fsrc, err := core.FormatFile(fname)
			core.DieIf(err)
			fmt.Print(fsrc)
			if opts["--rewrite"].(bool) {
				ioutil.WriteFile(fname, []byte(fsrc), 0644)
			}
		}
	}
}

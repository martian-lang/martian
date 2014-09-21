//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario command-line formatter. Enforces the one true style.
//
package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
<<<<<<< HEAD
	"io/ioutil"
	"mario/core"
=======
	"mario/core"
	"os"
	"path"
	"path/filepath"
>>>>>>> FETCH_HEAD
)

var __VERSION__ string = "<version not embedded>"

func main() {
	// Command-line arguments.
	doc := `Mario Formatter.

Usage:
<<<<<<< HEAD
    mrf <file.mro>... [--rewrite]
    mrf -h | --help | --version

Options:
    --rewrite     Rewrite the specified file(s) in place in addition to 
                  printing reformatted source to stdout.
=======
    mrf <file.mro>... | --all
    mrf -h | --help | --version

Options:
    --all         Format all files in $MROPATH.
>>>>>>> FETCH_HEAD
    -h --help     Show this message.
    --version     Show version.`
	opts, _ := docopt.Parse(doc, nil, true, __VERSION__, false)

<<<<<<< HEAD
	// Format just the specified MRO files.
	for _, fname := range opts["<file.mro>"].([]string) {
		fsrc, err := core.FormatFile(fname)
		core.DieIf(err)
		fmt.Print(fsrc)
		if opts["--rewrite"].(bool) {
			ioutil.WriteFile(fname, []byte(fsrc), 0600)
=======
	// Mario environment variables.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPath := cwd
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPath = value
	}

	if opts["--all"].(bool) {
		// Format all MRO files in MRO path.
		paths, err := filepath.Glob(mroPath + "/*.mro")
		core.DieIf(err)
		for _, p := range paths {
			_ = p
			//core.DieIf(err)
		}
	} else {
		// Format just the specified MRO files.
		for _, fname := range opts["<file.mro>"].([]string) {
			fsrc, err := core.FormatFile(fname)
			core.DieIf(err)
			fmt.Println(fsrc)
>>>>>>> FETCH_HEAD
		}
	}
}

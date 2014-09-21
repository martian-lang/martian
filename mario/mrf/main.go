//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario command-line formatter. Enforces the one true style.
//
package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"io/ioutil"
	"mario/core"
)

var __VERSION__ string = "<version not embedded>"

func main() {
	// Command-line arguments.
	doc := `Mario Formatter.

Usage:
    mrf <file.mro>... [--rewrite]
    mrf -h | --help | --version

Options:
    --rewrite     Rewrite the specified file(s) in place in addition to 
                  printing reformatted source to stdout.
    -h --help     Show this message.
    --version     Show version.`
	opts, _ := docopt.Parse(doc, nil, true, __VERSION__, false)

	// Format just the specified MRO files.
	for _, fname := range opts["<file.mro>"].([]string) {
		fsrc, err := core.FormatFile(fname)
		core.DieIf(err)
		fmt.Print(fsrc)
		if opts["--rewrite"].(bool) {
			ioutil.WriteFile(fname, []byte(fsrc), 0600)
		}
	}
}

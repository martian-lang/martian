//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//

// Martian command-line code formatter for MRO files.
//
// Most of the time, it is invoked with a command line such as
//
//	mrf *.mrp --rewrite
//
// mrf is an opinionated code formatter, meaning its style output is not
// configurable.  This is a deliberate choice.  By preventing users from
// making different style choices, pointless whitespace-only diffs should
// be prevented and arguments about style can be avoided.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/martian-lang/docopt.go"
	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

func main() {
	util.SetPrintLogger(os.Stderr)
	util.SetupSignalHandlers()
	// Command-line arguments.
	doc := `Martian Formatter.

Usage:
    mrf [--rewrite] [--includes] <file.mro>...
    mrf --all [--includes]
    mrf -h | --help | --version

Options:
    --rewrite     Rewrite the specified file(s) in place.
    --includes    Add and remove includes as appropriate.
    --all         Rewrite all files in MROPATH.
    -h --help     Show this message.
    --version     Show version.`
	martianVersion := util.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, martianVersion, false)

	// Martian environment variables.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPaths := util.ParseMroPath(cwd)
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPaths = util.ParseMroPath(value)
	}

	fixIncludes := opts["--includes"].(bool)
	if opts["--all"].(bool) {
		// Format all MRO files in MRO path.
		fileNames := make([]string, 0, len(mroPaths)*3)
		for _, mroPath := range mroPaths {
			fnames, err := filepath.Glob(mroPath + "/*.mro")
			util.DieIf(err)
			fileNames = append(fileNames, fnames...)
		}
		var parser syntax.Parser
		for _, fname := range fileNames {
			fsrc, err := parser.FormatFile(fname, fixIncludes, mroPaths)
			util.DieIf(err)
			ioutil.WriteFile(fname, []byte(fsrc), 0644)
		}
		fmt.Printf("Successfully reformatted %d files.\n", len(fileNames))
	} else {
		// Format just the specified MRO files.
		for _, fname := range opts["<file.mro>"].([]string) {
			fsrc, err := syntax.FormatFile(fname, fixIncludes, mroPaths)
			util.DieIf(err)
			if opts["--rewrite"].(bool) {
				ioutil.WriteFile(fname, []byte(fsrc), 0644)
			} else {
				fmt.Print(fsrc)
			}
		}
	}
}

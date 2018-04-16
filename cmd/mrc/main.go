//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian command-line compiler. Primarily used for unit testing.
//
package main

import (
	"fmt"
	"github.com/martian-lang/martian/martian/core"
	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/martian-lang/docopt.go"
)

func main() {
	util.SetupSignalHandlers()
	// Command-line arguments.
	doc := `Martian Compiler.

Usage:
    mrc [options] <file.mro>...
    mrc [options]
    mrc -h | --help | --version

Options:
    --all           Compile all files in $MROPATH.
    --json          Output abstract syntax tree as JSON.
    --strict        Strict syntax validation
    --no-check-src  Do not check that stage source paths exist.

    -h --help       Show this message.
    --version       Show version.`
	martianVersion := util.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, martianVersion, false)

	// Martian environment variables.
	cwd, _ := os.Getwd()
	mroPaths := util.ParseMroPath(cwd)
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPaths = util.ParseMroPath(value)
	}
	checkSrcPath := true
	if opts["--no-check-src"].(bool) {
		checkSrcPath = false
	}

	// Setup strictness
	syntax.SetEnforcementLevel(syntax.EnforceLog)
	if opts["--strict"].(bool) {
		syntax.SetEnforcementLevel(syntax.EnforceError)
	} else if flags := os.Getenv("MROFLAGS"); flags != "" {
		re := regexp.MustCompile(`-strict=(log|alarm|error)`)
		if match := re.FindStringSubmatch(flags); len(match) > 1 {
			syntax.SetEnforcementLevel(syntax.ParseEnforcementLevel(match[1]))
		}
	}
	mkjson := opts["--json"].(bool)

	count := 0
	if opts["--all"].(bool) {
		// Compile all MRO files in MRO path.
		num, asts, err := core.CompileAll(mroPaths, checkSrcPath)

		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		if mkjson {
			fmt.Printf("%s", syntax.JsonDumpAsts(asts))
		}

		count += num
	} else {
		// Compile just the specified MRO files.
		var asts []*syntax.Ast
		for _, fname := range opts["<file.mro>"].([]string) {
			if !filepath.IsAbs(fname) {
				fname = path.Join(cwd, fname)
			}
			_, _, ast, err := syntax.Compile(fname, mroPaths, checkSrcPath)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
			} else {
				if mkjson {
					asts = append(asts, ast)
				}
				count++
			}
		}
		if mkjson {
			fmt.Printf("%s\n", syntax.JsonDumpAsts(asts))
		}
	}
	fmt.Fprintln(os.Stderr, "Successfully compiled", count, "mro files.")
}

//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian command-line invocation generator.
//
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/martian-lang/docopt.go"
	"github.com/martian-lang/martian/martian/core"
	"github.com/martian-lang/martian/martian/util"
)

func main() {
	util.SetupSignalHandlers()
	// Command-line arguments.
	doc := `Martian Invocation Generator.

Usage:
    mrg
    mrg --reverse
    mrg -h | --help | --version

Options:
    --reverse       Generate invocation data from mro source.
    -h --help       Show this message.
    --version       Show version.`
	martianVersion := util.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, martianVersion, false)

	util.ENABLE_LOGGING = false

	// Martian environment variables.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPaths := util.ParseMroPath(cwd)
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPaths = util.ParseMroPath(value)
	}

	if opts["--reverse"].(bool) {
		src, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		invocation, err := core.InvocationDataFromSource(src, mroPaths)
		if err != nil {
			os.Stderr.WriteString("Error parsing source: ")
			os.Stderr.WriteString(err.Error())
			os.Stderr.WriteString("\n")
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(invocation); err != nil {
			os.Stderr.WriteString("Error generating json: ")
			os.Stderr.WriteString(err.Error())
			os.Stderr.WriteString("\n")
			os.Exit(1)
		}
	}

	// Read and parse JSON from stdin.
	dec := json.NewDecoder(os.Stdin)
	dec.UseNumber()
	var input core.InvocationData
	if err := dec.Decode(&input); err == nil {
		src, bldErr := input.BuildCallSource(mroPaths)

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

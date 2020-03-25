// Command mro serves as a front-end for various tools for modifying and
// analyzing mro files.
package main

import (
	"fmt"
	"os"
	"path"

	"github.com/martian-lang/martian/cmd/mro/check"
	"github.com/martian-lang/martian/cmd/mro/edit"
	"github.com/martian-lang/martian/cmd/mro/format"
)

const usage = "Usage: mro [help] [check | edit | format] ..."

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
	// Support running from a symlink from the old tool name.
	switch path.Base(os.Args[0]) {
	case "mrc":
		check.Main(os.Args[1:])
	case "mrdr":
		edit.Main(os.Args[1:])
	case "mrf":
		format.Main(os.Args[1:])
	}
	if os.Args[1] == "help" {
		if len(os.Args) == 2 {
			fmt.Fprintln(os.Stderr, usage+`

	check:
		Perform static analysis tasks.

	edit:
		Perform various refactoring tasks.

	format:
		Reformat an mro file according to the canonical formatting.
`)
		} else {
			delegateMain(append([]string{os.Args[2], "--help"}, os.Args[3:]...))
		}
	}
	delegateMain(os.Args[1:])
}

func delegateMain(argv []string) {
	switch os.Args[1] {
	case "check":
		check.Main(os.Args[1:])
	case "edit":
		check.Main(os.Args[1:])
	case "format":
		check.Main(os.Args[1:])
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
}

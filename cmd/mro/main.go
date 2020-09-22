// Command mro serves as a front-end for various tools for modifying and
// analyzing mro files.
package main

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"runtime/trace"

	"github.com/martian-lang/martian/cmd/mro/check"
	"github.com/martian-lang/martian/cmd/mro/edit"
	"github.com/martian-lang/martian/cmd/mro/format"
	"github.com/martian-lang/martian/cmd/mro/graph"
	"github.com/martian-lang/martian/martian/util"
)

const usage = "Usage: mro [help] [check | edit | format | graph] ..."

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
	// Support running from a symlink from the old tool name.
	switch path.Base(os.Args[0]) {
	case "mrc":
		check.Main(os.Args[1:])
		os.Exit(0)
	case "mrdr":
		edit.Main(os.Args[1:])
		os.Exit(0)
	case "mrf":
		format.Main(os.Args[1:])
		os.Exit(0)
	}
	switch os.Args[1] {
	case "help":
		if len(os.Args) == 2 {
			fmt.Fprintln(os.Stderr, usage+`

	check:
		Perform static analysis tasks.

	edit:
		Perform various refactoring tasks.

	format:
		Reformat an mro file according to the canonical formatting.

	graph:
		Render a call graph, or query information about it.

	version:
		Print the version and exit.
`)
		} else {
			delegateMain(append([]string{os.Args[2], "--help"}, os.Args[3:]...))
			os.Exit(0)
		}
	case "version", "--version":
		fmt.Println(util.GetVersion())
		os.Exit(0)
	}
	delegateMain(os.Args[1:])
	os.Exit(0)
}

func delegateMain(argv []string) {
	switch argv[0] {
	case "check":
		check.Main(argv[1:])
	case "edit":
		edit.Main(argv[1:])
	case "format":
		format.Main(argv[1:])
	case "graph":
		graph.Main(argv[1:])
	case "-cpuprofile":
		cpuProfile(argv[1], argv[2:])
	case "-memprofile":
		memProfile(argv[1], argv[2:])
	case "-trace":
		traceProf(argv[1], argv[2:])
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
}

func cpuProfile(dest string, argv []string) {
	f, err := os.Create(dest)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not create CPU profile: ", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		fmt.Fprintln(os.Stderr, "could not start CPU profile: ", err)
		os.Exit(1)
	}
	delegateMain(argv)
	pprof.StopCPUProfile()
	if err := f.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "could not close cpu profile: ", err)
	}
}

func memProfile(dest string, argv []string) {
	delegateMain(argv)
	runtime.GC()
	f, err := os.Create(dest)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not create memory profile: ", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := pprof.WriteHeapProfile(f); err != nil {
		fmt.Fprintln(os.Stderr, "could not write memory profile: ", err)
	} else if err := f.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "could not close memory profile: ", err)
	}
}

func traceProf(dest string, argv []string) {
	f, err := os.Create(dest)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not create trace: ", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := trace.Start(f); err != nil {
		fmt.Fprintln(os.Stderr, "could not start trace: ", err)
		os.Exit(1)
	}
	delegateMain(argv)
	trace.Stop()
	if err := f.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "could not close trace: ", err)
	}
}

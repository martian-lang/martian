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
	"strings"

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
		os.Exit(check.Main(os.Args[1:]))
	case "mrdr":
		os.Exit(edit.Main(os.Args[1:]))
	case "mrf":
		os.Exit(format.Main(os.Args[1:]))
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
		Print the version and exit.`)
		} else {
			os.Exit(delegateMain(append([]string{
				os.Args[2], "--help"}, os.Args[3:]...)))
		}
	case "version", "--version":
		fmt.Println(util.GetVersion())
		os.Exit(0)
	}
	os.Exit(delegateMain(os.Args[1:]))
}

func delegateMain(argv []string) int {
	switch argv[0] {
	case "check":
		return check.Main(argv[1:])
	case "edit":
		return edit.Main(argv[1:])
	case "format":
		return format.Main(argv[1:])
	case "graph":
		return graph.Main(argv[1:])
	case "-cpuprofile":
		return cpuProfile(argv[1], argv[2:])
	case "-memprofile":
		return memProfile(argv[1], argv[2:])
	case "-trace":
		return traceProf(argv[1], argv[2:])
	default:
		fmt.Fprintln(os.Stderr, usage)
		return 1
	}
}

// For purposese of automating profile collection, if there is a * in
// the destination profile name, make a new temp file, so we don't have to
// specify a fresh filename for every run.
func openMaybeTemp(dest string) (*os.File, error) {
	dir, name := path.Split(dest)
	if strings.ContainsRune(name, '*') {
		return os.CreateTemp(dir, name)
	} else {
		return os.Create(dest)
	}
}

func cpuProfile(dest string, argv []string) int {
	f, err := openMaybeTemp(dest)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not create CPU profile: ", err)
		return 1
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		fmt.Fprintln(os.Stderr, "could not start CPU profile: ", err)
		return 1
	}
	returnCode := delegateMain(argv)
	pprof.StopCPUProfile()
	if err := f.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "could not close cpu profile: ", err)
	}
	return returnCode
}

func memProfile(dest string, argv []string) int {
	returnCode := delegateMain(argv)
	runtime.GC()
	f, err := openMaybeTemp(dest)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not create memory profile: ", err)
		return 1
	}
	defer f.Close()
	if err := pprof.WriteHeapProfile(f); err != nil {
		fmt.Fprintln(os.Stderr, "could not write memory profile: ", err)
	} else if err := f.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "could not close memory profile: ", err)
	}
	return returnCode
}

func traceProf(dest string, argv []string) int {
	f, err := openMaybeTemp(dest)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not create trace: ", err)
		return 1
	}
	defer f.Close()
	if err := trace.Start(f); err != nil {
		fmt.Fprintln(os.Stderr, "could not start trace: ", err)
		return 1
	}
	returnCode := delegateMain(argv)
	trace.Stop()
	if err := f.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "could not close trace: ", err)
	}
	return returnCode
}

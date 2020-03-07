// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

// Martian command-line code refactoring tool.
//
// mrdr makes semantic edits to pipelines, including removing input parameters,
// eliminating unused calls, and so on.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/syntax/refactoring"
	"github.com/martian-lang/martian/martian/util"
)

// stringListValue wraps a refactoring.StringSet to implement flag.Value
type stringListValue struct {
	set *refactoring.StringSet
}

func (s stringListValue) String() string {
	if s.set == nil || len(*s.set) == 0 {
		return ""
	}
	items := make([]string, 0, len(*s.set))
	for i := range *s.set {
		items = append(items, i)
	}
	sort.Strings(items)
	return strings.Join(items, ",")
}

func (s stringListValue) Set(v string) error {
	if len(v) == 0 && s.set != nil && len(*s.set) != 0 {
		for k := range *s.set {
			delete(*s.set, k)
		}
		return nil
	}
	vals := strings.Split(v, ",")
	if *s.set == nil {
		if len(vals) != 0 {
			*s.set = make(refactoring.StringSet, len(vals))
		}
	} else {
		for k := range *s.set {
			delete(*s.set, k)
		}
	}
	for _, val := range vals {
		(*s.set)[val] = struct{}{}
	}
	return nil
}

func main() {
	util.SetPrintLogger(os.Stderr)
	util.SetupSignalHandlers()

	var flags flag.FlagSet
	flags.Init("mrdr", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(),
			"Usage: mrdr [options] <file1.mro> [<file2.mro>...]")
		fmt.Fprintln(flags.Output())
		flags.PrintDefaults()
	}

	var removeParams, topCalls refactoring.StringSet
	var listUnused, removeCalls, rewrite bool
	flags.Var(stringListValue{set: &removeParams}, "remove-input",
		"Remove an input parameter from a stage, e.g. `STAGE.input_name`."+
			"  Multiple parameters may be provided, separated with commas.")
	flags.BoolVar(&removeCalls, "remove-unused-calls", false,
		"Remove calls in pipelines if the called stage or pipeline "+
			"has outputs but none of them are used.")
	flags.Var(stringListValue{set: &topCalls}, "top-calls",
		"A comma-separated `list` of pipeline names to treat as top-level "+
			"calls for unused output analysis.")
	flags.BoolVar(&listUnused, "list-unused", false,
		"Print a list of stages or pipelines which are not called "+
			"from any of the given top-calls.")
	flags.BoolVar(&rewrite, "rewrite", false,
		"Write the modified content back to the original file.")
	flags.BoolVar(&rewrite, "w", rewrite,
		"Write the modified content back to the original file.")
	version := flags.Bool("v", false, "Print the version and exit.")
	if err := flags.Parse(os.Args[1:]); err != nil {
		panic(err)
	}
	if *version {
		fmt.Println(util.GetVersion())
		os.Exit(0)
	}

	if flags.NArg() < 1 {
		flags.Usage()
		os.Exit(1)
	}

	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPaths := util.ParseMroPath(cwd)
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPaths = util.ParseMroPath(value)
	}

	var parser syntax.Parser
	fileBytes, compiledAsts := loadFiles(flags.Args(), mroPaths, &parser)

	edit, err := refactoring.Refactor(compiledAsts,
		topCalls,
		validateParams(removeParams, &flags),
		removeCalls)
	if err != nil {
		fmt.Fprintln(flags.Output(),
			err.Error())
		os.Exit(10)
	}

	if listUnused && len(topCalls) > 0 {
		unused := refactoring.FindUnusedCallables(topCalls, compiledAsts)
		pwd, _ := os.Getwd()
		if pwd != "" {
			pwd += "/"
		}
		width := 0
		for _, c := range unused {
			if w := len(c.GetId()); w > width {
				width = w
			}
		}
		if width > 30 {
			width = 30
		}
		for _, c := range unused {
			var p string
			if len(mroPaths) < 2 {
				// Use path relative to MROPATH or current directory.
				p, _, _ = syntax.IncludeFilePath(c.File().FullPath, mroPaths)
			} else {
				// Use absolute path or path relative to current directory.
				p = c.File().FullPath
				if strings.HasPrefix(p, pwd) {
					p = p[len(pwd):]
				}
			}
			fmt.Fprintf(os.Stderr, "Unused %-8s %-*s defined at %s:%d\n",
				c.Type(), width, c.GetId(), p, c.Line())
		}
	}

	if edit != nil {
		for i, fname := range flags.Args() {
			editFile(fileBytes[i], fname, mroPaths, edit, rewrite, &parser)
		}
	}
}

func loadFiles(names, mroPaths []string, parser *syntax.Parser) ([][]byte, []*syntax.Ast) {
	fileBytes := make([][]byte, len(names))
	var compiledAsts []*syntax.Ast
	var fail bool
	for i, fname := range names {
		var ast *syntax.Ast
		var err error
		if fname == "-" {
			fileBytes[i], err = ioutil.ReadAll(os.Stdin)
			fname = "standard input"
		} else {
			fileBytes[i], err = ioutil.ReadFile(fname)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr,
				"Error reading from %s: %s\n",
				fname, err.Error())
			fail = true
		} else if _, _, ast, err = parser.ParseSourceBytes(
			fileBytes[i], fname, mroPaths, false); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %s\n",
				fname, err.Error())
			fail = true
		} else {
			compiledAsts = append(compiledAsts, ast)
		}
	}
	if fail {
		os.Exit(3)
	}
	return fileBytes, compiledAsts
}

func validateParams(params refactoring.StringSet, flags *flag.FlagSet) []refactoring.CallableParam {
	result := make([]refactoring.CallableParam, 0, len(params))
	for param := range params {
		i := strings.IndexByte(param, '.')
		if i < 1 {
			fmt.Fprintln(flags.Output(),
				"Parameter name must be specified as PIPELINE.input_name")
			flags.Usage()
			os.Exit(4)
		}
		result = append(result, refactoring.CallableParam{
			Callable: param[:i],
			Param:    param[i+1:],
		})
	}
	return result
}

func editFile(data []byte, filename string, mroPaths []string,
	edit refactoring.Edit, rewrite bool, parser *syntax.Parser) {
	ast, err := parser.UncheckedParse(data, filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %s\n",
			filename, err.Error())
		return
	}
	count, err := edit.Apply(ast)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error appying edit to %s: %s\n",
			filename, err.Error())
	}
	if count == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, count, "edits to", filename)
	if rewrite && filename != "-" {
		dir, base := filepath.Split(filename)
		if dir == "" {
			dir = "."
		}
		f, err := ioutil.TempFile(dir, base)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening temporary file for %s: %s\n",
				filename, err.Error())
			return
		}
		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()
		if _, err := f.WriteString(ast.Format()); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing updated source for %s: %s\n",
				filename, err.Error())
			return
		}
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing updated source file for %s: %s\n",
				filename, err.Error())
			return
		}
		if err := os.Rename(f.Name(), filename); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error renaming updated source file for %s: %s\n",
				filename, err.Error())
		}
	} else {
		fmt.Println(ast.Format())
	}
}

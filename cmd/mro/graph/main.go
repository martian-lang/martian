// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Package check implements the command line interface for querying and
// extracting static call graph information from pipeline definitions.
package graph

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/syntax/graph"
	"github.com/martian-lang/martian/martian/util"
)

func Main(argv []string) {
	util.SetPrintLogger(os.Stderr)
	syntax.SetEnforcementLevel(syntax.EnforceError)

	var flags flag.FlagSet
	flags.Init("mro graph", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(),
			"Usage: mro graph [options] <file1.mro>]")
		fmt.Fprintln(flags.Output())
		flags.PrintDefaults()
	}

	var asJson, asDot bool
	flags.BoolVar(&asJson, "json", false,
		"Render the call graph as json.")
	flags.BoolVar(&asDot, "dot", false,
		"Render the call graph in graphviz dot format.")
	var stageInput, stageOutput string
	flags.StringVar(&stageInput, "trace-input", "",
		"Show the resolved inputs to the given `STAGE`.")
	flags.StringVar(&stageOutput, "trace-output", "",
		"List any input parameters to any stages which resolve "+
			"to the given `STAGE.output`")
	if err := flags.Parse(argv); err != nil {
		panic(err)
	}

	cg, lookup := getGraph(flags.Arg(0))
	if stageInput != "" || stageOutput != "" {
		if asJson || asDot {
			fmt.Fprintln(flags.Output(),
				"Cannot render input/output traces as json or dot.")
			flags.Usage()
			os.Exit(1)
		}
		if stageInput != "" {
			if !traceInput(stageInput, cg, lookup) {
				fmt.Fprintln(os.Stderr, "Callable", stageInput, "not found")
			}
		}
		if stageOutput != "" {
			if !traceOutput(stageOutput, cg) {
				fmt.Fprintln(os.Stderr, "Callable", stageOutput, "not found")
			}
		}
		os.Exit(0)
	}
	pcg, ok := cg.(*syntax.CallGraphPipeline)
	if !ok {
		fmt.Fprintln(os.Stderr,
			"Best call found was not a pipeline:",
			cg.GetFqid())
		os.Exit(1)
	}
	if asDot {
		if asJson {
			fmt.Fprintln(flags.Output(),
				"Cannot render both json and dot.")
			flags.Usage()
			os.Exit(1)
		}
		renderDot(pcg)
		os.Exit(0)
	}
	renderJson(pcg)
	os.Exit(0)
}

func getGraph(fname string) (syntax.CallGraphNode, *syntax.TypeLookup) {
	cwd, _ := os.Getwd()
	mroPaths := util.ParseMroPath(cwd)
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPaths = util.ParseMroPath(value)
	}
	_, _, ast, err := syntax.Compile(fname, mroPaths, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}
	call := getBestCall(ast)
	if call == nil {
		fmt.Fprintln(os.Stderr, "No callable objects found.")
		os.Exit(3)
	}
	cg, err := ast.MakeCallGraph("", call)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error building call graph:", err.Error())
		os.Exit(3)
	}
	return cg, &ast.TypeTable
}

// If the AST has a call, return it.  Otherwise, return the last
// callable defined in the top-level file.
func getBestCall(ast *syntax.Ast) *syntax.CallStm {
	if ast.Call != nil {
		return ast.Call
	}
	var found syntax.Callable
	for _, c := range ast.Pipelines {
		if f := c.File(); f == nil || len(f.IncludedFrom) == 0 {
			found = c
		}
	}
	if found == nil {
		for _, c := range ast.Stages {
			if f := c.File(); f == nil || len(f.IncludedFrom) == 0 {
				found = c
			}
		}
	}
	if found == nil && len(ast.Callables.List) > 0 {
		found = ast.Callables.List[len(ast.Callables.List)-1]
	}
	if found == nil {
		return nil
	}
	return syntax.GenerateAbstractCall(found, &ast.TypeTable)
}

// Given a string PIPELINE.INNER_PIPELINE.STAGE.output.member, attempt to
// return the call graph node for PIPELINE.INNER_PIPELINE.STAGE, along with
// the tail component output.member.  Note that it uses greedy matching so if
// INNER_PIPELINE also had a call named `output` there would be ambiguity about
// which was meant; the algorithm has no backtracking so descending into child
// nodes has priority.
//
// A nil value for the returned node indicates that no match was found.
func findNode(id string, cg syntax.CallGraphNode) (string, syntax.CallGraphNode) {
	for _, child := range cg.GetChildren() {
		if tail, node := findNode(id, child); node != nil {
			return tail, node
		}
	}
	if id == cg.GetFqid() {
		return "", cg
	}
	if strings.HasPrefix(id, cg.GetFqid()) {
		tail := strings.TrimPrefix(id, cg.GetFqid())
		// tail has the prefix but is not equal to the fqid, so it must have
		// at least one character.
		if tail[0] == '.' {
			//trim off leading .
			return tail[1:], cg
		}
	}
	return "", nil
}

func traceInput(stage string, cg syntax.CallGraphNode, lookup *syntax.TypeLookup) bool {
	param, cg := findNode(stage, cg)
	if cg == nil {
		return false
	}
	if param == "" {
		for k, v := range cg.ResolvedInputs() {
			t := v.Type.TypeId()
			fmt.Print(cg.GetFqid(), ".", k, " (", t.String(), ") = ",
				syntax.FormatExp(v.Exp, "\t"))
			fmt.Println()
		}
		return true
	}
	i := strings.IndexByte(param, '.')
	var path string
	if i >= 0 {
		path = param[i+1:]
		param = param[:i]
	}
	rb := cg.ResolvedInputs()[param]
	if rb == nil {
		fmt.Fprintln(os.Stderr, "No input", param, "in", cg.GetFqid())
		os.Exit(4)
	}
	exp := rb.Exp
	if path != "" {
		if rp, err := exp.BindingPath(path, nil, lookup); err != nil {
			fmt.Fprint(os.Stderr, "Invalid path ", path, " in ",
				stage, ": ", err.Error())
		} else {
			exp = rp
		}
	}
	fmt.Println(syntax.FormatExp(exp, ""))
	return true
}

// Returns true if either ID is a reference to the other, to a member of the
// other, or to a member containing the other.
func tailMatch(id1, id2 string) bool {
	if id1 == id2 {
		return true
	}
	if id1 == "" || id2 == "" {
		return true
	}
	if len(id1) < len(id2) {
		return strings.HasPrefix(id2, id1) && id2[len(id1)] == '.'
	} else if len(id2) < len(id1) {
		return strings.HasPrefix(id1, id2) && id1[len(id2)] == '.'
	}
	return false
}

func traceOutput(stage string, cg syntax.CallGraphNode) bool {
	tail, source := findNode(stage, cg)
	if source == nil {
		return false
	}
	traceOutputResolved(tail, source, cg)
	return true
}

func traceOutputResolved(tail string, source, cg syntax.CallGraphNode) {
	if cg == source {
		// Don't descend into children of the selected pipeline.  They can't
		// possibly depend on the pipeline's outputs.
		return
	}
	switch cg := cg.(type) {
	case *syntax.CallGraphStage:
		for k, v := range cg.ResolvedInputs() {
			refs := v.Exp.FindRefs()
			if len(refs) > 0 {
				var mentioned map[string]struct{}
				for _, ref := range refs {
					if ref.Id == source.GetFqid() && tailMatch(ref.OutputId, tail) {
						id := ref.GoString()
						if _, ok := mentioned[id]; !ok {
							if mentioned == nil {
								mentioned = make(map[string]struct{})
							}
							mentioned[id] = struct{}{}
							fmt.Println("Stage", cg.GetFqid(), "input", k,
								"depends on", id)
						}
					}
				}
			}
		}
	case *syntax.CallGraphPipeline:
		for _, child := range cg.Children {
			traceOutputResolved(tail, source, child)
		}
		outputs := cg.ResolvedOutputs()

		if outputs != nil {
			refs := outputs.Exp.FindRefs()
			if len(refs) > 0 {
				var mentioned map[string]struct{}
				for _, ref := range refs {
					if ref.Id == source.GetFqid() && tailMatch(ref.OutputId, tail) {
						id := ref.GoString()
						if _, ok := mentioned[id]; !ok {
							if mentioned == nil {
								mentioned = make(map[string]struct{})
							}
							mentioned[id] = struct{}{}
							fmt.Println("Pipeline", cg.GetFqid(),
								"outputs depend on", id)
						}
					}
				}
			}
		}
	}
}

func renderDot(pcg *syntax.CallGraphPipeline) {
	if err := graph.RenderDot(pcg, os.Stdout, "", "  "); err != nil {
		fmt.Fprintln(os.Stderr, "Error rendering dot:", err.Error())
		os.Exit(4)
	}
}

func renderJson(pcg *syntax.CallGraphPipeline) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	// This is frequently enough for human consumption that we don't want the
	// type names to be mangled.
	enc.SetEscapeHTML(false)
	if err := enc.Encode(pcg); err != nil {
		fmt.Fprintln(os.Stderr, "Error rendering json:", err.Error())
		os.Exit(4)
	}
}

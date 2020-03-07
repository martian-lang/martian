// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"fmt"
	"os"

	"github.com/martian-lang/martian/martian/syntax"
)

// Finds calls which have outputs but for which none of the outputs are bound,
// either to a pipeline output, another call input, or to a retain.
//
// The pipeline must be fully compiled.
//
// The set asts must be fully compiled, but are only required in order to remove
// parameter bindings for calls to this pipeline with inputs which are no longer
// required as a result of removing the calls.
func RemoveUnusedCalls(pipe *syntax.Pipeline, asts []*syntax.Ast) Edit {
	return removeUnusedCalls(pipe, asts)
}

// Create an edit for removing unused calls for all pipelines in the given asts.
func RemoveAllUnusedCalls(asts []*syntax.Ast) Edit {
	if len(asts) == 0 {
		return nil
	}
	var edits editSet
	pipelines := make(map[decId]struct{}, len(asts)*(len(asts[0].Pipelines)+1))
	for _, ast := range asts {
		for _, pipe := range ast.Pipelines {
			id := makeDecId(pipe)
			if _, ok := pipelines[id]; !ok {
				pipelines[id] = struct{}{}
				newEdits := removeUnusedCalls(pipe, asts)
				if len(edits) == 0 {
					edits = newEdits
				} else if len(newEdits) > 0 {
					edits = append(edits, newEdits...)
				}
			}
		}
	}
	return edits
}

// Returns true if the callable is expected to have side effects.
//
// Even if the outputs don't make it out of the pipeline, some of them may be
// retained, in which case the call must be kept.
//
// Preflights should always be kept.
func hasSideEffects(c syntax.Callable) bool {
	if c == nil {
		return false
	}
	switch c := c.(type) {
	case *syntax.Stage:
		if c.Retain != nil && len(c.Retain.Params) != 0 {
			return true
		}
	case *syntax.Pipeline:
		if c.Retain != nil && len(c.Retain.Refs) != 0 {
			return true
		}
		for _, call := range c.Calls {
			if call.Modifiers != nil && call.Modifiers.Preflight {
				return true
			}
			if sc := c.Callables.Table[call.Id]; sc == nil {
				panic(fmt.Sprint("unknown callable ", call.DecId))
			} else if p, ok := sc.(*syntax.Pipeline); ok {
				if hasSideEffects(p) {
					return true
				}
			}
		}
	}
	return false
}

// Calls without outputs are assumed to be called only for their side effects.
func hasOutputs(c syntax.Callable) bool {
	p := c.GetOutParams()
	return p != nil && len(p.List) > 0
}

func removeUnusedCalls(pipe *syntax.Pipeline, asts []*syntax.Ast) editSet {
	if pipe.Callables == nil {
		panic("pipeline was not fully compiled")
	}
	// Collect all of the calls we might remove.
	calls := make(StringSet, len(pipe.Calls))
	for _, call := range pipe.Calls {
		if call.Modifiers != nil && call.Modifiers.Preflight {
			// Always keep preflight stages.
			continue
		}
		if HasKeepComment(call) {
			continue
		}
		if c := pipe.Callables.Table[call.Id]; c == nil {
			panic(fmt.Sprint("unknown callable ", call.DecId))
		} else if hasOutputs(c) && !HasKeepComment(c) && !hasSideEffects(c) {
			calls.Add(call.Id)
		}
	}

	for _, binding := range pipe.Ret.Bindings.List {
		removeIdRefs(calls, binding.Exp, syntax.KindCall)
	}
	if pipe.Retain != nil {
		for _, binding := range pipe.Retain.Refs {
			if binding.Kind == syntax.KindCall {
				calls.Remove(binding.Id)
			}
		}
	}
	for _, c := range pipe.Calls {
		for _, b := range c.Bindings.List {
			removeIdRefs(calls, b.Exp, syntax.KindCall)
		}
		if c.Modifiers.Bindings != nil {
			for _, b := range c.Modifiers.Bindings.List {
				removeIdRefs(calls, b.Exp, syntax.KindCall)
			}
		}
	}
	if len(calls) > 0 {
		return removeCallSet(pipe, asts, calls)
	}
	return nil
}

func removeCallSet(pipe *syntax.Pipeline,
	asts []*syntax.Ast,
	calls StringSet) editSet {
	edits := make(editSet, 0, len(calls))
	for _, c := range pipe.Calls {
		if calls.Contains(c.Id) {
			fmt.Fprintf(os.Stderr,
				"Outputs of call %s of pipeline %s at %s:%d are not used\n",
				c.Id, pipe.Id, c.File().FileName, c.Line())
			edits = append(edits, &removeCall{
				Pipeline: pipe,
				Call:     c,
			})
		}
	}
	// Check for inputs which are now unbound.
	edits = removeUnboundPipelineInputs(pipe, nil, calls, asts, edits)
	return edits
}

type removeCall struct {
	Pipeline *syntax.Pipeline
	Call     *syntax.CallStm
}

// Apply removes a parameter binding from a call in the given AST.
//
// The first return value indicates the number places where a change was
// made.
//
// The AST is not required to have been compiled.
func (e removeCall) Apply(ast *syntax.Ast) (int, error) {
	for _, pipe := range ast.Pipelines {
		if pipe.Id == e.Pipeline.Id &&
			pipe.Node.File().FullPath == e.Pipeline.Node.File().FullPath {
			return e.remove(pipe)
		}
	}
	return 0, nil
}

func (e removeCall) remove(pipe *syntax.Pipeline) (int, error) {
	for i, c := range pipe.Calls {
		if c.Id == e.Call.Id {
			if i == 0 {
				pipe.Calls = pipe.Calls[1:]
			} else if i == len(pipe.Calls)-1 {
				pipe.Calls = pipe.Calls[:i]
			} else {
				pipe.Calls = append(pipe.Calls[:i:i], pipe.Calls[i+1:]...)
			}
			return 1, nil
		}
	}
	return 0, nil
}

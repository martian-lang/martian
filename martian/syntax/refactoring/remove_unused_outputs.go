// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"fmt"
	"os"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
)

// RemoveUnusedOutputs finds outputs for pipelines and removes them if they are
// not bound in any reference, and the pipeline in question is not listed in
// topCalls.
//
// The asts must be fully compiled, and in order to be accurite should contian
// the entire set of mros.
//
// Pipelines will only be edited if they're in the call graph for one or more
// topCalls.
func RemoveUnusedOutputs(topCalls StringSet, asts []*syntax.Ast) Edit {
	if len(asts) == 0 {
		return nil
	}
	outputs := make(map[decId]StringSet, len(asts[0].Pipelines)+1)

	usedPipes := make(map[decId]*syntax.Pipeline)
	for _, ast := range asts {
		for _, pipe := range ast.Pipelines {
			if topCalls.Contains(pipe.Id) {
				usedPipes[makeDecId(pipe)] = pipe
				// Add child pipeline outputs to the set of pipelines to
				// possibly trim.  Pipelines which aren't in the call graph
				// for any top-level pipeline won't get changed.
				populateChildPipelineOuts(pipe, outputs)
			}
		}
		// Remove top-call pipelines which might also be children of other top
		// calls
		for _, pipe := range ast.Pipelines {
			if topCalls.Contains(pipe.Id) {
				delete(outputs, makeDecId(pipe))
			}
		}
	}
	for len(usedPipes) > 0 && len(outputs) > 0 {
		newUsed := make(map[decId]*syntax.Pipeline,
			len(asts[0].Pipelines)+len(usedPipes))
		for _, pipe := range usedPipes {
			usePipeOuts(pipe, newUsed, outputs)
		}
		usedPipes = newUsed
	}
	if len(outputs) == 0 {
		return nil
	}
	edits := make(editSet, 0, len(outputs))
	for id, outputs := range outputs {
		pipe := getPipeline(id, asts)
		if pipe != nil {
			for output := range outputs {
				fmt.Fprintf(os.Stderr,
					"Output %s of pipeline %s at %s:%d is not used.\n",
					output, pipe.Id, pipe.File().FileName, pipe.Line())
				edits = append(edits, &removePipelineOutput{
					Pipeline: pipe,
					Output:   output,
				})
			}
			// Remove inputs which are no longer bound.
			edits = removeUnboundPipelineInputs(pipe,
				outputs, nil, asts, edits)
		}
	}
	return edits
}

func populateChildPipelineOuts(pipe *syntax.Pipeline, outputs map[decId]StringSet) {
	for _, callable := range pipe.Callables.Table {
		if p, ok := callable.(*syntax.Pipeline); ok && p != nil {
			if p.OutParams != nil && len(pipe.OutParams.List) > 0 {
				dec := makeDecId(pipe)
				if _, ok := outputs[dec]; !ok {
					outputs[dec] = makePipeOutSet(pipe)
				}
			}
			populateChildPipelineOuts(p, outputs)
		}
	}
}

func getPipeline(id decId, asts []*syntax.Ast) *syntax.Pipeline {
	for _, ast := range asts {
		if ast != nil {
			for _, pipe := range ast.Pipelines {
				if pipe.Id == id.Name && syntax.DefiningFile(pipe) == id.File {
					return pipe
				}
			}
		}
	}
	return nil
}

func makePipeOutSet(pipe *syntax.Pipeline) StringSet {
	outs := make(StringSet, len(pipe.OutParams.List))
	for _, param := range pipe.OutParams.List {
		if !HasKeepComment(param) {
			outs.Add(param.Id)
		}
	}
	return outs
}

func usePipeOuts(pipe *syntax.Pipeline,
	usedPipes map[decId]*syntax.Pipeline,
	outputs map[decId]StringSet) {
	if pipe.Retain != nil {
		for _, ref := range pipe.Retain.Refs {
			removeCallRef(ref, pipe.Callables.Table, usedPipes, outputs)
		}
	}
	if pipe.Ret != nil {
		removeBoundCallRefs(pipe.Ret.Bindings, pipe.Callables.Table,
			usedPipes, outputs)
	}
	for _, call := range pipe.Calls {
		removeBoundCallRefs(call.Bindings, pipe.Callables.Table,
			usedPipes, outputs)
		if call.Modifiers != nil && call.Modifiers.Bindings != nil {
			removeBoundCallRefs(call.Modifiers.Bindings, pipe.Callables.Table,
				usedPipes, outputs)
		}
	}
}

func removeBoundCallRefs(bindings *syntax.BindStms, callables map[string]syntax.Callable,
	usedPipes map[decId]*syntax.Pipeline, outputs map[decId]StringSet) {
	if bindings == nil {
		return
	}
	for _, b := range bindings.List {
		if b != nil {
			for _, ref := range b.Exp.FindRefs() {
				removeCallRef(ref, callables, usedPipes, outputs)
			}
		}
	}
}

func removeCallRef(ref *syntax.RefExp, callables map[string]syntax.Callable,
	usedPipes map[decId]*syntax.Pipeline, outputs map[decId]StringSet) {
	if ref.Kind == syntax.KindCall {
		c := callables[ref.Id]
		if c != nil {
			dec := makeDecId(c)
			if p, ok := c.(*syntax.Pipeline); ok {
				usedPipes[dec] = p
			}
			if ref.OutputId == "" {
				// Use all the outs
				delete(outputs, dec)
			} else if set, ok := outputs[dec]; ok {
				if i := strings.IndexByte(ref.OutputId, '.'); i < 0 {
					set.Remove(ref.OutputId)
				} else {
					set.Remove(ref.OutputId[:i])
				}
				if len(set) == 0 {
					delete(outputs, dec)
				}
			}
		}
	}
}

type removePipelineOutput struct {
	Pipeline *syntax.Pipeline
	Output   string
}

func (e removePipelineOutput) Apply(ast *syntax.Ast) (int, error) {
	for _, pipe := range ast.Pipelines {
		if pipe.Id == e.Pipeline.Id &&
			pipe.File().FullPath == e.Pipeline.File().FullPath {
			return e.removeParam(pipe) + e.removeRet(pipe), nil
		}
	}
	return 0, nil
}

func (e removePipelineOutput) removeParam(pipe *syntax.Pipeline) int {
	if pipe.OutParams == nil {
		return 0
	}
	for i, param := range pipe.OutParams.List {
		if param.Id == e.Output {
			delete(pipe.OutParams.Table, e.Output)
			if i == 0 {
				pipe.OutParams.List = pipe.OutParams.List[1:]
			} else if i == len(pipe.OutParams.List)-1 {
				pipe.OutParams.List = pipe.OutParams.List[:i]
			} else {
				pipe.OutParams.List = append(pipe.OutParams.List[:i:i],
					pipe.OutParams.List[i+1:]...)
			}
			return 1
		}
	}
	return 0
}

func (e removePipelineOutput) removeRet(pipe *syntax.Pipeline) int {
	if pipe.Ret == nil || pipe.Ret.Bindings == nil {
		return 0
	}
	for i, param := range pipe.Ret.Bindings.List {
		if param.Id == e.Output {
			delete(pipe.Ret.Bindings.Table, e.Output)
			if i == 0 {
				pipe.Ret.Bindings.List = pipe.Ret.Bindings.List[1:]
			} else if i == len(pipe.Ret.Bindings.List)-1 {
				pipe.Ret.Bindings.List = pipe.Ret.Bindings.List[:i]
			} else {
				pipe.Ret.Bindings.List = append(pipe.Ret.Bindings.List[:i:i],
					pipe.Ret.Bindings.List[i+1:]...)
			}
			return 1
		}
	}
	return 0
}

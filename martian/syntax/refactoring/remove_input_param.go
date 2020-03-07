// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"fmt"
	"os"

	"github.com/martian-lang/martian/martian/syntax"
)

type removeCallableInput struct {
	Callable syntax.Callable
	Param    string
}

// Apply removes an input parameter from a callable in the given AST object.
//
// The first return value indicates the number places where a change was
// made.
//
// The AST is not required to have been compiled.
func (e removeCallableInput) Apply(ast *syntax.Ast) (int, error) {
	count := 0
	for _, target := range ast.Callables.List {
		if target.GetId() == e.Callable.GetId() &&
			target.File().FullPath == e.Callable.File().FullPath {
			if err := e.remove(target.GetInParams()); err != nil {
				return count, err
			} else {
				count++
			}
		}
	}
	return count, nil
}

func (e removeCallableInput) remove(params *syntax.InParams) error {
	for i, p := range params.List {
		if p.Id == e.Param {
			delete(params.Table, e.Param)
			if i == 0 {
				params.List = params.List[1:]
			} else if i == len(params.List)-1 {
				params.List = params.List[:i]
			} else {
				params.List = append(params.List[:i:i], params.List[i+1:]...)
			}
			return nil
		}
	}
	return nil
}

type removeCallInput struct {
	Pipeline *syntax.Pipeline
	Call     *syntax.CallStm
	Param    string
}

// Apply removes a parameter binding from a call in the given AST.
//
// The first return value indicates the number places where a change was
// made.
//
// The AST is not required to have been compiled.
func (e removeCallInput) Apply(ast *syntax.Ast) (int, error) {
	count := 0
	if e.Pipeline == nil {
		if ast.Call != nil &&
			ast.Call.DecId == e.Call.DecId &&
			ast.Call.File().FullPath == e.Call.File().FullPath {
			e.remove(ast.Call.Bindings)
			return 1, nil
		}
		return 0, nil
	}
	for _, pipe := range ast.Pipelines {
		if pipe.Id == e.Pipeline.Id &&
			pipe.Node.File().FullPath == e.Pipeline.Node.File().FullPath {
			for _, call := range pipe.Calls {
				if call.Id == e.Call.Id {
					if err := e.remove(call.Bindings); err != nil {
						return count, err
					} else {
						count++
					}
				}
			}
		}
	}
	return count, nil
}

func (e removeCallInput) remove(params *syntax.BindStms) error {
	for i, p := range params.List {
		if p.Id == e.Param {
			delete(params.Table, e.Param)
			if i == 0 {
				params.List = params.List[1:]
			} else if i == len(params.List)-1 {
				params.List = params.List[:i]
			} else {
				params.List = append(params.List[:i:i], params.List[i+1:]...)
			}
			return nil
		}
	}
	return nil
}

// RemoveInputParam creates an Edit which will remove the given parameter from
// the given callable object and every place where the callable is called.
//
// If removing the parameter from a call results in a pipeline no longer binding
// one of its inputs to anything, that input parameter is also removed from the
// pipeline, recursively.
//
// The asts must be fully compiled.
func RemoveInputParam(callable syntax.Callable, param string, asts []*syntax.Ast) Edit {
	match := matchCallable(callable)
	return removeInputParam(match, callable, param, asts, nil)
}

func removeInputParam(match matcher,
	callable syntax.Callable, param string,
	asts []*syntax.Ast, edits editSet) editSet {
	modified := make(map[decId]struct{})
	edits = append(edits, &removeCallableInput{
		Callable: callable,
		Param:    param,
	})
	for _, ast := range asts {
		if match(ast) {
			for _, pipe := range ast.Pipelines {
				dec := makeDecId(pipe)
				if _, ok := modified[dec]; ok {
					// This pipeline already edited, possibly in a different
					// AST.
					continue
				}
				modified[dec] = struct{}{}
				inputs := make(StringSet, len(pipe.InParams.List))
				for _, param := range pipe.InParams.List {
					inputs.Add(param.Id)
				}
				for _, binding := range pipe.Ret.Bindings.List {
					removeIdRefs(inputs, binding.Exp, syntax.KindSelf)
				}
				if pipe.Retain != nil {
					for _, binding := range pipe.Retain.Refs {
						if binding.Kind == syntax.KindSelf {
							inputs.Remove(binding.Id)
						}
					}
				}
				for _, c := range pipe.Calls {
					if c.DecId == callable.GetId() {
						id := makeDecId(c)
						if _, ok := modified[id]; !ok {
							edits = append(edits, &removeCallInput{
								Pipeline: pipe,
								Call:     c,
								Param:    param,
							})
							modified[id] = struct{}{}
						}
						for _, b := range c.Bindings.List {
							if b.Id != param {
								removeIdRefs(inputs, b.Exp, syntax.KindSelf)
							}
						}
					} else {
						for _, b := range c.Bindings.List {
							removeIdRefs(inputs, b.Exp, syntax.KindSelf)
						}
					}
					if c.Modifiers.Bindings != nil {
						for _, b := range c.Modifiers.Bindings.List {
							removeIdRefs(inputs, b.Exp, syntax.KindSelf)
						}
					}
				}
				if len(inputs) > 0 {
					// Remove inputs which are no longer bound.
					for input := range inputs {
						fmt.Fprintf(os.Stderr,
							"Input %s of pipeline %s in %s:%d is no longer used\n",
							input, pipe.Id, pipe.File().FileName, pipe.Line())
						edits = removeInputParam(match, pipe, input,
							asts, edits)
					}
				}
			}
		}
		if ast.Call != nil && ast.Call.DecId == callable.GetId() {
			id := makeDecId(ast.Call)
			if _, ok := modified[id]; !ok {
				edits = append(edits, &removeCallInput{
					Call:  ast.Call,
					Param: param,
				})
			}
		}
	}
	return edits
}

func removeUnboundPipelineInputs(pipe *syntax.Pipeline,
	removedOuts, removedCalls StringSet, asts []*syntax.Ast, edits editSet) editSet {
	inputs := make(StringSet, len(pipe.InParams.List))
	for _, param := range pipe.InParams.List {
		inputs.Add(param.Id)
	}
	for _, binding := range pipe.Ret.Bindings.List {
		if !removedOuts.Contains(binding.Id) {
			removeIdRefs(inputs, binding.Exp, syntax.KindSelf)
		}
	}
	if pipe.Retain != nil {
		for _, binding := range pipe.Retain.Refs {
			if binding.Kind == syntax.KindSelf {
				inputs.Remove(binding.Id)
			}
		}
	}
	for _, c := range pipe.Calls {
		if !removedCalls.Contains(c.Id) {
			for _, b := range c.Bindings.List {
				removeIdRefs(inputs, b.Exp, syntax.KindSelf)
			}
			if c.Modifiers.Bindings != nil {
				for _, b := range c.Modifiers.Bindings.List {
					removeIdRefs(inputs, b.Exp, syntax.KindSelf)
				}
			}
		}
	}
	// Remove inputs which are no longer bound.
	if len(inputs) > 0 {
		match := matchCallable(pipe)
		for input := range inputs {
			fmt.Fprintf(os.Stderr,
				"Input %s of pipeline %s in %s:%d is no longer used\n",
				input, pipe.Id, pipe.File().FileName, pipe.Line())
			edits = removeInputParam(match,
				pipe, input,
				asts, edits)
		}
	}
	return edits
}

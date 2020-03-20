// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"github.com/martian-lang/martian/martian/syntax"
)

func RenameOutput(callable syntax.Callable,
	oldParam, newParam string, asts []*syntax.Ast) Edit {
	modified := make(map[decId]struct{}, 2*len(asts))
	var edits editSet
	for _, ast := range asts {
		if c := ast.Callables.Table[callable.GetId()]; c == nil ||
			c.File().FullPath != callable.File().FullPath {
			continue
		}
		for _, c := range ast.Callables.List {
			dec := makeDecId(c)
			if _, ok := modified[dec]; ok {
				continue
			}
			modified[dec] = struct{}{}
			if c.GetId() == callable.GetId() &&
				c.File().FullPath == callable.File().FullPath {
				edits = append(edits, renameCallableOutputEdit{
					Callable: c,
					OldParam: oldParam,
					NewParam: newParam,
				})
			} else if pipe, ok := c.(*syntax.Pipeline); ok {
				edits = renameOutputInCalls(callable,
					oldParam, newParam, pipe, edits)
			}
		}
		// Fix up top-level call if needed.
		if ast.Call != nil && ast.Call.DecId == callable.GetId() {
			edits = append(edits, renameCallParamEdit{
				File:     syntax.DefiningFile(ast.Call),
				Id:       ast.Call.Id,
				OldParam: oldParam,
				NewParam: newParam,
			})
		}
	}
	if len(edits) == 0 {
		return nil
	}
	return edits
}

func renameOutputInCalls(callable syntax.Callable,
	oldName, newName string, pipe *syntax.Pipeline, edits editSet) editSet {
	for cid, c := range pipe.Callables.Table {
		if c.GetId() == callable.GetId() {
			for _, call := range pipe.Calls {
				// Check for references that need to be updated.
				for _, binding := range call.Bindings.List {
					if binding.Exp.HasRef() {
						edits = updateRefsOutNamesFromBinding(edits, binding, pipe, call,
							cid, oldName, newName, false)
					}
				}
				if call.Modifiers != nil && call.Modifiers.Bindings != nil {
					for _, binding := range call.Modifiers.Bindings.List {
						if binding.Exp.HasRef() {
							edits = updateRefsOutNamesFromBinding(edits, binding, pipe, call,
								cid, oldName, newName, true)
						}
					}
				}
			}
			if pipe.Ret != nil && pipe.Ret.Bindings != nil {
				for _, binding := range pipe.Ret.Bindings.List {
					edits = updateRefsOutNamesFromBinding(edits, binding, pipe, nil,
						cid, oldName, newName, false)
				}
			}
			if pipe.Retain != nil {
				for _, ref := range pipe.Retain.Refs {
					if u := updateRef(ref, syntax.KindCall, cid, oldName, newName); u != ref {
						edits = append(edits, updatePipelineRetain{
							Pipeline: pipe,
							OldRef:   ref,
							NewRef:   u,
						})
					}
				}
			}
		}
	}
	return edits
}

func updateRefsOutNamesFromBinding(edits editSet, binding *syntax.BindStm,
	pipe *syntax.Pipeline, call *syntax.CallStm,
	callableId, oldId, newId string, isMods bool) editSet {
	exp := updateRefInExp(binding.Exp, syntax.KindCall, callableId, oldId, newId)
	if exp == binding.Exp {
		return edits
	}
	if exp == binding.Exp {
		return edits
	}
	// Must edit the original AST here or else other edits will be operating on
	// the incorrect expression.
	binding.Exp = exp
	return append(edits, &editBinding{
		Pipeline: pipe,
		Call:     call,
		Binding:  binding,
		Mods:     isMods,
		Exp:      exp,
	})
}

type (
	renameCallableOutputEdit struct {
		Callable syntax.Callable
		OldParam string
		NewParam string
	}

	updatePipelineRetain struct {
		Pipeline *syntax.Pipeline
		OldRef   *syntax.RefExp
		NewRef   *syntax.RefExp
	}
)

func (e renameCallableOutputEdit) Apply(ast *syntax.Ast) (int, error) {
	count := 0
	for _, callable := range ast.Callables.List {
		if callable.GetId() == e.Callable.GetId() &&
			syntax.DefiningFile(callable) == syntax.DefiningFile(e.Callable) {
			count += e.applyOuts(callable.GetOutParams())
			if pipe, ok := callable.(*syntax.Pipeline); ok &&
				pipe != nil && pipe.Ret != nil && pipe.Ret.Bindings != nil {
				for _, b := range pipe.Ret.Bindings.List {
					if b.Id == e.OldParam {
						if pipe.Ret.Bindings.Table != nil {
							p, ok := pipe.Ret.Bindings.Table[e.OldParam]
							if ok {
								delete(pipe.Ret.Bindings.Table, e.OldParam)
								pipe.Ret.Bindings.Table[e.NewParam] = p
							}
						}
						b.Id = e.NewParam
						count++
					}
				}
			}
		}
	}
	return count, nil
}

func (e renameCallableOutputEdit) applyOuts(params *syntax.OutParams) int {
	if params == nil {
		return 0
	}
	for _, param := range params.List {
		if param.Id == e.OldParam {
			param.Id = e.NewParam
			if params.Table != nil {
				p, ok := params.Table[e.OldParam]
				if ok {
					delete(params.Table, e.OldParam)
					params.Table[e.NewParam] = p
				}
			}
			return 1
		}
	}
	return 0
}

func (e updatePipelineRetain) Apply(ast *syntax.Ast) (int, error) {
	count := 0
	for _, target := range ast.Pipelines {
		if target.GetId() == e.Pipeline.Id &&
			target.File().FullPath == e.Pipeline.File().FullPath {
			count += e.update(target)
		}
	}
	return count, nil
}

func (e updatePipelineRetain) update(target *syntax.Pipeline) int {
	for i, r := range target.Retain.Refs {
		if r.Kind == e.OldRef.Kind && r.Id == e.OldRef.Id && r.OutputId == e.OldRef.OutputId {
			target.Retain.Refs[i] = e.NewRef
			return 1
		}
	}
	return 0
}

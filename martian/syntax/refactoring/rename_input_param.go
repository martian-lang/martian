// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"github.com/martian-lang/martian/martian/syntax"
)

func RenameInput(callable syntax.Callable,
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
			if c.GetId() == callable.GetId() && c.File().FullPath == callable.File().FullPath {
				edits = append(edits, renameCallableInputEdit{
					Callable: c,
					OldParam: oldParam,
					NewParam: newParam,
				})
				if pipe, ok := c.(*syntax.Pipeline); ok {
					edits = renameSelfInputInCalls(pipe,
						oldParam, newParam, edits)
				}
			} else if pipe, ok := c.(*syntax.Pipeline); ok {
				edits = renameInputInCalls(callable,
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

func renameInputInCalls(callable syntax.Callable,
	oldName, newName string, pipe *syntax.Pipeline, edits editSet) editSet {
	for _, call := range pipe.Calls {
		if call.DecId == callable.GetId() {
			edits = append(edits, renameCallParamEdit{
				Pipeline: pipe,
				File:     syntax.DefiningFile(call),
				Id:       call.Id,
				OldParam: oldName,
				NewParam: newName,
			})
		}
	}
	return edits
}

func renameSelfInputInCalls(pipe *syntax.Pipeline,
	oldName, newName string, edits editSet) editSet {
	for _, call := range pipe.Calls {
		// Check for references that need to be updated.
		for _, binding := range call.Bindings.List {
			if binding.Exp.HasRef() {
				edits = updateSelfRefsFromBinding(edits, binding, pipe, call,
					oldName, newName, false)
			}
		}
		if call.Modifiers != nil && call.Modifiers.Bindings != nil {
			for _, binding := range call.Modifiers.Bindings.List {
				if binding.Exp.HasRef() {
					edits = updateSelfRefsFromBinding(edits, binding, pipe, call,
						oldName, newName, true)
				}
			}
		}
	}
	if pipe.Ret != nil && pipe.Ret.Bindings != nil {
		for _, binding := range pipe.Ret.Bindings.List {
			edits = updateSelfRefsFromBinding(edits, binding, pipe, nil,
				oldName, newName, false)
		}
	}
	return edits
}

func updateSelfRefsFromBinding(edits editSet, binding *syntax.BindStm,
	pipe *syntax.Pipeline, call *syntax.CallStm,
	oldId, newId string, isMods bool) editSet {
	exp := updateRefInExp(binding.Exp, syntax.KindSelf, "", oldId, newId)
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
	renameCallableInputEdit struct {
		Callable syntax.Callable
		OldParam string
		NewParam string
	}

	renameCallParamEdit struct {
		Pipeline *syntax.Pipeline
		File     string
		Id       string
		OldParam string
		NewParam string
	}
)

func (e renameCallableInputEdit) Apply(ast *syntax.Ast) (int, error) {
	count := 0
	for _, callable := range ast.Callables.List {
		if callable.GetId() == e.Callable.GetId() &&
			syntax.DefiningFile(callable) == syntax.DefiningFile(e.Callable) {
			count += e.applyIns(callable.GetInParams())
			if pipe, ok := callable.(*syntax.Pipeline); ok &&
				pipe != nil && pipe.Retain != nil {
				for _, ref := range pipe.Retain.Refs {
					if ref.Kind == syntax.KindSelf && ref.Id == e.OldParam {
						ref.Id = e.NewParam
						count++
					}
				}
			}
		}
	}
	return count, nil
}

func (e renameCallableInputEdit) applyIns(params *syntax.InParams) int {
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

func (e renameCallParamEdit) Apply(ast *syntax.Ast) (int, error) {
	if e.Pipeline == nil {
		if ast.Call == nil {
			return 0, nil
		}
		if ast.Call.Id != e.Id ||
			syntax.DefiningFile(ast.Call) != e.File {
			return 0, nil
		}
		return e.apply(ast.Call), nil
	}
	edits := 0
	for _, p := range ast.Pipelines {
		if p.Id != e.Pipeline.Id ||
			syntax.DefiningFile(p) != syntax.DefiningFile(e.Pipeline) {
			continue
		}
		for _, call := range p.Calls {
			if call.Id == e.Id {
				edits += e.apply(call)
			}
		}
	}
	return edits, nil
}

func (e renameCallParamEdit) apply(call *syntax.CallStm) int {
	if call.Bindings == nil {
		return 0
	}
	for _, b := range call.Bindings.List {
		if b.Id == e.OldParam {
			b.Id = e.NewParam

			if call.Bindings.Table != nil {
				b, ok := call.Bindings.Table[e.OldParam]
				if ok {
					delete(call.Bindings.Table, e.OldParam)
					call.Bindings.Table[e.OldParam] = b
				}
			}
			return 1
		}
	}
	return 0
}

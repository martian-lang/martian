// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"fmt"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
)

func RenameCallable(callable syntax.Callable,
	newName string, asts []*syntax.Ast) Edit {
	if callable.GetId() == newName {
		return nil
	}
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
				edits = append(edits, renameCallableEdit{
					Callable: c.GetId(),
					File:     syntax.DefiningFile(c),
					NewName:  newName,
				})
			} else if pipe, ok := c.(*syntax.Pipeline); ok {
				edits = renameCallsToCallable(callable,
					newName, pipe, edits)
			}
		}
		// Fix up top-level call if needed.
		if ast.Call != nil && ast.Call.DecId == callable.GetId() {
			id := ast.Call.Id
			if id == ast.Call.DecId {
				id = newName
			}
			edits = append(edits, renameCallEdit{
				File:  syntax.DefiningFile(ast.Call),
				OldId: ast.Call.Id,
				Id:    id,
				DecId: newName,
			})
		}
	}
	if len(edits) == 0 {
		return nil
	}
	return edits
}

func renameCallsToCallable(callable syntax.Callable,
	newName string, pipe *syntax.Pipeline, edits editSet) editSet {
	newIds := make(map[string]string)
	for _, call := range pipe.Calls {
		if call.DecId == callable.GetId() {
			if call.DecId != call.Id || pipe.Callables.Table[newName] != nil {
				// Either the call was already aliased or changing the name
				// would cause a collision.
				edits = append(edits, renameCallEdit{
					Pipeline: pipe,
					File:     syntax.DefiningFile(call),
					OldId:    call.Id,
					Id:       call.Id,
					DecId:    newName,
				})
			} else {
				newIds[call.Id] = newName
				edits = append(edits, renameCallEdit{
					Pipeline: pipe,
					File:     syntax.DefiningFile(call),
					OldId:    call.Id,
					Id:       newName,
					DecId:    newName,
				})
			}
		}
	}
	if len(newIds) > 0 {
		for _, call := range pipe.Calls {
			// Check for references that need to be updated.
			for _, binding := range call.Bindings.List {
				if binding.Exp.HasRef() {
					edits = updateRefsFromBinding(edits, binding, pipe, call,
						newIds, false)
				}
			}
			if call.Modifiers != nil && call.Modifiers.Bindings != nil {
				for _, binding := range call.Modifiers.Bindings.List {
					if binding.Exp.HasRef() {
						edits = updateRefsFromBinding(edits, binding, pipe, call,
							newIds, true)
					}
				}
			}
		}
		if pipe.Ret != nil && pipe.Ret.Bindings != nil {
			for _, binding := range pipe.Ret.Bindings.List {
				edits = updateRefsFromBinding(edits, binding, pipe, nil,
					newIds, false)
			}
		}
	}
	return edits
}

func updateRefsFromBinding(edits editSet, binding *syntax.BindStm,
	pipe *syntax.Pipeline, call *syntax.CallStm,
	newIds map[string]string, isMods bool) editSet {
	exp := binding.Exp
	for oldId, newId := range newIds {
		exp = updateRefInExp(exp, syntax.KindCall, "", oldId, newId)
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

func updateRef(ref *syntax.RefExp, kind syntax.ExpKind,
	callId, oldName, newName string) *syntax.RefExp {
	if ref.Kind != kind {
		return ref
	}
	if callId == "" {
		if ref.Id == oldName {
			r := *ref
			r.Id = newName
			return &r
		}
		return ref
	} else if callId != ref.Id {
		return ref
	}
	if ref.OutputId == oldName {
		r := *ref
		r.OutputId = newName
		return &r
	}
	if len(ref.OutputId) < len(oldName) {
		return ref
	}
	if i := strings.IndexByte(ref.OutputId, '.'); i > 0 && ref.OutputId[:i] == oldName {
		r := *ref
		r.OutputId = newName + ref.OutputId[i:]
		return &r
	}
	return ref
}

func updateRefInExp(exp syntax.Exp, kind syntax.ExpKind,
	callId, oldName, newName string) syntax.Exp {
	if !exp.HasRef() {
		return exp
	}
	switch exp := exp.(type) {
	case *syntax.RefExp:
		return updateRef(exp, kind, callId, oldName, newName)
	case *syntax.SplitExp:
		e := updateRefInExp(exp.Value, kind, callId, oldName, newName)
		if e == exp.Value {
			return exp
		}
		ee := *exp
		ee.Value = e
		return &ee
	case *syntax.ArrayExp:
		arr := make([]syntax.Exp, 0, len(exp.Value))
		change := false
		for _, v := range exp.Value {
			e := updateRefInExp(v, kind, callId, oldName, newName)
			arr = append(arr, e)
			if e != v {
				change = true
			}
		}
		if !change {
			return exp
		}
		ee := *exp
		ee.Value = arr
		return &ee
	case *syntax.MapExp:
		m := make(map[string]syntax.Exp, len(exp.Value))
		change := false
		for k, v := range exp.Value {
			e := updateRefInExp(v, kind, callId, oldName, newName)
			m[k] = e
			if e != v {
				change = true
			}
		}
		if !change {
			return exp
		}
		ee := *exp
		ee.Value = m
		return &ee
	}
	return exp
}

type (
	renameCallableEdit struct {
		File     string
		Callable string
		NewName  string
	}

	renameCallEdit struct {
		Pipeline *syntax.Pipeline
		File     string
		OldId    string
		Id       string
		DecId    string
	}
)

func (e renameCallableEdit) Apply(ast *syntax.Ast) (int, error) {
	for _, callable := range ast.Callables.List {
		if callable.GetId() == e.Callable &&
			syntax.DefiningFile(callable) == e.File {
			if ast.Callables.Table != nil {
				c, ok := ast.Callables.Table[e.Callable]
				if ok {
					delete(ast.Callables.Table, e.Callable)
					ast.Callables.Table[e.NewName] = c
				}
			}
			switch c := callable.(type) {
			case *syntax.Pipeline:
				c.Id = e.NewName
				return 1, nil
			case *syntax.Stage:
				c.Id = e.NewName
				return 1, nil
			default:
				return 0, fmt.Errorf("unexpected callable type %T", callable)
			}
		}
	}
	return 0, nil
}

func (e renameCallEdit) Apply(ast *syntax.Ast) (int, error) {
	if e.Pipeline == nil {
		if ast.Call == nil {
			return 0, nil
		}
		if ast.Call.Id != e.OldId ||
			syntax.DefiningFile(ast.Call) != e.File {
			return 0, nil
		}
		ast.Call.DecId = e.DecId
		ast.Call.Id = e.Id
		return 1, nil
	}
	edits := 0
	for _, p := range ast.Pipelines {
		if p.Id != e.Pipeline.Id ||
			syntax.DefiningFile(p) != syntax.DefiningFile(e.Pipeline) {
			continue
		}
		for _, call := range p.Calls {
			if call.Id == e.OldId {
				call.Id = e.Id
				call.DecId = e.DecId
				edits++
			}
		}
		if p.Callables != nil && p.Callables.Table != nil {
			c, ok := p.Callables.Table[e.OldId]
			if ok {
				delete(p.Callables.Table, e.OldId)
				p.Callables.Table[e.Id] = c
			}
		}
		if p.Retain != nil && e.OldId != e.Id {
			for _, ref := range p.Retain.Refs {
				if ref.Kind == syntax.KindCall && ref.Id == e.OldId {
					ref.Id = e.Id
					edits++
				}
			}
		}
	}
	return edits, nil
}

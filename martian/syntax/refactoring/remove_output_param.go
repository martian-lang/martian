// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
)

// RemoveOutputParam removes an output parameter from a stage or pipeline, and
// attempts to fix every location where that output is used.
//
// If the output is retained, the retain directive is removed.
//
// If the output is bound as a pipeline output, that pipeline output will be
// be removed as well.
//
// If the output is bound as the value in a typed map literal, or as an element
// in an array literal, that key or element is removed from the map or array.
// If the map or array becomes empty, then if it was used as a pipeline output
// the pipeline output will be removed as well.
//
// If the output is bound directly to a call input, the input is assigned
// null instead.
//
// If the output is bound to a disable statement, the disable statement is
// removed.
//
// Note that unlike most of the other refactorings, this can leave the mro in
// a broken state in one of two ways.  First, null is not a valid replacement
// value for a call binding if the call was to a pipeline which uses that input
// to disable a child call.  Second, a map call may be left with no parameters
// to map over.  There is not an obviously correct fix in either of these cases;
// it is left to the user to fix them up manually.
func RemoveOutputParam(callable syntax.Callable, param string, asts []*syntax.Ast) (Edit, error) {
	match := matchCallable(callable)
	return removeOutputParam(match, callable, param, asts, nil)
}

type (
	// removePipelineRetain is an Edit which removes a retained value from
	// a pipeline.
	removePipelineRetain struct {
		Pipeline *syntax.Pipeline
		Ref      *syntax.RefExp
	}

	// removeStageRetain is an Edit which removes a retained parameter from
	// a stage.
	removeStageOutput struct {
		Stage *syntax.Stage
		Param string
	}

	// removeMod is an Edit which removes a call modifier reference.
	removeMod struct {
		Pipeline *syntax.Pipeline
		Call     *syntax.CallStm
		Ref      *syntax.RefExp
	}

	// editBinding is an Edit which updates the expression used in a binding.
	editBinding struct {
		Pipeline *syntax.Pipeline
		Call     *syntax.CallStm
		Binding  *syntax.BindStm
		Mods     bool
		Exp      syntax.Exp
	}
)

// Apply removes an input parameter from a callable in the given AST object.
//
// The first return value indicates the number places where a change was
// made.
//
// The AST is not required to have been compiled.
func (e removePipelineRetain) Apply(ast *syntax.Ast) (int, error) {
	count := 0
	for _, target := range ast.Pipelines {
		if target.GetId() == e.Pipeline.Id &&
			target.File().FullPath == e.Pipeline.File().FullPath {
			if err := e.remove(target); err != nil {
				return count, err
			} else {
				count++
			}
		}
	}
	return count, nil
}

func (e removePipelineRetain) remove(target *syntax.Pipeline) error {
	for i, r := range target.Retain.Refs {
		if r.Kind == e.Ref.Kind && r.Id == e.Ref.Id && r.OutputId == e.Ref.OutputId {
			if len(target.Retain.Refs) == 1 {
				target.Retain = nil
			} else if i == 0 {
				target.Retain.Refs = target.Retain.Refs[1:]
			} else if i == len(target.Retain.Refs)-1 {
				target.Retain.Refs = target.Retain.Refs[:i]
			} else {
				target.Retain.Refs = append(
					target.Retain.Refs[:i:i],
					target.Retain.Refs[i+1:]...)
			}
			return nil
		}
	}
	return nil
}

// Apply removes an input parameter from a callable in the given AST object.
//
// The first return value indicates the number places where a change was
// made.
//
// The AST is not required to have been compiled.
func (e removeStageOutput) Apply(ast *syntax.Ast) (int, error) {
	count := 0
	for _, target := range ast.Stages {
		if target.Id == e.Stage.Id &&
			target.File().FullPath == e.Stage.File().FullPath {
			count += e.removeOutputParam(target.OutParams)
			count += e.removeRetain(target)
		}
	}
	return count, nil
}

func (e removeStageOutput) removeOutputParam(params *syntax.OutParams) int {
	for i, p := range params.List {
		if p.Id == e.Param {
			delete(params.Table, p.Id)
			if i == 0 {
				params.List = params.List[1:]
			} else if i == len(params.List)-1 {
				params.List = params.List[:i]
			} else {
				params.List = append(params.List[:i:i],
					params.List[i+1:]...)
			}
			return 1
		}
	}
	return 0
}

func (e removeStageOutput) removeRetain(stage *syntax.Stage) int {
	if stage.Retain == nil {
		return 0
	}
	for i, p := range stage.Retain.Params {
		if p.Id == e.Param {
			if len(stage.Retain.Params) == 1 {
				stage.Retain = nil
			} else if i == 0 {
				stage.Retain.Params = stage.Retain.Params[1:]
			} else if i == len(stage.Retain.Params)-1 {
				stage.Retain.Params = stage.Retain.Params[:i]
			} else {
				stage.Retain.Params = append(stage.Retain.Params[:i:i],
					stage.Retain.Params[i+1:]...)
			}
			return 1
		}
	}
	return 0
}

// Apply removes a modifier binding from a call in the given AST.
//
// The first return value indicates the number places where a change was
// made.
//
// The AST is not required to have been compiled.
func (e removeMod) Apply(ast *syntax.Ast) (int, error) {
	for _, pipe := range ast.Pipelines {
		if pipe.Id == e.Pipeline.Id &&
			pipe.Node.File().FullPath == e.Pipeline.Node.File().FullPath {
			for _, call := range pipe.Calls {
				if call.Id == e.Call.Id {
					return e.remove(call.Modifiers)
				}
			}
		}
	}
	return 0, nil
}

func (e removeMod) remove(mod *syntax.Modifiers) (int, error) {
	if mod == nil || mod.Bindings == nil {
		return 0, nil
	}
	for i, c := range mod.Bindings.List {
		if ref, ok := c.Exp.(*syntax.RefExp); ok && ref.Kind == e.Ref.Kind &&
			ref.Id == e.Ref.Id && ref.OutputId == e.Ref.OutputId {
			if i == 0 {
				mod.Bindings.List = mod.Bindings.List[1:]
			} else if i == len(mod.Bindings.List)-1 {
				mod.Bindings.List = mod.Bindings.List[:i]
			} else {
				mod.Bindings.List = append(mod.Bindings.List[:i:i],
					mod.Bindings.List[i+1:]...)
			}
			return 1, nil
		}
	}
	return 0, nil
}

// Apply removes a parameter binding from a call in the given AST.
//
// The first return value indicates the number places where a change was
// made.
//
// The AST is not required to have been compiled.
func (e editBinding) Apply(ast *syntax.Ast) (int, error) {
	if e.Pipeline == nil && ast.Call != nil &&
		e.Call.File().FullPath == ast.Call.File().FullPath &&
		e.Call.Id == ast.Call.Id {
		return e.apply(ast.Call.Bindings.List), nil
	}
	for _, pipe := range ast.Pipelines {
		if pipe.Id == e.Pipeline.Id &&
			pipe.Node.File().FullPath == e.Pipeline.Node.File().FullPath {
			if e.Call != nil {
				return e.applyToCalls(pipe.Calls), nil
			} else {
				return e.apply(pipe.Ret.Bindings.List), nil
			}
		}
	}
	return 0, nil
}

func (e editBinding) applyToCalls(calls []*syntax.CallStm) int {
	for _, call := range calls {
		if call.Id == e.Call.Id {
			if e.Mods {
				return e.apply(call.Modifiers.Bindings.List)
			} else {
				return e.apply(call.Bindings.List)
			}
		}
	}
	return 0
}

func (e editBinding) apply(bindings []*syntax.BindStm) int {
	for _, bind := range bindings {
		if bind.Id == e.Binding.Id {
			bind.Exp = e.Exp
			return 1
		}
	}
	return 0
}

func isCallRefTo(ref *syntax.RefExp, pipe *syntax.Pipeline,
	callable syntax.Callable, param string) bool {
	if ref.Kind == syntax.KindCall {
		if c := pipe.Callables.Table[ref.Id]; c != nil &&
			c.GetId() == callable.GetId() {
			if param == "" {
				return true
			}
			if ref.OutputId == "" {
				return param == ""
			} else if i := strings.IndexByte(ref.OutputId, '.'); i < 0 {
				return ref.OutputId == param
			} else {
				return ref.OutputId[:i] == param
			}
		}
	}
	return false
}

func removeOutputParam(match matcher,
	callable syntax.Callable, param string,
	asts []*syntax.Ast, edits editSet) (editSet, error) {
	modified := make(map[decId]struct{})
	switch callable := callable.(type) {
	case *syntax.Stage:
		edits = append(edits, &removeStageOutput{
			Stage: callable,
			Param: param,
		})
	case *syntax.Pipeline:
		edits = append(edits, &removePipelineOutput{
			Pipeline: callable,
			Output:   param,
		})
		edits = removeUnboundPipelineInputs(
			callable, StringSet{param: struct{}{}}, nil, asts, edits)
	}
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
				if pipe.Retain != nil {
					// Remove any retains for the given param.
					for _, ref := range pipe.Retain.Refs {
						if isCallRefTo(ref, pipe, callable, param) {
							edits = append(edits, &removePipelineRetain{
								Pipeline: pipe,
								Ref:      ref,
							})
						}
					}
				}
				for _, call := range pipe.Calls {
					for _, binding := range call.Bindings.List {
						if binding.Id == "*" {
							break
						}
						edits = removeRefFromBinding(edits,
							binding, pipe, call, callable, param,
						)
					}
					if call.Modifiers != nil && call.Modifiers.Bindings != nil {
						for _, binding := range call.Modifiers.Bindings.List {
							if ref, ok := binding.Exp.(*syntax.RefExp); ok {
								if isCallRefTo(ref, pipe, callable, param) {
									edits = append(edits, &removeMod{
										Pipeline: pipe,
										Call:     call,
										Ref:      ref,
									})
								}
							}
						}
					}
				}
				if pipe.Ret != nil {
					for _, binding := range pipe.Ret.Bindings.List {
						if binding.Id == "*" {
							break
						}
						var err error
						if shouldRemoveExpCallRef(binding.Exp, pipe, callable, param) {
							edits, err = removeOutputParam(
								match, pipe, binding.Id, asts, edits)
						} else {
							edits = removeRefFromBinding(edits,
								binding, pipe, nil, callable,
								param)
						}
						if err != nil {
							return edits, err
						}
					}
				}
			}
		}
	}
	return edits, nil
}

// shouldRemoveExpCallRef returns true if exp is a reference to the given output
// of the given callable, or a (possibly nested) array with a single element
// containing such a reference, a map containing such a reference, or a split
// over such a collection.
func shouldRemoveExpCallRef(exp syntax.Exp,
	pipe *syntax.Pipeline,
	callable syntax.Callable, param string) bool {
	switch exp := exp.(type) {
	case *syntax.RefExp:
		if exp.Kind == syntax.KindCall {
			return isCallRefTo(exp, pipe, callable, param)
		}
	case *syntax.SplitExp:
		return shouldRemoveExpCallRef(exp.Value, pipe, callable, param)
	case *syntax.ArrayExp:
		return len(exp.Value) == 1 &&
			shouldRemoveExpCallRef(exp.Value[0], pipe, callable, param)
	case *syntax.MapExp:
		if exp.Kind == syntax.KindMap && len(exp.Value) == 1 {
			for _, v := range exp.Value {
				return shouldRemoveExpCallRef(v, pipe, callable, param)
			}
		}
	}
	return false
}

func removeRefFromBinding(edits editSet,
	binding *syntax.BindStm, pipe *syntax.Pipeline, call *syntax.CallStm,
	callable syntax.Callable,
	param string) editSet {
	exp := removeRefFromExp(binding.Exp, pipe, callable, param)
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
		Exp:      exp,
	})
}

func removeRefFromExp(exp syntax.Exp,
	pipe *syntax.Pipeline,
	callable syntax.Callable,
	param string) syntax.Exp {
	if !exp.HasRef() {
		return exp
	}
	switch exp := exp.(type) {
	case *syntax.RefExp:
		if isCallRefTo(exp, pipe, callable, param) {
			e := new(syntax.NullExp)
			e.Node = exp.Node
			return e
		}
	case *syntax.SplitExp:
		if shouldRemoveExpCallRef(exp.Value, pipe, callable, param) {
			e := new(syntax.NullExp)
			e.Node = exp.Node
			return e
		}
		e := removeRefFromExp(exp.Value, pipe, callable, param)
		if e == exp.Value {
			return exp
		}
		ee := *exp
		ee.Value = e
		return &ee
	case *syntax.ArrayExp:
		if shouldRemoveExpCallRef(exp, pipe, callable, param) {
			e := new(syntax.NullExp)
			e.Node = exp.Node
			return e
		}
		arr := make([]syntax.Exp, 0, len(exp.Value))
		change := false
		for _, v := range exp.Value {
			if !shouldRemoveExpCallRef(v, pipe, callable, param) {
				e := removeRefFromExp(v, pipe, callable, param)
				arr = append(arr, e)
				if e != v {
					change = true
				}
			} else {
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
		if shouldRemoveExpCallRef(exp, pipe, callable, param) {
			e := new(syntax.NullExp)
			e.Node = exp.Node
			return e
		}
		m := make(map[string]syntax.Exp, len(exp.Value))
		change := false
		for k, v := range exp.Value {
			if !shouldRemoveExpCallRef(v, pipe, callable, param) {
				e := removeRefFromExp(v, pipe, callable, param)
				m[k] = e
				if e != v {
					change = true
				}
			} else {
				change = true
				if exp.Kind == syntax.KindStruct {
					e := new(syntax.NullExp)
					e.Node.Comments = syntax.GetComments(v)
					e.Node.Loc.Line = v.Line()
					e.Node.Loc.File = v.File()
					m[k] = e
				}
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

// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"fmt"
	"os"

	"github.com/martian-lang/martian/martian/syntax"
)

// An Edit represents a modification to an AST.
//
// Edits may be created on different Ast objects than they are applied to.  For
// example, fully-compiled Ast objects with processed includes may be required
// for determining the desired set of edits, but then the edits are only applied
// on a subset of the files or are applied to files without processing includes
// (for example if one wishes to then format the AST to a new mro file).
type Edit interface {
	// Apply makes the requested change to the given AST object.
	//
	// The first return value indicates the number places where a change was
	// made.
	//
	// The AST is not required to have been compiled.
	Apply(*syntax.Ast) (int, error)
}

type editSet []Edit

func (e editSet) Apply(ast *syntax.Ast) (int, error) {
	var errs syntax.ErrorList
	var count int
	for _, edit := range e {
		c, err := edit.Apply(ast)
		count += c
		if err != nil {
			errs = append(errs, err)
		}
	}
	return count, errs.If()
}

func makeEditSet(m map[decId]Edit) editSet {
	if len(m) == 0 {
		return nil
	}
	set := make(editSet, 0, len(m))
	for _, v := range m {
		set = append(set, v)
	}
	return set
}

// A Matcher is used to determine whether an edit applies to a given Ast.
type matcher func(*syntax.Ast) bool

func matchCallable(callable syntax.Callable) matcher {
	fn := syntax.DefiningFile(callable)
	return func(ast *syntax.Ast) bool {
		if c := ast.Callables.Table[callable.GetId()]; c != nil &&
			c.Type() == callable.Type() &&
			syntax.DefiningFile(c) == fn {
			return true
		}
		return false
	}
}

type StringSet map[string]struct{}

func (set StringSet) Contains(s string) bool {
	_, ok := set[s]
	return ok
}

func (set StringSet) Add(s string) {
	set[s] = struct{}{}
}

func (set StringSet) Remove(s string) {
	delete(set, s)
}

func removeIdRefs(names StringSet, exp syntax.Exp, kind syntax.ExpKind) {
	if exp == nil {
		return
	}
	if exp, ok := exp.(*syntax.RefExp); ok && exp.Kind == kind {
		names.Remove(exp.Id)
		return
	}
	for _, ref := range exp.FindRefs() {
		if ref.Kind == kind {
			names.Remove(ref.Id)
		}
	}
}

// Used as a map key.  Name is insufficient because the same callable might be
// defined in two different files, so long as those files aren't both included
// in the same AST.
type decId struct {
	Name string
	Kind string
	File string
	Line int
}

func makeDecId(node syntax.NamedNode) decId {
	var kind string
	switch node := node.(type) {
	case syntax.Callable:
		kind = node.Type()
	case *syntax.CallStm:
		kind = syntax.KindCall
	case *syntax.StructType:
		kind = syntax.KindStruct
	default:
		fmt.Fprintf(os.Stderr, "Unexpected type %T for %s", node, node.GetId())
	}
	return decId{
		Name: node.GetId(),
		Kind: kind,
		File: syntax.DefiningFile(node),
		Line: node.Line(),
	}
}

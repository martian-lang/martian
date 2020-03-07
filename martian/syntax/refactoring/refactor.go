// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Package refactoring includes methods for modifying martian ASTs.
package refactoring

import (
	"fmt"

	"github.com/martian-lang/martian/martian/syntax"
)

func getCallable(id string, asts []*syntax.Ast) syntax.Callable {
	for _, ast := range asts {
		if ast != nil && ast.Callables != nil {
			if c := ast.Callables.Table[id]; c != nil {
				return c
			}
		}
	}
	return nil
}

type CallableParam struct {
	Callable string
	Param    string
}

// Refactor modifies a set of ASTs.
//
// If removeCalls is true, RemoveAllUnusedCalls is applied repeatedly until
// no further changes are made.
//
// If topCalls is non-empty, the RemoveUnusedOutputs will be applied repeatedly
// until no further changes are made.  If removeCalls is also specified, the
// calls to RemoveAllUnusedCalls and RemoveUnusedOutputs are alternated.
//
// The returned Edit will apply the same changes to another Ast.
func Refactor(asts []*syntax.Ast,
	topCalls StringSet, removeParams []CallableParam,
	removeCalls bool) (Edit, error) {
	edits := make(editSet, 0, 2+len(topCalls))
	for _, removeParam := range removeParams {
		cname := removeParam.Callable
		param := removeParam.Param
		callable := getCallable(cname, asts)
		if callable == nil {
			return edits, fmt.Errorf("callable %s not found", cname)
		}
		edit := RemoveInputParam(callable, param, asts)
		if edit != nil {
			edits = append(edits, edit)
			if removeCalls || len(topCalls) > 0 {
				for _, ast := range asts {
					// Run this on the compiled ASTs so removeCalls has
					// an up-to-date version of the pipelines.
					if _, err := edit.Apply(ast); err != nil {
						return edits, fmt.Errorf("applying edit: %w", err)
					}
				}
			}
		}
	}
	if removeCalls || len(topCalls) > 0 {
		changes := true
		for changes {
			changes = false
			if removeCalls {
				edit := RemoveAllUnusedCalls(asts)
				if edit != nil {
					count := 0
					for _, ast := range asts {
						if c, err := edit.Apply(ast); err != nil {
							return edits, fmt.Errorf("applying edit: %w", err)
						} else {
							count += c
						}
					}
					if count > 0 {
						// Go around for another pass, in case some of the
						// removed calls were the only dependencies keeping
						// another call alive
						changes = true
						edits = append(edits, edit)
					}
				}
			}
			if len(topCalls) > 0 {
				edit := RemoveUnusedOutputs(topCalls, asts)
				if edit != nil {
					count := 0
					for _, ast := range asts {
						if c, err := edit.Apply(ast); err != nil {
							return edits, fmt.Errorf("applying edit: %w", err)
						} else {
							count += c
						}
					}
					if count > 0 {
						// Go around for another pass, in case some of the
						// removed outputs were the only dependencies keeping
						// another call alive
						changes = true
						edits = append(edits, edit)
					}
				}
			}
		}
	}
	if len(edits) == 0 {
		return nil, nil
	}
	return edits, nil
}

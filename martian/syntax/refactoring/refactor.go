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

type Rename struct {
	Callable string
	NewName  string
}

type RenameParam struct {
	CallableParam
	NewName string
}

// RefactorConfig contains options to be passed to Refactor.
type RefactorConfig struct {
	// If topCalls is non-empty, the RemoveUnusedOutputs will be applied repeatedly
	// until no further changes are made.  If RemoveCalls is also specified, the
	// calls to RemoveAllUnusedCalls and RemoveUnusedOutputs are alternated.
	TopCalls StringSet

	// Remove the given input parameters.
	RemoveInParams []CallableParam

	// Remove the given output parameters.
	RemoveOutParams []CallableParam

	// If removeCalls is true, RemoveAllUnusedCalls is applied repeatedly until
	// no further changes are made.
	RemoveCalls bool

	// Rename the given callable objects.
	Rename []Rename

	// Rename the given input parameters.
	RenameInParam []RenameParam

	// Rename the given output parameters.
	RenameOutParam []RenameParam
}

// Refactor modifies a set of ASTs.
//
// The returned Edit will apply the same changes to another Ast.
func Refactor(asts []*syntax.Ast,
	opt RefactorConfig) (Edit, error) {
	edits := make(editSet, 0, 2+len(opt.TopCalls))
	for _, rename := range opt.Rename {
		callable := getCallable(rename.Callable, asts)
		if callable == nil {
			return edits, fmt.Errorf("callable %s not found", rename.Callable)
		}
		edit := RenameCallable(callable, rename.NewName, asts)
		if edit != nil {
			edits = append(edits, edit)
			for _, ast := range asts {
				// Run this on the compiled ASTs so removeCalls has
				// an up-to-date version of the pipelines.
				if _, err := edit.Apply(ast); err != nil {
					return edits, fmt.Errorf("applying edit: %w", err)
				}
			}
		}
	}
	for _, rename := range opt.RenameInParam {
		callable := getCallable(rename.Callable, asts)
		if callable == nil {
			return edits, fmt.Errorf("callable %s not found", rename.Callable)
		}
		edit := RenameInput(callable, rename.Param, rename.NewName, asts)
		if edit != nil {
			edits = append(edits, edit)
			for _, ast := range asts {
				// Run this on the compiled ASTs so removeCalls has
				// an up-to-date version of the pipelines.
				if _, err := edit.Apply(ast); err != nil {
					return edits, fmt.Errorf("applying edit: %w", err)
				}
			}
		}
	}
	for _, rename := range opt.RenameOutParam {
		callable := getCallable(rename.Callable, asts)
		if callable == nil {
			return edits, fmt.Errorf("callable %s not found", rename.Callable)
		}
		edit := RenameOutput(callable, rename.Param, rename.NewName, asts)
		if edit != nil {
			edits = append(edits, edit)
			for _, ast := range asts {
				// Run this on the compiled ASTs so removeCalls has
				// an up-to-date version of the pipelines.
				if _, err := edit.Apply(ast); err != nil {
					return edits, fmt.Errorf("applying edit: %w", err)
				}
			}
		}
	}
	for _, removeParam := range opt.RemoveInParams {
		cname := removeParam.Callable
		param := removeParam.Param
		callable := getCallable(cname, asts)
		if callable == nil {
			return edits, fmt.Errorf("callable %s not found", cname)
		}
		edit := RemoveInputParam(callable, param, asts)
		if edit != nil {
			edits = append(edits, edit)
			if opt.RemoveCalls || len(opt.TopCalls) > 0 {
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
	for _, removeParam := range opt.RemoveOutParams {
		cname := removeParam.Callable
		param := removeParam.Param
		callable := getCallable(cname, asts)
		if callable == nil {
			return edits, fmt.Errorf("callable %s not found", cname)
		}
		edit, err := RemoveOutputParam(callable, param, asts)
		if err != nil {
			return edits, err
		}
		if edit != nil {
			edits = append(edits, edit)
			if opt.RemoveCalls || len(opt.TopCalls) > 0 {
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
	if opt.RemoveCalls || len(opt.TopCalls) > 0 {
		changes := true
		for changes {
			changes = false
			if opt.RemoveCalls {
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
			if len(opt.TopCalls) > 0 {
				edit := RemoveUnusedOutputs(opt.TopCalls, asts)
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

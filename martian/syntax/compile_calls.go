// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// compile/check calls and callables.

package syntax

import (
	"fmt"
	"strings"
)

func (callables *Callables) compile(global *Ast) error {
	var errs ErrorList
	if len(callables.List) > 0 && callables.Table == nil {
		callables.Table = make(map[string]Callable, len(callables.List))
	}
	for _, callable := range callables.List {
		// Check for duplicates
		if existing, ok := callables.Table[callable.GetId()]; ok {
			var msg strings.Builder
			fmt.Fprintf(&msg,
				"DuplicateNameError: %s '%s' was already declared when encountered again",
				callable.Type(), callable.GetId())
			if node := existing.getNode(); node != nil {
				msg.WriteString(".\n  Previous declaration at ")
				node.Loc.writeTo(&msg, "      ")
				msg.WriteRune('\n')
			}
			errs = append(errs, global.err(callable, msg.String()))
		} else {
			callables.Table[callable.GetId()] = callable
		}
	}
	return errs.If()
}

// If call statement present, check the call and its bindings.
func (global *Ast) compileCall() error {
	if global.Call != nil {
		callable, ok := global.Callables.Table[global.Call.DecId]
		if !ok {
			return global.err(global.Call,
				"ScopeNameError: '%s' is not defined in this scope",
				global.Call.DecId)
		}
		if err := global.Call.Bindings.compile(global,
			nil, callable.GetInParams()); err != nil {
			return err
		}
		if err := global.Call.Modifiers.compile(global,
			nil, global.Call); err != nil {
			return err
		}
		if global.Call.Modifiers.Bindings != nil {
			if _, ok := global.Call.Modifiers.Bindings.Table[disabled]; ok {
				return global.err(global.Call,
					"UnsupportedTagError: Top-level call cannot be disabled.")
			}
			if global.Call.Modifiers.Preflight {
				return global.err(global.Call,
					"UnsupportedTagError: Top-level call cannot be preflight.")
			}
		}
		if err := global.Call.checkMappings(global, nil); err != nil {
			return err
		}
	}
	return nil
}

// checkMappings fills the MapOver list for the call and verifies (to the extent
// possible at check time) that the split parameters are either all arrays with
// the same lengths or all maps with the same keys.
//
// While it is always possible to verify that the mapped parameters are either
// all arrays or all maps, it is only possible to verify the array lengths or
// map keys if the source maps or arrays are either literals or references to
// map calls in the same pipeline.
func (call *CallStm) checkMappings(global *Ast, pipeline *Pipeline) error {
	if call.Bindings == nil || call.Mapping == nil {
		return nil
	}
	var firstSplit *BindStm
	var errs ErrorList
	for _, binding := range call.Bindings.List {
		if spe, ok := binding.Exp.(*SplitExp); ok {
			if spe.Call == nil {
				spe.Call = call
			}
			if firstSplit == nil {
				firstSplit = binding
			}
			switch exp := spe.Value.(type) {
			case *ArrayExp:
				if src, err := MergeMapCallSources(call.Mapping, exp); err != nil {
					errs = append(errs, &InconsistentMapCallError{
						Call:     call,
						Pipeline: pipeline.GetId(),
						Inner:    err,
					})
				} else {
					call.Mapping = src
				}
			case *MapExp:
				if src, err := MergeMapCallSources(call.Mapping, exp); err != nil {
					errs = append(errs, &InconsistentMapCallError{
						Call:     call,
						Pipeline: pipeline.GetId(),
						Inner:    err,
					})
				} else {
					call.Mapping = src
				}
			case *RefExp:
				if t, mapping, err := exp.resolveType(global, pipeline); err != nil {
					errs = append(errs, &wrapError{
						innerError: err,
						loc:        exp.Node.Loc,
					})
				} else if mapping != nil {
					if src, err := MergeMapCallSources(mapping, call); err != nil {
						errs = append(errs, &InconsistentMapCallError{
							Call:     call,
							Pipeline: pipeline.GetId(),
							Inner:    err,
						})
					} else {
						call.Mapping = src
						exp.MergeOver = []MapCallSource{src}
					}
				} else {
					if t.ArrayDim > 0 {
						exp.MergeOver = []MapCallSource{new(placeholderArrayMapSource)}
						if src, err := MergeMapCallSources(call.Mapping, exp); err != nil {
							errs = append(errs, &InconsistentMapCallError{
								Call:     call,
								Pipeline: pipeline.GetId(),
								Inner:    err,
							})
						} else {
							call.Mapping = src
							if src != exp {
								exp.MergeOver[0] = src
							}
						}
					} else if t.MapDim > 0 {
						exp.MergeOver = []MapCallSource{new(placeholderMapMapSource)}
						if src, err := MergeMapCallSources(call.Mapping, exp); err != nil {
							errs = append(errs, &InconsistentMapCallError{
								Call:     call,
								Pipeline: pipeline.GetId(),
								Inner:    err,
							})
						} else {
							call.Mapping = src
							if src != exp {
								exp.MergeOver[0] = src
							}
						}
					} else {
						errs = append(errs, &wrapError{
							innerError: &IncompatibleTypeError{
								Message: "SplitTypeMismatch: cannot split over a " + t.Tname,
							},
							loc: spe.Node.Loc,
						})
					}
				}
			}
			spe.Source = call.Mapping
		}
	}
	return errs.If()
}

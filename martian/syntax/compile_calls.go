// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// compile/check calls and callables.

package syntax

import (
	"fmt"
	"strings"
)

func (callables *Callables) compile(global *Ast) error {
	var errs ErrorList
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
	}
	return nil
}

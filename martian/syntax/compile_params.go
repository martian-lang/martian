// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// compile/check params, bindings, and expressions.

package syntax

import (
	"fmt"
	"regexp"
	"strings"
)

func (params *InParams) compile(global *Ast) error {
	var errs ErrorList
	for _, param := range params.List {
		// Check for duplicates
		if _, ok := params.Table[param.GetId()]; ok {
			errs = append(errs, global.err(param,
				"DuplicateNameError: parameter '%s' was already declared when encountered again",
				param.GetId()))
		} else {
			params.Table[param.GetId()] = param
		}

		// Check that types exist.
		if _, ok := global.TypeTable[param.GetTname()]; !ok {
			errs = append(errs, global.err(param,
				"TypeError: undefined type '%s'",
				param.GetTname()))
		}

		// Cache if param is file or path.
		t, ok := global.TypeTable[param.GetTname()]
		param.setIsFile(ok && t.IsFile())
	}
	return errs.If()
}

func checkLegalUnixFilename(name string) error {
	if len(name) > 255 {
		return fmt.Errorf("too long")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("reserved name")
	}
	for _, c := range name {
		if c == '/' {
			return fmt.Errorf("'/' is not allowed in filenames")
		} else if c == 0 {
			return fmt.Errorf("null characters are not allowed in filenames")
		}
	}
	return nil
}

func (param *OutParam) compile(global *Ast) error {
	var errs ErrorList
	// Check that types exist.
	if _, ok := global.TypeTable[param.GetTname()]; !ok {
		errs = append(errs, global.err(param,
			"TypeError: undefined type '%s'",
			param.GetTname()))
	}

	// Cache if param is file or path.
	t, ok := global.TypeTable[param.GetTname()]
	param.setIsFile(ok && t.IsFile())
	if param.IsFile() {
		if param.OutName != "" {
			if err := checkLegalUnixFilename(param.OutName); err != nil {
				errs = append(errs, global.err(
					param,
					"OutName: illegal filename %q: %v",
					param.OutName, err))
			}
		}
	}
	return errs.If()
}

func (params *OutParams) compile(global *Ast) error {
	var errs ErrorList
	for _, param := range params.List {
		// Check for duplicates
		if _, ok := params.Table[param.GetId()]; ok {
			errs = append(errs, global.err(param,
				"DuplicateNameError: parameter '%s' was already declared when encountered again",
				param.GetId()))
		} else {
			params.Table[param.GetId()] = param
		}
		if err := param.compile(global); err != nil {
			errs = append(errs, err)
		}
	}
	return errs.If()
}

var windowsDeviceNameRe = regexp.MustCompile(`^(?:(?i:CON|PRN|AUX|NUL)|(?i:COM|LPT)[0-9])(?:$|\.)`)

func checkLegalFilename(name string) error {
	if len(name) > 128 {
		return fmt.Errorf("too long")
	}
	for _, c := range name {
		switch c {
		case '|', '/', '\\', '<', '>', '?', '*', ':', '"',
			'\a', '\b', '\f', '\n', '\r', '\t', '\v', 0:
			return fmt.Errorf("'%c' is not a legal character", c)
		}
	}
	if strings.HasSuffix(name, " ") {
		return fmt.Errorf("name cannot end with space")
	}
	if strings.HasSuffix(name, ".") {
		return fmt.Errorf("name cannot end with .")
	}
	if n := windowsDeviceNameRe.FindString(name); n != "" {
		return fmt.Errorf(
			"%s conflicts with a reserved windows device name",
			n)
	}
	return nil
}

func (param *OutParam) checkFilename() error {
	if !param.IsFile() {
		return nil
	}
	if param.OutName != "" {
		if err := checkLegalFilename(param.OutName); err != nil {
			return &wrapError{
				innerError: fmt.Errorf("out file name %q for parameter %s is not "+
					"legal under Microsoft Windows operating systems "+
					"and may cause issues for users who export their "+
					"results to such filesystems: %v",
					param.OutName, param.Id, err),
				loc: param.Node.Loc,
			}
		}
	} else if windowsDeviceNameRe.MatchString(param.Id) {
		return &wrapError{
			innerError: fmt.Errorf("parameter %s, which is a file output, "+
				"conflicts with a 'device file' name on Microsoft Windows, "+
				"and will cause issues for users on such filesystems",
				param.Id),
			loc: param.Node.Loc,
		}
	}
	return nil
}

// Returns an error if one or more of the output parameters will generate
// file names which are potentially problematic.
func (params *OutParams) CheckFilenames() error {
	if params == nil {
		return nil
	}
	var errs ErrorList
	for _, param := range params.List {
		if err := param.checkFilename(); err != nil {
			errs = append(errs, err)
		}
	}
	return errs.If()
}

func (exp *ValExp) resolveType(global *Ast, pipeline *Pipeline) ([]string, int, error) {
	switch exp.getKind() {

	// Handle scalar types.
	case KindInt, KindFloat, KindBool, KindMap, KindNull, KindPath:
		return []string{string(exp.getKind())}, 0, nil

	// Handle strings (which could be files too).
	case KindString:
		return []string{"string"}, 0, nil

	// Array: [ 1, 2 ]
	case KindArray:
		subexps := exp.Value.([]Exp)
		if len(subexps) == 0 {
			return []string{KindNull}, 1, nil
		}
		arrayTypes := make([]string, 0, len(subexps))
		commonArrayDim := -1
		var errs ErrorList
		for _, subexp := range subexps {
			arrayKind, arrayDim, err := subexp.resolveType(global, pipeline)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			arrayTypes = append(arrayTypes, arrayKind...)
			if commonArrayDim == -1 {
				commonArrayDim = arrayDim
			} else if commonArrayDim != arrayDim {
				errs = append(errs, global.err(exp,
					"TypeMismatchError: inconsistent array dimensions %d vs %d",
					commonArrayDim, arrayDim))
			}
		}
		return arrayTypes, commonArrayDim + 1, errs.If()
	// File: look for matching t in user/file type table
	case KindFile:
		for userType := range global.UserTypeTable {
			if strings.HasSuffix(exp.Value.(string), userType) {
				return []string{userType}, 0, nil
			}
		}
	}
	return []string{"unknown"}, 0, nil
}

func (exp *RefExp) resolveType(global *Ast, pipeline *Pipeline) ([]string, int, error) {
	if pipeline == nil {
		return []string{""}, 0, global.err(exp,
			"ReferenceError: this binding cannot be resolved outside of a stage or pipeline.")
	}

	switch exp.getKind() {

	// Param: self.myparam
	case KindSelf:
		param, ok := pipeline.GetInParams().Table[exp.Id]
		if !ok {
			return []string{""}, 0, global.err(exp,
				"ScopeNameError: '%s' is not an input parameter of pipeline '%s'",
				exp.Id, pipeline.GetId())
		}
		return []string{param.GetTname()}, param.GetArrayDim(), nil

	// Call: STAGE.myoutparam or STAGE
	case KindCall:
		callable, ok := pipeline.Callables.Table[exp.Id]
		if !ok {
			return []string{""}, 0, global.err(exp,
				"ScopeNameError: '%s' is not called in pipeline '%s'",
				exp.Id, pipeline.Id)
		}
		// Check referenced output is actually an output of the callable.
		param, ok := callable.GetOutParams().Table[exp.OutputId]
		if !ok {
			return []string{""}, 0, global.err(exp,
				"NoSuchOutputError: '%s' is not an output parameter of '%s'",
				exp.OutputId, callable.GetId())
		}

		return []string{param.GetTname()}, param.GetArrayDim(), nil
	}
	return []string{"unknown"}, 0, nil
}

func (bindings *BindStms) compile(global *Ast, pipeline *Pipeline, params *InParams) error {
	// Check the bindings
	var errs ErrorList
	for _, binding := range bindings.List {
		// Collect bindings by id so we can check that all params are bound.
		if _, ok := bindings.Table[binding.Id]; ok {
			errs = append(errs, global.err(binding,
				"DuplicateBinding: '%s' already bound in this call",
				binding.Id))
		}
		// Building the bindings table could also happen in the grammar rules,
		// but then we lose the ability to detect duplicate parameters as we're
		// doing right above this comment. So leave this here.
		bindings.Table[binding.Id] = binding

		if err := binding.compile(global, pipeline, params); err != nil {
			errs = append(errs, err)
		}
	}

	if params != nil {
		// Check that all input params of the called segment are bound.
		for _, param := range params.List {
			if _, ok := bindings.Table[param.GetId()]; !ok {
				errs = append(errs, global.err(bindings,
					"ArgumentNotSuppliedError: no argument supplied for parameter '%s'",
					param.GetId()))
			}
		}
	}
	return errs.If()
}

func (binding *BindStm) compile(global *Ast, pipeline *Pipeline, params *InParams) error {
	// Make sure the bound-to id is a declared parameter of the callable.
	param, ok := params.Table[binding.Id]
	if !ok {
		return global.err(binding, "ArgumentError: '%s' is not a valid parameter",
			binding.Id)
	}

	// Typecheck the binding and cache the type.
	valueTypes, arrayDim, err := binding.Exp.resolveType(global, pipeline)
	if err != nil {
		return err
	}

	// Check for array match
	if binding.Sweep {
		if arrayDim == 0 {
			return global.err(binding,
				"TypeMismatchError: got non-array value for sweep parameter '%s'",
				param.GetId())
		}
		arrayDim -= 1
	}
	if param.GetArrayDim() != arrayDim {
		if param.GetArrayDim() == 0 && arrayDim > 0 {
			return global.err(binding,
				"TypeMismatchError: got array value for non-array parameter '%s'",
				param.GetId())
		} else if param.GetArrayDim() > 0 && arrayDim == 0 {
			// Allow an array-decorated parameter to accept null values.
			if len(valueTypes) < 1 || valueTypes[0] != KindNull {
				return global.err(binding,
					"TypeMismatchError: expected array of '%s' for '%s'",
					param.GetTname(), param.GetId())
			}
		} else {
			return global.err(binding,
				"TypeMismatchError: got %d-dimensional array value for %d-dimensional array parameter '%s'",
				arrayDim, param.GetArrayDim(), param.GetId())
		}
	}

	for _, valueType := range valueTypes {
		if !global.checkTypeMatch(param.GetTname(), valueType) {
			return global.err(binding,
				"TypeMismatchError: expected type '%s' for '%s' but got '%s' instead",
				param.GetTname(), param.GetId(), valueType)
		}
	}
	binding.Tname = param.GetTname()
	return nil
}

func (bindings *BindStms) compileReturns(global *Ast, pipeline *Pipeline, params *OutParams) error {
	// Check the bindings
	var errs ErrorList
	for _, binding := range bindings.List {
		// Collect bindings by id so we can check that all params are bound.
		if _, ok := bindings.Table[binding.Id]; ok {
			errs = append(errs, global.err(binding,
				"DuplicateBinding: '%s' already bound in this call",
				binding.Id))
		}
		// Building the bindings table could also happen in the grammar rules,
		// but then we lose the ability to detect duplicate parameters as we're
		// doing right above this comment. So leave this here.
		bindings.Table[binding.Id] = binding

		if err := binding.compileReturns(global, pipeline, params); err != nil {
			errs = append(errs, err)
		}
	}

	if params != nil {
		// Check that all input params of the called segment are bound.
		for _, param := range params.List {
			if _, ok := bindings.Table[param.GetId()]; !ok {
				errs = append(errs, global.err(bindings,
					"ArgumentNotSuppliedError: no argument supplied for parameter '%s'",
					param.GetId()))
			}
		}
	}
	return errs.If()
}

func (binding *BindStm) compileReturns(global *Ast, pipeline *Pipeline, params *OutParams) error {
	// Make sure the bound-to id is a declared parameter of the callable.
	param, ok := params.Table[binding.Id]
	if !ok {
		return global.err(binding, "ArgumentError: '%s' is not a valid parameter",
			binding.Id)
	}

	// Typecheck the binding and cache the type.
	valueTypes, arrayDim, err := binding.Exp.resolveType(global, pipeline)
	if err != nil {
		return err
	}

	// Check for array match
	if binding.Sweep {
		if arrayDim == 0 {
			return global.err(binding,
				"TypeMismatchError: got non-array value for sweep parameter '%s'",
				param.GetId())
		}
		arrayDim -= 1
	}
	if param.GetArrayDim() != arrayDim {
		if param.GetArrayDim() == 0 && arrayDim > 0 {
			return global.err(binding,
				"TypeMismatchError: got array value for non-array parameter '%s'",
				param.GetId())
		} else if param.GetArrayDim() > 0 && arrayDim == 0 {
			// Allow an array-decorated parameter to accept null values.
			if len(valueTypes) < 1 || valueTypes[0] != KindNull {
				return global.err(binding,
					"TypeMismatchError: expected array of '%s' for '%s'",
					param.GetTname(), param.GetId())
			}
		} else {
			return global.err(binding,
				"TypeMismatchError: got %d-dimensional array value for %d-dimensional array parameter '%s'",
				arrayDim, param.GetArrayDim(), param.GetId())
		}
	}

	for _, valueType := range valueTypes {
		if !global.checkTypeMatch(param.GetTname(), valueType) {
			return global.err(binding,
				"TypeMismatchError: expected type '%s' for '%s' but got '%s' instead",
				param.GetTname(), param.GetId(), valueType)
		}
	}
	binding.Tname = param.GetTname()
	return nil
}

func getBoundParamIds(uexp Exp) []string {
	switch exp := uexp.(type) {
	case *RefExp:
		if exp.Kind == KindSelf {
			return []string{exp.Id}
		}
	case *ValExp:
		if exp.Kind == KindArray {
			var ids []string
			for _, subExp := range exp.Value.([]Exp) {
				ids = append(ids, getBoundParamIds(subExp)...)
			}
			return ids
		}
	}
	return nil
}

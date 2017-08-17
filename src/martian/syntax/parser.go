//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO semantic checking.
//
package syntax

import (
	"fmt"
	"io/ioutil"
	"martian/util"
	"os"
	"path/filepath"
	"strings"
)

//
// Semantic Checking Methods
//
func (global *Ast) err(nodable AstNodable, msg string, v ...interface{}) error {
	return &AstError{global, nodable.getNode(), fmt.Sprintf(msg, v...)}
}

func (callables *Callables) check(global *Ast) error {
	for _, callable := range callables.List {
		// Check for duplicates
		if _, ok := callables.Table[callable.GetId()]; ok {
			return global.err(callable, "DuplicateNameError: stage or pipeline '%s' was already declared when encountered again", callable.GetId())
		}
		callables.Table[callable.GetId()] = callable
	}
	return nil
}

func (params *Params) check(global *Ast) error {
	for _, param := range params.List {
		// Check for duplicates
		if _, ok := params.Table[param.GetId()]; ok {
			return global.err(param, "DuplicateNameError: parameter '%s' was already declared when encountered again", param.GetId())
		}
		params.Table[param.GetId()] = param

		// Check that types exist.
		if _, ok := global.TypeTable[param.GetTname()]; !ok {
			return global.err(param, "TypeError: undefined type '%s'", param.GetTname())
		}

		// Cache if param is file or path.
		_, ok := global.UserTypeTable[param.GetTname()]
		param.setIsFile(ok)
	}
	return nil
}

func (exp *ValExp) resolveType(global *Ast, callable Callable) ([]string, int, error) {
	switch exp.getKind() {

	// Handle scalar types.
	case "int", "float", "bool", "path", "map", "null":
		return []string{exp.getKind()}, 0, nil

	// Handle strings (which could be files too).
	case "string":
		for userType := range global.UserTypeTable {
			if strings.HasSuffix(exp.Value.(string), userType) {
				return []string{"string", userType}, 0, nil
			}
		}
		return []string{"string"}, 0, nil

	// Array: [ 1, 2 ]
	case "array":
		for _, subexp := range exp.Value.([]Exp) {
			arrayKind, arrayDim, err := subexp.resolveType(global, callable)
			return arrayKind, arrayDim + 1, err
		}
		return []string{"null"}, 1, nil
	// File: look for matching t in user/file type table
	case "file":
		for userType := range global.UserTypeTable {
			if strings.HasSuffix(exp.Value.(string), userType) {
				return []string{userType}, 0, nil
			}
		}
	}
	return []string{"unknown"}, 0, nil
}

func (exp *RefExp) resolveType(global *Ast, callable Callable) ([]string, int, error) {
	if callable == nil {
		return []string{""}, 0, global.err(exp, "ReferenceError: this binding cannot be resolved outside of a stage or pipeline.")
	}

	switch exp.getKind() {

	// Param: self.myparam
	case "self":
		param, ok := callable.GetInParams().Table[exp.Id]
		if !ok {
			return []string{""}, 0, global.err(exp, "ScopeNameError: '%s' is not an input parameter of pipeline '%s'", exp.Id, callable.GetId())
		}
		return []string{param.GetTname()}, param.GetArrayDim(), nil

	// Call: STAGE.myoutparam or STAGE
	case "call":
		// Check referenced callable is acutally called in this scope.
		pipeline, ok := callable.(*Pipeline)
		if !ok {
			return []string{""}, 0, global.err(exp, "ScopeNameError: '%s' is not called in pipeline '%s'", exp.Id, callable.GetId())
		} else {
			callable, ok := pipeline.Callables.Table[exp.Id]
			if !ok {
				return []string{""}, 0, global.err(exp, "ScopeNameError: '%s' is not called in pipeline '%s'", exp.Id, pipeline.Id)
			}
			// Check referenced output is actually an output of the callable.
			param, ok := callable.GetOutParams().Table[exp.OutputId]
			if !ok {
				return []string{""}, 0, global.err(exp, "NoSuchOutputError: '%s' is not an output parameter of '%s'", exp.OutputId, callable.GetId())
			}

			return []string{param.GetTname()}, param.GetArrayDim(), nil
		}
	}
	return []string{"unknown"}, 0, nil
}

func checkTypeMatch(paramType string, valueType string) bool {
	return (valueType == "null" ||
		paramType == valueType ||
		(paramType == "path" && valueType == "string") ||
		(paramType == "float" && valueType == "int"))
}

func (bindings *BindStms) check(global *Ast, callable Callable, params *Params) error {
	// Check the bindings
	for _, binding := range bindings.List {
		// Collect bindings by id so we can check that all params are bound.
		if _, ok := bindings.Table[binding.Id]; ok {
			return global.err(binding, "DuplicateBinding: '%s' already bound in this call", binding.Id)
		}
		// Building the bindings table could also happen in the grammar rules,
		// but then we lose the ability to detect duplicate parameters as we're
		// doing right above this comment. So leave this here.
		bindings.Table[binding.Id] = binding

		// Make sure the bound-to id is a declared parameter of the callable.
		param, ok := params.Table[binding.Id]
		if !ok {
			return global.err(binding, "ArgumentError: '%s' is not a valid parameter", binding.Id)
		}

		// Typecheck the binding and cache the type.
		valueTypes, arrayDim, err := binding.Exp.resolveType(global, callable)
		if err != nil {
			return err
		}

		// Check for array match
		if binding.Sweep {
			if arrayDim == 0 {
				return global.err(binding, "TypeMismatchError: got non-array value for sweep parameter '%s'", param.GetId())
			}
			arrayDim -= 1
		}
		if param.GetArrayDim() != arrayDim {
			if param.GetArrayDim() == 0 && arrayDim > 0 {
				return global.err(binding, "TypeMismatchError: got array value for non-array parameter '%s'", param.GetId())
			} else if param.GetArrayDim() > 0 && arrayDim == 0 {
				// Allow an array-decorated parameter to accept null values.
				if len(valueTypes) < 1 || valueTypes[0] != "null" {
					return global.err(binding, "TypeMismatchError: expected array of '%s' for '%s'", param.GetTname(), param.GetId())
				}
			} else {
				return global.err(binding, "TypeMismatchError: got %d-dimensional array value for %d-dimensional array parameter '%s'", arrayDim, param.GetArrayDim(), param.GetId())
			}
		}

		anymatch := false
		lastType := ""
		for _, valueType := range valueTypes {
			anymatch = anymatch || checkTypeMatch(param.GetTname(), valueType)
			lastType = valueType
		}
		if !anymatch {
			return global.err(binding, "TypeMismatchError: expected type '%s' for '%s' but got '%s' instead", param.GetTname(), param.GetId(), lastType)
		}
		binding.Tname = param.GetTname()
	}

	// Check that all input params of the called segment are bound.
	for _, param := range params.List {
		if _, ok := bindings.Table[param.GetId()]; !ok {
			return global.err(bindings, "ArgumentNotSuppliedError: no argument supplied for parameter '%s'", param.GetId())
		}
	}
	return nil
}

func (global *Ast) check(stagecodePaths []string, checkSrcPath bool) error {
	// Build type table, starting with builtins. Duplicates allowed.
	builtinTypes := []*BuiltinType{
		{"string"},
		{"int"},
		{"float"},
		{"bool"},
		{"path"},
		{"file"},
		{"map"},
	}
	for _, builtinType := range builtinTypes {
		global.TypeTable[builtinType.Id] = builtinType
	}
	for _, userType := range global.UserTypes {
		global.TypeTable[userType.Id] = userType
		global.UserTypeTable[userType.Id] = userType
	}

	// Check for duplicate names amongst callables.
	if err := global.Callables.check(global); err != nil {
		return err
	}

	// Check stage declarations.
	for _, stage := range global.Stages {
		// Check in parameters.
		if err := stage.InParams.check(global); err != nil {
			return err
		}

		// Check out parameters.
		if err := stage.OutParams.check(global); err != nil {
			return err
		}

		if checkSrcPath {
			// Check existence of src path.
			if _, found := util.SearchPaths(stage.Src.Path, stagecodePaths); !found {
				// Exempt exec stages
				if stage.Src.Lang != "exec" {
					stagecodePathsList := strings.Join(stagecodePaths, ", ")
					return global.err(stage, "SourcePathError: searched (%s) but stage source path not found '%s'", stagecodePathsList, stage.Src.Path)
				}
			}
		}
		// Check split parameters.
		if stage.SplitParams != nil {
			if err := stage.SplitParams.check(global); err != nil {
				return err
			}
		}
	}

	// Check pipeline declarations.
	for _, pipeline := range global.Pipelines {
		// Check in parameters.
		if err := pipeline.InParams.check(global); err != nil {
			return err
		}

		// Check out parameters.
		if err := pipeline.OutParams.check(global); err != nil {
			return err
		}

		// Check calls.
		for _, call := range pipeline.Calls {
			// Check for duplicate calls.
			if _, ok := pipeline.Callables.Table[call.Id]; ok {
				return global.err(call, "DuplicateCallError: '%s' was already called when encountered again", call.Id)
			}
			// Check we're calling something declared.
			callable, ok := global.Callables.Table[call.Id]
			if !ok {
				return global.err(call, "ScopeNameError: '%s' is not defined in this scope", call.Id)
			}
			// Save the valid callables for this scope.
			pipeline.Callables.Table[call.Id] = callable

			// Check to make sure if local, preflight or volatile is declared, callable is a stage
			if _, ok := callable.(*Stage); !ok {
				if call.Modifiers.Local {
					return global.err(call, "UnsupportedTagError: Pipeline '%s' cannot be called with 'local' tag", call.Id)
				}
				if call.Modifiers.Preflight {
					return global.err(call, "UnsupportedTagError: Pipeline '%s' cannot be called with 'preflight' tag", call.Id)
				}
				if call.Modifiers.Volatile {
					return global.err(call, "UnsupportedTagError: Pipeline '%s' cannot be called with 'volatile' tag", call.Id)
				}
			}
			if call.Modifiers.Preflight {
				for _, binding := range call.Bindings.List {
					if binding.Exp.getKind() == "call" {
						return global.err(call, "PreflightBindingError: Preflight stage '%s' cannot have input parameter bound to output parameter of another stage or pipeline", call.Id)
					}
				}
				if len(callable.GetOutParams().List) > 0 {
					return global.err(call, "PreflightOutputError: Preflight stage '%s' cannot have any output parameters", call.Id)
				}
			}

			// Check the bindings
			if err := call.Bindings.check(global, pipeline, callable.GetInParams()); err != nil {
				return err
			}

			// Check that all input params of the callable are bound.
			for _, param := range callable.GetInParams().List {
				if _, ok := call.Bindings.Table[param.GetId()]; !ok {
					return global.err(call, "ArgumentNotSuppliedError: no argument supplied for parameter '%s'", param.GetId())
				}
			}
		}
	}

	// Doing these in a separate loop gives the user better incremental
	// error messages while writing a long pipeline declaration.
	for _, pipeline := range global.Pipelines {
		// Check all pipeline input params are bound in a call statement.
		boundParamIds := map[string]bool{}
		for _, call := range pipeline.Calls {
			for _, binding := range call.Bindings.List {
				refexp, ok := binding.Exp.(*RefExp)
				if ok {
					boundParamIds[refexp.Id] = true
				}
			}
		}
		for _, param := range pipeline.InParams.List {
			if _, ok := boundParamIds[param.GetId()]; !ok {
				return global.err(param, "UnusedInputError: no calls use pipeline input parameter '%s'", param.GetId())
			}
		}

		// Check all pipeline output params are returned.
		returnedParamIds := map[string]bool{}
		for _, binding := range pipeline.Ret.Bindings.List {
			returnedParamIds[binding.Id] = true
		}
		for _, param := range pipeline.OutParams.List {
			if _, ok := returnedParamIds[param.GetId()]; !ok {
				return global.err(pipeline.Ret, "ReturnError: pipeline output parameter '%s' is not returned", param.GetId())
			}
		}

		// Check return bindings.
		if err := pipeline.Ret.Bindings.check(global, pipeline, pipeline.OutParams); err != nil {
			return err
		}
	}

	// If call statement present, check the call and its bindings.
	if global.Call != nil {
		callable, ok := global.Callables.Table[global.Call.Id]
		if !ok {
			return global.err(global.Call, "ScopeNameError: '%s' is not defined in this scope", global.Call.Id)
		}
		if err := global.Call.Bindings.check(global, callable, callable.GetInParams()); err != nil {
			return err
		}
	}
	return nil
}

//
// Parser interface, called by runtime.
//
func ParseSource(src string, srcPath string, incPaths []string, checkSrc bool) (string, []string, *Ast, error) {
	// Add the source file's own folder to the include path for
	// resolving both @includes and stage src paths.
	incPaths = append([]string{filepath.Dir(srcPath)}, incPaths...)

	// Add PATH environment variable to the stage code path
	stagecodePaths := append(incPaths, strings.Split(os.Getenv("PATH"), ":")...)

	// Preprocess: generate new source and a locmap.
	postsrc, ifnames, locmap, err := preprocess(src, filepath.Base(srcPath), incPaths)
	if err != nil {
		return "", nil, nil, err
	}
	//printSourceMap(postsrc, locmap)

	// Parse the source into an AST and attach the locmap.
	ast, perr := yaccParse(postsrc, locmap)
	if perr != nil { // perr is an mmLexInfo struct
		// Guard against index out of range, which can happen if there is syntax error
		// at the end of the file, e.g. forgetting to put a close paren at the end of
		// and invocation call/file.
		if perr.loc >= len(locmap) {
			perr.loc = len(locmap) - 1
		}
		return "", nil, nil, &ParseError{perr.token, locmap[perr.loc].fname, locmap[perr.loc].loc}
	}

	// Run semantic checks.
	if err := ast.check(stagecodePaths, checkSrc); err != nil {
		return "", nil, nil, err
	}

	return postsrc, ifnames, ast, nil
}

// Compile an MRO file in cwd or mroPaths.
func Compile(fpath string, mroPaths []string, checkSrcPath bool) (string, []string, *Ast, error) {
	if data, err := ioutil.ReadFile(fpath); err != nil {
		return "", nil, nil, err
	} else {
		return ParseSource(string(data), fpath, mroPaths, checkSrcPath)
	}
}

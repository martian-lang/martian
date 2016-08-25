//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO semantic checking.
//
package core

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

//
// Semantic Checking Methods
//
func (global *Ast) err(nodable AstNodable, msg string, v ...interface{}) {
	global.Errors = append(global.Errors, &AstError{global, nodable.getNode(), fmt.Sprintf(msg, v...)})
}

func (callables *Callables) check(global *Ast) {
	for _, callable := range callables.List {
		// Check for duplicates
		if _, ok := callables.Table[callable.getId()]; ok {
			global.err(callable, "DuplicateNameError: stage or pipeline '%s' was already declared when encountered again", callable.getId())
		}
		callables.Table[callable.getId()] = callable
	}
}

func (params *Params) check(global *Ast) {
	for _, param := range params.List {
		// Check for duplicates
		if _, ok := params.Table[param.getId()]; ok {
			global.err(param, "DuplicateNameError: parameter '%s' was already declared when encountered again", param.getId())
		}
		params.Table[param.getId()] = param

		// Check that types exist.
		if _, ok := global.TypeTable[param.getTname()]; !ok {
			global.err(param, "TypeError: undefined type '%s'", param.getTname())
		}

		// Cache if param is file or path.
		_, ok := global.UserTypeTable[param.getTname()]
		param.setIsFile(ok)

	}
}

func (exp *ValExp) resolveType(global *Ast, callable Callable) ([]string, int) {
	switch exp.getKind() {

	// Handle scalar types.
	case "int", "float", "bool", "path", "map", "null":
		return []string{exp.getKind()}, 0

	// Handle strings (which could be files too).
	case "string":
		for t, _ := range global.UserTypeTable {
			if strings.HasSuffix(exp.Value.(string), t) {
				return []string{"string", t}, 0
			}
		}
		return []string{"string"}, 0

	// Array: [ 1, 2 ]
	case "array":
		for _, subexp := range exp.Value.([]Exp) {
			arrayKind, arrayDim := subexp.resolveType(global, callable)
			return arrayKind, arrayDim + 1
		}
		return []string{"null"}, 1
	// File: look for matching t in user/file type table
	case "file":
		for t, _ := range global.UserTypeTable {
			if strings.HasSuffix(exp.Value.(string), t) {
				return []string{t}, 0
			}
		}
	}
	return []string{"unknown"}, 0
}

func (exp *RefExp) resolveType(global *Ast, callable Callable) ([]string, int) {
	if callable == nil {
		global.err(exp, "ReferenceError: this binding cannot be resolved outside of a stage or pipeline.")
	}

	switch exp.getKind() {

	// Param: self.myparam
	case "self":
		param, ok := callable.getInParams().Table[exp.Id]
		if !ok {
			global.err(exp, "ScopeNameError: '%s' is not an input parameter of pipeline '%s'", exp.Id, callable.getId())
		}
		return []string{param.getTname()}, param.getArrayDim()

	// Call: STAGE.myoutparam or STAGE
	case "call":
		// Check referenced callable is acutally called in this scope.
		pipeline, ok := callable.(*Pipeline)
		if !ok {
			global.err(exp, "ScopeNameError: '%s' is not called in pipeline '%s'", exp.Id, callable.getId())
		} else {
			callable, ok := pipeline.Callables.Table[exp.Id]
			if !ok {
				global.err(exp, "ScopeNameError: '%s' is not called in pipeline '%s'", exp.Id, pipeline.Id)
			}
			// Check referenced output is actually an output of the callable.
			param, ok := callable.getOutParams().Table[exp.OutputId]
			if !ok {
				global.err(exp, "NoSuchOutputError: '%s' is not an output parameter of '%s'", exp.OutputId, callable.getId())
			}

			return []string{param.getTname()}, param.getArrayDim()
		}
	}
	return []string{"unknown"}, 0
}

func checkTypeMatch(paramType string, valueType string) bool {
	return (valueType == "null" ||
		paramType == valueType ||
		(paramType == "path" && valueType == "string") ||
		(paramType == "float" && valueType == "int"))
}

func (bindings *BindStms) check(global *Ast, callable Callable, params *Params) {
	// Check the bindings
	for _, binding := range bindings.List {
		// Collect bindings by id so we can check that all params are bound.
		if _, ok := bindings.Table[binding.Id]; ok {
			global.err(binding, "DuplicateBinding: '%s' already bound in this call", binding.Id)
		}
		// Building the bindings table could also happen in the grammar rules,
		// but then we lose the ability to detect duplicate parameters as we're
		// doing right above this comment. So leave this here.
		bindings.Table[binding.Id] = binding

		// Make sure the bound-to id is a declared parameter of the callable.
		param, ok := params.Table[binding.Id]
		if !ok {
			global.err(binding, "ArgumentError: '%s' is not a valid parameter", binding.Id)
		}

		// Typecheck the binding and cache the type.
		valueTypes, arrayDim := binding.Exp.resolveType(global, callable)

		// Check for array match
		if binding.Sweep {
			if arrayDim == 0 {
				global.err(binding, "TypeMismatchError: got non-array value for sweep parameter '%s'", param.getId())
			}
			arrayDim -= 1
		}
		if param.getArrayDim() != arrayDim {
			if param.getArrayDim() == 0 && arrayDim > 0 {
				global.err(binding, "TypeMismatchError: got array value for non-array parameter '%s'", param.getId())
			} else if param.getArrayDim() > 0 && arrayDim == 0 {
				// Allow an array-decorated parameter to accept null values.
				if len(valueTypes) < 1 || valueTypes[0] != "null" {
					global.err(binding, "TypeMismatchError: expected array of '%s' for '%s'", param.getTname(), param.getId())
				}
			} else {
				global.err(binding, "TypeMismatchError: got %d-dimensional array value for %d-dimensional array parameter '%s'", arrayDim, param.getArrayDim(), param.getId())
			}
		}

		anymatch := false
		lastType := ""
		for _, valueType := range valueTypes {
			anymatch = anymatch || checkTypeMatch(param.getTname(), valueType)
			lastType = valueType
		}
		if !anymatch {
			global.err(binding, "TypeMismatchError: expected type '%s' for '%s' but got '%s' instead", param.getTname(), param.getId(), lastType)
		}
		binding.Tname = param.getTname()
	}

	// Check that all input params of the called segment are bound.
	for _, param := range params.List {
		if _, ok := bindings.Table[param.getId()]; !ok {
			global.err(bindings, "ArgumentNotSuppliedError: no argument supplied for parameter '%s'", param.getId())
		}
	}
}

func (global *Ast) check(stagecodePaths []string, checkSrcPath bool) {
	// Build type table, starting with builtins. Duplicates allowed.
	builtinTypes := []*BuiltinType{
		&BuiltinType{"string"},
		&BuiltinType{"int"},
		&BuiltinType{"float"},
		&BuiltinType{"bool"},
		&BuiltinType{"path"},
		&BuiltinType{"file"},
		&BuiltinType{"map"},
	}
	for _, builtinType := range builtinTypes {
		global.TypeTable[builtinType.Id] = builtinType
	}
	for _, userType := range global.UserTypes {
		global.TypeTable[userType.Id] = userType
		global.UserTypeTable[userType.Id] = userType
	}

	// Check for duplicate names amongst callables.
	global.Callables.check(global)

	// Check stage declarations.
	for _, stage := range global.Stages {
		// Check in parameters.
		stage.InParams.check(global)

		// Check out parameters.
		stage.OutParams.check(global)

		if checkSrcPath {
			// Check existence of src path.
			if _, found := SearchPaths(stage.Src.Path, stagecodePaths); !found {
				stagecodePathsList := strings.Join(stagecodePaths, ", ")
				global.err(stage, "SourcePathError: searched (%s) but stage source path not found '%s'", stagecodePathsList, stage.Src.Path)
			}
		}
		// Check split parameters.
		if stage.SplitParams != nil {
			stage.SplitParams.check(global)
		}
	}

	// Check pipeline declarations.
	for _, pipeline := range global.Pipelines {
		// Check in parameters.
		pipeline.InParams.check(global)

		// Check out parameters.
		pipeline.OutParams.check(global)

		preflightCalls := []*CallStm{}

		// Check calls.
		for _, call := range pipeline.Calls {
			// Check for duplicate calls.
			if _, ok := pipeline.Callables.Table[call.Id]; ok {
				global.err(call, "DuplicateCallError: '%s' was already called when encountered again", call.Id)
			}
			// Check we're calling something declared.
			callable, ok := global.Callables.Table[call.Id]
			if !ok {
				global.err(call, "ScopeNameError: '%s' is not defined in this scope", call.Id)
			}
			// Save the valid callables for this scope.
			pipeline.Callables.Table[call.Id] = callable

			// Check to make sure if local, preflight or volatile is declared, callable is a stage
			if _, ok := callable.(*Stage); !ok {
				if call.Modifiers.Local {
					global.err(call, "UnsupportedTagError: Pipeline '%s' cannot be called with 'local' tag", call.Id)
				}
				if call.Modifiers.Preflight {
					global.err(call, "UnsupportedTagError: Pipeline '%s' cannot be called with 'preflight' tag", call.Id)
				}
				if call.Modifiers.Volatile {
					global.err(call, "UnsupportedTagError: Pipeline '%s' cannot be called with 'volatile' tag", call.Id)
				}
			}
			if call.Modifiers.Preflight {
				for _, binding := range call.Bindings.List {
					if binding.Exp.getKind() == "call" {
						global.err(call, "PreflightBindingError: Preflight stage '%s' cannot have input parameter bound to output parameter of another stage or pipeline", call.Id)
					}
				}
				if len(callable.getOutParams().List) > 0 {
					global.err(call, "PreflightOutputError: Preflight stage '%s' cannot have any output parameters", call.Id)
				}

				preflightCalls = append(preflightCalls, call)
			}

			// Check the bindings
			call.Bindings.check(global, pipeline, callable.getInParams())

			// Check that all input params of the callable are bound.
			for _, param := range callable.getInParams().List {
				if _, ok := call.Bindings.Table[param.getId()]; !ok {
					global.err(call, "ArgumentNotSuppliedError: no argument supplied for parameter '%s'", param.getId())
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
			if _, ok := boundParamIds[param.getId()]; !ok {
				global.err(param, "UnusedInputError: no calls use pipeline input parameter '%s'", param.getId())
			}
		}

		// Check all pipeline output params are returned.
		returnedParamIds := map[string]bool{}
		for _, binding := range pipeline.Ret.Bindings.List {
			returnedParamIds[binding.Id] = true
		}
		for _, param := range pipeline.OutParams.List {
			if _, ok := returnedParamIds[param.getId()]; !ok {
				global.err(pipeline.Ret, "ReturnError: pipeline output parameter '%s' is not returned", param.getId())
			}
		}

		// Check return bindings.
		pipeline.Ret.Bindings.check(global, pipeline, pipeline.OutParams)
	}

	// If call statement present, check the call and its bindings.
	if global.Call != nil {
		callable, ok := global.Callables.Table[global.Call.Id]
		if !ok {
			global.err(global.Call, "ScopeNameError: '%s' is not defined in this scope", global.Call.Id)
		}
		global.Call.Bindings.check(global, callable, callable.getInParams())
	}
}

//
// Parser interface, called by runtime.
//
func parseSource(src string, srcPath string, incPaths []string, checkSrc bool) (string, []string, *Ast, []error) {
	// Add the source file's own folder to the include path for
	// resolving both @includes and stage src paths.
	incPaths = append([]string{filepath.Dir(srcPath)}, incPaths...)

	// Add PATH environment variable to the stage code path
	stagecodePaths := append(incPaths, strings.Split(os.Getenv("PATH"), ":")...)

	// Preprocess: generate new source and a locmap.
	postsrc, ifnames, locmap, err := preprocess(src, filepath.Base(srcPath), incPaths)
	if err != nil {
		return "", nil, nil, []error{err}
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
		return "", nil, nil, []error{&ParseError{perr.token, locmap[perr.loc].fname, locmap[perr.loc].loc}}
	}

	// Run semantic checks.
	ast.check(stagecodePaths, checkSrc)
	return postsrc, ifnames, ast, ast.Errors
}

// Compile an MRO file in cwd or mroPaths.
func Compile(fpath string, mroPaths []string, checkSrcPath bool) (string, []string, *Ast, []error) {
	if data, err := ioutil.ReadFile(fpath); err != nil {
		return "", nil, nil, []error{err}
	} else {
		return parseSource(string(data), fpath, mroPaths, checkSrcPath)
	}
}

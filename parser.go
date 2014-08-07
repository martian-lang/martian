package main

import (
	"fmt"
	"io/ioutil"
    "path"
    "strings"
	"os"
)

//
// Semantic Checking Helpers
//
func (global *Ast) err(locable Locatable, msg string, v ...interface{}) error {
	return &MarioError{global, locable, fmt.Sprintf(msg, v...)}
}

func (callables *Callables) check(global *Ast) error {
	for _, callable := range callables.list {
		// Check for duplicates
		if _, ok := callables.table[callable.Id()]; ok {
			return global.err(callable, "DuplicateNameError: stage or pipeline '%s' was already declared when encountered again", callable.Id())
		}
		callables.table[callable.Id()] = callable
	}
	return nil
}

func (params *Params) check(global *Ast) error {
	for _, param := range params.list {
		// Check for duplicates
		if _, ok := params.table[param.Id()]; ok {
			return global.err(param, "DuplicateNameError: parameter '%s' was already declared when encountered again", param.Id())
		}
		params.table[param.Id()] = param

		// Check that types exist.
		if _, ok := global.typeTable[param.Tname()]; !ok {
			return global.err(param, "TypeError: undefined type '%s'", param.Tname())
		}

	}
	return nil
}

func (exp *ValExp) ResolveType(global *Ast, pipeline *Pipeline) (string, error) {
	switch exp.Kind() {

	// Handle scalar types.
	case "int", "float", "string", "bool", "path", "null":
		return exp.Kind(), nil

	// Array: [ 1, 2 ]
	case "array":
	case "file":
	}
	return "unknown", nil
}

func (exp *RefExp) ResolveType(global *Ast, pipeline *Pipeline) (string, error) {
	switch exp.Kind() {

	// Param: self.myparam
	case "self":
		param, ok := pipeline.inParams.table[exp.id]
		if !ok {
			return "", global.err(exp, "ScopeNameError: '%s' is not an input parameter of pipeline '%s'", exp.id, pipeline.id)
		}
		return param.Tname(), nil

	// Call: STAGE.myoutparam or STAGE
	case "call":
		// Check referenced callable is acutally called in this scope.
		callable, ok := pipeline.callables.table[exp.id]
		if !ok {
			return "", global.err(exp, "ScopeNameError: '%s' is not called in pipeline '%s'", exp.id, pipeline.id)
		}

		// Check referenced output is actually an output of the callable.
		param, ok := callable.OutParams().table[exp.outputId]
		if !ok {
			return "", global.err(exp, "NoSuchOutputError: '%s' is not an output parameter of '%s'", exp.outputId, callable.Id())
		}
		return param.Tname(), nil
	}
	return "call", nil
}

func checkTypeMatch(t1 string, t2 string) bool {
	return t1 == "null" || t2 == "null" || t1 == t2
}

func (bindings *Bindings) check(global *Ast, pipeline *Pipeline, params *Params) error {
	// Check the bindings
	for _, binding := range bindings.list {
		// Collect bindings by id so we can check that all params are bound.
		if _, ok := bindings.table[binding.id]; ok {
			return global.err(binding, "DuplicateBinding: '%s' already bound in this call", binding.id)
		}
		bindings.table[binding.id] = binding

		// Make sure the bound-to id is a declared parameter of the callable.
		param, ok := params.table[binding.id]
		if !ok {
			return global.err(binding, "ArgumentError: '%s' is not a valid parameter", binding.id)
		}

		// Typecheck the binding and cache the type.
		valueType, err := binding.exp.ResolveType(global, pipeline)
		if err != nil {
			return err
		}
		if !checkTypeMatch(param.Tname(), valueType) {
			return global.err(param, "TypeMismatchError: expected type '%s' for '%s' but got '%s' instead", param.Tname(), param.Id(), valueType)
		}
		binding.tname = param.Tname()
	}
	return nil
}

func (global *Ast) check() error {
	// Build type table, starting with builtins. Duplicates allowed.
	types := []string{"string", "int", "float", "bool", "path", "file"}
	for _, filetype := range global.filetypes {
		types = append(types, filetype.id)
	}
	for _, t := range types {
		global.typeTable[t] = true
	}

	// Check for duplicate names amongst callables.
	if err := global.callables.check(global); err != nil {
		return err
	}

	// Check stage declarations.
	for _, stage := range global.stages {
		// Check in parameters.
		if err := stage.inParams.check(global); err != nil {
			return err
		}
		// Check out parameters.
		if err := stage.outParams.check(global); err != nil {
			return err
		}
		// Check split parameters.
		if stage.splitParams != nil {
			if err := stage.splitParams.check(global); err != nil {
				return err
			}
		}
	}

	// Check pipeline declarations.
	for _, pipeline := range global.pipelines {
		// Check in parameters.
		if err := pipeline.inParams.check(global); err != nil {
			return err
		}
		// Check out parameters.
		if err := pipeline.outParams.check(global); err != nil {
			return err
		}

		// Check calls.
		for _, call := range pipeline.calls {
			// Check for duplicate calls.
			if _, ok := pipeline.callables.table[call.id]; ok {
				return global.err(call, "DuplicateCallError: '%s' was already called when encountered again", call.id)
			}
			// Check we're calling something declared.
			callable, ok := global.callables.table[call.id]
			if !ok {
				return global.err(call, "ScopeNameError: '%s' is not defined in this scope", call.id)
			}
			// Save the valid callables for this scope.
			pipeline.callables.table[call.id] = callable

			// Check the bindings
			if err := call.bindings.check(global, pipeline, callable.InParams()); err != nil {
				return err
			}

			// Check that all input params of the callable are bound.
			for _, param := range callable.InParams().list {
				if _, ok := call.bindings.table[param.Id()]; !ok {
					return global.err(call, "ArgumentNotSuppliedError: no argument supplied for parameter '%s'", param.Id())
				}
			}
		}
	}

	// Doing these in a separate loop gives the user better incremental
	// error messages while writing a long pipeline declaration.
	for _, pipeline := range global.pipelines {
		// Check all pipeline input params are bound in a call statement.
		boundParamIds := map[string]bool{}
		for _, call := range pipeline.calls {
			for _, binding := range call.bindings.list {
				refexp, ok := binding.exp.(*RefExp)
				if ok {
					boundParamIds[refexp.id] = true
				}
			}
		}
		for _, param := range pipeline.inParams.list {
			if _, ok := boundParamIds[param.Id()]; !ok {
				return global.err(param, "UnusedInputError: no calls use pipeline input parameter '%s'", param.Id())
			}
		}

		// Check all pipeline output params are returned.
		returnedParamIds := map[string]bool{}
		for _, binding := range pipeline.ret.bindings.list {
			returnedParamIds[binding.id] = true
		}
		for _, param := range pipeline.outParams.list {
			if _, ok := returnedParamIds[param.Id()]; !ok {
				return global.err(pipeline.ret, "ReturnError: pipeline output parameter '%s' is not returned", param.Id())
			}
		}

		// Check return bindings.
		if err := pipeline.ret.bindings.check(global, pipeline, pipeline.outParams); err != nil {
			return err
		}
	}
	return nil
}

//
// Package Exports
//
func ParseString(src string, locmap []FileLoc) (*Ast, error) {
	global, err := yaccParse(src)
	if err != nil { // err is an mmLexInfo struct
		return nil, &ParseError{err.token, locmap[err.loc].fname, locmap[err.loc].loc}
	}
	global.locmap = locmap

	if err := global.check(); err != nil {
		return nil, err
	}
	return global, nil
}

func ParseFile(filename string) (string, *Ast, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", nil, err
	}
	postsrc, locmap := preprocess(string(data), filename)
	//printSourceMap(postsrc, locmap)
	global, err := ParseString(postsrc, locmap)
	return postsrc, global, err
}

func main() {
    dirs, _ := ioutil.ReadDir("../pipelines/src/mro")
    count := 0
    for i := 0; i < 1; i ++ {
        for _, dir := range dirs {
            if strings.HasPrefix(dir.Name(), "_") {
                continue
            }
            p := path.Join("../pipelines/src/mro", dir.Name())
            //_, _, err := ParseFile("../pipelines/src/mro/analytics_phasing.mro")
            _, _, err := ParseFile(p)
            count += 1
            if err != nil {
                fmt.Println(err.Error())
                os.Exit(1)
            }
        }
    }
    fmt.Printf("Successfully compiled %d mro files.", count)
}

//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO semantic checking.
//

package syntax

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/martian-lang/martian/martian/util"
)

//
// Semantic Checking Methods
//
func (global *Ast) err(nodable AstNodable, msg string, v ...interface{}) error {
	return &AstError{global, nodable.getNode(), fmt.Sprintf(msg, v...)}
}

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

func (params *Params) compile(global *Ast) error {
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

func (exp *ValExp) resolveType(global *Ast, callable Callable) ([]string, int, error) {
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
			arrayKind, arrayDim, err := subexp.resolveType(global, callable)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			arrayTypes = append(arrayTypes, arrayKind...)
			if commonArrayDim == -1 {
				commonArrayDim = arrayDim
			} else if commonArrayDim != arrayDim {
				errs = append(errs, global.err(exp,
					"Inconsistent array dimensions %d vs %d",
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

func (exp *RefExp) resolveType(global *Ast, callable Callable) ([]string, int, error) {
	if callable == nil {
		return []string{""}, 0, global.err(exp, "ReferenceError: this binding cannot be resolved outside of a stage or pipeline.")
	}

	switch exp.getKind() {

	// Param: self.myparam
	case KindSelf:
		param, ok := callable.GetInParams().Table[exp.Id]
		if !ok {
			return []string{""}, 0, global.err(exp, "ScopeNameError: '%s' is not an input parameter of pipeline '%s'", exp.Id, callable.GetId())
		}
		return []string{param.GetTname()}, param.GetArrayDim(), nil

	// Call: STAGE.myoutparam or STAGE
	case KindCall:
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

func (global *Ast) isUserType(t string) bool {
	_, ok := global.UserTypeTable[t]
	return ok
}

func (global *Ast) checkTypeMatch(paramType string, valueType string) bool {
	return (valueType == KindNull ||
		paramType == valueType ||
		(paramType == KindPath && valueType == KindString) ||
		(paramType == KindFile && valueType == KindString) ||
		(paramType == KindFloat && valueType == KindInt) ||
		// Allow implicit cast between string and user file type
		(global.isUserType(paramType) &&
			(valueType == KindString || valueType == KindFile)) ||
		(global.isUserType(valueType) &&
			(paramType == KindString || paramType == KindFile)))
}

func (bindings *BindStms) compile(global *Ast, callable Callable, params *Params) error {
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

		if err := binding.compile(global, callable, params); err != nil {
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

func (binding *BindStm) compile(global *Ast, callable Callable, params *Params) error {
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
			if len(valueTypes) < 1 || valueTypes[0] != KindNull {
				return global.err(binding, "TypeMismatchError: expected array of '%s' for '%s'", param.GetTname(), param.GetId())
			}
		} else {
			return global.err(binding, "TypeMismatchError: got %d-dimensional array value for %d-dimensional array parameter '%s'", arrayDim, param.GetArrayDim(), param.GetId())
		}
	}

	for _, valueType := range valueTypes {
		if !global.checkTypeMatch(param.GetTname(), valueType) {
			return global.err(binding, "TypeMismatchError: expected type '%s' for '%s' but got '%s' instead", param.GetTname(), param.GetId(), valueType)
		}
	}
	binding.Tname = param.GetTname()
	return nil
}

// Do a stable sort of the calls in topological order.  Returns an error
// if there is a dependency cycle or self-dependency.
func (pipeline *Pipeline) topoSort() error {
	// While there are probably better algorithms out there, most of them
	// don't produce a stable sort, which is important here.

	if len(pipeline.Calls) == 0 {
		return nil
	}

	// Find the direct dependencies of each call.
	depsMap, err := func(calls []*CallStm) (map[*CallStm]map[*CallStm]struct{}, error) {
		deps := make(map[*CallStm]map[*CallStm]struct{}, len(calls))

		callMap := make(map[string]*CallStm, len(calls))
		for _, call := range calls {
			callMap[call.Id] = call
		}
		var findDeps func(*CallStm, Exp) error
		findDeps = func(src *CallStm, uexp Exp) error {
			switch exp := uexp.(type) {
			case *RefExp:
				if exp.Kind == KindCall {
					depSet := deps[src]
					if depSet == nil {
						depSet = make(map[*CallStm]struct{})
						deps[src] = depSet
					}
					dep := callMap[exp.Id]
					if dep == src {
						return &wrapError{
							innerError: fmt.Errorf("Call %s input bound to its own output in pipeline %s.",
								src.Id, pipeline.Id),
							loc: exp.getNode().Loc,
						}
					}
					depSet[dep] = struct{}{}
				}
			case *ValExp:
				if exp.Kind == KindArray {
					for _, subExp := range exp.Value.([]Exp) {
						if err := findDeps(src, subExp); err != nil {
							return err
						}
					}
				}
			}
			return nil
		}
		var errs ErrorList
		for _, call := range calls {
			for _, bind := range call.Bindings.List {
				if err := findDeps(call, bind.Exp); err != nil {
					errs = append(errs, err)
				}
			}
			if call.Modifiers.Bindings != nil {
				for _, bind := range call.Modifiers.Bindings.List {
					if err := findDeps(call, bind.Exp); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}
		return deps, errs.If()
	}(pipeline.Calls)
	if err != nil {
		return err
	}

	// Find the next level of transitive dependencies.
	missingDeps := func(src *CallStm, deps map[*CallStm]struct{},
		depsMap map[*CallStm]map[*CallStm]struct{}) ([]*CallStm, error) {
		var missing []*CallStm
		for dep := range deps {
			for transDep := range depsMap[dep] {
				if _, ok := deps[transDep]; !ok {
					if transDep == src {
						return nil, &wrapError{
							innerError: fmt.Errorf(
								"Call depends transitively on itself (%s -> ... -> %s -> %s) in pipeline %s.",
								src.Id, dep.Id, transDep.Id, pipeline.Id),
							loc: src.getNode().Loc,
						}
					}
					missing = append(missing, transDep)
				}
			}
		}
		return missing, nil
	}
	if err := func(depsMap map[*CallStm]map[*CallStm]struct{}) error {
		changes := true
		for changes {
			extraDeps := make(map[*CallStm][]*CallStm)
			var errs ErrorList
			for src, deps := range depsMap {
				if missing, err := missingDeps(src, deps, depsMap); err != nil {
					errs = append(errs, err)
				} else if len(missing) > 0 {
					extraDeps[src] = missing
				}
			}
			if err := errs.If(); err != nil {
				return err
			}
			for src, deps := range extraDeps {
				for _, dep := range deps {
					depsMap[src][dep] = struct{}{}
				}
			}
			changes = len(extraDeps) > 0
		}
		return nil
	}(depsMap); err != nil {
		return err
	}
	// checkIndex is the last index known to be in the sorted region.
	checkIndex := 0
	for checkIndex+1 < len(pipeline.Calls) {
		call := pipeline.Calls[checkIndex]
		deps := depsMap[call]
		if deps == nil || len(deps) == 0 {
			checkIndex++
			continue
		}
		// If call depends on any calls which appear later in the sort order,
		// move it down to there.  Otherwise, increment the index.
		maxIndex := -1
		for i, maybeDep := range pipeline.Calls[checkIndex+1:] {
			if _, ok := deps[maybeDep]; ok {
				maxIndex = i
			}
		}
		if maxIndex >= 0 {
			// Shift
			copy(pipeline.Calls[checkIndex:checkIndex+maxIndex+1],
				pipeline.Calls[checkIndex+1:checkIndex+maxIndex+2])
			// Replace
			pipeline.Calls[checkIndex+maxIndex+1] = call
		} else {
			checkIndex++
		}
	}
	return nil
}

// Build type table, starting with builtins. Duplicates allowed.
func (global *Ast) compileTypes() error {
	builtinTypes := []*BuiltinType{
		{KindString},
		{KindInt},
		{KindFloat},
		{KindBool},
		{KindPath},
		{KindFile},
		{KindMap},
	}
	for _, builtinType := range builtinTypes {
		global.TypeTable[builtinType.Id] = builtinType
	}
	for _, userType := range global.UserTypes {
		global.TypeTable[userType.Id] = userType
		global.UserTypeTable[userType.Id] = userType
	}
	return nil
}

// Check stage declarations.
func (global *Ast) compileStages() error {
	var errs ErrorList
	for _, stage := range global.Stages {
		if err := stage.compile(global); err != nil {
			errs = append(errs, err)
		}
	}
	return errs.If()
}

func (stage *Stage) compile(global *Ast) error {
	var errs ErrorList
	// Check in parameters.
	if err := stage.InParams.compile(global); err != nil {
		errs = append(errs, err)
	}

	// Check out parameters.
	if err := stage.OutParams.compile(global); err != nil {
		errs = append(errs, err)
	}

	// Check split parameters.
	if stage.ChunkIns != nil {
		if err := stage.ChunkIns.compile(global); err != nil {
			errs = append(errs, err)
		}
		if GetEnforcementLevel() > EnforceDisable {
			for paramName := range stage.ChunkIns.Table {
				if _, ok := stage.InParams.Table[paramName]; ok {
					if GetEnforcementLevel() >= EnforceError {
						errs = append(errs, global.err(stage,
							"DuplicateNameError: '%s' appears as both a stage and split input",
							paramName))
					} else {
						util.PrintInfo("compile",
							"WARNING: '%s' appears as both a stage and split input for stage %s",
							paramName, stage.Id)
					}
				}
			}
		}
	}
	if stage.ChunkOuts != nil {
		if err := stage.ChunkOuts.compile(global); err != nil {
			errs = append(errs, err)
		}
		// Check that chunk outs don't duplicate stage outs.
		for _, param := range stage.ChunkOuts.List {
			if _, ok := stage.OutParams.Table[param.GetId()]; ok {
				errs = append(errs, global.err(param,
					"DuplicateNameError: parameter name '%s' of stage %s is used for both chunk and stage outs",
					param.GetId(), stage.Id))
			}
		}
	}
	if stage.Retain != nil {
		if err := stage.Retain.compile(global, stage); err != nil {
			errs = append(errs, err)
		}
	}
	return errs.If()
}

func (retains *RetainParams) compile(global *Ast, stage *Stage) error {
	var errs ErrorList
	ids := make(map[string]AstNode, len(retains.Params))
	for _, param := range retains.Params {
		if out := stage.OutParams.Table[param.Id]; out == nil {
			errs = append(errs, global.err(param,
				"RetainParamError: stage %s does not have an out parameter named %s to retain.",
				stage.Id, param.Id))
		} else if !out.IsFile() {
			errs = append(errs, global.err(param,
				"RetainParamError: out parameter %s of %s is not of file type.",
				param.Id, stage.Id))
		} else {
			ids[param.Id] = param.Node
		}
	}
	if len(ids) != len(retains.Params) {
		retains.Params = make([]*RetainParam, 0, len(ids))
		for id, node := range ids {
			retains.Params = append(retains.Params, &RetainParam{
				Node: node,
				Id:   id,
			})
		}
	}
	sort.Slice(retains.Params, func(i, j int) bool {
		return retains.Params[i].Id < retains.Params[j].Id
	})
	return errs.If()
}

func (retains *PipelineRetains) compile(global *Ast, pipeline *Pipeline) error {
	var errs ErrorList
	for _, param := range retains.Refs {
		if types, _, err := param.resolveType(global, pipeline); err != nil {
			errs = append(errs, err)
		} else {
			any := false
			for _, tName := range types {
				if t, ok := global.TypeTable[tName]; ok && t.IsFile() {
					any = true
					break
				}
			}
			if !any {
				errs = append(errs, global.err(param,
					"RetainParamError: parameter %s of %s is not of file type.",
					param.OutputId, param.Id))
			}
		}
	}
	return errs.If()
}

const (
	disabled  = "disabled"
	local     = "local"
	preflight = "preflight"
	volatile  = "volatile"
	strict    = "strict"
)

// For checking modifier bindings.  Modifiers are optional so
// only the list is set.
var modParams = Params{
	Table: map[string]Param{
		disabled:  &InParam{Id: disabled, Tname: "bool"},
		local:     &InParam{Id: local, Tname: "bool"},
		preflight: &InParam{Id: preflight, Tname: "bool"},
		volatile:  &InParam{Id: volatile, Tname: "bool"},
	},
}

func (mods *Modifiers) compile(global *Ast, parent Callable, call *CallStm) error {
	var errs ErrorList
	if mods.Bindings != nil {
		if err := mods.Bindings.compile(global, parent, &modParams); err != nil {
			return err
		}
		// Simplifly value expressions down.
		if binding := mods.Bindings.Table[volatile]; binding != nil {
			if mods.Volatile {
				errs = append(errs, global.err(call,
					"ConflictingModifiers: Cannot specify modifiers in more than one way."))
			}
			// grammar only allows bool literals.
			mods.Volatile = binding.Exp.ToInterface().(bool)
			delete(mods.Bindings.Table, volatile)
		}
		if binding := mods.Bindings.Table[local]; binding != nil {
			if mods.Local {
				errs = append(errs, global.err(call,
					"ConflictingModifiers: Cannot specify modifiers in more than one way."))
			}
			// grammar only allows bool literals.
			mods.Local = binding.Exp.ToInterface().(bool)
			delete(mods.Bindings.Table, local)
		}
		if binding := mods.Bindings.Table[preflight]; binding != nil {
			if mods.Preflight {
				errs = append(errs, global.err(call,
					"ConflictingModifiers: Cannot specify modifiers in more than one way."))
			}
			// grammar only allows bool literals.
			mods.Preflight = binding.Exp.ToInterface().(bool)
			delete(mods.Bindings.Table, preflight)
		}
	}

	callable := global.Callables.Table[call.DecId]
	// Check to make sure if local, preflight or volatile is declared, callable is a stage
	if _, ok := callable.(*Stage); !ok {
		if call.Modifiers.Local {
			errs = append(errs, global.err(call,
				"UnsupportedTagError: Pipeline '%s' cannot be called with 'local' tag",
				call.DecId))
		}
		if call.Modifiers.Preflight {
			errs = append(errs, global.err(call,
				"UnsupportedTagError: Pipeline '%s' cannot be called with 'preflight' tag",
				call.DecId))
		}
		if call.Modifiers.Volatile {
			errs = append(errs, global.err(call,
				"UnsupportedTagError: Pipeline '%s' cannot be called with 'volatile' tag",
				call.DecId))
		}
	}

	if mods.Preflight {
		if mods.Bindings != nil && mods.Bindings.Table[disabled] != nil {
			errs = append(errs, global.err(call,
				"UnsupportedTagError: Preflight stages cannot be declared disabled."))
		}
		for _, binding := range call.Bindings.List {
			if binding.Exp.getKind() == KindCall {
				errs = append(errs, global.err(call,
					"PreflightBindingError: Preflight stage '%s' cannot have input parameter bound to output parameter of another stage or pipeline",
					call.Id))
			}
		}
		if mods.Bindings != nil {
			for _, binding := range mods.Bindings.Table {
				if binding.Exp.getKind() == KindCall {
					errs = append(errs, global.err(call,
						"PreflightBindingError: Preflight stage '%s' cannot have input parameter bound to output parameter of another stage or pipeline",
						call.Id))
				}
			}
		}

		if len(callable.GetOutParams().List) > 0 {
			errs = append(errs, global.err(call,
				"PreflightOutputError: Preflight stage '%s' cannot have any output parameters",
				call.Id))
		}
	}

	return errs.If()
}

func (pipeline *Pipeline) compile(global *Ast) error {
	var errs ErrorList
	// Check in parameters.
	if err := pipeline.InParams.compile(global); err != nil {
		errs = append(errs, err)
	}

	// Check out parameters.
	if err := pipeline.OutParams.compile(global); err != nil {
		errs = append(errs, err)
	}

	// Check calls.
	for _, call := range pipeline.Calls {
		// Check for duplicate calls.
		if _, ok := pipeline.Callables.Table[call.Id]; ok {
			errs = append(errs, global.err(call,
				"DuplicateCallError: '%s' was already called when encountered again",
				call.Id))
		}
		// Check we're calling something declared.
		callable, ok := global.Callables.Table[call.DecId]
		if !ok {
			errs = append(errs, global.err(call,
				"ScopeNameError: '%s' is not defined in this scope",
				call.DecId))
			continue
		}
		// Save the valid callables for this scope.
		pipeline.Callables.Table[call.Id] = callable
	}
	if err := errs.If(); err != nil {
		return err
	}
	// Check call bindings after all calls are checked, so that the Callables
	// table is fully populated.
	for _, call := range pipeline.Calls {
		if err := call.Modifiers.compile(global, pipeline, call); err != nil {
			errs = append(errs, err)
		}

		// Check the bindings
		callable := global.Callables.Table[call.DecId]
		if err := call.Bindings.compile(
			global, pipeline, callable.GetInParams()); err != nil {
			errs = append(errs, err)
			continue
		}

		// Check that all input params of the callable are bound.
		for _, param := range callable.GetInParams().List {
			if _, ok := call.Bindings.Table[param.GetId()]; !ok {
				errs = append(errs, global.err(call,
					"ArgumentNotSuppliedError: no argument supplied for parameter '%s'",
					param.GetId()))
			}
		}
	}
	if err := errs.If(); err != nil {
		return err
	}
	return pipeline.topoSort()
}

// Check pipeline declarations.
func (global *Ast) compilePipelineDecs() error {
	var errs ErrorList
	for _, pipeline := range global.Pipelines {
		if err := pipeline.compile(global); err != nil {
			errs = append(errs, err)
		}
	}
	return errs.If()
}

func getBoundParamIds(uexp Exp) []string {
	switch exp := uexp.(type) {
	case *RefExp:
		return []string{exp.Id}
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

// Check all pipeline input params are bound in a call statement.
func (global *Ast) compilePipelineArgs() error {
	// Doing these in a separate loop gives the user better incremental
	// error messages while writing a long pipeline declaration.
	for _, pipeline := range global.Pipelines {
		boundParamIds := map[string]bool{}
		for _, call := range pipeline.Calls {
			for _, binding := range call.Bindings.List {
				for _, id := range getBoundParamIds(binding.Exp) {
					boundParamIds[id] = true
				}
			}
			if call.Modifiers.Bindings != nil {
				for _, binding := range call.Modifiers.Bindings.List {
					refexp, ok := binding.Exp.(*RefExp)
					if ok {
						boundParamIds[refexp.Id] = true
					}
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
		if err := pipeline.Ret.Bindings.compile(global, pipeline, pipeline.OutParams); err != nil {
			return err
		}

		// Check retain bindings.
		if pipeline.Retain != nil {
			if err := pipeline.Retain.compile(global, pipeline); err != nil {
				return err
			}
		}
	}
	return nil
}

// If call statement present, check the call and its bindings.
func (global *Ast) compileCall() error {
	if global.Call != nil {
		callable, ok := global.Callables.Table[global.Call.DecId]
		if !ok {
			return global.err(global.Call, "ScopeNameError: '%s' is not defined in this scope", global.Call.DecId)
		}
		if err := global.Call.Bindings.compile(global, nil, callable.GetInParams()); err != nil {
			return err
		}
		if err := global.Call.Modifiers.compile(global, nil, global.Call); err != nil {
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

func (global *Ast) compile() error {
	if err := global.compileTypes(); err != nil {
		return err
	}

	// Check for duplicate names amongst callables.
	if err := global.Callables.compile(global); err != nil {
		return err
	}

	if err := global.compileStages(); err != nil {
		return err
	}

	if err := global.compilePipelineDecs(); err != nil {
		return err
	}

	if err := global.compilePipelineArgs(); err != nil {
		return err
	}

	if err := global.compileCall(); err != nil {
		return err
	}

	return nil
}

func (global *Ast) checkSrcPaths(stagecodePaths []string) error {
	var errs ErrorList
	for _, stage := range global.Stages {
		// Exempt exec stages
		if stage.Src.Lang != "exec" && stage.Src.Lang != "comp" {
			if _, found := util.SearchPaths(stage.Src.Path, stagecodePaths); !found {
				stagecodePathsList := strings.Join(stagecodePaths, ", ")
				errs = append(errs, global.err(stage,
					"SourcePathError: searched (%s) but stage source path not found '%s'",
					stagecodePathsList, stage.Src.Path))
			}
		}
	}
	return errs.If()
}

func (src *SourceFile) checkIncludes(fullPath string, inc *SourceLoc) error {
	var errs ErrorList
	if fullPath == src.FullPath {
		errs = append(errs, &wrapError{
			innerError: fmt.Errorf("Include cycle: %s included", src.FullPath),
			loc:        *inc,
		})
	} else {
		for _, parent := range src.IncludedFrom {
			if err := parent.File.checkIncludes(fullPath, inc); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs.If()
}

// ParseSource parses a souce string into an AST.
//
// src is the mro source code.
//
// srcPath is the path to the source code file (if applicable), used for
// debugging information.
//
// incPaths is the orderd set of search paths to use when resolving include
// directives.
//
// If checkSrc is true, then the parser will verify that stage src values
// refer to code that actually exists.
func ParseSource(src string, srcPath string, incPaths []string, checkSrc bool) (string, []string, *Ast, error) {
	return ParseSourceBytes([]byte(src), srcPath, incPaths, checkSrc)
}

func ParseSourceBytes(src []byte, srcPath string, incPaths []string, checkSrc bool) (string, []string, *Ast, error) {
	fname := filepath.Base(srcPath)
	absPath, _ := filepath.Abs(srcPath)
	srcFile := SourceFile{
		FileName: fname,
		FullPath: absPath,
	}
	if ast, err := parseSource(src, &srcFile, incPaths,
		map[string]*SourceFile{absPath: &srcFile}); err != nil {
		return "", nil, ast, err
	} else {
		err := ast.compile()
		ifnames := make([]string, len(ast.Includes))
		for i, inc := range ast.Includes {
			ifnames[i] = inc.Value
		}
		if checkSrc {
			stagecodePaths := filepath.SplitList(os.Getenv("PATH"))
			seenPaths := make(map[string]struct{}, len(incPaths)+len(stagecodePaths))
			for f := range ast.Files {
				p := filepath.Dir(f)
				if _, ok := seenPaths[p]; !ok {
					stagecodePaths = append(stagecodePaths, p)
					seenPaths[p] = struct{}{}
				}
			}
			if srcerr := ast.checkSrcPaths(stagecodePaths); err != nil {
				err = ErrorList{err, srcerr}.If()
			}
		}
		return ast.format(false), ifnames, ast, err
	}
}

func parseSource(src []byte, srcFile *SourceFile, incPaths []string,
	processedIncludes map[string]*SourceFile) (*Ast, error) {
	// Add the source file's own folder to the include path for
	// resolving both @includes and stage src paths.
	incPaths = append([]string{filepath.Dir(srcFile.FullPath)}, incPaths...)

	// Parse the source into an AST and attach the comments.
	ast, err := yaccParse(src, srcFile)
	if err != nil {
		return nil, err
	}

	iasts, err := getIncludes(srcFile, ast.Includes, incPaths, processedIncludes)
	if iasts != nil {
		ast.merge(iasts)
	}
	return ast, err
}

func getIncludes(srcFile *SourceFile, includes []*Include, incPaths []string,
	processedIncludes map[string]*SourceFile) (*Ast, error) {
	var errs ErrorList
	var iasts *Ast
	seen := make(map[string]struct{}, len(includes))
	for _, inc := range includes {
		if ifpath, found := util.SearchPaths(inc.Value, incPaths); !found {
			errs = append(errs, &FileNotFoundError{
				name: inc.Value,
				loc:  inc.Node.Loc,
			})
		} else {
			absPath, _ := filepath.Abs(ifpath)
			if _, ok := seen[absPath]; ok {
				errs = append(errs, &wrapError{
					innerError: fmt.Errorf("%s included multiple times",
						inc.Value),
					loc: inc.Node.Loc,
				})
			}
			seen[absPath] = struct{}{}

			if absPath == srcFile.FullPath {
				errs = append(errs, &wrapError{
					innerError: fmt.Errorf("%s includes itself", srcFile.FullPath),
					loc:        inc.Node.Loc,
				})
			} else if iSrcFile := processedIncludes[absPath]; iSrcFile != nil {
				iSrcFile.IncludedFrom = append(iSrcFile.IncludedFrom, &inc.Node.Loc)
				if err := srcFile.checkIncludes(absPath, &inc.Node.Loc); err != nil {
					errs = append(errs, err)
				}
			} else {
				iSrcFile = &SourceFile{
					FileName:     inc.Value,
					FullPath:     absPath,
					IncludedFrom: []*SourceLoc{&inc.Node.Loc},
				}
				processedIncludes[absPath] = iSrcFile
				if b, err := ioutil.ReadFile(iSrcFile.FullPath); err != nil {
					errs = append(errs, &wrapError{
						innerError: err,
						loc:        inc.Node.Loc,
					})
				} else {
					iast, err := parseSource(b, iSrcFile,
						incPaths[1:], processedIncludes)
					errs = append(errs, err)
					if iast != nil {
						if iasts == nil {
							iasts = iast
						} else {
							// x.merge(y) puts y's stuff before x's.
							iast.merge(iasts)
							iasts = iast
						}
					}
				}
			}
		}
	}
	return iasts, errs.If()
}

// Compile an MRO file in cwd or mroPaths.
func Compile(fpath string, mroPaths []string, checkSrcPath bool) (string, []string, *Ast, error) {
	if data, err := ioutil.ReadFile(fpath); err != nil {
		return "", nil, nil, err
	} else {
		return ParseSourceBytes(data, fpath, mroPaths, checkSrcPath)
	}
}

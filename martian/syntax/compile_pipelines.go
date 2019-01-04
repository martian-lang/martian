// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// compile/check pipelines.

package syntax

import (
	"fmt"
)

func (pipeline *Pipeline) directDepsMap() (map[*CallStm]map[*CallStm]struct{}, error) {
	calls := pipeline.Calls
	deps := make(map[*CallStm]map[*CallStm]struct{}, len(calls))

	callMap := make(map[string]*CallStm, len(calls))
	for _, call := range calls {
		if call.DecId == pipeline.Id {
			return deps, &wrapError{
				innerError: fmt.Errorf(
					"RecursiveCallError: Pipeline %s calls itself.",
					pipeline.Id),
				loc: call.getNode().Loc,
			}
		}
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
						innerError: fmt.Errorf(
							"CyclicDependencyError: call %s input bound to its own output in pipeline %s.",
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
}

// Finds dependencies of src which aren't in depsMap
func (pipeline *Pipeline) findMissingDeps(src *CallStm, deps map[*CallStm]struct{},
	depsMap map[*CallStm]map[*CallStm]struct{}) ([]*CallStm, error) {
	var missing []*CallStm
	for dep := range deps {
		for transDep := range depsMap[dep] {
			if _, ok := deps[transDep]; !ok {
				if transDep == src {
					return nil, &wrapError{
						innerError: fmt.Errorf(
							"CyclicDependencyError: Call depends transitively on itself (%s -> ... -> %s -> %s) in pipeline %s.",
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

// Add the breadth-first next level of transitive dependencies to depsMap.
func (pipeline *Pipeline) addNextDeps(depsMap map[*CallStm]map[*CallStm]struct{}) error {
	changes := true
	for changes {
		extraDeps := make(map[*CallStm][]*CallStm)
		var errs ErrorList
		for src, deps := range depsMap {
			if missing, err := pipeline.findMissingDeps(src, deps, depsMap); err != nil {
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
	depsMap, err := pipeline.directDepsMap()
	if err != nil {
		return err
	}

	// Find the next level of transitive dependencies.
	if err := pipeline.addNextDeps(depsMap); err != nil {
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
				return global.err(param,
					"UnusedInputError: no calls use pipeline input parameter '%s'",
					param.GetId())
			}
		}

		// Check all pipeline output params are returned.
		returnedParamIds := map[string]bool{}
		for _, binding := range pipeline.Ret.Bindings.List {
			returnedParamIds[binding.Id] = true
		}

		// Check return bindings.
		if err := pipeline.Ret.Bindings.compileReturns(global,
			pipeline, pipeline.OutParams); err != nil {
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

// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// compile/check stages.

package syntax

import (
	"sort"

	"github.com/martian-lang/martian/martian/util"
)

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

const (
	disabled  = "disabled"
	local     = "local"
	preflight = "preflight"
	volatile  = "volatile"
	strict    = "strict"
)

// For checking modifier bindings.  Modifiers are optional so
// only the list is set.
var modParams = InParams{
	Table: map[string]*InParam{
		disabled:  &InParam{Id: disabled, Tname: "bool"},
		local:     &InParam{Id: local, Tname: "bool"},
		preflight: &InParam{Id: preflight, Tname: "bool"},
		volatile:  &InParam{Id: volatile, Tname: "bool"},
	},
}

func (mods *Modifiers) compile(global *Ast, parent *Pipeline, call *CallStm) error {
	// Error message strings
	const (
		ConflictingModifiers = "ConflictingModifiers: Cannot specify " +
			"modifiers in more than one way."
		UnsupportedTagError = "UnsupportedTagError: Pipeline '%s' " +
			"cannot be called with "
		PreflightBindingError = "PreflightBindingError: Preflight stage " +
			"'%s' cannot have input parameter bound to output " +
			"parameter of another stage or pipeline"
		PreflightOutputError = "PreflightOutputError: Preflight stage " +
			"'%s' cannot have any output parameters"
	)

	var errs ErrorList
	if mods.Bindings != nil {
		if err := mods.Bindings.compile(global, parent, &modParams); err != nil {
			return err
		}
		// Simplifly value expressions down.
		if binding := mods.Bindings.Table[volatile]; binding != nil {
			if mods.Volatile {
				errs = append(errs, global.err(call,
					ConflictingModifiers))
			}
			// grammar only allows bool literals.
			mods.Volatile = binding.Exp.ToInterface().(bool)
			delete(mods.Bindings.Table, volatile)
		}
		if binding := mods.Bindings.Table[local]; binding != nil {
			if mods.Local {
				errs = append(errs, global.err(call,
					ConflictingModifiers))
			}
			// grammar only allows bool literals.
			mods.Local = binding.Exp.ToInterface().(bool)
			delete(mods.Bindings.Table, local)
		}
		if binding := mods.Bindings.Table[preflight]; binding != nil {
			if mods.Preflight {
				errs = append(errs, global.err(call,
					ConflictingModifiers))
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
				UnsupportedTagError+"'local' tag",
				call.DecId))
		}
		if call.Modifiers.Preflight {
			errs = append(errs, global.err(call,
				UnsupportedTagError+"'preflight' tag",
				call.DecId))
		}
		if call.Modifiers.Volatile {
			errs = append(errs, global.err(call,
				UnsupportedTagError+"'volatile' tag",
				call.DecId))
		}
	}

	if mods.Preflight {
		for _, binding := range call.Bindings.List {
			if binding.Exp.getKind() == KindCall {
				errs = append(errs, global.err(call,
					PreflightBindingError,
					call.Id))
			}
		}
		if mods.Bindings != nil {
			for _, binding := range mods.Bindings.Table {
				if binding.Exp.getKind() == KindCall {
					errs = append(errs, global.err(call,
						PreflightBindingError,
						call.Id))
				}
			}
		}

		if len(callable.GetOutParams().List) > 0 {
			errs = append(errs, global.err(call,
				PreflightOutputError,
				call.Id))
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
		} else if !out.IsFile() && out.GetTname() != KindMap {
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

// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// Utilities for testing semantic equivalence of AST nodes.

package syntax

import (
	"math"
	"reflect"
	"strings"

	"github.com/martian-lang/martian/martian/util"
)

// Returns true if the two compiled ASTs represent semantically equivalent
// calls.
//
// Calls are semantically equivalent if they will generate the same pipestance.
// A stage of pipeline name may change if the call is aliased to the same name
// as was previously present.  There must not be new arguments or outputs.
// Stages must not change their split, local, or preflight status, but may
// change split in/out parameters or VDR mode.  File types may change their
// names, and include structure may change.
//
// Stages or pipelines outside the transitive closure from the top-level call
// are ignored.
func (ast *Ast) EquivalentCall(other *Ast) bool {
	if ast.Callables == nil || ast.Callables.Table == nil ||
		other.Callables == nil || other.Callables.Table == nil {
		return false
	}
	return ast.Call.EquivalentTo(other.Call, ast.Callables, other.Callables)
}

// Two calls are semantically equivalent if their (possibly aliased) names are
// equal, all of their argument bindings are equal, and their call modifiers
// are semantically equivalent.
func (call *CallStm) EquivalentTo(other *CallStm,
	myCallables, otherCallables *Callables) bool {
	if call == nil {
		return other == nil
	} else if other == nil {
		return false
	}
	// Check the call names match.
	if call.Id != other.Id {
		util.PrintInfo("compare",
			"Call name %s does not match %s",
			call.Id, other.Id)
		return false
	}
	if !call.Bindings.Equals(other.Bindings) {
		util.PrintInfo("compare",
			"Call bindings for %s do not match.",
			call.Id)
		return false
	}
	if !call.Modifiers.EquivalentTo(other.Modifiers) {
		util.PrintInfo("compare",
			"Call modifiers do not match.")
		return false
	}

	if callable := myCallables.Table[call.DecId]; callable == nil {
		return otherCallables.Table[other.DecId] == nil
	} else if oc := otherCallables.Table[other.DecId]; oc == nil {
		util.PrintInfo("compare",
			"Callable %s not found",
			other.Id)
		return false
	} else {
		return callable.EquivalentTo(oc, myCallables, otherCallables)
	}
}

func (bindings *BindStms) Equals(other *BindStms) bool {
	if bindings == nil || len(bindings.List) == 0 {
		return other == nil || len(other.List) == 0
	} else if other == nil || other.Table == nil ||
		len(other.Table) != len(bindings.List) {
		return false
	}
	for _, b := range bindings.List {
		if ob := other.Table[b.Id]; ob == nil {
			util.PrintInfo("compare",
				"Binding ID %s not found",
				b.Id)
			return false
		} else if !b.Equals(ob) {
			return false
		}
	}
	return true
}

// Two call modifier sets are equivalent if the values for preflight, local,
// and disable are equal.  volatile is ignored.
func (mods *Modifiers) EquivalentTo(other *Modifiers) bool {
	if mods == nil {
		if other == nil {
			return true
		} else {
			return other.EquivalentTo(mods)
		}
	} else if other == nil {
		if mods.Local || mods.Preflight {
			return false
		} else {
			return mods.Bindings == nil || mods.Bindings.Table == nil ||
				mods.Bindings.Table[disabled] == nil
		}
	} else if mods.Local != other.Local || mods.Preflight != other.Preflight {
		return false
	} else if mods.Bindings != nil && mods.Bindings.Table != nil {
		if b := mods.Bindings.Table[disabled]; b != nil {
			if other.Bindings == nil || other.Bindings.Table == nil {
				return false
			} else if ob := mods.Bindings.Table[disabled]; ob == nil {
				return false
			} else {
				return b.Equals(ob)
			}
		}
	}
	if other.Bindings != nil && other.Bindings.Table != nil {
		return other.Bindings.Table[disabled] == nil
	} else {
		return true
	}
}

// Equals returns true if the two parameter sets share the same parameter
// names and types.  Changes to file type names are ignored.
func (params *InParams) Equals(other *InParams) bool {
	if params == nil || len(params.List) == 0 {
		return other == nil || len(other.List) == 0
	} else if other == nil || other.Table == nil ||
		len(other.Table) != len(params.List) {
		util.PrintInfo("compare",
			"Argument length mismatch.")
		return false
	}
	for _, arg := range params.List {
		if oa := other.Table[arg.GetId()]; oa == nil {
			util.PrintInfo("compare",
				"Argument %s not found.",
				arg.GetId())
			return false
		} else if arg.GetArrayDim() != oa.GetArrayDim() {
			return false
		} else if arg.IsFile() != oa.IsFile() {
			return false
		} else if !arg.IsFile() && arg.GetTname() != oa.GetTname() {
			return false
		}
	}
	return true
}

// Equals returns true if the two parameter sets share the same parameter
// names and types.  Changes to file type names are ignored.  If checkOutNames
// is true, the output name for the parameters are also compared.
func (params *OutParams) Equals(other *OutParams, checkOutNames bool) bool {
	if params == nil || len(params.List) == 0 {
		return other == nil || len(other.List) == 0
	} else if other == nil || other.Table == nil ||
		len(other.Table) != len(params.List) {
		util.PrintInfo("compare",
			"Argument length mismatch.")
		return false
	}
	for _, arg := range params.List {
		if oa := other.Table[arg.GetId()]; oa == nil {
			util.PrintInfo("compare",
				"Argument %s not found.",
				arg.GetId())
			return false
		} else if arg.GetArrayDim() != oa.GetArrayDim() {
			return false
		} else if arg.IsFile() != oa.IsFile() {
			return false
		} else if !arg.IsFile() && arg.GetTname() != oa.GetTname() {
			return false
		} else if checkOutNames && arg.GetOutName() != oa.GetOutName() {
			return false
		}
	}
	return true
}

// Two pipelines are semantically equivalent if their input and output argument
// names are the same and all of their calls are semantically equivalent and
// their return bindings are the same.  Changes to VDR retention are ignored.
func (pipeline *Pipeline) EquivalentTo(other Callable,
	myCallables, otherCallables *Callables) bool {
	if pipeline == nil {
		return other == nil
	} else if other == nil {
		return false
	}
	if op, ok := other.(*Pipeline); !ok {
		return false
	} else if !pipeline.InParams.Equals(op.InParams) {
		util.PrintInfo("compare",
			"Pipeline %s in params unequal.",
			pipeline.Id)
		return false
	} else if !pipeline.OutParams.Equals(op.OutParams, true) {
		util.PrintInfo("compare",
			"Pipeline %s out params unequal.",
			pipeline.Id)
		return false
	} else if len(pipeline.Calls) != len(op.Calls) {
		util.PrintInfo("compare",
			"Pipeline %s call count changed.",
			pipeline.Id)
		return false
	} else if pipeline.Ret == nil || op.Ret == nil ||
		!pipeline.Ret.Bindings.Equals(op.Ret.Bindings) {
		util.PrintInfo("compare",
			"Pipeline %s return bindings unequal.",
			pipeline.Id)
		return false
	} else {
		oCalls := make(map[string]*CallStm, len(op.Calls))
		for _, call := range op.Calls {
			oCalls[call.Id] = call
		}
		for _, call := range pipeline.Calls {
			if !call.EquivalentTo(oCalls[call.Id], myCallables, otherCallables) {
				return false
			}
		}
		return true
	}
}

// Two stages are equivalent if they have the same inputs and outputs with the
// same types, and share the same splitting behavior.  All file types are
// considered equal.  Resources, stage source code, and split ins/outs are
// ignored.
func (stage *Stage) EquivalentTo(other Callable, _, _ *Callables) bool {
	if stage == nil {
		return other == nil
	} else if other == nil {
		return false
	} else if os, ok := other.(*Stage); !ok {
		return false
	} else if stage.Split != os.Split {
		util.PrintInfo("compare",
			"Stage %s split status different.",
			stage.Id)
		return false
	} else if !stage.InParams.Equals(os.InParams) {
		util.PrintInfo("compare",
			"Stage %s in parameters unequal.",
			stage.Id)
		return false
	} else if !stage.OutParams.Equals(os.OutParams, false) {
		util.PrintInfo("compare",
			"Stage %s out parameters unequal.",
			stage.Id)
		return false
	} else {
		return true
	}
}

func (binding *BindStm) Equals(other *BindStm) bool {
	if binding == nil {
		return other == nil
	} else if other == nil {
		return false
	}
	if binding.Id != other.Id {
		util.PrintInfo("compare",
			"Binding %s name does not match %s",
			binding.Id, other.Id)
		return false
	}
	if binding.Sweep != other.Sweep {
		util.PrintInfo("compare",
			"Binding %s sweep status different.",
			binding.Id)
		return false
	}
	if binding.Exp == nil {
		return other.Exp == nil
	} else if other.Exp == nil {
		return false
	} else if !binding.Exp.equal(other.Exp) {
		util.PrintInfo("compare",
			"Binding %s values differ.",
			binding.Id)
		return false
	}
	return true
}

func (exp *ValExp) equal(other Exp) bool {
	if exp == nil {
		return other == nil
	} else if other == nil {
		return false
	} else if ov, ok := other.(*ValExp); !ok {
		util.PrintInfo("compare",
			"Value are not both literals.")
		return false
	} else {
		return exp.equalVal(ov)
	}
}

func (exp *ValExp) equalVal(ov *ValExp) bool {
	if exp.Kind == KindArray {
		// unlike other types, empty array is equivalent to nil
		return exp.equalArray(ov)
	} else if exp.Value == nil {
		return ov.Value == nil
	} else if ov.Value == nil {
		return false
	} else if exp.Kind == KindInt {
		return exp.equalInt(ov)
	} else if exp.Kind == KindFloat {
		return exp.equalFloat(ov)
	} else if exp.Kind != ov.Kind {
		util.PrintInfo("compare",
			"Value type %v != %v",
			exp.Kind, ov.Kind)
		return false
	} else if exp.Value == nil {
		return ov.Value == nil
	} else if reflect.DeepEqual(exp.Value, ov.Value) {
		return true
	} else {
		// Check format equivalence.  This is the most reliable
		// way to compare two values without replicating a lot of
		// code.
		var buf1 strings.Builder
		exp.format(&buf1, "")
		var buf2 strings.Builder
		buf2.Grow(buf1.Len())
		ov.format(&buf2, "")
		if buf1.String() != buf2.String() {
			util.PrintInfo("compare",
				"Values do not match:\n%s\nvs\n%s",
				buf1.String(), buf2.String())
			return false
		}
		return true
	}
}

func (exp *ValExp) equalArray(ov *ValExp) bool {
	if ov.Kind != KindArray {
		return false
	}
	var a1, a2 []Exp
	if v, ok := exp.Value.([]Exp); ok {
		a1 = v
	}
	if v, ok := ov.Value.([]Exp); ok {
		a2 = v
	}
	if len(a1) != len(a2) {
		util.PrintInfo("compare", "Lengths %d != %d",
			len(a1), len(a2))
	}
	for i, v := range a1 {
		if !v.equal(a2[i]) {
			return false
		}
	}
	return true
}

func (exp *ValExp) equalInt(ov *ValExp) bool {
	if ov.Kind == KindInt {
		if i, ok := exp.Value.(int64); !ok {
			if _, ok := ov.Value.(int64); ok {
				util.PrintInfo("compare", "Inconsistent types %T != %T", exp.Value, ov.Value)
				return false
			}
			return true
		} else if i2, ok := ov.Value.(int64); !ok {
			util.PrintInfo("compare", "Inconsistent types %T != %T", exp.Value, ov.Value)
			return false
		} else {
			return i == i2
		}
	} else if ov.Kind == KindFloat {
		if i, ok := exp.Value.(int64); !ok {
			if _, ok := ov.Value.(float64); ok {
				util.PrintInfo("compare",
					"Inconsistent types %T != %T", exp.Value, ov.Value)
				return false
			}
			return true
		} else if f, ok := ov.Value.(float64); !ok {
			util.PrintInfo("compare",
				"Inconsistent types %T != %T", exp.Value, ov.Value)
			return false
		} else {
			return math.Abs(float64(i)-f) < 1e-15
		}
	} else {
		util.PrintInfo("compare",
			"Value type %v != %v",
			exp.Kind, ov.Kind)
		return false
	}
}

func (exp *ValExp) equalFloat(ov *ValExp) bool {
	if ov.Kind == KindInt {
		return ov.equalInt(exp)
	} else if ov.Kind == KindFloat {
		if f1, ok := exp.Value.(float64); !ok {
			if _, ok := ov.Value.(float64); ok {
				util.PrintInfo("compare",
					"Inconsistent types %T != %T", exp.Value, ov.Value)
				return false
			}
			return true
		} else if f2, ok := ov.Value.(float64); !ok {
			util.PrintInfo("compare",
				"Inconsistent types %T != %T", exp.Value, ov.Value)
			return false
		} else {
			return math.Abs(f1-f2) < f1*1e-15
		}
	} else {
		util.PrintInfo("compare",
			"Value type %v != %v",
			exp.Kind, ov.Kind)
		return false
	}
}

func (exp *RefExp) equal(other Exp) bool {
	if exp == nil {
		return other == nil
	} else if other == nil {
		return false
	} else if ov, ok := other.(*RefExp); !ok {
		util.PrintInfo("compare",
			"Values are not both references.  Other is %T",
			other)
		return false
	} else if exp.Kind != ov.Kind {
		util.PrintInfo("compare",
			"Reference type %v != %v",
			exp.Kind, ov.Kind)
		return false
	} else {
		return exp.Id == ov.Id && exp.OutputId == ov.OutputId
	}
}

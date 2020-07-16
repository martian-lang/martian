// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Utilities for testing semantic equivalence of AST nodes.

package syntax

import (
	"errors"
	"fmt"
	"math"

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
		len(other.List) != len(bindings.List) {
		return false
	}
	for _, b := range bindings.List {
		if b.Id == "*" {
			continue
		}
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
		} else if arg.IsFile() != KindIsFile && arg.GetTname() != oa.GetTname() {
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
		} else if fk := arg.IsFile(); fk != KindIsFile &&
			arg.GetTname() != oa.GetTname() {
			return false
		} else if (fk == KindIsFile || fk == KindIsDirectory) &&
			checkOutNames && arg.GetOutName() != oa.GetOutName() {
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

var notEqualError = fmt.Errorf("expression != nil")

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
	if binding.Exp == nil {
		return other.Exp == nil
	} else if other.Exp == nil {
		return false
	} else if err := binding.Exp.equal(other.Exp); err != nil {
		if errors.Is(err, notEqualError) {
			util.PrintInfo("compare",
				"Binding %s values differ.",
				binding.Id)
		} else {
			util.PrintInfo("compare",
				"Binding %s values differ:\n%v",
				binding.Id, err)
		}
		return false
	}
	return true
}

func (exp *StringExp) equal(other Exp) error {
	if s, ok := other.(*StringExp); !ok {
		return fmt.Errorf(
			"Values are not both strings.  Other is %T",
			other)
	} else if s.Value != exp.Value {
		return fmt.Errorf(
			"%q != %q",
			exp.Value, s.Value)
	} else {
		return nil
	}
}

func (exp *BoolExp) equal(other Exp) error {
	if b, ok := other.(*BoolExp); !ok {
		return fmt.Errorf(
			"Values are not both boolean.  Other is %T",
			b)
	} else if b.Value != exp.Value {
		return fmt.Errorf(
			"Differing boolean values")
	} else {
		return nil
	}
}

func (exp *IntExp) equal(other Exp) error {
	switch other := other.(type) {
	case *IntExp:
		if other.Value != exp.Value {
			return fmt.Errorf(
				"%d != %d",
				other.Value, exp.Value)
		}
		return nil
	case *FloatExp:
		if other.Value != float64(exp.Value) {
			return fmt.Errorf(
				"%g != %d",
				other.Value, exp.Value)
		}
		return nil
	}
	return notEqualError
}

func (exp *FloatExp) equal(other Exp) error {
	switch other := other.(type) {
	case *IntExp:
		if float64(other.Value) != exp.Value {
			return fmt.Errorf(
				"%d != %g",
				other.Value, exp.Value)
		}
		return nil
	case *FloatExp:
		if math.Abs(other.Value-exp.Value) <= math.Abs(exp.Value)*1e-15 {
			return nil
		}
		return fmt.Errorf(
			"%g != %g",
			other.Value, exp.Value)
	}
	return notEqualError
}

func (exp *NullExp) equal(other Exp) error {
	_, ok := other.(*NullExp)
	if !ok {
		return fmt.Errorf(
			"Values are not both null.  Other is %T",
			other)
	}
	return nil
}
func (exp *NullExp) Equal(uother MapCallSource) bool {
	_, ok := uother.(*NullExp)
	if ok {
		return true
	}
	return equivalentSource(exp, uother)
}

func (exp *MapExp) equal(uother Exp) error {
	other, ok := uother.(*MapExp)
	if !ok {
		return fmt.Errorf(
			"Values are not both %s.  Other is %T",
			exp.Kind, other)
	}
	if len(exp.Value) != len(other.Value) {
		return fmt.Errorf(
			"Map sizes differ: %d != %d",
			len(exp.Value), len(other.Value))
	}
	for k, v := range exp.Value {
		if ov, ok := other.Value[k]; !ok {
			return fmt.Errorf(
				"Missing map key %s",
				k)
		} else if err := v.equal(ov); err != nil {
			return err
		}
	}
	return nil
}
func (exp *MapExp) Equal(uother MapCallSource) bool {
	other, ok := uother.(*MapExp)
	if !ok {
		return false
	}
	return exp.equal(other) == nil
}

func (exp *ArrayExp) equal(uother Exp) error {
	other, ok := uother.(*ArrayExp)
	if !ok {
		return fmt.Errorf(
			"Values are not both arrays.  Other is %T",
			other)
	}
	if len(exp.Value) != len(other.Value) {
		return fmt.Errorf(
			"Array lengths differ: %d != %d",
			len(exp.Value), len(other.Value))
	}
	for i, v := range exp.Value {
		if err := v.equal(other.Value[i]); err != nil {
			return err
		}
	}
	return nil
}
func (exp *ArrayExp) Equal(uother MapCallSource) bool {
	other, ok := uother.(*ArrayExp)
	if !ok {
		return false
	}
	return exp.equal(other) == nil
}

func (exp *SplitExp) equal(uother Exp) error {
	other, ok := uother.(*SplitExp)
	if !ok {
		return fmt.Errorf(
			"Values are not both sweeps.  Other is %T",
			other)
	}
	return exp.Value.equal(other.Value)
}

func (m *MergeExp) equal(other Exp) error {
	if m == nil {
		if other == nil {
			return nil
		}
		return notEqualError
	} else if other == nil {
		return notEqualError
	}
	o, ok := other.(*MergeExp)
	if !ok {
		return notEqualError
	}
	if m.Value == nil {
		if o.Value == nil {
			return nil
		}
		return notEqualError
	}
	if err := m.Value.equal(o.Value); err != nil {
		return err
	}
	if m.MergeOver == o.MergeOver {
		return nil
	}
	if m.MergeOver.CallMode() != o.MergeOver.CallMode() ||
		m.MergeOver.KnownLength() != o.MergeOver.KnownLength() ||
		m.MergeOver.ArrayLength() != o.MergeOver.ArrayLength() {
		return notEqualError
	}
	mkeys := m.MergeOver.Keys()
	okeys := o.MergeOver.Keys()
	if len(mkeys) != len(okeys) {
		return notEqualError
	}
	for k := range mkeys {
		if _, ok := okeys[k]; !ok {
			return notEqualError
		}
	}
	return nil
}

func equivalentSource(s1, s2 MapCallSource) bool {
	if s1 == s2 {
		return true
	}
	if s1 == nil || s2 == nil {
		return false
	}
	if s1.CallMode() != s2.CallMode() {
		return false
	}
	if s1.KnownLength() {
		if !s2.KnownLength() {
			return false
		}
		switch s1.CallMode() {
		case ModeSingleCall, ModeNullMapCall, ModeUnknownMapCall:
			return true
		case ModeArrayCall:
			return s1.ArrayLength() == s2.ArrayLength()
		case ModeMapCall:
			k1, k2 := s1.Keys(), s2.Keys()
			if len(k1) != len(k2) {
				return false
			}
			for k := range k1 {
				if _, ok := k2[k]; !ok {
					return false
				}
			}
		}
	}
	return true
}

func (exp *RefExp) equal(other Exp) error {
	if exp == nil {
		if other == nil {
			return nil
		} else {
			return notEqualError
		}
	} else if other == nil {
		return notEqualError
	} else if ov, ok := other.(*RefExp); !ok {
		return fmt.Errorf("Values are not both references.  Other is %T",
			other)
	} else if exp.Kind != ov.Kind {
		return fmt.Errorf(
			"Reference type %v != %v",
			exp.Kind, ov.Kind)
	} else if exp.Id != ov.Id || exp.OutputId != ov.OutputId {
		return notEqualError
	} else if len(exp.Forks) != len(ov.Forks) {
		return nil
	} else {
		for c, i := range exp.Forks {
			j := ov.Forks[c]
			if i == nil && j != nil {
				return fmt.Errorf("nil fork dim %s", c.Id)
			} else if j == nil {
				return fmt.Errorf("missing fork dim %s", c.Id)
			}
			if !indexEqual(i, j) {
				return fmt.Errorf("fork index %s != %s",
					i.GoString(), j.GoString())
			}
		}
		return nil
	}
}

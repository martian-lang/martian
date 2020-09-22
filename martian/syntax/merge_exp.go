// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Merge expressions wrap outputs of pipelines which were called in a map call.

package syntax

import (
	"fmt"
	"sort"
)

// A MergeExp represents wraps a struct value that is the output of a mapped
// pipeline call.
//
// Given a pipeline
//
//   pipeline FOO(
//       in  int   x,
//       out int[] y,
//   )
//   {
//       return (
//           y = [self.x],
//       )
//   }
//
// the output of a normal (non-mapped) call to the pipeline would be a struct
// containing the element `y`.  When pipeline is called in a map, however, the
// result is an array of such structures.  A MergeExp represents this; the
// first pass output of the above pipeline, if called with `x = split [1, 2, 3]`
// would be
//
//   merge {
//       y: [split [1, 2, 3]],
//   }
//
// which would resolve to
//
//   [
//       {y: [1]},
//       {y: [2]},
//       {y: [3]},
//   ]
//
// A merge expression represents such a merge over a single level  of the
// forking hierarchy, as represented by the call graph node where the split
// happened.
//
// MergeOver specifies the MapCallSource used for the merge.  If it does not
// have a length that can be computed at mro compile time, then ForkNode
// will specify the fully-qualified name of a stage node which forks on the
// specified call.  If there are any references to such stages inside Value,
// ForkNode will specify one of those nodes; otherwise one will be picked
// arbitrarily from the set of stages which feed inputs to the pipeline
// represented by the Call.
//
// This expression type does not exist in mro source code - it is only produced
// during call graph resolution.  The formatting methods exist to implement the
// expression interface and for convenience in error reporting.
type MergeExp struct {
	Call      *CallGraphStage `json:"-"`
	Value     Exp             `json:"merge_value"`
	MergeOver MapCallSource   `json:"merge_over,omitempty"`
	ForkNode  *RefExp         `json:"fork_node,omitempty"`
}

func (s *MergeExp) getNode() *AstNode { return &s.Call.Call().Node }
func (s *MergeExp) File() *SourceFile { return s.Call.Call().Node.Loc.File }
func (s *MergeExp) Line() int         { return s.Call.Call().Node.Loc.Line }
func (*MergeExp) getKind() ExpKind    { return KindMerge }

func (*MergeExp) inheritComments() bool     { return false }
func (*MergeExp) getSubnodes() []AstNodable { return nil }

func (s *MergeExp) HasRef() bool {
	if s.ForkNode != nil {
		return true
	}
	return s.Value.HasRef()
}

func (s *MergeExp) HasSplit() bool {
	return s.Value.HasSplit()
}

// CallMode Returns the call mode for a call which depends on this source.
func (m *MergeExp) CallMode() CallMode {
	return m.MergeOver.CallMode()
}

// KnownLength returns true if the array length or map keys of the merge are
// known.
func (m *MergeExp) KnownLength() bool {
	return m.MergeOver.KnownLength()
}

// ArrayLength returns the array length, if known, or -1.
func (m *MergeExp) ArrayLength() int {
	return m.MergeOver.ArrayLength()
}

// Keys returns the map keys, if known.
func (m *MergeExp) Keys() map[string]Exp {
	return m.MergeOver.Keys()
}

func (m *MergeExp) GetCall() *CallStm {
	return m.Call.Call()
}

// Recursively searches the expression for references.
func (m *MergeExp) FindRefs() []*RefExp {
	if m == nil || m.Value == nil {
		return nil
	}
	refs := m.Value.FindRefs()
	if m.ForkNode != nil {
		for _, ref := range refs {
			if ref.Id == m.ForkNode.Id {
				return refs
			}
		}
		refs = append(refs, m.ForkNode)
	}
	return refs
}

func (m *MergeExp) filter(t Type, lookup *TypeLookup) (Exp, error) {
	if m == nil || m.Value == nil {
		return m, nil
	}
	var innerType Type
	switch t := t.(type) {
	case *ArrayType:
		if m.MergeOver.CallMode() == ModeMapCall {
			return m, &IncompatibleTypeError{
				Message: fmt.Sprintf("unexpected typed-map merge expression for %s\n%s",
					t.TypeId().str(),
					FormatExp(m, "")),
			}
		}
		innerType = t.Elem
	case *TypedMapType:
		if m.MergeOver.CallMode() == ModeArrayCall {
			return m, &IncompatibleTypeError{
				Message: fmt.Sprintf("unexpected array merge expression for %s\n%s",
					t.TypeId().str(),
					FormatExp(m, "")),
			}
		}
		innerType = t.Elem
	default:
		return m, &IncompatibleTypeError{
			Message: fmt.Sprintf("unexpected merge expression for %s\n%s",
				t.TypeId().str(),
				FormatExp(m, "")),
		}
	}
	val, err := m.Value.filter(innerType, lookup)
	if err != nil {
		err = &bindingError{
			Msg: "filtering merge",
			Err: err,
		}
	}
	if val == m.Value {
		return m, err
	}
	result := *m
	result.Value = val
	return &result, err
}

func (m *MergeExp) FindTypedRefs(list []*BoundReference,
	t Type, lookup *TypeLookup) ([]*BoundReference, error) {
	var innerType Type
	switch t := t.(type) {
	case *ArrayType:
		if m.MergeOver.CallMode() == ModeMapCall {
			return list, &IncompatibleTypeError{
				Message: fmt.Sprintf("unexpected typed-map merge expression for %s\n%s",
					t.TypeId().str(),
					FormatExp(m, "")),
			}
		}
		innerType = lookup.GetArray(t, -1)
	case *TypedMapType:
		if m.MergeOver.CallMode() == ModeArrayCall {
			return list, &IncompatibleTypeError{
				Message: fmt.Sprintf("unexpected array merge expression for %s\n%s",
					t.TypeId().str(),
					FormatExp(m, "")),
			}
		}
		innerType = t.Elem
	default:
		return list, &IncompatibleTypeError{
			Message: fmt.Sprintf("unexpected merge expression for %s\n%s",
				t.TypeId().str(),
				FormatExp(m, "")),
		}
	}
	list, err := m.Value.FindTypedRefs(list, innerType, lookup)
	if err != nil {
		err = &bindingError{
			Msg: "in merge",
			Err: err,
		}
	}
	return list, err
}

// IsEmpty returns true if the merge expression is over an empty array or map.
func (m *MergeExp) IsEmpty() bool {
	return m.MergeOver.KnownLength() &&
		len(m.MergeOver.Keys()) == 0 &&
		m.MergeOver.ArrayLength() <= 0
}

func (exp *MergeExp) wrapError(err error) error {
	if err == nil {
		return nil
	}
	if exp.Call == nil {
		return &bindingError{
			Msg: "top-level merge",
			Err: err,
		}
	}
	return &bindingError{
		Msg: "merge for " + exp.Call.GetFqid(),
		Err: err,
	}
}

func (exp *MergeExp) resolveRefs(self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) (Exp, error) {
	v, err := exp.Value.resolveRefs(self, siblings, lookup)
	if err != nil {
		return exp, exp.wrapError(err)
	}
	fn := findMergeForkNode(v, exp.Call)
	if fn != nil && fn.Id == exp.Call.Fqid {
		fn = nil
	}
	if v != exp.Value || fn != exp.ForkNode {
		e := *exp
		e.Value = v
		e.ForkNode = fn
		return &e, nil
	}
	return exp, nil
}

func sourceForFork(src MapCallSource, fork map[*CallStm]CollectionIndex,
	lookup *TypeLookup) (MapCallSource, error) {
	if src.KnownLength() {
		return src, nil
	}
	switch se := src.(type) {
	case *MergeExp:
		return sourceForFork(se.MergeOver, fork, lookup)
	case Exp:
		if se, err := se.BindingPath("", fork, lookup); err != nil {
			return src, err
		} else if ss, ok := se.(MapCallSource); ok {
			return ss, nil
		} else {
			return src, fmt.Errorf("source fork %s did not resolve to a source",
				se.GoString())
		}
	case *MapCallSet:
		return sourceForFork(se.Master, fork, lookup)
	}
	return src, nil
}

func findMergeForkExpNode(v Exp, call *CallStm) *RefExp {
	switch v := v.(type) {
	case *RefExp:
		for c := range v.Forks {
			if c == call {
				return v
			}
		}
	case *ArrayExp:
		for _, e := range v.Value {
			if m := findMergeForkExpNode(e, call); m != nil {
				return m
			}
		}
	case *MapExp:
		keys := make([]string, 0, len(v.Value))
		for k := range v.Value {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if m := findMergeForkExpNode(v.Value[k], call); m != nil {
				return m
			}
		}
	case *DisabledExp:
		if m := findMergeForkExpNode(v.Value, call); m != nil {
			return m
		}
		return findMergeForkExpNode(v.Disabled, call)
	case *SplitExp:
		if m := findMergeForkExpNode(v.Value, call); m != nil {
			return m
		}
		if v.Call == call {
			if vm, ok := v.Value.(*MergeExp); ok {
				return findMergeForkExpNode(vm.Value, vm.Call.call)
			}
		}
	case *MergeExp:
		return findMergeForkExpNode(v.Value, call)
	}
	return nil
}

func findMergeForkNode(v Exp, call *CallGraphStage) *RefExp {
	if m := findMergeForkExpNode(v, call.call); m != nil {
		return m
	}
	for _, e := range call.Disable {
		if m := findMergeForkExpNode(e, call.call); m != nil {
			return m
		}
	}
	// Search the inputs for a fork.
	if len(call.Inputs) > 0 {
		ins := make([]string, 0, len(call.Inputs))
		for i := range call.Inputs {
			ins = append(ins, i)
		}
		sort.Strings(ins)
		for _, i := range ins {
			e := call.Inputs[i]
			if e != nil {
				if m := findMergeForkExpNode(e.Exp, call.call); m != nil {
					return m
				}
			}
		}
	}
	return nil
}

func (s *MergeExp) BindingPath(bindPath string,
	fork map[*CallStm]CollectionIndex,
	lookup *TypeLookup) (Exp, error) {
	if s == nil || s.Value == nil {
		return nil, nil
	}
	v, err := s.Value.BindingPath(bindPath, fork, lookup)
	if i := fork[s.GetCall()]; i != nil && i.IndexSource() == nil {
		return v, s.wrapError(err)
	}
	src := s.MergeOver
	if se, serr := sourceForFork(src, fork, lookup); serr != nil {
		if err == nil {
			err = serr
		} else {
			err = ErrorList{err, serr}
		}
	} else {
		src = se
	}
	for ms, ok := src.(*MergeExp); ok; ms, ok = src.(*MergeExp) {
		src = ms.MergeOver
	}
	if err != nil || !src.KnownLength() {
		fn := findMergeForkNode(v, s.Call)
		if fn != nil && fn.Id == s.Call.Fqid {
			fn = nil
		}
		if v != s.Value ||
			(fn != s.ForkNode &&
				(fn == nil || s.ForkNode == nil || fn.Id != s.ForkNode.Id)) {
			sc := *s
			sc.Value = v
			sc.ForkNode = fn
			s = &sc
		}
		return s, s.wrapError(err)
	}
	// Static merge, do it now.
	switch src.CallMode() {
	case ModeArrayCall:
		if src.ArrayLength() == 0 {
			return &NullExp{
				valExp: valExp{Node: *v.getNode()},
			}, s.wrapError(err)
		}
		var errs ErrorList
		arr := ArrayExp{
			valExp: valExp{Node: *v.getNode()},
			Value:  make([]Exp, src.ArrayLength()),
		}
		// Put aside existing fork value
		ov, ok := fork[s.GetCall()]
		if !ok && fork != nil {
			defer delete(fork, s.GetCall())
		} else {
			defer func() { fork[s.GetCall()] = ov }()
		}
		if fork == nil {
			fork = make(map[*CallStm]CollectionIndex)
		}
		for i := range arr.Value {
			fork[s.GetCall()] = arrayIndex(i)
			iv, err := v.BindingPath("", fork, lookup)
			if err != nil {
				errs = append(errs, err)
			}
			arr.Value[i] = iv
		}
		return &arr, s.wrapError(errs.If())
	case ModeMapCall:
		keys := src.Keys()
		if len(keys) == 0 {
			return &NullExp{
				valExp: valExp{Node: *v.getNode()},
			}, s.wrapError(err)
		}

		var errs ErrorList
		arr := MapExp{
			valExp: valExp{Node: *v.getNode()},
			Kind:   KindMap,
			Value:  make(map[string]Exp, len(keys)),
		}
		// Put aside existing fork value
		ov, ok := fork[s.GetCall()]
		if !ok && fork != nil {
			defer delete(fork, s.GetCall())
		} else {
			defer func() { fork[s.GetCall()] = ov }()
		}
		if fork == nil {
			fork = make(map[*CallStm]CollectionIndex)
		}
		for i := range keys {
			fork[s.GetCall()] = mapKeyIndex(i)
			iv, err := v.BindingPath("", fork, lookup)
			if err != nil {
				errs = append(errs, err)
			}
			arr.Value[i] = iv
		}
		return &arr, s.wrapError(errs.If())
	default:
		panic("invalid merge kind " + src.CallMode().String())
	}
}

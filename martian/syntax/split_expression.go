// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Expression splitting arrays or maps for a map call.

package syntax

import (
	"strconv"
)

// A SplitExp represents an expression which is either a typed map or array,
// where a call is made repeatedly, once for each element in the collection.
// Only top-level expressions can be split.
type SplitExp struct {
	valExp
	// Either a MapExp, ArrayExp, or RefExp.  In a fully-resolved pipeline,
	// it could also be another nested SplitExp.
	Value Exp
	// The type for the binding expression.
	Type Type
	// The call that this split was made on.  This becomes relevant when
	// resolving a map call of a pipeline - calls within that pipeline
	// may split further, but potentially over a different set of keys.
	Call *CallStm

	// The dimensionality source
	Source MapCallSource
}

func (s *SplitExp) getKind() ExpKind { return KindSplit }
func (s *SplitExp) val() interface{} { return s.Value }

func (s *SplitExp) getSubnodes() []AstNodable {
	return []AstNodable{s.Value}
}

func (e *SplitExp) HasRef() bool {
	if e == nil || e.Value == nil {
		return false
	}
	return e.Value.HasRef()
}

func (e *SplitExp) HasSplit() bool {
	return true
}

func (e *SplitExp) GetCall() *CallStm {
	return e.Call
}

func (e *SplitExp) innerValue() Exp {
	return e.Value
}

func (e *SplitExp) mapSource() MapCallSource {
	return e.Source
}

func (e *SplitExp) setSource(src MapCallSource) {
	e.Source = src
}

// IsEmpty returns true if the split expression is over an empty array or map.
func (e *SplitExp) IsEmpty() bool {
	switch exp := e.Value.(type) {
	case *ArrayExp:
		if len(exp.Value) == 0 {
			return true
		}
	case *MapExp:
		if len(exp.Value) == 0 {
			return true
		}
	case *NullExp:
		return true
	case *SplitExp:
		if exp.IsEmpty() {
			return true
		}
	case *MergeExp:
		if exp.IsEmpty() {
			return true
		}
	}
	return e.Source.KnownLength() &&
		len(e.Source.Keys()) == 0 &&
		e.Source.ArrayLength() <= 0
}

func (e *SplitExp) FindRefs() []*RefExp {
	refs := e.Value.FindRefs()
	if e.Source != nil {
		switch s := e.Source.(type) {
		case *MapCallSet:
			switch r := s.Master.(type) {
			case *RefExp:
				refs = append(refs, r)
			case *BoundReference:
				refs = append(refs, r.Exp)
			}
		case *ArrayExp, *MapExp:
		case Exp:
			refs = append(refs, s.FindRefs()...)
		}
	}
	return refs
}

func (exp *SplitExp) FindTypedRefs(list []*BoundReference,
	t Type, lookup *TypeLookup) ([]*BoundReference, error) {
	tid := t.TypeId()
	var innerType Type
	switch val := exp.Value.(type) {
	case *MapExp:
		innerType = lookup.GetMap(t)
	case *ArrayExp:
		innerType = lookup.GetArray(t, 1)
	case *MergeExp:
		tt, err := lookup.AddDim(t, exp.CallMode())
		if err != nil {
			return list, err
		}
		return val.Value.FindTypedRefs(list, tt, lookup)
	case *RefExp:
		if tid.ArrayDim > 0 {
			tid.ArrayDim--
		} else if tid.MapDim > 0 {
			tid.ArrayDim = tid.MapDim - 1
			tid.MapDim = 0
		}
		innerType = lookup.Get(tid)
	case *NullExp:
		innerType = &builtinNull
	case *SplitExp:
		var err error
		innerType, err = lookup.AddDim(t, val.CallMode())
		if err != nil {
			return list, err
		}
	case *DisabledExp:
		list, err := val.Value.FindTypedRefs(list, t, lookup)
		if err != nil {
			return list, err
		}
		return val.Disabled.FindTypedRefs(list, &builtinBool, lookup)
	default:
		return list, &wrapError{
			innerError: &bindingError{
				Msg: "split was not over an array, map, or ref: " +
					string(exp.Value.getKind()),
			},
			loc: exp.Node.Loc,
		}
	}
	list, err := exp.Value.FindTypedRefs(list, innerType, lookup)
	if err != nil {
		err = &bindingError{
			Msg: "in split",
			Err: err,
		}
	}
	return list, err
}

func (exp *SplitExp) GetExp() Exp {
	return exp.Value
}

func (exp *SplitExp) InnerMapSource() MapCallSource {
	switch e := exp.Value.(type) {
	case *MergeExp:
		if is, ok := e.Value.(MapCallSource); ok {
			return is
		}
	case *ArrayExp:
		if len(e.Value) == 0 {
			return &NullExp{valExp: exp.valExp}
		}
		var inner MapCallSource
		for _, ev := range e.Value {
			if is, ok := ev.(MapCallSource); !ok || is == nil {
				return nil
			} else if inner == nil {
				inner = is
			} else if m := inner.CallMode(); m != is.CallMode() {
				return &mapSourcePlaceholder
			} else if m == ModeArrayCall && inner.ArrayLength() != is.ArrayLength() {
				if inner.KnownLength() {
					inner = new(placeholderArrayMapSource)
				}
			} else if m == ModeMapCall {
				ka, kb := inner.Keys(), is.Keys()
				if len(ka) != len(kb) {
					if ka != nil {
						inner = new(placeholderMapSource)
					}
				} else {
					for k := range ka {
						if _, ok := kb[k]; !ok {
							inner = new(placeholderMapSource)
							break
						}
					}
				}
			}
		}
		return inner
	case *MapExp:
		if len(e.Value) == 0 {
			return &NullExp{valExp: exp.valExp}
		}
		var inner MapCallSource
		for _, ev := range e.Value {
			if is, ok := ev.(MapCallSource); !ok || is == nil {
				return nil
			} else if inner == nil {
				inner = is
			} else if m := inner.CallMode(); m != is.CallMode() {
				return &mapSourcePlaceholder
			} else if m == ModeArrayCall && inner.ArrayLength() != is.ArrayLength() {
				if inner.KnownLength() {
					inner = new(placeholderMapMapSource)
				}
			} else if m == ModeMapCall {
				ka, kb := inner.Keys(), is.Keys()
				if len(ka) != len(kb) {
					if ka != nil {
						inner = new(placeholderMapSource)
					}
				} else {
					for k := range ka {
						if _, ok := kb[k]; !ok {
							inner = new(placeholderMapSource)
							break
						}
					}
				}
			}
		}
		return inner
	}
	return nil
}

func (exp *SplitExp) KnownLength() bool {
	if src := exp.InnerMapSource(); src != nil {
		return src.KnownLength()
	}
	return false
}

func (exp *SplitExp) ArrayLength() int {
	if src := exp.InnerMapSource(); src != nil {
		return src.ArrayLength()
	}
	return -1
}

func (exp *SplitExp) Keys() map[string]Exp {
	if src := exp.InnerMapSource(); src != nil {
		return src.Keys()
	}
	return nil
}

func (exp *SplitExp) CallMode() CallMode {
	if exp.Type != nil {
		t := exp.Type
		switch tt := t.(type) {
		case *ArrayType:
			if tt.Dim > 1 {
				return ModeArrayCall
			} else if tt.Dim == 1 {
				t = tt.Elem
			}
		case *TypedMapType:
			t = tt.Elem
		}
		switch tt := t.(type) {
		case *ArrayType:
			if tt.Dim > 0 {
				return ModeArrayCall
			}
		case *TypedMapType:
			return ModeMapCall
		}
	}
	switch e := exp.Value.(type) {
	case *RefExp:
		return ModeUnknownMapCall
	case *MergeExp:
		if is, ok := e.Value.(MapCallSource); ok {
			return is.CallMode()
		}
	case *ArrayExp:
		if len(e.Value) == 0 {
			return ModeNullMapCall
		}
		var inner CallMode = -1
		for _, ev := range e.Value {
			if is, ok := ev.(MapCallSource); !ok || is == nil {
				if inner == -1 || inner == ModeSingleCall {
					inner = ModeSingleCall
				} else {
					return ModeUnknownMapCall
				}
			} else if inner == -1 {
				inner = is.CallMode()
			} else if m := is.CallMode(); inner != m {
				if inner == ModeUnknownMapCall || inner == ModeNullMapCall {
					inner = m
				} else if m != ModeUnknownMapCall && m != ModeNullMapCall {
					return ModeUnknownMapCall
				}
			}
		}
		return inner
	case *MapExp:
		if len(e.Value) == 0 {
			return ModeNullMapCall
		}
		var inner CallMode = -1
		for _, ev := range e.Value {
			if is, ok := ev.(MapCallSource); !ok || is == nil {
				if inner == -1 || inner == ModeSingleCall {
					inner = ModeSingleCall
				} else {
					return ModeUnknownMapCall
				}
			} else if inner == -1 {
				inner = is.CallMode()
			} else if m := is.CallMode(); inner != m {
				if inner == ModeUnknownMapCall || inner == ModeNullMapCall {
					inner = m
				} else if m != ModeUnknownMapCall && m != ModeNullMapCall {
					return ModeUnknownMapCall
				}
			}
		}
		return inner
	}
	return ModeSingleCall
}

func (exp *SplitExp) wrapError(err error) error {
	if err == nil {
		return nil
	}
	if exp.Call == nil {
		return &bindingError{
			Msg: "top-level split",
			Err: err,
		}
	}
	return &bindingError{
		Msg: exp.GoString() + " for " + exp.Call.Id,
		Err: err,
	}
}

func (exp *SplitExp) resolveRefs(self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) (Exp, error) {
	v, err := exp.Value.resolveRefs(self, siblings, lookup)
	if err != nil {
		return exp, exp.wrapError(err)
	}
	src := exp.Source
	if s, ok := src.(*MapCallSet); ok {
		// Break out of the set, because we don't want to propagate merges back
		// to the call's set.  It's possible for one call to appear in more
		// than one CallGraphNode, with different sources.
		src = s.Master
	}
	switch s := src.(type) {
	case *RefExp:
		re, err := s.resolveRefs(self, siblings, lookup)
		if err == nil {
			if rs, ok := re.(MapCallSource); ok {
				src = rs
			}
		}
	case *BoundReference:
		re, err := s.Exp.resolveRefs(self, siblings, lookup)
		if err == nil {
			switch rs := re.(type) {
			case *RefExp:
				src = &BoundReference{
					Exp:  rs,
					Type: s.Type,
				}
			case MapCallSource:
				src = rs
			}
		}
	}
	if vs, ok := v.(MapCallSource); ok {
		src, err = MergeMapCallSources(vs, src)
	}
	if v != exp.Value || src != exp.Source {
		e := *exp
		e.Value = v
		e.Source = src
		return &e, exp.wrapError(err)
	}
	return exp, exp.wrapError(err)
}

func (s *SplitExp) BindingPath(bindPath string,
	fork map[*CallStm]CollectionIndex,
	lookup *TypeLookup) (Exp, error) {
	if s == nil || s.Value == nil {
		return s, nil
	}
	if s.IsEmpty() {
		return &NullExp{valExp: s.valExp}, nil
	}
	i, ok := fork[s.Call]
	if !ok {
		if fork == nil {
			fork = make(map[*CallStm]CollectionIndex)
		} else {
			defer delete(fork, s.Call)
		}
		i = unknownIndex{src: s.Source}
		fork[s.Call] = i
	} else if i.IndexSource() == nil {
		done, v, err := getElement(s.Value, i, false)
		if done {
			if err != nil {
				return s, s.wrapError(err)
			}
			v, err := v.BindingPath(bindPath, fork, lookup)
			return v, s.wrapError(err)
		}
	}
	src := s.Source
	v, err := s.Value.BindingPath(bindPath, fork, lookup)
	if err != nil {
		return s, s.wrapError(err)
	}
	switch val := v.(type) {
	case *NullExp:
		return s.Value, nil
	case *MergeExp:
		if i != nil && i.IndexSource() == nil {
			if val.GetCall() != s.Call {
				oldFork, oldForkOk := fork[val.GetCall()]
				fork[val.GetCall()] = i
				if oldForkOk {
					defer func() { fork[val.GetCall()] = oldFork }()
				} else {
					defer delete(fork, val.GetCall())
				}
			}
			return val.Value.BindingPath("", fork, lookup)
		} else if nsrc, err := MergeMapCallSources(val.MergeOver, src); err != nil {
			return s, s.wrapError(err)
		} else {
			src = nsrc
			val.MergeOver = src
		}
	case *ArrayExp:
		src = val
	case *MapExp:
		src = val
	case *RefExp:
		if _, ok := val.Forks[s.Call]; ok {
			// This is just the one fork of the ref.
			return val, nil
		}
		// Otherwise we're splitting an array or map output of the ref.
	case *SplitExp:
		if ss := val.InnerMapSource(); ss != nil && ss.KnownLength() {
			src = ss
		} else if !src.KnownLength() {
			src = val
		}
	}
	if ok && i.IndexSource() == nil {
		done, v, err := getElement(v, i, true)
		if done {
			if err != nil {
				return s, s.wrapError(err)
			}
			return v, nil
		}
	}
	if v == s.Value && src == s.Source {
		return s, nil
	}
	e := *s
	e.Value = v
	e.Source = src
	if bindPath != "" && s.Type != nil {
		tt, err := bindingType(bindPath, s.Type, lookup)
		if err != nil {
			e.Type = nil
		} else {
			e.Type = tt
		}
	}
	return &e, nil
}

func getElement(v Exp, i CollectionIndex, decendSplit bool) (bool, Exp, error) {
	switch val := v.(type) {
	case *ArrayExp:
		if i.Mode() != ModeArrayCall {
			return true, v, &bindingError{
				Msg: "cannot use " + i.Mode().String() + " key to index array",
			}
		}
		i := i.ArrayIndex()
		if len(val.Value) <= i {
			return true, v, &bindingError{
				Msg: "array index " + strconv.Itoa(i) +
					" out of range for array of length " +
					strconv.Itoa(len(val.Value)),
			}
		}
		if i < 0 {
			return true, v, &bindingError{
				Msg: "array index " + strconv.Itoa(i) + " out of range.",
			}
		}
		return true, val.Value[i], nil
	case *MapExp:
		if i.Mode() != ModeMapCall {
			return true, v, &bindingError{
				Msg: "cannot use " + i.Mode().String() + " key to index array",
			}
		}
		key := i.MapKey()
		if v, ok := val.Value[key]; !ok {
			return true, v, &bindingError{
				Msg: "key " + key + " not in map",
			}
		} else {
			return true, v, nil
		}
	case *SplitExp:
		if decendSplit {
			return invertSplit(val, i)
		}
	}
	return false, v, nil
}

func invertSplit(sp *SplitExp, i CollectionIndex) (bool, Exp, error) {
	if !sp.Source.KnownLength() {
		return false, sp, nil
	}
	var errs ErrorList
	switch v := sp.Value.(type) {
	case *ArrayExp:
		arr := *v
		arr.Value = make([]Exp, len(v.Value))
		done := true
		change := false
		for j, vv := range v.Value {
			d, e, err := getElement(vv, i, true)
			if err != nil {
				errs = append(errs, err)
			}
			done = done && d
			arr.Value[j] = e
			change = change || (e != vv)
		}
		if !change {
			return done, sp, errs.If()
		}
		r := *sp
		r.Value = &arr
		return done, &r, errs.If()
	case *MapExp:
		m := *v
		m.Value = make(map[string]Exp, len(v.Value))
		done := true
		change := false
		for k, vv := range v.Value {
			d, e, err := getElement(vv, i, true)
			if err != nil {
				errs = append(errs, err)
			}
			done = done && d
			m.Value[k] = e
			change = change || (e != vv)
		}
		if !change {
			return done, sp, errs.If()
		}
		r := *sp
		r.Value = &m
		return done, &r, errs.If()
	}
	return false, sp, nil
}

func (s *SplitExp) filter(t Type, lookup *TypeLookup) (Exp, error) {
	if _, ok := baseType(t).(*StructType); !ok {
		return s, nil
	}
	if s == nil || s.Value == nil {
		return s, nil
	}
	switch s.Source.CallMode() {
	case ModeArrayCall:
		t = lookup.GetArray(t, 1)
	case ModeMapCall:
		t = lookup.GetMap(t)
	default:
		if ss, ok := s.Value.(MapCallSource); ok {
			switch ss.CallMode() {
			case ModeArrayCall:
				t = lookup.GetArray(t, 1)
			case ModeMapCall:
				t = lookup.GetMap(t)
			default:
				return s, &wrapError{
					innerError: &bindingError{
						Msg: "invalid split type " + s.CallMode().String() +
							" (value " + s.GoString() + ")",
					},
					loc: s.Node.Loc,
				}
			}
		} else {
			return s, &wrapError{
				innerError: &bindingError{
					Msg: "invalid split type " + s.CallMode().String() +
						" (" + s.GoString() + ")",
				},
				loc: s.Node.Loc,
			}
		}
	}
	v, err := s.Value.filter(t, lookup)
	if v == s.Value {
		return s, err
	}
	e := *s
	e.Value = v
	return &e, err
}

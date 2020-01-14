// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"fmt"
	"strings"
)

func (exp *RefExp) resolveRefs(self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) (Exp, error) {
	var res *ResolvedBinding
	if exp.Kind == KindSelf {
		res = self[exp.Id]
		if res == nil {
			return exp, &bindingError{
				Msg: "unknown parameter " + exp.Id,
			}
		}
	} else {
		res = siblings[exp.Id]
		if res == nil {
			return exp, &bindingError{
				Msg: "unknown call name " + exp.Id,
			}
		}
	}
	return res.Exp.BindingPath(exp.OutputId, exp.ForkIndex, exp.OutputIndex)
}

func (exp *ArrayExp) resolveRefs(self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) (Exp, error) {
	var errs ErrorList
	result := ArrayExp{
		valExp: valExp{Node: exp.Node},
		Value:  make([]Exp, len(exp.Value)),
	}
	for i, subexp := range exp.Value {
		e, err := subexp.resolveRefs(self, siblings, lookup)
		if err != nil {
			errs = append(errs, &bindingError{
				Msg: fmt.Sprintf("element %d", i),
				Err: err,
			})
		}
		result.Value[i] = e
	}
	return &result, errs.If()
}

func (exp *SplitExp) resolveRefs(self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) (Exp, error) {
	v, err := exp.Value.resolveRefs(self, siblings, lookup)
	r := exp.Source
	if rr, ok := v.(refMapResolver); ok && rr != nil {
		r = rr.resolveMapSource(exp.CallMode())
	}
	if err == nil && (r != exp.Source || v != exp.Value) {
		e := *exp
		e.Value = v
		if r == nil {
			panic("nil source")
		}
		e.Source = r
		return &e, nil
	}
	return exp, err
}

func (exp *MapExp) resolveRefs(self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) (Exp, error) {
	var errs ErrorList
	result := MapExp{
		valExp: valExp{Node: exp.Node},
		Kind:   exp.Kind,
		Value:  make(map[string]Exp, len(exp.Value)),
	}
	for i, subexp := range exp.Value {
		e, err := subexp.resolveRefs(self, siblings, lookup)
		if err != nil {
			errs = append(errs, &bindingError{
				Msg: "key " + i,
				Err: err,
			})
		}
		result.Value[i] = e
	}
	return &result, errs.If()
}

func (exp *FloatExp) resolveRefs(self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) (Exp, error) {
	return exp, nil
}

func (exp *IntExp) resolveRefs(self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) (Exp, error) {
	return exp, nil
}

func (exp *BoolExp) resolveRefs(self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) (Exp, error) {
	return exp, nil
}
func (exp *StringExp) resolveRefs(self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) (Exp, error) {
	return exp, nil
}

func (exp *NullExp) resolveRefs(self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) (Exp, error) {
	return exp, nil
}

func (s *ArrayExp) BindingPath(bindPath string,
	fork map[MapCallSource]CollectionIndex,
	index []CollectionIndex) (Exp, error) {
	if len(index) > 0 {
		i := index[0]
		if i.IndexSource() != nil {
			i = fork[i.IndexSource()]
		}
		if m := i.Mode(); m != ModeArrayCall {
			return s, &bindingError{
				Msg: "taking " + m.String() + " key of array",
			}
		}
		if i != nil && i.IndexSource() == nil {
			if i.ArrayIndex() >= len(s.Value) {
				return s, &bindingError{
					Msg: fmt.Sprintf("index %d > array length %d",
						i.ArrayIndex(), len(s.Value)),
				}
			}
			if e := s.Value[i.ArrayIndex()]; e == nil {
				return e, nil
			} else {
				return e.BindingPath(bindPath, fork, index[1:])
			}
		}
		index = index[1:]
	}
	// handle projection
	result := ArrayExp{
		valExp: valExp{Node: s.Node},
		Value:  make([]Exp, len(s.Value)),
	}
	if (bindPath == "" && len(fork) == 0 && len(index) == 0) ||
		s == nil || len(s.Value) == 0 {
		return s, nil
	}
	change := false
	var errs ErrorList
	for i, sub := range s.Value {
		e, err := sub.BindingPath(bindPath, fork, index)
		if err != nil {
			errs = append(errs, &wrapError{
				innerError: &bindingError{
					Msg: fmt.Sprint("element ", i),
					Err: err,
				},
				loc: sub.getNode().Loc,
			})
		}
		result.Value[i] = e
		if e != sub {
			change = true
		}
	}
	if !change {
		return s, errs.If()
	}
	return &result, errs.If()
}

func (s *SplitExp) BindingPath(bindPath string,
	fork map[MapCallSource]CollectionIndex,
	index []CollectionIndex) (Exp, error) {
	if s == nil || s.Value == nil {
		return s, nil
	}
	if _, ok := s.Value.(*NullExp); ok {
		return s.Value, nil
	}
	if id, ok := fork[s.Source]; ok && id.IndexSource() == nil {
		return s.Value.BindingPath(bindPath,
			fork, append(index, id))
	}
	if bindPath == "" {
		if v, err := s.Value.BindingPath(bindPath, fork, index); v == s.Value {
			return s, nil
		} else if v != s.Value {
			r := *s
			r.Value = v
			return &r, err
		} else {
			return s, err
		}
	}
	// handle projection
	// Skip this level of nesting.
	if len(index) > 0 {
		index = index[1:]
	}
	v, err := s.Value.BindingPath(bindPath, fork, index)
	if v == s.Value {
		return s, err
	}
	e := *s
	e.Value = v
	return &e, err
}

func (s *MapExp) BindingPath(bindPath string,
	fork map[MapCallSource]CollectionIndex,
	index []CollectionIndex) (Exp, error) {

	if (bindPath == "" && len(fork) == 0 && len(index) == 0) ||
		s == nil || len(s.Value) == 0 {
		return s, nil
	}
	if s.Kind == KindStruct && bindPath != "" {
		parts := strings.SplitN(bindPath, ".", 2)
		var remainder string
		if len(parts) == 2 {
			remainder = parts[1]
		}
		sub, ok := s.Value[parts[0]]
		if !ok {
			return nil, &wrapError{
				innerError: &bindingError{
					Msg: "no element " + parts[0],
				},
				loc: s.Node.Loc,
			}
		}
		return sub.BindingPath(remainder, fork, index)
	}
	if len(index) > 0 {
		i := index[0]
		if i.IndexSource() != nil {
			i = fork[i.IndexSource()]
		}
		if m := i.Mode(); m != ModeMapCall {
			return s, &bindingError{
				Msg: "taking " + m.String() + " key of map",
			}
		}
		if i != nil {
			if e, ok := s.Value[i.MapKey()]; !ok {
				return e, &bindingError{
					Msg: "missing key " + i.MapKey(),
				}
			} else if e == nil {
				return e, nil
			} else {
				return e.BindingPath(bindPath, fork, index[1:])
			}
		}
		index = index[1:]
	}
	// handle projection
	result := MapExp{
		valExp: valExp{Node: s.Node},
		Kind:   s.Kind,
		Value:  make(map[string]Exp, len(s.Value)),
	}
	var errs ErrorList
	for i, sub := range s.Value {
		e, err := sub.BindingPath(bindPath, fork, index)
		if err != nil {
			errs = append(errs, &wrapError{
				innerError: &bindingError{
					Msg: fmt.Sprint("key ", i),
					Err: err,
				},
				loc: sub.getNode().Loc,
			})
		}
		result.Value[i] = e
	}
	return &result, errs.If()
}
func (s *StringExp) BindingPath(bindPath string,
	fork map[MapCallSource]CollectionIndex,
	index []CollectionIndex) (Exp, error) {
	if bindPath != "" {
		return s, &wrapError{
			innerError: &bindingError{
				Msg: "binding within a string",
			},
			loc: s.Node.Loc,
		}
	}
	if len(index) > 0 {
		return s, &wrapError{
			innerError: &bindingError{
				Msg: "indexing into a string",
			},
			loc: s.Node.Loc,
		}
	}
	return s, nil
}
func (s *BoolExp) BindingPath(bindPath string,
	fork map[MapCallSource]CollectionIndex,
	index []CollectionIndex) (Exp, error) {
	if bindPath != "" {
		return s, &wrapError{
			innerError: &bindingError{
				Msg: "binding within a boolean",
			},
			loc: s.Node.Loc,
		}
	}
	if len(index) > 0 {
		return s, &wrapError{
			innerError: &bindingError{
				Msg: "indexing into a bool",
			},
			loc: s.Node.Loc,
		}
	}
	return s, nil
}
func (s *IntExp) BindingPath(bindPath string,
	fork map[MapCallSource]CollectionIndex,
	index []CollectionIndex) (Exp, error) {
	if bindPath != "" {
		return s, &wrapError{
			innerError: &bindingError{
				Msg: "binding within an int",
			},
			loc: s.Node.Loc,
		}
	}
	if len(index) > 0 {
		return s, &wrapError{
			innerError: &bindingError{
				Msg: "indexing into an int",
			},
			loc: s.Node.Loc,
		}
	}
	return s, nil
}
func (s *FloatExp) BindingPath(bindPath string,
	fork map[MapCallSource]CollectionIndex,
	index []CollectionIndex) (Exp, error) {
	if bindPath != "" {
		return s, &wrapError{
			innerError: &bindingError{
				Msg: "binding within a float",
			},
			loc: s.Node.Loc,
		}
	}
	if len(index) > 0 {
		return s, &wrapError{
			innerError: &bindingError{
				Msg: "indexing into a float",
			},
			loc: s.Node.Loc,
		}
	}
	return s, nil
}
func (s *NullExp) BindingPath(string,
	map[MapCallSource]CollectionIndex, []CollectionIndex) (Exp, error) {
	return s, nil
}
func (s *RefExp) BindingPath(bindPath string,
	fork map[MapCallSource]CollectionIndex,
	index []CollectionIndex) (Exp, error) {
	if bindPath == "" {
		s, err := s.updateForks(fork)
		if err != nil {
			return s, err
		}
		if len(index) > 0 {
			r := *s
			r.OutputIndex = append(index, r.OutputIndex...)
			return &r, err
		}
		return s, err
	}
	s, err := s.updateForks(fork)
	if err != nil {
		return s, err
	}
	if s.OutputId != "" {
		var buf strings.Builder
		buf.Grow(len(s.OutputId) + 1 + len(bindPath))
		buf.WriteString(s.OutputId)
		buf.WriteRune('.')
		buf.WriteString(bindPath)
		bindPath = buf.String()
	}
	r := *s
	if len(index) > 0 {
		r.OutputIndex = append(index[:len(index):len(index)], s.OutputIndex...)
	}
	r.OutputId = bindPath
	return &r, nil
}

func (s *ArrayExp) filter(t Type, lookup *TypeLookup) (Exp, error) {
	if _, ok := baseType(t).(*StructType); !ok {
		return s, nil
	}
	if s == nil || len(s.Value) == 0 {
		return s, nil
	}
	if at, ok := t.(*ArrayType); !ok {
		return s, &IncompatibleTypeError{
			Message: t.GetId().str() + " is not an array",
		}
	} else if at.Dim == 1 {
		t = at.Elem
	} else {
		id := t.GetId()
		id.ArrayDim--
		t = lookup.Get(id)
	}
	// handle projection
	result := ArrayExp{
		valExp: valExp{Node: s.Node},
		Value:  make([]Exp, len(s.Value)),
	}
	anyChange := false
	var errs ErrorList
	for i, sub := range s.Value {
		e, err := sub.filter(t, lookup)
		if err != nil {
			errs = append(errs, &wrapError{
				innerError: &bindingError{
					Msg: fmt.Sprint("element ", i),
					Err: err,
				},
				loc: sub.getNode().Loc,
			})
		}
		result.Value[i] = e
		if e != sub {
			anyChange = true
		}
	}
	if !anyChange {
		return s, errs.If()
	}
	return &result, errs.If()
}

func (s *SplitExp) filter(t Type, lookup *TypeLookup) (Exp, error) {
	if _, ok := baseType(t).(*StructType); !ok {
		return s, nil
	}
	if s == nil || s.Value == nil {
		return s, nil
	}
	switch s.CallMode() {
	case ModeArrayCall:
		t = lookup.GetArray(t, 1)
	case ModeMapCall:
		t = lookup.GetMap(t)
	default:
		panic("invalid split type")
	}
	v, err := s.Value.filter(t, lookup)
	if v == s.Value {
		return s, err
	}
	e := *s
	e.Value = v
	return &e, err
}

func (s *MapExp) filter(t Type, lookup *TypeLookup) (Exp, error) {
	if st, ok := t.(*StructType); ok {
		result := MapExp{
			valExp: valExp{Node: s.Node},
			Kind:   KindStruct,
			Value:  make(map[string]Exp, len(s.Value)),
		}
		anyChange := len(st.Members) == len(s.Value)
		var errs ErrorList
		for _, m := range st.Members {
			if t := lookup.Get(m.Tname); t == nil {
				errs = append(errs, &bindingError{
					Msg: "unknown type " + m.Tname.String() + " for member " + m.Id,
				})
			} else if sub := s.Value[m.Id]; sub == nil {
				errs = append(errs, &bindingError{
					Msg: "missing member " + m.Id,
				})
			} else {
				e, err := sub.filter(t, lookup)
				if err != nil {
					errs = append(errs, &wrapError{
						innerError: &bindingError{
							Msg: "field " + m.Id,
							Err: err,
						},
						loc: sub.getNode().Loc,
					})
				}
				result.Value[m.Id] = e
				if e != sub {
					anyChange = true
				}
			}
		}
		if err := errs.If(); err != nil {
			return s, &bindingError{
				Msg: "filtering " +
					FormatExp(s, "          ") +
					"\n          as " + t.GetId().str(),
				Err: err,
			}
		}
		if !anyChange {
			return s, nil
		}
		return &result, nil
	}
	if s == nil || len(s.Value) == 0 || t == &builtinMap {
		return s, nil
	}
	if _, ok := baseType(t).(*StructType); !ok {
		return s, nil
	}
	if at, ok := t.(*TypedMapType); !ok {
		return s, &IncompatibleTypeError{
			Message: fmt.Sprintf("unexpected map expression for %s\n%s",
				t.GetId().str(),
				FormatExp(s, "")),
		}
	} else {
		t = at.Elem
	}
	// handle projection
	result := MapExp{
		valExp: valExp{Node: s.Node},
		Kind:   KindMap,
		Value:  make(map[string]Exp, len(s.Value)),
	}
	anyChange := false
	var errs ErrorList
	for i, sub := range s.Value {
		e, err := sub.filter(t, lookup)
		if err != nil {
			errs = append(errs, &wrapError{
				innerError: &bindingError{
					Msg: fmt.Sprint("element ", i),
					Err: err,
				},
				loc: sub.getNode().Loc,
			})
		}
		result.Value[i] = e
		if e != sub {
			anyChange = true
		}
	}
	if !anyChange {
		return s, errs.If()
	}
	return &result, errs.If()
}
func (s *StringExp) filter(_ Type, _ *TypeLookup) (Exp, error) {
	return s, nil
}
func (s *BoolExp) filter(_ Type, _ *TypeLookup) (Exp, error) {
	return s, nil
}
func (s *IntExp) filter(_ Type, _ *TypeLookup) (Exp, error) {
	return s, nil
}
func (s *FloatExp) filter(_ Type, _ *TypeLookup) (Exp, error) {
	return s, nil
}
func (s *NullExp) filter(_ Type, _ *TypeLookup) (Exp, error) {
	return s, nil
}
func (s *RefExp) filter(_ Type, _ *TypeLookup) (Exp, error) {
	return s, nil
}

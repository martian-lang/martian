// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"fmt"
	"strings"
)

func (s *ArrayExp) BindingPath(bindPath string) (Exp, error) {
	if bindPath == "" || s == nil || len(s.Value) == 0 {
		return s, nil
	}
	// handle projection
	result := ArrayExp{
		valExp: valExp{Node: s.Node},
		Value:  make([]Exp, len(s.Value)),
	}
	var errs ErrorList
	for i, sub := range s.Value {
		e, err := sub.BindingPath(bindPath)
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
	}
	return &result, errs.If()
}
func (s *SweepExp) BindingPath(bindPath string) (Exp, error) {
	if bindPath == "" || s == nil || len(s.Value) == 0 {
		return s, nil
	}
	// handle projection
	result := SweepExp{
		valExp: valExp{Node: s.Node},
		Value:  make([]Exp, len(s.Value)),
	}
	var errs ErrorList
	for i, sub := range s.Value {
		e, err := sub.BindingPath(bindPath)
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
	}
	return &result, errs.If()
}
func (s *MapExp) BindingPath(bindPath string) (Exp, error) {
	if bindPath == "" || s == nil {
		return s, nil
	}
	if s.Kind == KindStruct {
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
		return sub.BindingPath(remainder)
	}
	// handle projection
	result := MapExp{
		valExp: valExp{Node: s.Node},
		Value:  make(map[string]Exp, len(s.Value)),
	}
	var errs ErrorList
	for i, sub := range s.Value {
		e, err := sub.BindingPath(bindPath)
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
func (s *StringExp) BindingPath(bindPath string) (Exp, error) {
	if bindPath != "" {
		return s, &wrapError{
			innerError: &bindingError{
				Msg: "binding within a string",
			},
			loc: s.Node.Loc,
		}
	}
	return s, nil
}
func (s *BoolExp) BindingPath(bindPath string) (Exp, error) {
	if bindPath != "" {
		return s, &wrapError{
			innerError: &bindingError{
				Msg: "binding within a boolean",
			},
			loc: s.Node.Loc,
		}
	}
	return s, nil
}
func (s *IntExp) BindingPath(bindPath string) (Exp, error) {
	if bindPath != "" {
		return s, &wrapError{
			innerError: &bindingError{
				Msg: "binding within an int",
			},
			loc: s.Node.Loc,
		}
	}
	return s, nil
}
func (s *FloatExp) BindingPath(bindPath string) (Exp, error) {
	if bindPath != "" {
		return s, &wrapError{
			innerError: &bindingError{
				Msg: "binding within a float",
			},
			loc: s.Node.Loc,
		}
	}
	return s, nil
}
func (s *NullExp) BindingPath(bindPath string) (Exp, error) {
	return s, nil
}
func (s *RefExp) BindingPath(bindPath string) (Exp, error) {
	if bindPath == "" {
		return s, nil
	}
	if s.OutputId != "" {
		var buf strings.Builder
		buf.Grow(len(s.OutputId) + 1 + len(bindPath))
		buf.WriteString(s.OutputId)
		buf.WriteRune('.')
		buf.WriteString(bindPath)
		bindPath = buf.String()
	}
	return &RefExp{
		Node:     s.Node,
		Kind:     s.Kind,
		Id:       s.Id,
		OutputId: bindPath,
	}, nil
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
			Message: "not an array",
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
func (s *SweepExp) filter(t Type, lookup *TypeLookup) (Exp, error) {
	if _, ok := baseType(t).(*StructType); !ok {
		return s, nil
	}
	if s == nil || len(s.Value) == 0 {
		return s, nil
	}
	// handle projection
	result := SweepExp{
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
func (s *MapExp) filter(t Type, lookup *TypeLookup) (Exp, error) {
	if s.Kind == KindStruct {
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
						Msg: "unknown type " + m.Tname.String(),
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
			if !anyChange {
				return s, errs.If()
			}
			return &result, errs.If()
		}
	}
	if s == nil || len(s.Value) == 0 || t == &builtinMap {
		return s, nil
	}
	if _, ok := baseType(t).(*StructType); !ok {
		return s, nil
	}
	if at, ok := t.(*TypedMapType); !ok {
		return s, &IncompatibleTypeError{
			Message: "unexpected map",
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

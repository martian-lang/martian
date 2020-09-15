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
	return res.Exp.BindingPath(exp.OutputId, exp.Forks, lookup)
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

func (exp *FloatExp) resolveRefs(_, _ map[string]*ResolvedBinding,
	_ *TypeLookup) (Exp, error) {
	return exp, nil
}

func (exp *IntExp) resolveRefs(_, _ map[string]*ResolvedBinding,
	_ *TypeLookup) (Exp, error) {
	return exp, nil
}

func (exp *BoolExp) resolveRefs(_, _ map[string]*ResolvedBinding,
	_ *TypeLookup) (Exp, error) {
	return exp, nil
}
func (exp *StringExp) resolveRefs(_, _ map[string]*ResolvedBinding,
	_ *TypeLookup) (Exp, error) {
	return exp, nil
}

func (exp *NullExp) resolveRefs(_, _ map[string]*ResolvedBinding,
	_ *TypeLookup) (Exp, error) {
	return exp, nil
}

func (s *ArrayExp) BindingPath(bindPath string,
	forks map[*CallStm]CollectionIndex,
	lookup *TypeLookup) (Exp, error) {
	if (bindPath == "" && len(forks) == 0) ||
		s == nil || len(s.Value) == 0 {
		return s, nil
	}
	// handle projection
	result := ArrayExp{
		valExp: valExp{Node: s.Node},
		Value:  make([]Exp, len(s.Value)),
	}
	change := false
	var errs ErrorList
	for i, sub := range s.Value {
		e, err := sub.BindingPath(bindPath, forks, lookup)
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

func (s *MapExp) BindingPath(bindPath string,
	forks map[*CallStm]CollectionIndex,
	lookup *TypeLookup) (Exp, error) {
	if (bindPath == "" && len(forks) == 0) ||
		s == nil || len(s.Value) == 0 {
		return s, nil
	}
	if s.Kind == KindStruct {
		if bindPath != "" {
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
			r, err := sub.BindingPath(remainder, forks, lookup)
			if err != nil {
				err = &wrapError{
					innerError: &bindingError{
						Msg: "struct element " + parts[0],
						Err: err,
					},
					loc: sub.getNode().Loc,
				}
			}
			return r, err
		}
	}
	// handle projection
	result := MapExp{
		valExp: valExp{Node: s.Node},
		Kind:   s.Kind,
		Value:  make(map[string]Exp, len(s.Value)),
	}
	var errs ErrorList
	for i, sub := range s.Value {
		e, err := sub.BindingPath(bindPath, forks, lookup)
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
	_ map[*CallStm]CollectionIndex,
	_ *TypeLookup) (Exp, error) {
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
func (s *BoolExp) BindingPath(bindPath string,
	_ map[*CallStm]CollectionIndex,
	_ *TypeLookup) (Exp, error) {
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
func (s *IntExp) BindingPath(bindPath string,
	_ map[*CallStm]CollectionIndex,
	_ *TypeLookup) (Exp, error) {
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
func (s *FloatExp) BindingPath(bindPath string,
	_ map[*CallStm]CollectionIndex,
	_ *TypeLookup) (Exp, error) {
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
func (s *NullExp) BindingPath(string, map[*CallStm]CollectionIndex,
	*TypeLookup) (Exp, error) {
	return s, nil
}
func (s *RefExp) BindingPath(bindPath string,
	forks map[*CallStm]CollectionIndex,
	_ *TypeLookup) (Exp, error) {
	ref, err := s.updateForks(forks)
	if err != nil {
		err = &bindingError{
			Msg: "binding " + s.GoString(),
			Err: err,
		}
	}
	if bindPath == "" {
		return ref, err
	}
	if ref == s {
		r := *ref
		ref = &r
	}
	if ref.OutputId != "" {
		var buf strings.Builder
		buf.Grow(len(ref.OutputId) + 1 + len(bindPath))
		buf.WriteString(ref.OutputId)
		buf.WriteRune('.')
		buf.WriteString(bindPath)
		bindPath = buf.String()
	}
	ref.OutputId = bindPath
	return ref, nil
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
			Message: t.TypeId().str() + " is not an array, but have " + s.GoString(),
		}
	} else if at.Dim == 1 {
		t = at.Elem
	} else {
		id := t.TypeId()
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
					"\n          as " + t.TypeId().str(),
				Err: err,
			}
		}
		if !anyChange {
			return s, nil
		}
		return &result, nil
	}
	if s == nil || len(s.Value) == 0 {
		return s, nil
	}
	if t == &builtinMap {
		if s.HasRef() {
			refs := s.FindRefs()
			return s, &IncompatibleTypeError{
				Message: fmt.Sprintf(
					"%s cannot be assinged to untyped map: "+
						"contains reference to %s",
					s.Kind,
					refs[0].GoString()),
			}
		}
		return s, nil
	}
	if _, ok := baseType(t).(*StructType); !ok {
		return s, nil
	}
	if at, ok := t.(*TypedMapType); !ok {
		return s, &IncompatibleTypeError{
			Message: fmt.Sprintf("unexpected %s expression for %s\n%s",
				s.Kind,
				t.TypeId().str(),
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

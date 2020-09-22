// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Code for resolving the pipeline graph.

package syntax

import (
	"fmt"
	"sort"
	"strings"
)

type (
	// ResolvedBinding contains information about how a binding gets resolved
	// in a call graph.
	ResolvedBinding struct {
		// If the binding resolved to a literal expression in mro, this returns
		// that expression.  Any parts of the expression are bound to the outputs
		// of a stage, the ID in the resulting *RefExp will be the Fqid of the
		// stage, not the base ID of the stage as it would be for a *RefExp found
		// in an AST.
		Exp Exp `json:"expression"`
		// The type for the binding.  For inputs, this is the expected input
		// type, not the output type of the bound node.
		Type Type `json:"type"`
	}

	// Map of bindings, used for input arguments.  Keys are sorted in JSON
	// output.
	ResolvedBindingMap map[string]*ResolvedBinding

	// BoundReference contains information about a reference with type
	// information.
	BoundReference struct {
		Exp  *RefExp
		Type Type
	}
)

type bindingError struct {
	Msg string
	Err error
}

func (err *bindingError) Error() string {
	if e := err.Err; e == nil {
		return err.Msg
	} else {
		var buf strings.Builder
		buf.Grow(len(err.Msg) + 25)
		err.writeTo(&buf)
		return buf.String()
	}
}

func (err *bindingError) Unwrap() error {
	if err == nil {
		return err
	}
	return err.Err
}

func (err *bindingError) writeTo(w stringWriter) {
	if e := err.Err; e == nil {
		mustWriteString(w, err.Msg)
	} else {
		mustWriteString(w, err.Msg)
		if len(err.Msg) > 20 {
			mustWriteString(w, ":\n\t")
		} else {
			mustWriteString(w, ": ")
		}
		if ew, ok := e.(errorWriter); ok {
			ew.writeTo(w)
		} else {
			mustWriteString(w, e.Error())
		}
	}
}

func (bindings *BindStms) resolve(self, calls map[string]*ResolvedBinding,
	lookup *TypeLookup) (map[string]*ResolvedBinding, error) {
	if len(bindings.List) == 0 {
		return nil, nil
	}
	result := make(map[string]*ResolvedBinding, len(bindings.List))
	var errs ErrorList
	for _, binding := range bindings.List {
		if binding.Id == "*" {
			// Compilation should have expanded this one out.
			continue
		}
		tid := binding.Tname
		r, err := resolveExp(binding.Exp, tid, self, calls, lookup)
		if err != nil {
			errs = append(errs, &bindingError{
				Msg: "BindingError: parameter " + binding.Id,
				Err: err,
			})
		}
		result[binding.Id] = r
	}
	return result, errs.If()
}

func resolveExp(exp Exp, tname TypeId, self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) (*ResolvedBinding, error) {
	t := lookup.Get(tname)
	if t == nil {
		return nil, fmt.Errorf("unknown type " + tname.String())
	}
	rexp, err := exp.resolveRefs(self, siblings, lookup)
	if err != nil {
		return &ResolvedBinding{
			Exp:  exp,
			Type: t,
		}, err
	}
	fexp, err := rexp.filter(t, lookup)
	if err != nil {
		return &ResolvedBinding{
			Exp:  fexp,
			Type: t,
		}, err
	}
	if s, ok := fexp.(*SplitExp); ok {
		if it, err := lookup.AddDim(t, s.Source.CallMode()); err == nil {
			if s.Type != nil && s.Type.IsAssignableFrom(it, lookup) == nil {
				// broader type
				s.Type = it
			} else if ok {
				s.Type = it
			}
		}
	}
	return &ResolvedBinding{
		Exp:  fexp,
		Type: t,
	}, err
}

func bindingType(p string, t Type, lookup *TypeLookup) (Type, error) {
	if p == "" {
		return t, nil
	}
	switch t := t.(type) {
	case *TypedMapType:
		r, err := bindingType(p, t.Elem, lookup)
		if r != nil {
			return lookup.GetMap(r), err
		}
		return r, err
	case *ArrayType:
		r, err := bindingType(p, t.Elem, lookup)
		if r != nil {
			return lookup.GetArray(r, t.Dim), err
		}
		return r, err
	case *StructType:
		element := p
		rest := ""
		if i := strings.IndexRune(p, '.'); i > 0 {
			element = p[:i]
			rest = p[i+1:]
		}
		member := t.Table[element]
		if member == nil {
			return t, &bindingError{
				Msg: "no member " + element + " in " + t.Id,
			}
		}
		return bindingType(rest, lookup.Get(member.Tname), lookup)
	}
	return t, &bindingError{
		Msg: "can't resolve path through " + t.TypeId().str(),
	}
}

func (b *ResolvedBinding) BindingPath(p string,
	forks map[*CallStm]CollectionIndex,
	lookup *TypeLookup) (*ResolvedBinding, error) {
	t, err := bindingType(p, b.Type, lookup)
	if err != nil {
		return b, err
	}
	e, err := b.Exp.BindingPath(p, forks, lookup)
	if err != nil || (e == b.Exp && t == b.Type) {
		return b, err
	}
	return &ResolvedBinding{
		Exp:  e,
		Type: t,
	}, nil
}

func (*BoundReference) KnownLength() bool {
	return false
}

func (*BoundReference) ArrayLength() int {
	return -1
}

func (*BoundReference) Keys() map[string]Exp {
	return nil
}

func (b *BoundReference) CallMode() CallMode {
	switch b.Type.(type) {
	case *ArrayType:
		return ModeArrayCall
	case *TypedMapType:
		return ModeMapCall
	case nullType:
		return ModeNullMapCall
	default:
		return ModeSingleCall
	}
}

func (b *BoundReference) GoString() string {
	if b == nil || b.Exp == nil {
		return "null ref"
	}
	if b.Type == nil {
		return b.Exp.GoString() + " (unknown type)"
	}
	return b.Exp.GoString() + " (" + b.Type.TypeId().str() + ")"
}

// Finds all of the expressions in this binding which are reference expressions,
// with types attached.
//
// This is distsinct from Exp.FindRefs() in that it propagates type information,
// which is relevant if any type conversions are taking place.  However, any
// references inside a type a untyped map will not have complete type
// information.
func (b *ResolvedBinding) FindRefs(lookup *TypeLookup) ([]*BoundReference, error) {
	if !b.Exp.HasRef() {
		return nil, nil
	}
	return b.Exp.FindTypedRefs(nil, b.Type, lookup)
}

func (exp *RefExp) FindTypedRefs(list []*BoundReference,
	t Type, lookup *TypeLookup) ([]*BoundReference, error) {
	return append(list, &BoundReference{
		Exp:  exp,
		Type: t,
	}), nil
}

func (exp *ArrayExp) FindTypedRefs(list []*BoundReference,
	t Type, lookup *TypeLookup) ([]*BoundReference, error) {
	tid := t.TypeId()
	if tid.ArrayDim == 0 {
		return nil, &wrapError{
			innerError: &bindingError{
				Msg: "unexpected array",
			},
			loc: exp.Node.Loc,
		}
	}
	tid.ArrayDim--
	nt := lookup.Get(tid)
	if nt == nil {
		panic("invalid type " + tid.String())
	}
	var errs ErrorList
	if cap(list) == 0 {
		list = make([]*BoundReference, 0, len(exp.Value))
	}
	for _, e := range exp.Value {
		if e == nil || !e.HasRef() {
			continue
		}
		var err error
		list, err = e.FindTypedRefs(list, nt, lookup)
		if err != nil {
			errs = append(errs, &bindingError{
				Msg: "in array",
				Err: err,
			})
		}
	}
	return list, errs.If()
}

func (exp *DisabledExp) FindTypedRefs(list []*BoundReference,
	t Type, lookup *TypeLookup) ([]*BoundReference, error) {
	list, err := exp.Value.FindTypedRefs(list, t, lookup)
	if err != nil {
		return list, err
	}
	return exp.Disabled.FindTypedRefs(list, &builtinBool, lookup)
}

func (exp *MapExp) FindTypedRefs(list []*BoundReference,
	t Type, lookup *TypeLookup) ([]*BoundReference, error) {
	switch t := t.(type) {
	case *TypedMapType:
		var errs ErrorList
		keys := make([]string, 0, len(exp.Value))
		for key, val := range exp.Value {
			if val != nil && val.HasRef() {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		if cap(list) == 0 {
			list = make([]*BoundReference, 0, len(keys))
		}
		for _, key := range keys {
			e := exp.Value[key]
			if e == nil || !e.HasRef() {
				continue
			}
			var err error
			list, err = e.FindTypedRefs(list, t.Elem, lookup)
			if err != nil {
				errs = append(errs, &bindingError{
					Msg: "map key " + key,
					Err: err,
				})
			}
		}
		return list, errs.If()
	case *StructType:
		var errs ErrorList
		for _, m := range t.Members {
			tt := lookup.Get(m.Tname)
			if tt == nil {
				errs = append(errs, &bindingError{
					Msg: "struct field " + m.Id + " has unknown type " + m.Tname.str(),
				})
			} else if val, ok := exp.Value[m.Id]; !ok {
				errs = append(errs, &bindingError{
					Msg: "struct missing field " + m.Id,
				})
			} else if val != nil {
				var err error
				list, err = val.FindTypedRefs(list, tt, lookup)
				if err != nil {
					errs = append(errs, &bindingError{
						Msg: "field " + m.Id,
						Err: err,
					})
				}
			}
		}
		return list, errs.If()
	case *ArrayType:
		return list, &wrapError{
			innerError: &bindingError{
				Msg: "unexpected " + string(exp.Kind) +
					" (expected " + t.TypeId().str() + ")",
			},
			loc: exp.Node.Loc,
		}
	case *BuiltinType:
		if t.Id != KindMap {
			return list, &wrapError{
				innerError: &bindingError{
					Msg: "unexpected " + string(exp.Kind) +
						" (expected " + t.TypeId().str() + ")",
				},
				loc: exp.Node.Loc,
			}
		}
		// Untyped map type.  Generally this can't be allowed.
		refs := exp.FindRefs()
		if cap(list) == 0 {
			list = make([]*BoundReference, 0, len(refs))
		}
		for _, r := range refs {
			if r.OutputId == "" && (r.Kind == KindCall || r.Id == "") {
				// In these cases we know it's a struct so we can tolerate.
				list = append(list, &BoundReference{
					Exp:  r,
					Type: &builtinMap,
				})
			} else {
				return list, &wrapError{
					innerError: &wrapError{
						innerError: &bindingError{
							Msg: "reference " + r.GoString() +
								" cannot be bound inside an untyped map",
						},
						loc: r.Node.Loc,
					},
					loc: exp.Node.Loc,
				}
			}
		}
		return list, nil
	default:
		return list, &wrapError{
			innerError: &bindingError{
				Msg: "unexpected " + string(exp.Kind) +
					" (expected " + t.TypeId().str() + ")",
			},
			loc: exp.Node.Loc,
		}
	}
}

func (exp *valExp) FindTypedRefs(list []*BoundReference, _ Type, _ *TypeLookup) ([]*BoundReference, error) {
	return list, nil
}

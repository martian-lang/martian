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
		mustWriteString(w, ": ")
		if ew, ok := e.(errorWriter); ok {
			ew.writeTo(w)
		} else {
			mustWriteString(w, e.Error())
		}
	}
}

func (bindings *BindStms) resolve(self, calls map[string]*ResolvedBinding,
	lookup *TypeLookup, keepSplit bool) (map[string]*ResolvedBinding, error) {
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
		r, err := resolveExp(binding.Exp, tid, self, calls, lookup, keepSplit)
		if err != nil {
			errs = append(errs, &bindingError{
				Msg: "BindingError: input parameter " + binding.Id,
				Err: err,
			})
		}
		result[binding.Id] = r
	}
	return result, errs.If()
}

func resolveExp(exp Exp, tname TypeId, self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup, keepSplit bool) (*ResolvedBinding, error) {
	t := lookup.Get(tname)
	if t == nil {
		return nil, fmt.Errorf("unknown type " + tname.String())
	}
	rexp, err := exp.resolveRefs(self, siblings, lookup, keepSplit)
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
	return &ResolvedBinding{
		Exp:  fexp,
		Type: t,
	}, err
}

func bindingType(p string, t Type, lookup *TypeLookup) (Type, error) {
	if p == "" {
		return t, nil
	}
	i := strings.IndexRune(p, '.')
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
		if i > 0 {
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
	fork map[MapCallSource]CollectionIndex, index []CollectionIndex,
	lookup *TypeLookup) (*ResolvedBinding, error) {
	t, err := bindingType(p, b.Type, lookup)
	if err != nil {
		return b, err
	}
	e, err := b.Exp.BindingPath(p, fork, index)
	if err != nil || (e == b.Exp && t == b.Type) {
		return b, err
	}
	return &ResolvedBinding{
		Exp:  e,
		Type: t,
	}, nil
}

// Finds all of the expressions in this binding which are reference expressions,
// with types attached.
//
// This is distsinct from Exp.FindRefs() in that it propagates type information,
// which is relevent if any type conversions are taking place.  However, any
// references inside a type a untyped map will not have complete type
// information.
func (b *ResolvedBinding) FindRefs(lookup *TypeLookup) ([]*BoundReference, error) {
	if !b.Exp.HasRef() {
		return nil, nil
	}
	switch exp := b.Exp.(type) {
	case *RefExp:
		return []*BoundReference{{
			Exp:  exp,
			Type: b.Type,
		}}, nil
	case *ArrayExp:
		t := b.Type.TypeId()
		if t.ArrayDim == 0 {
			return nil, &wrapError{
				innerError: &bindingError{
					Msg: "unexpected array",
				},
				loc: exp.Node.Loc,
			}
		}
		t.ArrayDim--
		nt := lookup.Get(t)
		if nt == nil {
			panic("invalid type " + t.String())
		}
		var errs ErrorList
		result := make([]*BoundReference, 0, len(exp.Value))
		for _, e := range exp.Value {
			if e == nil || !e.HasRef() {
				continue
			}
			rb := ResolvedBinding{
				Exp:  e,
				Type: nt,
			}
			if refs, err := rb.FindRefs(lookup); err != nil {
				errs = append(errs, &bindingError{
					Msg: "in array",
					Err: err,
				})
			} else if len(refs) > 0 {
				result = append(result, refs...)
			}
		}
		return result, errs.If()
	case *DisabledExp:
		rb := *b
		rb.Exp = exp.Value
		refs, err := rb.FindRefs(lookup)
		if err != nil {
			return refs, err
		}
		return append(refs, &BoundReference{
			Exp:  exp.Disabled,
			Type: &builtinBool,
		}), nil
	case *SplitExp:
		t := b.Type.TypeId()
		var innerType Type
		switch exp.InnerValue().(type) {
		case *MapExp:
			innerType = lookup.GetMap(b.Type)
		case *ArrayExp:
			innerType = lookup.GetArray(b.Type, 1)
		case *RefExp:
			if t.ArrayDim > 0 {
				t.ArrayDim--
			} else if t.MapDim > 0 {
				t.ArrayDim = t.MapDim - 1
				t.MapDim = 0
			}
			innerType = lookup.Get(t)
		case *NullExp:
			innerType = &builtinNull
		default:
			return nil, &wrapError{
				innerError: &bindingError{
					Msg: "split was not over an array, map, or ref",
				},
				loc: exp.Node.Loc,
			}
		}
		rb := ResolvedBinding{
			Exp:  exp.Value,
			Type: innerType,
		}
		result, err := rb.FindRefs(lookup)
		if err != nil {
			err = &bindingError{
				Msg: "in split",
				Err: err,
			}
		}
		return result, err
	case *MapExp:
		switch t := b.Type.(type) {
		case *TypedMapType:
			if exp.Kind == KindStruct {
				// To avoid special handling of references, pipeline output
				// bindings for mapped calls of pipelines will be structs of
				// maps rather than maps of structs.  But, that means we have
				// to have special handling here instead.
				if t, ok := t.Elem.(*StructType); ok {
					return findStructRefs(lookup, t, exp, false, true)
				}
			}
			var errs ErrorList
			keys := make([]string, 0, len(exp.Value))
			for key, val := range exp.Value {
				if val != nil && val.HasRef() {
					keys = append(keys, key)
				}
			}
			sort.Strings(keys)
			result := make([]*BoundReference, 0, len(keys))
			for _, key := range keys {
				e := exp.Value[key]
				if e == nil || !e.HasRef() {
					continue
				}
				rb := ResolvedBinding{
					Exp:  e,
					Type: t.Elem,
				}
				if refs, err := rb.FindRefs(lookup); err != nil {
					errs = append(errs, &bindingError{
						Msg: "map key " + key,
						Err: err,
					})
				} else if len(refs) > 0 {
					result = append(result, refs...)
				}
			}
			return result, errs.If()
		case *StructType:
			return findStructRefs(lookup, t, exp, false, false)
		case *ArrayType:
			// To avoid special handling of references, pipeline output
			// bindings for mapped calls of pipelines will be structs of
			// arrays rather than arrays of structs.  But, that means we have
			// to have special handling here instead.
			if t, ok := t.Elem.(*StructType); ok {
				return findStructRefs(lookup, t, exp, true, false)
			}
			return nil, &wrapError{
				innerError: &bindingError{
					Msg: "unexpected " + string(exp.Kind) +
						" (expected " + t.TypeId().str() + ")",
				},
				loc: exp.Node.Loc,
			}
		case *BuiltinType:
			if t.Id != KindMap {
				return nil, &wrapError{
					innerError: &bindingError{
						Msg: "unexpected " + string(exp.Kind) +
							" (expected " + t.TypeId().str() + ")",
					},
					loc: exp.Node.Loc,
				}
			}
			// Untyped map type.  Generally this can't be allowed.
			refs := exp.FindRefs()
			result := make([]*BoundReference, 0, len(refs))
			for _, r := range refs {
				if r.OutputId == "" && (r.Kind == KindCall || r.Id == "") {
					// In these cases we know it's a struct so we can tolerate.
					result = append(result, &BoundReference{
						Exp:  r,
						Type: &builtinMap,
					})
				} else {
					return nil, &wrapError{
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
			return result, nil
		default:
			return nil, &wrapError{
				innerError: &bindingError{
					Msg: "unexpected " + string(exp.Kind) +
						" (expected " + t.TypeId().str() + ")",
				},
				loc: exp.Node.Loc,
			}
		}
	default:
		panic(fmt.Sprintf("invalid reference type %T", exp))
	}
}

func findStructRefs(lookup *TypeLookup, t *StructType, exp *MapExp, arr, typedMap bool) ([]*BoundReference, error) {
	var errs ErrorList
	result := make([]*BoundReference, 0, len(t.Members))
	for _, member := range t.Members {
		if v, ok := exp.Value[member.Id]; !ok {
			errs = append(errs, &bindingError{
				Msg: "missing " + member.Id,
			})
		} else if v.HasRef() {
			tn := member.Tname
			if arr {
				tn.ArrayDim++
			}
			if typedMap {
				if tn.MapDim != 0 {
					errs = append(errs, &bindingError{
						Msg: "can't nest map for field " + member.Id + " in a map call",
					})
				} else {
					tn.MapDim = tn.ArrayDim + 1
					tn.ArrayDim = 0
				}
			}
			rb := ResolvedBinding{
				Exp:  v,
				Type: lookup.Get(tn),
			}
			if refs, err := rb.FindRefs(lookup); err != nil {
				errs = append(errs, &bindingError{
					Msg: "struct field " + member.Id,
					Err: err,
				})
			} else if len(refs) > 0 {
				result = append(result, refs...)
			}
		}
	}
	return result, errs.If()
}

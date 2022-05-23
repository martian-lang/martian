//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Methods for resolving bindings at runtime to concrete values.
//

package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

type BindingInfo struct {
	Mode        string        `json:"mode"`
	Node        *string       `json:"node"`
	MatchedFork interface{}   `json:"matchedFork"`
	Value       interface{}   `json:"value"`
	Id          string        `json:"id"`
	Type        syntax.TypeId `json:"type"`
	Waiting     bool          `json:"waiting"`
}

func (node *Node) inputBindingInfo(fork ForkId) []BindingInfo {
	readSize := node.top.rt.FreeMemBytes() / int64(len(node.prenodes)+1)
	result := make([]BindingInfo, 0, len(node.call.ResolvedInputs()))
	for _, input := range node.call.Call().Bindings.List {
		if input.Id == "*" {
			continue
		}
		result = append(result, BindingInfo{
			Id: input.Id,
		})
		r := &result[len(result)-1]
		rb := node.call.ResolvedInputs()[input.Id]
		r.Type = input.Tname
		if refs, err := rb.FindRefs(node.top.types); err != nil {
			panic(err)
		} else if len(refs) == 1 {
			r.Mode = "reference"
			fqname := node.top.allNodes[refs[0].Exp.Id].GetFQName()
			r.Node = &fqname
		}
		ready, val, _ := node.top.resolve(rb.Exp, rb.Type, fork, readSize)
		r.Value = val
		r.Waiting = !ready
	}
	return result
}

func (node *Node) resolveInputs(fork ForkId, keepSplit bool) ([]string, MarshalerMap, error) {
	readSize := node.top.rt.FreeMemBytes() / int64(len(node.prenodes)+1)
	result := make(MarshalerMap, len(node.call.ResolvedInputs()))
	var errs syntax.ErrorList
	var mapped []string
	for k, v := range node.call.ResolvedInputs() {
		_, r, err := node.top.resolve(v.Exp, v.Type, fork, readSize)
		if err != nil {
			if keepSplit {
				var forkErr *forkResolutionError
				if !errors.As(err, &forkErr) {
					if s, ok := v.Exp.(*syntax.SplitExp); ok {
						r, err = node.top.resolveKeepSplit(s, v.Type, fork, readSize)
						mapped = append(mapped, k)
					}
				}
			}
			if err != nil {
				errs = append(errs, &elementError{
					element: "parameter " + k,
					inner:   err,
				})
			}
		}
		result[k] = r
	}
	return mapped, result, errs.If()
}

// Remove the first unmatched split in the expression chain.
func (node *TopNode) resolveKeepSplit(s *syntax.SplitExp, t syntax.Type,
	fork ForkId, readSize int64) (json.Marshaler, error) {
	for _, f := range fork {
		if f.Split.Call == s.Call {
			if f.Split.Source != nil {
				outerT := t
				switch f.Split.Source.CallMode() {
				case syntax.ModeArrayCall:
					outerT = node.types.GetArray(t, 1)
				case syntax.ModeMapCall:
					outerT = node.types.GetMap(t)
				case syntax.ModeSingleCall:
				case syntax.ModeNullMapCall:
					return nil, nil
				default:
					panic("invalid source kind" + f.Split.Source.CallMode().String())
				}
				if outerT == nil {
					tid := t.TypeId()
					panic("invalid type " + tid.String() +
						" for map over " + f.Split.Source.CallMode().String())
				}
				_, r, err := node.resolve(s.Value, outerT, fork, readSize)
				if err != nil {
					err = &elementError{
						element: "resolving matched split",
						inner:   err,
					}
				}
				return r, err
			}
		}
	}
	_, r, err := node.resolve(s.Value, t, fork, readSize)
	if err != nil {
		err = &elementError{
			element: "resolving unmatched split",
			inner:   err,
		}
	}
	return r, err
}

// baseType returns the inner type after stripping off map or array wrappers.
func baseType(t syntax.Type) *syntax.StructType {
	switch t := t.(type) {
	case *syntax.ArrayType:
		return baseType(t.Elem)
	case *syntax.TypedMapType:
		return baseType(t.Elem)
	case *syntax.StructType:
		return t
	}
	tid := t.TypeId()
	panic("Unexpected resolved output type" + tid.String())
}

func (node *Node) outputBindingInfo(fork ForkId) []BindingInfo {
	readSize := node.top.rt.FreeMemBytes() / int64(len(node.prenodes)+1)
	ro := node.call.ResolvedOutputs()
	if ro == nil || ro.Type == nil ||
		ro.Type.TypeId().Tname == syntax.KindNull || ro.Exp == nil {
		return nil
	}
	if _, ok := ro.Exp.(*syntax.NullExp); ok {
		return nil
	}
	members := baseType(ro.Type).Members
	if len(members) > 0 {
		fm := fork.SourceIndexMap()
		result := make([]BindingInfo, len(members))
		for i, member := range members {
			rb, err := ro.BindingPath(member.Id, fm, node.top.types)
			if err == nil {
				result[i].Id = member.Id
				result[i].Type = rb.Type.TypeId()
				if refs, err := rb.FindRefs(node.top.types); err != nil {
					panic(err)
				} else if len(refs) == 1 {
					result[i].Mode = "reference"
					fqname := node.top.allNodes[refs[0].Exp.Id].GetFQName()
					result[i].Node = &fqname
				}
				ready, val, _ := node.top.resolve(rb.Exp, rb.Type, fork, readSize)
				result[i].Value = val
				result[i].Waiting = !ready
			}
		}
		return result
	}
	return nil
}

func (node *Node) resolvePipelineOutputs(fork ForkId) (json.Marshaler, syntax.Type, error) {
	readSize := node.top.rt.FreeMemBytes() / int64(len(node.prenodes)+1)
	ro := node.call.ResolvedOutputs()
	_, r, err := node.top.resolve(ro.Exp, ro.Type, fork, readSize)
	if err != nil {
		tid := ro.Type.TypeId()
		err = &elementError{
			element: node.GetFQName() + " fork " + fork.GoString() + " (" + tid.String() + ")",
			inner:   err,
		}
	}
	return r, ro.Type, err
}

// resolve will either return true and the concrete value represented by the
// expression, false if some of the referred-to nodes are still running, or
// an error if there was a problem resolving a reference.
func (node *TopNode) resolve(binding syntax.Exp, t syntax.Type,
	fork ForkId,
	readSize int64) (bool, json.Marshaler, error) {
	if binding == nil {
		return true, nil, nil
	}
	if !binding.HasRef() && !binding.HasSplit() {
		return true, binding, nil
	}
	if len(fork) > 0 {
		br := syntax.ResolvedBinding{
			Exp:  binding,
			Type: t,
		}
		if b, err := br.BindingPath("", fork.SourceIndexMap(), node.Types()); err != nil {
			return true, nil, &elementError{
				element: "resolving binding",
				inner:   err,
			}
		} else if binding != b.Exp {
			binding = b.Exp
			if !binding.HasRef() && !binding.HasSplit() {
				return true, binding, nil
			}
			t = b.Type
		}
	}
	switch binding := binding.(type) {
	case *syntax.RefExp:
		return node.resolveRef(binding, t, fork, readSize)
	case *syntax.ArrayExp:
		return node.resolveArray(binding, t, fork, readSize)
	case *syntax.MapExp:
		return node.resolveMap(binding, t, fork, readSize)
	case *syntax.SplitExp:
		return node.resolveSplit(binding, t, fork, readSize)
	case *syntax.MergeExp:
		return node.resolveMerge(binding, t, fork, readSize)
	case *syntax.DisabledExp:
		return node.resolveDisabledExp(binding, t, fork, readSize)
	default:
		tid := t.TypeId()
		panic(fmt.Sprintf("unexpected ref or sweep type %T, wanted %s",
			binding, tid.String()))
	}
}

func (node *TopNode) resolveDisabledExp(binding *syntax.DisabledExp, t syntax.Type,
	fork ForkId, readSize int64) (bool, json.Marshaler, error) {
	ready, disabled, err := node.resolve(binding.Disabled,
		node.Types().Get(syntax.TypeId{Tname: syntax.KindBool}), fork, readSize)
	if err != nil {
		return ready, nil, &elementError{
			element: "disabled binding",
			inner:   err,
		}
	} else if !ready {
		return ready, nil, nil
	}
	switch disabled := disabled.(type) {
	case *syntax.BoolExp:
		if disabled.Value {
			return true, nil, nil
		}
	case json.RawMessage:
		var db bool
		if err := json.Unmarshal(disabled, &db); err != nil {
			return true, nil, &elementError{
				element: "disabled binding",
				inner:   err,
			}
		}
		if db {
			return true, nil, nil
		}
	default:
		return true, nil, &elementError{
			element: fmt.Sprintf(
				"invalid type %T for disabled binding from %s",
				disabled, binding.Disabled.GoString()),
		}
	}
	ready, result, err := node.resolve(binding.Value, t, fork, readSize)
	if err != nil {
		return ready, result, &elementError{
			element: "enabled value",
			inner:   err,
		}
	}
	return ready, result, err
}

func (node *TopNode) resolveMerge(binding *syntax.MergeExp, t syntax.Type,
	fork ForkId, readSize int64) (bool, json.Marshaler, error) {
	var innerT syntax.Type
	switch t := t.(type) {
	case *syntax.ArrayType:
		if binding.MergeOver.CallMode() != syntax.ModeArrayCall {
			return false, nil, fmt.Errorf(
				"expected array but mapping is %s",
				binding.MergeOver.CallMode().String())
		}
		if t.Dim > 1 {
			nt := *t
			nt.Dim--
			innerT = &nt
		} else {
			innerT = t.Elem
		}
	case *syntax.TypedMapType:
		if binding.MergeOver.CallMode() != syntax.ModeMapCall {
			tid := t.TypeId()
			return false, nil, fmt.Errorf(
				"expected %s but mapping is %s",
				tid.String(),
				binding.MergeOver.CallMode().String())
		}
		innerT = t.Elem
	default:
		tid := t.TypeId()
		panic("invalid type for " + binding.GoString() +
			" for " + binding.Call.GetFqid() + ": " + tid.String())
	}
	for _, part := range fork {
		if part.Split.Call == binding.GetCall() {
			return node.resolve(binding.Value, innerT, fork, readSize)
		}
	}
	var forkRefId string
	if callRef := binding.ForkNode; callRef != nil {
		forkRefId = callRef.Id
	} else {
		forkRefId = binding.Call.GetFqid()
	}
	parts, errs := node.getParts(binding.GetCall(), fork, forkRefId)
	if err := errs.If(); err != nil {
		util.PrintError(err, "runtime",
			"Resolving parts for %s.  This will likely result in further errors.",
			binding.GetCall().GoString())
	}
	allReady := true
	switch binding.CallMode() {
	case syntax.ModeMapCall:
		if len(parts) == 1 {
			if _, ok := parts[0].Id.(emptyFork); ok {
				err := errs.If()
				if err != nil {
					err = &elementError{
						element: "merge of map " + binding.Call.GetFqid(),
						inner:   err,
					}
				}
				return true, MarshalerMap{}, err
			}
		}
		result := make(MarshalerMap, len(parts))
		for _, part := range parts {
			if part.Id.IndexSource() != nil {
				errs = append(errs, &elementError{
					element: "unresolved fork source for mapped " +
						part.Split.Call.GoString(),
				})
				continue
			}
			switch part.Id.Mode() {
			case syntax.ModeArrayCall:
				return true, result, fmt.Errorf("unexpected array fork")
			case syntax.ModeMapCall:
				ready, v, err := node.resolve(binding.Value,
					innerT,
					append(fork, part), readSize)
				if err != nil {
					errs = append(errs, &elementError{
						element: "key " + part.Id.GoString(),
						inner:   err,
					})
				}
				allReady = allReady && ready
				result[part.Id.MapKey()] = v
			}
		}
		err := errs.If()
		if err != nil {
			err = &elementError{
				element: "merge of map " + binding.Call.GetFqid(),
				inner:   err,
			}
		}
		return allReady, result, err
	case syntax.ModeNullMapCall:
		return true, nil, nil
	case syntax.ModeArrayCall:
		if len(parts) == 1 {
			if _, ok := parts[0].Id.(emptyFork); ok {
				err := errs.If()
				if err != nil {
					err = &elementError{
						element: "merge of array " + binding.Call.GetFqid(),
						inner:   err,
					}
				}
				return true, marshallerArray{}, err
			}
		}
		result := make(marshallerArray, len(parts))
		for i, part := range parts {
			if part.Id.IndexSource() != nil {
				errs = append(errs, &elementError{
					element: "unresolved fork for array-mapped " +
						part.Split.Call.GoString(),
				})
				continue
			}
			switch part.Id.Mode() {
			case syntax.ModeMapCall:
				return true, result, fmt.Errorf("unexpected map fork")
			case syntax.ModeArrayCall:
				ready, v, err := node.resolve(binding.Value,
					innerT,
					append(fork, part), readSize)
				if err != nil {
					errs = append(errs, &elementError{
						element: "index " + part.Id.GoString(),
						inner:   err,
					})
				}
				allReady = allReady && ready
				result[i] = v
			}
		}
		err := errs.If()
		if err != nil {
			err = &elementError{
				element: "merge of array " + binding.Call.GetFqid(),
				inner:   err,
			}
		}
		return allReady, result, err
	}
	panic("invalid mapping mode")
}

// getParts returns the ForkSourcePart corresponding the the given call for
// every fork of the given node which matches the given fork ID.
func (node *TopNode) getParts(src *syntax.CallStm,
	forkId ForkId,
	id string) ([]*ForkSourcePart, syntax.ErrorList) {
	boundNode := node.allNodes[id]
	if boundNode == nil {
		panic("unknown bound node - this should not be possible in properly-compiled code")
	}
	boundNode.expandForks(true)
	var errs syntax.ErrorList
	parts := boundNode.forkIds.Table[src]
	if len(parts) == 1 && parts[0].Id.IndexSource() != nil &&
		(parts[0].Range == nil || parts[0].Range.Length() >= 0) {
		matchingParts := make([]*ForkSourcePart, 0, len(boundNode.forks))
		for _, fork := range boundNode.forks {
			if p, err := fork.forkId.matchPart(parts[0].Split.Call); err != nil {
				if parts[0].Split.Call == src {
					errs = append(errs, &forkResolutionError{
						Msg: "circular fork sources",
					})
				} else {
					errs = append(errs, &elementError{
						element: "unmatched call " + parts[0].Split.Call.GoString(),
						inner:   err,
					})
				}
			} else if fork.forkId.Matches(forkId) {
				matchingParts = append(matchingParts, p)
			}
		}
		parts = matchingParts
	} else if len(parts) == 0 && src.KnownLength() {
		switch src.CallMode() {
		case syntax.ModeArrayCall:
			if src.ArrayLength() > 0 {
				parts = make([]*ForkSourcePart, src.ArrayLength())
				partStore := make([]ForkSourcePart, src.ArrayLength())
				for i := range parts {
					partStore[i] = ForkSourcePart{
						Split: &syntax.SplitExp{
							Value:  &syntax.MergeExp{MergeOver: src},
							Source: src,
							Call:   src,
						},
						Id: arrayIndexFork(i),
					}
					parts[i] = &partStore[i]
				}
			}
		case syntax.ModeMapCall:
			if keyMap := src.Keys(); len(keyMap) > 0 {
				keys := make([]string, 0, len(keyMap))
				for k := range keyMap {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				partStore := make([]ForkSourcePart, len(keys))
				parts = make([]*ForkSourcePart, len(keys))
				for i, k := range keys {
					partStore[i] = ForkSourcePart{
						Split: &syntax.SplitExp{
							Value:  &syntax.MergeExp{MergeOver: src},
							Source: src,
							Call:   src,
						},
						Id: mapKeyFork(k),
					}
					parts[i] = &partStore[i]
				}
			}
		}
	} else if len(boundNode.forks[0].forkId) > 1 {
		id := make(ForkId, len(forkId), len(forkId)+1)
		copy(id, forkId)
		allow := func(part *ForkSourcePart) bool {
			for _, fork := range boundNode.forks {
				if fork.forkId.Matches(forkId) {
					if p, err := fork.forkId.matchPart(part.Split.Call); err != nil {
						errs = append(errs, &elementError{
							element: "unmatched fork source " + part.Split.Call.GoString(),
							inner:   err,
						})
					} else if indexEqual(p.Id, part.Id) {
						return true
					}
				}
			}
			return false
		}
		for len(parts) > 0 && !allow(parts[0]) {
			parts = parts[1:]
		}
		for len(parts) > 1 && !allow(parts[len(parts)-1]) {
			parts = parts[:len(parts)-1]
		}
		alloc := false
		for i := len(parts) - 1; i > 1; i-- {
			if !alloc {
				parts = append(parts[:i:i], parts[i+1:]...)
				alloc = true
			} else {
				parts = append(parts[:i], parts[i+1:]...)
			}
		}
	} else if len(parts) == 0 {
		return forkId.getUnmatchedForkParts(boundNode), errs
	}
	return parts, errs
}

func (node *TopNode) resolveSplit(binding *syntax.SplitExp, t syntax.Type,
	fork ForkId, readSize int64) (bool, json.Marshaler, error) {
	// If the split is due to a common ancesstor pipeline splitting, use
	// that.
	var innerT syntax.Type
	switch binding.Source.CallMode() {
	case syntax.ModeArrayCall:
		innerT = node.types.GetArray(t, 1)
	case syntax.ModeMapCall:
		innerT = node.types.GetMap(t)
	}
	for i, part := range fork {
		if part.Split.Call == binding.Call || i == len(fork)-1 {
			ready, result, err := node.resolve(
				binding.Value,
				innerT, fork, readSize)
			if err != nil {
				return ready, result, &elementError{
					element: "split " + binding.Source.CallMode().String(),
					inner:   err,
				}
			}
			if !ready {
				return ready, result, nil
			}
			if ref, ok := binding.Value.(*syntax.RefExp); ok {
				// RefExp resolution has already done the element extraction.
				for c := range ref.Forks {
					if c == binding.Call {
						return ready, result, nil
					}
				}
			}
			e, err := getElement(result, part.Id)
			if err != nil {
				err = &elementError{
					element: "splitting " + binding.Source.CallMode().String() +
						" " + binding.Value.GoString() + " with fork part " +
						part.Split.Call.GoString(),
					inner: err,
				}
			}
			return ready, e, err
		}
	}
	// This fork's split is unresolved.
	ready, result, err := node.resolve(
		binding.Value,
		innerT, fork, readSize)
	if err != nil {
		return ready, result, &elementError{
			element: "self-split " + binding.Source.CallMode().String(),
			inner:   err,
		}
	}
	if !ready {
		return ready, result, nil
	}
	if result != binding.Value {
		var r syntax.Exp
		switch rr := result.(type) {
		case syntax.Exp:
			r = rr
		default:
			var parser syntax.Parser
			r, err = convertToExp(&parser, false, result, innerT.TypeId(), node.types)
			if err != nil {
				return ready, binding, &elementError{
					element: "unresolved split value",
					inner:   err,
				}
			}
		}
		b := *binding
		b.Value = r
		return ready, &b, nil
	}
	return ready, binding, nil
}

func getElement(result json.Marshaler,
	element syntax.CollectionIndex) (json.Marshaler, error) {
	switch element.Mode() {
	case syntax.ModeArrayCall:
		if element.IndexSource() != nil {
			m, err := getArrayElement(result, 0)
			return m, &elementError{
				element: "unknown index",
				inner:   err,
			}
		}
		return getArrayElement(result, element.ArrayIndex())
	case syntax.ModeMapCall:
		if element.IndexSource() != nil {
			m, err := getMapElement(result, "")
			return m, &elementError{
				element: "unknown key",
				inner:   err,
			}
		}
		return getMapElement(result, element.MapKey())
	default:
		if element.IndexSource() != nil {
			return nil, &forkResolutionError{
				Msg: "unresolved index for map call of unknown type",
			}
		}
		return nil, &elementError{
			element: "unknown element type " + element.Mode().String() +
				" (" + element.GoString() + ")",
		}
	}
}

func getArrayElement(array json.Marshaler, i int) (json.Marshaler, error) {
	if i < 0 {
		return nil, &elementError{
			element: "array index " + strconv.Itoa(i),
		}
	}
	switch array := array.(type) {
	case *syntax.NullExp:
		return array, nil
	case json.RawMessage:
		var r marshallerArray
		if err := r.UnmarshalJSON(array); err != nil {
			return array, &elementError{
				element: "array index " + strconv.Itoa(i),
				inner:   err,
			}
		}
		if i >= len(r) {
			return nil, &elementError{
				element: "array index " + strconv.Itoa(i) +
					" >= " + strconv.Itoa(len(r)),
			}
		}
		return r[i], nil
	case marshallerArray:
		if i >= len(array) {
			return nil, &elementError{
				element: "array index " + strconv.Itoa(i) +
					" >= " + strconv.Itoa(len(array)),
			}
		}
		return array[i], nil
	case *syntax.ArrayExp:
		if i >= len(array.Value) {
			return nil, &elementError{
				element: "array index " + strconv.Itoa(i) +
					" >= " + strconv.Itoa(len(array.Value)),
			}
		}
		return array.Value[i], nil
	default:
		return nil, &syntax.IncompatibleTypeError{
			Message: fmt.Sprintf("can't take array element from %T", array),
		}
	}
}

func getMapElement(m json.Marshaler, k string) (json.Marshaler, error) {
	switch m := m.(type) {
	case *syntax.NullExp:
		return m, nil
	case json.RawMessage:
		var r LazyArgumentMap
		if err := json.Unmarshal(m, &r); err != nil {
			return m, &elementError{
				element: "map key " + k,
				inner:   err,
			}
		}
		v, ok := r[k]
		if !ok {
			return v, &elementError{
				element: "map key " + k + " not found",
				inner:   missingValueError,
			}
		}
		return v, nil
	case LazyArgumentMap:
		v, ok := m[k]
		if !ok {
			return v, &elementError{
				element: "map key " + k + " not found",
				inner:   missingValueError,
			}
		}
		return v, nil
	case MarshalerMap:
		v, ok := m[k]
		if !ok {
			return v, &elementError{
				element: "map key " + k + " not found",
				inner:   missingValueError,
			}
		}
		return v, nil
	case *syntax.MapExp:
		v, ok := m.Value[k]
		if !ok {
			return v, &elementError{
				element: "map key " + k + " not found",
				inner:   missingValueError,
			}
		}
		return v, nil
	default:
		return nil, &syntax.IncompatibleTypeError{
			Message: fmt.Sprintf("can't take map element from %T", m),
		}
	}
}

func (node *Node) matchFork(ref map[*syntax.CallStm]syntax.CollectionIndex,
	fork ForkId) (*Fork, error) {
	if len(node.forkRoots) == 0 {
		return node.forks[0], nil
	}
	matchedFork, err := fork.Match(ref, node.forkRoots)
	if err != nil {
		return nil, err
	}
	for i, id := range node.forkIds.List {
		if id.Matches(matchedFork) {
			return node.forks[i], nil
		}
	}
	return nil, fmt.Errorf(
		"unresolved fork %s (from %s): no matches out of %d possible forks",
		matchedFork.GoString(), fork.GoString(), len(node.forkRoots))
}

// Find all of the forks for which the given fork could match a more-constrained
// version of the ID.
func (node *Node) matchForks(fork ForkId) []*Fork {
	if len(node.forkRoots) == 0 {
		return node.forks
	}
	match := func(i int, f ForkId, forks []*Fork) bool {
		if len(forks) > i {
			return f.Matches(forks[i].forkId)
		}
		return false
	}
	forks := node.forks
	// Trim the matched parts off of the front.
	for len(forks) > 0 && !match(0, fork, forks) {
		forks = forks[1:]
	}
	// Trim the matched parts off the end.
	for len(forks) > 1 && !match(len(forks)-1, fork, forks) {
		forks = forks[:len(forks)-1]
	}
	if len(forks) <= 2 {
		return forks
	}
	result := forks[:1]
	if cap(result) != cap(forks) {
		panic(cap(result))
	}
	for i := 1; i < len(forks)-1; i++ {
		if match(i, fork, forks) {
			// if we haven't reallocated, this is a no-op.
			result = append(result, forks[i])
		} else if cap(result) == cap(forks) {
			// Need to allocate a new buffer.  It's capacity will be sufficient
			// to hold all but one element of upstream.  We know we're
			// skipping at least one, so it won't ever need to grow, and being
			// smaller than the length of upstream means the capacity will no
			// longer match either.
			result = make([]*Fork, i-1, len(forks)-1)
			copy(result, forks[:i])
		}
	}
	return append(result, forks[len(forks)-1])
}

func (node *TopNode) resolveRef(binding *syntax.RefExp, t syntax.Type,
	fork ForkId,
	readSize int64) (bool, json.Marshaler, error) {
	boundNode := node.allNodes[binding.Id]
	if boundNode == nil {
		panic("unknown bound node - this should not be possible in properly-compiled code")
	}
	return boundNode.resolveRef(binding, t, fork, readSize)
}

func (boundNode *Node) resolveRef(binding *syntax.RefExp, t syntax.Type,
	fork ForkId,
	readSize int64) (bool, json.Marshaler, error) {
	f, err := boundNode.matchFork(binding.Forks, fork)
	if err != nil {
		return true, nil, &forkResolutionError{
			Msg: "could not match reference to a specific fork of " +
				boundNode.GetFQName(),
			Err: err,
		}
	}
	return f.resolveRef(binding, t,
		boundNode.call.Call().DecId, readSize)
}

func (f *Fork) resolveRef(binding *syntax.RefExp, t syntax.Type,
	decId string,
	readSize int64) (bool, json.Marshaler, error) {
	args, err := f.metadata.read(OutsFile, readSize)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil, nil
		}
		return true, nil, &elementError{
			element: "stage " + binding.Id,
			inner:   err,
		}
	}
	types := f.node.top.types
	value, err := args.Path(binding.OutputId,
		types.Get(syntax.TypeId{
			Tname: decId,
		}), t, types)
	if err != nil {
		err = &elementError{
			element: "stage " + binding.Id,
			inner:   err,
		}
	}
	return true, value, err
}

type elementError struct {
	element string
	inner   error
}

func (err *elementError) Error() string {
	if err.inner == nil {
		return err.element
	}
	suffix := err.inner.Error()
	if strings.IndexRune(suffix, '\n')+len(err.element) > 40 {
		return err.element + ":\n\t" + suffix
	}
	return err.element + ": " + suffix
}

func (err *elementError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.inner
}

type forkResolutionError struct {
	Msg string
	Err error
}

func (err *forkResolutionError) Error() string {
	if err.Err == nil {
		return err.Msg
	}
	suffix := err.Err.Error()
	if strings.IndexRune(suffix, '\n')+len(err.Msg) > 40 {
		return err.Msg + ":\n\t" + suffix
	}
	return err.Msg + ": " + suffix
}

func (err *forkResolutionError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Err
}

var missingValueError = errors.New("key was not present")

// Path looks up map keys, separated by '.' characters, and projecting through
// typed maps and arrays.
func (args LazyArgumentMap) Path(p string, source, dest syntax.Type,
	lookup *syntax.TypeLookup) (json.Marshaler, error) {
	if args == nil {
		return nil, nil
	}
	if p == "" {
		if dest == nil {
			return args, nil
		}
		return args.filter(dest, lookup)
	}
	switch t := source.(type) {
	case *syntax.TypedMapType:
		if d, ok := dest.(*syntax.TypedMapType); ok {
			dest = d.Elem
		}
		result := make(MarshalerMap, len(args))
		var errs syntax.ErrorList
		for k, v := range args {
			elem, err := resolvePath(v, p, t.Elem, dest, lookup)
			result[k] = elem
			if err != nil {
				errs = append(errs, &elementError{
					element: "key " + k,
					inner:   err,
				})
			}
		}
		return result, errs.If()
	case *syntax.StructType:
		key := p
		indexDot := strings.IndexRune(p, '.')
		if indexDot > 0 {
			key = p[:indexDot]
		}
		if v, ok := args[key]; !ok {
			return nil, &elementError{
				element: key,
				inner:   missingValueError,
			}
		} else {
			member := t.Table[key]
			if member == nil {
				panic(fmt.Sprintf("invalid binding path within %s to member %s",
					t.Id, key))
			}
			mt := lookup.Get(member.Tname)
			if mt == nil {
				panic("unknown element type for " + member.Tname.String())
			}
			m, err := func(v json.RawMessage, indexDot int, p string,
				mt, dest syntax.Type, lookup *syntax.TypeLookup) (json.Marshaler, error) {
				if indexDot > 0 {
					return resolvePath(v, p[indexDot+1:], mt, dest, lookup)
				} else {
					if dest == nil {
						dest = mt
					}
					m, fatal, err := dest.FilterJson(v, lookup)
					if fatal {
						if err != nil {
							err = &syntax.IncompatibleTypeError{
								Message: "cannot filter " + mt.String() + " to " + dest.String(),
								Reason:  err,
							}
						}
						return m, err
					} else {
						return m, nil
					}
				}
			}(v, indexDot, p, mt, dest, lookup)
			if err != nil {
				return m, &elementError{
					element: "field " + key,
					inner:   err,
				}
			}
			return m, nil
		}
	default:
		if strings.ContainsRune(p, '.') {
			panic(fmt.Sprintf("invalid type %T for binding path %s", t, p))
		}
		if v, ok := args[p]; !ok {
			return nil, &elementError{
				element: p,
				inner:   missingValueError,
			}
		} else {
			var builder strings.Builder
			return v, t.IsValidJson(v, &builder, lookup)
		}
	}
}

// jsonPath works like Path except it doesn't use type information.
func (m LazyArgumentMap) jsonPath(p string) json.Marshaler {
	if p == "" {
		return m
	}
	if i := strings.IndexRune(p, '.'); i < 0 {
		return m[p]
	} else {
		return jsonPath(m[p[:i]], p[i+1:])
	}
}

func jsonPath(msg json.RawMessage, p string) json.Marshaler {
	if p == "" {
		return msg
	}
	msg = json.RawMessage(bytes.TrimSpace(msg))
	if len(msg) == 0 || bytes.Equal(msg, nullBytes) {
		return msg
	}
	switch msg[0] {
	case '{':
		var m LazyArgumentMap
		if json.Unmarshal(msg, &m) != nil {
			return msg
		}
		return m.jsonPath(p)
	case '[':
		var arr []json.RawMessage
		if json.Unmarshal(msg, &arr) != nil {
			return msg
		}
		result := make(marshallerArray, len(arr))
		for i, v := range arr {
			result[i] = jsonPath(v, p)
		}
		return result
	default:
		return msg
	}
}

func (args LazyArgumentMap) filter(t syntax.Type,
	lookup *syntax.TypeLookup) (json.Marshaler, error) {
	if !t.CanFilter() {
		return args, nil
	}
	switch t := t.(type) {
	case *syntax.TypedMapType:
		var errs syntax.ErrorList
		result := make(MarshalerMap, len(args))
		for k, v := range args {
			b, _, err := t.Elem.FilterJson(v, lookup)
			if err != nil {
				errs = append(errs, &elementError{
					element: "key " + k,
					inner:   err,
				})
			}
			result[k] = b
		}
		return result, errs.If()
	case *syntax.StructType:
		var errs syntax.ErrorList
		result := make(MarshalerMap, len(t.Members))
		for _, member := range t.Members {
			et := lookup.Get(member.Tname)
			b, _, err := et.FilterJson(args[member.Id], lookup)
			if err != nil {
				errs = append(errs, &elementError{
					element: "key " + member.Id,
					inner:   err,
				})
			}
			result[member.Id] = b
		}
		return result, errs.If()
	default:
		panic(fmt.Sprintf("incorrect type %T for map filtering", t))
	}
}

func resolvePath(b json.RawMessage, p string,
	t, dest syntax.Type, lookup *syntax.TypeLookup) (json.Marshaler, error) {
	if bytes.Equal(b, nullBytes) {
		return nil, nil
	}
	if p == "" {
		v, _, err := t.FilterJson(b, lookup)
		return v, err
	}
	switch t := t.(type) {
	case *syntax.ArrayType:
		var arr []json.RawMessage
		if err := json.Unmarshal(b, &arr); err != nil {
			return nil, err
		}
		if dest != nil {
			dest = lookup.GetArray(dest, -1)
		}
		et := lookup.GetArray(t, -1)
		result := make(marshallerArray, 0, len(arr))
		var errs syntax.ErrorList
		for i, v := range arr {
			elem, err := resolvePath(v, p, et, dest, lookup)
			if err != nil {
				errs = append(errs, &elementError{
					element: fmt.Sprint("array index ", i),
					inner:   err,
				})
			}
			result = append(result, elem)
		}
		return result, errs.If()
	case *syntax.TypedMapType, *syntax.StructType:
		var args LazyArgumentMap
		if err := json.Unmarshal(b, &args); err != nil {
			return nil, err
		}
		return args.Path(p, t, dest, lookup)
	default:
		panic(fmt.Sprintf("invalid path through %T", t))
	}
}

type marshallerArray []json.Marshaler

func (arr marshallerArray) MarshalJSON() ([]byte, error) {
	if arr == nil {
		return nullBytes[:len(nullBytes):len(nullBytes)], nil
	} else if len(arr) == 0 {
		return []byte(`[]`), nil
	}
	var buf bytes.Buffer
	buf.Grow(2 + len(arr)*5)
	err := arr.encodeJSON(&buf)
	return buf.Bytes(), err
}

func (arr marshallerArray) EncodeJSON(buf *bytes.Buffer) error {
	if arr == nil {
		_, err := buf.Write(nullBytes)
		return err
	} else if len(arr) == 0 {
		_, err := buf.WriteString(`[]`)
		return err
	}
	return arr.encodeJSON(buf)
}

func (arr marshallerArray) encodeJSON(buf *bytes.Buffer) error {
	if _, err := buf.WriteRune('['); err != nil {
		return err
	}
	for i, elem := range arr {
		if i != 0 {
			if _, err := buf.WriteRune(','); err != nil {
				return err
			}
		}
		if elem == nil {
			if _, err := buf.WriteString("null"); err != nil {
				return err
			}
		} else {
			switch elem := elem.(type) {
			case syntax.JsonWriter:
				if err := elem.EncodeJSON(buf); err != nil {
					return err
				}
			case json.RawMessage:
				if _, err := buf.Write(elem); err != nil {
					return err
				}
			default:
				if b, err := elem.MarshalJSON(); err != nil {
					return err
				} else if _, err := buf.Write(b); err != nil {
					return err
				}
			}
		}
	}
	_, err := buf.WriteRune(']')
	return err
}

func (arr *marshallerArray) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, nullBytes) {
		*arr = nil
		return nil
	}
	var ra []json.RawMessage
	if err := json.Unmarshal(b, &ra); err != nil {
		return err
	}
	*arr = make(marshallerArray, len(ra))
	for i, v := range ra {
		(*arr)[i] = v
	}
	return nil
}

// CallMode Returns the call mode for a call which depends on this source.
func (arr marshallerArray) CallMode() syntax.CallMode {
	return syntax.ModeArrayCall
}

// KnownLength returns true if the source is an array with a known length
// or is a map with a known set of keys.
func (arr marshallerArray) KnownLength() bool {
	return true
}

// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
// the length of the array referred to by this source.  Otherwise it will
// return -1.
func (arr marshallerArray) ArrayLength() int {
	return len(arr)
}

// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
// a map[string]Exp with the same keys which any call mapping over this
// source would have.  The values are arbitrary.  Otherwise, it will return
// nil.
func (arr marshallerArray) Keys() map[string]syntax.Exp {
	return nil
}

func (arr marshallerArray) GoString() string {
	var buf strings.Builder
	if err := buf.WriteByte('['); err != nil {
		panic(err)
	}
	for i, v := range arr {
		if i != 0 {
			if err := buf.WriteByte(','); err != nil {
				panic(err)
			}
		}
		if _, err := fmt.Fprint(&buf, v); err != nil {
			panic(err)
		}
	}
	if err := buf.WriteByte(']'); err != nil {
		panic(err)
	}
	return buf.String()
}

func (node *TopNode) resolveArray(binding *syntax.ArrayExp, t syntax.Type,
	fork ForkId,
	readSize int64) (bool, json.Marshaler, error) {
	result := make(marshallerArray, len(binding.Value))
	if at, ok := t.(*syntax.ArrayType); !ok {
		id := t.TypeId()
		return true, nil, &syntax.IncompatibleTypeError{
			Message: id.String() + " is not an array type",
		}
	} else {
		t = node.types.GetArray(at.Elem, at.Dim-1)
	}
	allReady := true
	var errs syntax.ErrorList
	for i, exp := range binding.Value {
		if ready, v, err := node.resolve(exp, t,
			fork, readSize); err != nil {
			allReady = ready && allReady
			errs = append(errs, &elementError{
				element: "array index " + strconv.Itoa(i),
				inner:   err,
			})
		} else if !ready {
			allReady = false
		} else {
			result[i] = v
		}
	}
	return allReady, result, errs.If()
}

func (node *TopNode) resolveMap(binding *syntax.MapExp, t syntax.Type,
	fork ForkId,
	readSize int64) (bool, json.Marshaler, error) {
	result := make(MarshalerMap, len(binding.Value))
	allReady := true
	var errs syntax.ErrorList
	switch t := t.(type) {
	case *syntax.TypedMapType:
		for key, exp := range binding.Value {
			if ready, v, err := node.resolve(exp, t.Elem,
				fork, readSize); err != nil {
				allReady = ready && allReady
				errs = append(errs, &elementError{
					element: "key " + key,
					inner:   err,
				})
			} else if !ready {
				allReady = false
				result[key] = nil
			} else {
				result[key] = v
			}
		}
	case *syntax.StructType:
		for _, member := range t.Members {
			if ready, v, err := node.resolve(
				binding.Value[member.Id],
				node.types.Get(member.Tname),
				fork,
				readSize); err != nil {
				allReady = ready && allReady
				errs = append(errs, &elementError{
					element: "field " + member.Id,
					inner:   err,
				})
				result[member.Id] = nil
			} else if !ready {
				allReady = false
				result[member.Id] = nil
			} else {
				result[member.Id] = v
			}
		}
	default:
		return true, nil, &syntax.IncompatibleTypeError{
			Message: fmt.Sprintf("not a map type %T", t),
		}
	}
	return allReady, result, errs.If()
}

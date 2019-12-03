//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
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
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
)

type BindingInfo struct {
	Id          string        `json:"id"`
	Type        syntax.TypeId `json:"type"`
	Mode        string        `json:"mode"`
	Node        *string       `json:"node"`
	MatchedFork interface{}   `json:"matchedFork"`
	Value       interface{}   `json:"value"`
	Waiting     bool          `json:"waiting"`
}

func (node *Node) inputBindingInfo(fork ForkId) []BindingInfo {
	readSize := node.top.rt.FreeMemBytes() / int64(len(node.prenodes)+1)
	result := make([]BindingInfo, len(node.call.ResolvedInputs()))
	for i, input := range node.call.Call().Bindings.List {
		result[i].Id = input.Id
		rb := node.call.ResolvedInputs()[input.Id]
		result[i].Type = input.Tname
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
	return result
}

func (node *Node) resolveInputs(fork ForkId) (MarshalerMap, error) {
	readSize := node.top.rt.FreeMemBytes() / int64(len(node.prenodes)+1)
	result := make(MarshalerMap, len(node.call.ResolvedInputs()))
	for k, v := range node.call.ResolvedInputs() {
		if _, r, err := node.top.resolve(v.Exp, v.Type, fork, readSize); err != nil {
			return result, err
		} else {
			result[k] = r
		}
	}
	return result, nil
}

func (node *Node) outputBindingInfo(fork ForkId) []BindingInfo {
	readSize := node.top.rt.FreeMemBytes() / int64(len(node.prenodes)+1)
	ro := node.call.ResolvedOutputs()
	if ro == nil {
		return nil
	}
	if outs, ok := ro.Exp.(*syntax.MapExp); ok {
		members := ro.Type.(*syntax.StructType).Members
		result := make([]BindingInfo, len(members))
		for i, member := range members {
			result[i].Id = member.Id
			rb := syntax.ResolvedBinding{
				Exp:  outs.Value[member.Id],
				Type: node.top.types.Get(member.Tname),
			}
			result[i].Type = member.Tname
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
		return result
	} else {
		return nil
	}
}

func (node *Node) resolvePipelineOutputs(fork ForkId) (json.Marshaler, error) {
	readSize := node.top.rt.FreeMemBytes() / int64(len(node.prenodes)+1)
	ro := node.call.ResolvedOutputs()
	_, r, err := node.top.resolve(ro.Exp, ro.Type, fork, readSize)
	return r, err
}

// resolve will either return true and the concrete value represented by the
// expression, false if some of the referred-to nodes are still running, or
// an error if there was a problem resolving a reference.
func (node *TopNode) resolve(binding syntax.Exp, t syntax.Type,
	fork ForkId, readSize int64) (bool, json.Marshaler, error) {
	if binding == nil {
		return true, nil, nil
	}
	if !binding.HasRef() && !binding.HasSweep() {
		return true, binding, nil
	}
	switch binding := binding.(type) {
	case *syntax.RefExp:
		return node.resolveRef(binding, t, fork, readSize)
	case *syntax.ArrayExp:
		return node.resolveArray(binding, t, fork, readSize)
	case *syntax.MapExp:
		return node.resolveMap(binding, t, fork, readSize)
	case *syntax.SweepExp:
		for _, part := range fork {
			if part.Source == binding {
				return node.resolve(
					binding.Value[int(*part.Id.(*arrayIndexFork))],
					t, fork, readSize)
			}
		}
		panic("sweep is not bound to this fork")
	default:
		tid := t.GetId()
		panic(fmt.Sprintf("unexpected ref or sweep type %T, wanted %s",
			binding, tid.String()))
	}
}

func (node *Node) matchFork(fork ForkId) *Fork {
	if len(node.forkRoots) == 0 {
		return node.forks[0]
	}
	fork = fork.Match(node.forkRoots)
	for i, id := range node.forkIds {
		if id.Equal(fork) {
			return node.forks[i]
		}
	}
	id, err := fork.ForkIdString()
	if err != nil {
		panic(err)
	}
	panic("invalid fork id " + id)
}

func (node *TopNode) resolveRef(binding *syntax.RefExp, t syntax.Type,
	fork ForkId, readSize int64) (bool, json.Marshaler, error) {
	boundNode := node.allNodes[binding.Id]
	if boundNode == nil {
		panic("unknown bound node - this should not be possible in properly-compiled code")
	}
	metadata := boundNode.matchFork(fork).metadata
	args, err := metadata.read(OutsFile, readSize)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil, nil
		}
		return true, nil, &elementError{
			element: "stage " + binding.Id,
			inner:   err,
		}
	}
	value, err := args.Path(binding.OutputId,
		node.types.Get(syntax.TypeId{
			Tname: boundNode.call.Call().DecId,
		}), t, node.types)
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
	return err.element + ": " + err.inner.Error()
}

func (err *elementError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.inner
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
		return args.filter(dest, lookup)
	}
	switch t := source.(type) {
	case *syntax.TypedMapType:
		result := make(MarshalerMap, len(args))
		var errs syntax.ErrorList
		for k, v := range args {
			elem, err := resolvePath(v, p, t.Elem, lookup)
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
				mt syntax.Type, lookup *syntax.TypeLookup) (json.Marshaler, error) {
				if indexDot > 0 {
					return resolvePath(v, p[indexDot+1:], mt, lookup)
				} else {
					m, fatal, err := mt.FilterJson(v, lookup)
					if fatal {
						return m, err
					} else {
						return m, nil
					}
				}
			}(v, indexDot, p, mt, lookup)
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
	t syntax.Type, lookup *syntax.TypeLookup) (json.Marshaler, error) {
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
		et := lookup.GetArray(t.Elem, t.Dim-1)
		result := make(marshallerArray, 0, len(arr))
		var errs syntax.ErrorList
		for i, v := range arr {
			elem, err := resolvePath(v, p, et, lookup)
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
		return args.Path(p, t, t, lookup)
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

func (node *TopNode) resolveArray(binding *syntax.ArrayExp, t syntax.Type,
	fork ForkId, readSize int64) (bool, json.Marshaler, error) {
	result := make(marshallerArray, len(binding.Value))
	if at, ok := t.(*syntax.ArrayType); !ok {
		return true, nil, &syntax.IncompatibleTypeError{
			Message: "not an array type",
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
			errs = append(errs, err)
		} else if !ready {
			allReady = false
		} else {
			result[i] = v
		}
	}
	return allReady, result, errs.If()
}

func (node *TopNode) resolveMap(binding *syntax.MapExp, t syntax.Type,
	fork ForkId, readSize int64) (bool, json.Marshaler, error) {
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
			Message: "not a map type",
		}
	}
	return allReady, result, errs.If()
}

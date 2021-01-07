// Copyright (c) 2021 10X Genomics, Inc. All rights reserved.

package ast_builder

import (
	"encoding"
	"encoding/base64"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
)

// Bindings converts a Go map or struct to bindings suitable for use in a Call.
func Bindings(args interface{}) (*syntax.BindStms, error) {
	if a, ok := args.(reflect.Value); ok {
		return bindings(a)
	}
	return bindings(reflect.ValueOf(args))
}

func bindings(args reflect.Value) (*syntax.BindStms, error) {
	switch args.Kind() {
	case reflect.Ptr, reflect.Interface:
		if args.IsNil() {
			return nil, fmt.Errorf("expected non-nil args")
		}
		return Bindings(args.Elem())
	case reflect.Struct:
		bindings := syntax.BindStms{List: make([]*syntax.BindStm, 0, args.NumField())}
		bs, err := encodeBindings(args, args.Type(), bindings.List)
		bindings.List = bs
		return &bindings, err
	case reflect.Map:
		exp, err := ValExp(args)
		if err != nil {
			return nil, err
		}
		m, ok := exp.(*syntax.MapExp)
		if !ok {
			return nil, fmt.Errorf("expected a map or struct, but got %s", exp)
		}
		keys := make([]string, 0, len(m.Value))
		for k := range m.Value {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		bindings := syntax.BindStms{List: make([]*syntax.BindStm, len(keys))}
		for i, k := range keys {
			bindings.List[i] = &syntax.BindStm{
				Id:  k,
				Exp: m.Value[k],
			}
		}
		return &bindings, nil
	}
	return nil, fmt.Errorf("expected a map or struct, but got %v", args)
}

// ValExp converts a Go value to an mro AST expression.
func ValExp(arg interface{}) (syntax.ValExp, error) {
	if a, ok := arg.(reflect.Value); ok {
		return valExp(a, false)
	}
	return valExp(reflect.ValueOf(arg), false)
}

func valExp(arg reflect.Value, forceString bool) (syntax.ValExp, error) {
	t := arg.Type()
	if arg.Kind() != reflect.Ptr &&
		reflect.PtrTo(t).Implements(textMarshalerType) ||
		t.Implements(textMarshalerType) {
		return textMarshalerEncoder(arg)
	}
	switch arg.Kind() {
	case reflect.Ptr, reflect.Interface:
		if arg.IsNil() {
			return new(syntax.NullExp), nil
		}
		return valExp(arg.Elem(), forceString)
	case reflect.Slice:
		if arg.IsNil() {
			return new(syntax.NullExp), nil
		}
		if t.Elem().Kind() == reflect.Uint8 {
			if t.PkgPath() == "encoding/json" && t.Name() == "RawMessage" {
				var parser syntax.Parser
				return parser.ParseValExp(arg.Bytes())
			} else if !reflect.PtrTo(t.Elem()).Implements(textMarshalerType) {
				// []byte gets special treatment, just like in json
				return &syntax.StringExp{Value: encodeByteSlice(arg)}, nil
			}
		}
		fallthrough
	case reflect.Array:
		arr := syntax.ArrayExp{
			Value: make([]syntax.Exp, arg.Len()),
		}
		var errs syntax.ErrorList
		for i := 0; i < arg.Len(); i++ {
			v, err := ValExp(arg.Index(i))
			if err != nil {
				errs = append(errs, fmt.Errorf("element %d: %w", i, err))
			}
			arr.Value[i] = v
		}
		return &arr, errs.If()
	case reflect.Map:
		if arg.IsNil() {
			return new(syntax.NullExp), nil
		}
		m := syntax.MapExp{
			Kind:  syntax.KindMap,
			Value: make(map[string]syntax.Exp, arg.Len()),
		}
		var errs syntax.ErrorList
		i := arg.MapRange()
		for i.Next() {
			k, err := keyString(i.Key())
			if err != nil {
				errs = append(errs, err)
			}
			v, err := ValExp(i.Value())
			if err != nil {
				errs = append(errs, fmt.Errorf("element %q: %w", k, err))
			}
			m.Value[k] = v
		}
		return &m, errs.If()
	case reflect.Bool:
		if forceString {
			return &syntax.StringExp{Value: strconv.FormatBool(arg.Bool())}, nil
		}
		return &syntax.BoolExp{Value: arg.Bool()}, nil
	case reflect.Float32, reflect.Float64:
		if forceString {
			prec := 64
			if arg.Kind() == reflect.Float32 {
				prec = 32
			}
			return &syntax.StringExp{
				Value: strconv.FormatFloat(arg.Float(), 'g', -1, prec),
			}, nil
		}
		return &syntax.FloatExp{Value: arg.Float()}, nil
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		if forceString {
			return &syntax.StringExp{
				Value: strconv.FormatInt(arg.Int(), 10),
			}, nil
		}
		return &syntax.IntExp{Value: arg.Int()}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u := arg.Uint()
		if forceString {
			return &syntax.StringExp{
				Value: strconv.FormatUint(u, 10),
			}, nil
		}
		i := int64(u)
		if i < 0 {
			return &syntax.IntExp{Value: i}, fmt.Errorf("integer overflow %d", u)
		}
		return &syntax.IntExp{Value: i}, nil
	case reflect.String:
		return &syntax.StringExp{Value: arg.String()}, nil
	case reflect.Struct:
		return structExp(arg)
	}
	return nil, fmt.Errorf("invalid type %v", arg)
}

func textMarshalerEncoder(v reflect.Value) (syntax.ValExp, error) {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return new(syntax.NullExp), nil
	}
	m, ok := v.Interface().(encoding.TextMarshaler)
	if !ok {
		return new(syntax.NullExp), nil
	}
	b, err := m.MarshalText()
	return &syntax.StringExp{Value: string(b)}, err
}

func encodeByteSlice(v reflect.Value) string {
	s := v.Bytes()
	encodedLen := base64.StdEncoding.EncodedLen(len(s))
	if encodedLen <= 1024 {
		// If the encoded bytes are small, avoid an extra
		// allocation and use the cheaper Encoding.Encode.
		// Use stack allocation (probably, though it's up to the compiler).
		var scratch [1024]byte
		dst := scratch[:encodedLen]
		base64.StdEncoding.Encode(dst, s)
		return string(dst)
	} else {
		var dst strings.Builder
		// The encoded bytes are too long to cheaply allocate, and
		// Encoding.Encode is no longer noticeably cheaper.
		enc := base64.NewEncoder(base64.StdEncoding, &dst)
		if _, err := enc.Write(s); err != nil {
			panic(err)
		}
		if err := enc.Close(); err != nil {
			panic(err)
		}
		return dst.String()
	}
}

func structExp(s reflect.Value) (syntax.ValExp, error) {
	t := s.Type()
	bs, err := encodeBindings(s, t, make([]*syntax.BindStm, 0, t.NumField()))
	m := syntax.MapExp{
		Kind:  syntax.KindStruct,
		Value: make(map[string]syntax.Exp, len(bs)),
	}
	for _, b := range bs {
		m.Value[b.Id] = b.Exp
	}
	return &m, err
}

func encodeBindings(s reflect.Value, t reflect.Type, b []*syntax.BindStm) ([]*syntax.BindStm, error) {
	var errs syntax.ErrorList
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		isUnexported := sf.PkgPath != ""
		if sf.Anonymous {
			t := sf.Type
			for t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			if t.Kind() == reflect.Struct {
				var err error
				if b, err = encodeBindings(s.Field(i), t, b); err != nil {
					errs = append(errs, err)
				}
			}
			continue
		} else if isUnexported {
			// Ignore unexported non-embedded fields.
			continue
		}
		k, forceString := getFieldKey(sf)
		if k == "-" {
			continue
		}
		v, err := valExp(s.Field(i), forceString)
		if err != nil {
			errs = append(errs, fmt.Errorf("field %s: %w", sf.Name, err))
		}
		b = append(b, &syntax.BindStm{Id: k, Exp: v})
	}
	return b, errs.If()
}

func getFieldKey(sf reflect.StructField) (string, bool) {
	k := sf.Name
	jsTag := strings.Split(sf.Tag.Get("json"), ",")
	if len(jsTag) > 0 {
		if jsTag[0] == "-" {
			return "-", false
		} else if jsTag[0] != "" {
			k = jsTag[0]
		}
		for _, tag := range jsTag[1:] {
			if tag == "string" {
				return k, true
			}
		}
	}
	return k, false
}

func keyString(k reflect.Value) (string, error) {
	if k.Kind() == reflect.String {
		return k.String(), nil
	}
	if k.Kind() == reflect.Ptr && k.IsNil() {
		return "", nil
	}
	if tm, ok := k.Interface().(encoding.TextMarshaler); ok {
		buf, err := tm.MarshalText()
		return string(buf), err
	}
	switch k.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(k.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(k.Uint(), 10), nil
	case reflect.Ptr, reflect.Interface:
		if k.IsNil() {
			return "", nil
		}
		return keyString(k.Elem())
	default:
		return k.String(), fmt.Errorf("invalid %v type for map key %v",
			k.Kind(), k)
	}
}

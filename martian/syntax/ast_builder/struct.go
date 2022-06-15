// Copyright (c) 2021 10X Genomics, Inc. All rights reserved.

// Package ast_builder contains utility methods for building martian AST objects.
package ast_builder

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
)

var (
	NotStructTypeError = errors.New("not a struct type")
	AnonymousTypeError = errors.New("cannot convert anonymous type")
)

// StructType returns an mro StructType ast node declaring a struct
// corresponding to the given go struct type.
//
// Only public members are considered.
//
// The name of each struct field will be the name of the field, unless a
// `json` tag is present in which case that name will be used, to ensure
// that the struct json serialization is consistent with the struct definition.
// If the field tag is `json:"-"` then the field will be excluded from the MRO.
//
// The default struct member type will be
//
//	Go type:                                             MRO type:
//	bool                                                 bool
//	float32, float64                                     float
//	int, uint, int8, int16...                            int
//	string, []byte                                       string
//	map[string]json.RawMessage, map[string]interface{}   map
//
// This can be overridden with a tag `mro:"type_name"`, e.g. if the intended
// type is a user-defined file type.
//
// If the go field type is a map (other than map[string]json.RawMessage or
// map[string]interface{}), a typed map of the given type will be used.
//
// If the go field is an array or slice (other than []byte), an array of the
// approprriate type will be used.
//
// The `mro_help:"..."` tag may be used to add help text to the struct field.
//
// The `mro_out:"..."` tag may be used to add an out name for the struct field.
func StructType(t reflect.Type) (*syntax.StructType, error) {
	if t.Kind() == reflect.Ptr {
		return StructType(t.Elem())
	}
	if t.Kind() != reflect.Struct {
		return nil, NotStructTypeError
	}
	if t.Name() == "" {
		return nil, AnonymousTypeError
	}
	members, err := structMembers(t, make([]*syntax.StructMember, 0, t.NumField()))
	return &syntax.StructType{
		Id:      t.Name(),
		Members: members,
	}, err
}

func structMembers(t reflect.Type, list []*syntax.StructMember) ([]*syntax.StructMember, error) {
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
				list, err = structMembers(t, list)
				if err != nil {
					return list, err
				}
			}
			continue
		} else if isUnexported {
			// Ignore unexported non-embedded fields.
			continue
		}
		m, err := makeStructMember(sf)
		if err != nil {
			return list, err
		}
		if m != nil {
			list = append(list, m)
		}
	}
	return list, nil
}

func makeStructMember(sf reflect.StructField) (*syntax.StructMember, error) {
	result := syntax.StructMember{
		Id:      sf.Name,
		OutName: sf.Tag.Get("mro_out"),
		Help:    sf.Tag.Get("mro_help"),
	}
	typeName := sf.Tag.Get("mro_type")
	jsTag := strings.Split(sf.Tag.Get("json"), ",")
	if len(jsTag) > 0 {
		if jsTag[0] == "-" {
			return nil, nil
		} else if jsTag[0] != "" {
			result.Id = jsTag[0]
		}
		if typeName == "" {
			for _, tag := range jsTag[1:] {
				if tag == "string" {
					typeName = "string"
					break
				}
			}
		}
	}
	tid, err := getStructMemberType(sf.Type, typeName)
	if err != nil {
		err = fmt.Errorf("field %s: %w", sf.Name, err)
	}
	result.Tname = tid
	return &result, err
}

type InvalidTypeError string

func (err InvalidTypeError) Error() string {
	return string(err)
}

var (
	InterfaceTypeError = InvalidTypeError("invalid interface type")
	UnknownTypeError   = InvalidTypeError("invalid type json.RawMessage")
)

var textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()

func getStructMemberType(t reflect.Type, typeName string) (syntax.TypeId, error) {
	if t.Kind() != reflect.Ptr && reflect.PtrTo(t).Implements(textMarshalerType) ||
		t.Implements(textMarshalerType) {
		if typeName != "" {
			return syntax.TypeId{Tname: typeName}, nil
		}
		return syntax.TypeId{Tname: syntax.KindString}, nil
	}
	switch t.Kind() {
	case reflect.Ptr:
		return getStructMemberType(t.Elem(), typeName)
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 &&
			!reflect.PtrTo(t.Elem()).Implements(textMarshalerType) {
			// []byte type, treat like string and don't make an array.
			if typeName != "" {
				return syntax.TypeId{Tname: typeName}, nil
			} else if t.PkgPath() == "encoding/json" && t.Name() == "RawMessage" {
				return syntax.TypeId{Tname: syntax.KindMap}, UnknownTypeError
			}
			return syntax.TypeId{Tname: syntax.KindString}, nil
		}
		fallthrough
	case reflect.Array:
		tid, err := getStructMemberType(t.Elem(), typeName)
		tid.ArrayDim++
		return tid, err
	case reflect.Map:
		if err := checkKeyType(t.Key()); err != nil {
			return syntax.TypeId{Tname: syntax.KindMap}, err
		}
		tid, err := getStructMemberType(t.Elem(), typeName)
		var tErr InvalidTypeError
		if errors.As(err, &tErr) {
			if typeName != "" && typeName != syntax.KindMap {
				return syntax.TypeId{Tname: typeName, MapDim: 1}, nil
			}
			return syntax.TypeId{Tname: syntax.KindMap}, nil
		}
		if tid.MapDim > 0 {
			// MRO does not allow nested map types.  However, we might be using
			// a martian struct type that is represented in Go as a map.
			if typeName != "" && typeName != syntax.KindMap {
				return syntax.TypeId{Tname: typeName, MapDim: 1}, nil
			}
			return syntax.TypeId{Tname: syntax.KindMap}, err
		}
		tid.MapDim = tid.ArrayDim + 1
		tid.ArrayDim = 0
		return tid, err
	}
	if typeName != "" {
		return syntax.TypeId{Tname: typeName}, nil
	}
	switch t.Kind() {
	case reflect.Bool:
		return syntax.TypeId{Tname: syntax.KindBool}, nil
	case reflect.Float32, reflect.Float64:
		return syntax.TypeId{Tname: syntax.KindFloat}, nil
	case reflect.String:
		return syntax.TypeId{Tname: syntax.KindString}, nil
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return syntax.TypeId{Tname: syntax.KindInt}, nil
	case reflect.Struct:
		return syntax.TypeId{Tname: t.Name()}, nil
	case reflect.Interface:
		return syntax.TypeId{Tname: syntax.KindMap}, InterfaceTypeError
	}
	return syntax.TypeId{Tname: syntax.KindMap}, fmt.Errorf(
		"invalid type %s (kind %s)", t.Name(), t.Kind())
}

func checkKeyType(t reflect.Type) error {
	if t.Implements(textMarshalerType) {
		return nil
	}
	switch t.Kind() {
	case reflect.Ptr:
		return checkKeyType(t.Elem())
	case reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return nil
	default:
		return fmt.Errorf("invalid non-stringable key type %s", t.Name())
	}
}

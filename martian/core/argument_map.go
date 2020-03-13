//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

// Data structure for validating and converting arguments and outputs.

package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/martian-lang/martian/martian/syntax"
)

// Returns true if the given value has the correct mro type.
// Non-fatal errors are written to alarms.
func checkJsonType(types *syntax.TypeLookup, val json.RawMessage,
	typename syntax.TypeId,
	alarms *strings.Builder) error {
	// Null is always legal.
	if bytes.Equal(val, nullBytes) {
		return nil
	}
	t := types.Get(typename)
	if t == nil {
		panic("unknown type " + typename.String())
	}
	return t.IsValidJson(val, alarms, types)
}

// Mapping from argument or output names to values.
//
// LazyArgumentMap does not fully deserialize the arguments.
//
// Includes convenience methods to validate the arguments against parameter
// lists from MRO.
type LazyArgumentMap map[string]json.RawMessage

var nullBytes = []byte(syntax.KindNull)

// Validate that all of the arguments in the map are declared parameters, and
// that all declared parameters are set in the arguments to a value of the
// correct type, or null.
//
// Hard errors are returned as the first parameter.  "soft" error messages
// are returned in the second.
//
// Optional params are values which are permitted to be in the argument map
// (if they are of the correct type) but which are not required to be present.
// For example, for a stage defined as
//
//     stage STAGE(
//         in  int a,
//         out int b,
//     ) split (
//         in  int c,
//         out int d,
//     )
//
// then in the outputs from the chunks, d is required but b is optional.
func (self LazyArgumentMap) ValidateInputs(types *syntax.TypeLookup,
	expected *syntax.InParams, optional ...*syntax.InParams) (error, string) {
	var result, alarms strings.Builder
	tname := func(param syntax.Param) string {
		t := param.GetTname()
		return t.String()
	}
	for _, param := range expected.Table {
		if val, ok := self[param.GetId()]; !ok {
			fmt.Fprintf(&result, "Missing input parameter '%s'\n", param.GetId())
			continue
		} else if len(val) == 0 || bytes.Equal(val, nullBytes) {
			// Allow for null output parameters
			continue
		} else if err := checkJsonType(types,
			val,
			param.GetTname(),
			&alarms); err != nil {
			fmt.Fprintf(&result,
				"Expected %s input parameter '%s' %s\n",
				tname(param), param.GetId(),
				err.Error())
		}
	}
	for key, val := range self {
		if _, ok := expected.Table[key]; !ok {
			isOptional := false
			for _, params := range optional {
				if param, ok := params.Table[key]; ok {
					isOptional = true
					if len(val) > 0 && !bytes.Equal(val, nullBytes) {
						if err := checkJsonType(types,
							val,
							param.GetTname(),
							&alarms); err != nil {
							fmt.Fprintf(&result,
								"Optional %s input parameter '%s' %s\n",
								tname(param), param.GetId(),
								err.Error())
						}
					}
				}
			}
			if !isOptional {
				fmt.Fprintf(&result, "Unexpected parameter '%s'\n", key)
			}
		}
	}
	if result.Len() == 0 {
		return nil, alarms.String()
	} else {
		return errors.New(result.String()), alarms.String()
	}
}

// Validate that all of the arguments in the map are declared parameters, and
// that all declared parameters are set in the arguments to a value of the
// correct type, or null.
//
// Hard errors are returned as the first parameter.  "soft" error messages
// are returned in the second.
//
// Optional params are values which are permitted to be in the argument map
// (if they are of the correct type) but which are not required to be present.
// For example, for a stage defined as
//
//     stage STAGE(
//         in  int a,
//         out int b,
//     ) split (
//         in  int c,
//         out int d,
//     )
//
// then in the outputs from the chunks, d is required but b is optional.
func (self LazyArgumentMap) ValidateOutputs(types *syntax.TypeLookup,
	expected *syntax.OutParams, optional ...*syntax.OutParams) (error, string) {
	var result, alarms strings.Builder
	tname := func(param syntax.Param) string {
		t := param.GetTname()
		return t.String()
	}
	for _, param := range expected.Table {
		if val, ok := self[param.GetId()]; !ok {
			fmt.Fprintf(&result, "Missing output value '%s'\n", param.GetId())
			continue
		} else if len(val) == 0 || bytes.Equal(val, nullBytes) {
			// Allow for null output parameters
			continue
		} else if err := checkJsonType(types,
			val,
			param.GetTname(),
			&alarms); err != nil {
			fmt.Fprintf(&result,
				"Expected %s output value '%s' %s\n",
				tname(param), param.GetId(),
				err.Error())
		}
	}
	for key, val := range self {
		if _, ok := expected.Table[key]; !ok {
			isOptional := false
			for _, params := range optional {
				if param, ok := params.Table[key]; ok {
					isOptional = true
					if len(val) > 0 && !bytes.Equal(val, nullBytes) {
						if err := checkJsonType(types,
							val,
							param.GetTname(),
							&alarms); err != nil {
							fmt.Fprintf(&result,
								"Optional %s output value '%s' %s\n",
								tname(param), param.GetId(),
								err.Error())
						}
					}
				}
			}
			if !isOptional {
				fmt.Fprintf(&alarms, "Unexpected output '%s'\n", key)
			}
		}
	}
	if result.Len() == 0 {
		return nil, alarms.String()
	} else {
		return errors.New(result.String()), alarms.String()
	}
}

// Validate that all of the arguments in the map are a value of the
// correct type, or null.
//
// Hard errors are returned as the first parameter.  "soft" error messages
// are returned in the second.
//
// Unlike for stage outputs (which are resolved via LazyArgumentMap),
// pipeline outputs are generated by the runtime and so we don't need to
// check for missing or invalid keys.
func (self MarshalerMap) ValidatePipelineOutputs(types *syntax.TypeLookup,
	expected *syntax.StructType) (error, string) {
	var result, alarms strings.Builder
	tname := func(param *syntax.StructMember) string {
		t := param.Tname
		return t.String()
	}
	for _, param := range expected.Members {
		if val, ok := self[param.Id]; !ok {
			fmt.Fprintf(&result, "Missing output value '%s'\n", param.Id)
			continue
		} else {
			switch val := val.(type) {
			case json.RawMessage:
				if len(val) == 0 || bytes.Equal(val, nullBytes) {
					// Allow for null output parameters
					continue
				} else if err := checkJsonType(types,
					val,
					param.Tname,
					&alarms); err != nil {
					fmt.Fprintf(&result,
						"Expected %s output value '%s' %s\n",
						tname(param), param.Id,
						err.Error())
				}
			case syntax.Exp:
				// Don't need to do anything here, as the expression would have
				// been checked by the compiler.
			}
		}
	}
	if result.Len() == 0 {
		return nil, alarms.String()
	} else {
		return errors.New(result.String()), alarms.String()
	}
}

func (self LazyArgumentMap) MarshalJSON() ([]byte, error) {
	if self == nil {
		return nullBytes[:len(nullBytes):len(nullBytes)], nil
	}
	if len(self) == 0 {
		return []byte("{}"), nil
	}
	var buf bytes.Buffer
	err := self.encodeJSON(&buf)
	return buf.Bytes(), err
}

// EncodeJSON writes a json representation of the object to a buffer.
func (self LazyArgumentMap) EncodeJSON(buf *bytes.Buffer) error {
	if self == nil {
		_, err := buf.Write(nullBytes)
		return err
	}
	if len(self) == 0 {
		_, err := buf.WriteString("{}")
		return err
	}
	return self.encodeJSON(buf)
}

func (self LazyArgumentMap) encodeJSON(buf *bytes.Buffer) error {
	l := 2
	if len(self) > 1 {
		l += len(self) - 1
	}
	keys := make([]string, 0, len(self))
	for k, v := range self {
		l += 2 + len(k) + len(v)
		keys = append(keys, k)
	}
	sort.Strings(keys)
	buf.Grow(l)
	if _, err := buf.WriteRune('{'); err != nil {
		return err
	}
	for i, key := range keys {
		if i != 0 {
			if _, err := buf.WriteRune(','); err != nil {
				return err
			}
		}
		if b, err := json.Marshal(key); err != nil {
			return err
		} else if _, err := buf.Write(b); err != nil {
			return err
		}
		if _, err := buf.WriteRune(':'); err != nil {
			return err
		}
		if v := self[key]; v == nil {
			if _, err := buf.Write(nullBytes); err != nil {
				return err
			}
		} else if _, err := buf.Write(v); err != nil {
			return err
		}
	}
	_, err := buf.WriteRune('}')
	return err
}

// CallMode Returns the call mode for a call which depends on this source.
func (m LazyArgumentMap) CallMode() syntax.CallMode {
	return syntax.ModeMapCall
}

// KnownLength returns true if the source is an array with a known length
// or is a map with a known set of keys.
func (m LazyArgumentMap) KnownLength() bool {
	return true
}

// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
// the length of the array referred to by this source.  Otherwise it will
// return -1.
func (m LazyArgumentMap) ArrayLength() int {
	return -1
}

// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
// a map[string]Exp with the same keys which any call mapping over this
// source would have.  The values are arbitrary.  Otherwise, it will return
// nil.
func (m LazyArgumentMap) Keys() map[string]syntax.Exp {
	if len(m) == 0 {
		return nil
	}
	keys := make(map[string]syntax.Exp, len(m))
	for k := range m {
		keys[k] = nil
	}
	return keys
}

func (m LazyArgumentMap) GoString() string {
	var buf strings.Builder
	if err := buf.WriteByte('{'); err != nil {
		panic(err)
	}
	for i, v := range m {
		if err := buf.WriteByte(' '); err != nil {
			panic(err)
		}
		if _, err := buf.WriteString(i); err != nil {
			panic(err)
		}
		if err := buf.WriteByte(':'); err != nil {
			panic(err)
		}
		if _, err := buf.WriteString(fmt.Sprint(v)); err != nil {
			panic(err)
		}
	}
	if err := buf.WriteByte('}'); err != nil {
		panic(err)
	}
	return buf.String()
}

// MarshalerMap stores arbitrary json data.  The values are strings, and the
// keys are objects which know how to marshal themselves.
//
// Unmarshaling into a MarshalerMap results in values of type json.RawMessage.
//
// Marshaling outputs keys in sorted order.
type MarshalerMap map[string]json.Marshaler

func (m MarshalerMap) MarshalJSON() ([]byte, error) {
	if m == nil {
		return nullBytes[:len(nullBytes):len(nullBytes)], nil
	} else if len(m) == 0 {
		return []byte(`{}`), nil
	}
	var buf bytes.Buffer
	err := m.encodeJSON(&buf)
	return buf.Bytes(), err
}

// EncodeJSON writes a json representation of the object to a buffer.
func (m MarshalerMap) EncodeJSON(buf *bytes.Buffer) error {
	if m == nil {
		_, err := buf.Write(nullBytes)
		return err
	} else if len(m) == 0 {
		_, err := buf.WriteString("{}")
		return err
	}
	return m.encodeJSON(buf)
}

func (m MarshalerMap) encodeJSON(buf *bytes.Buffer) error {
	keyLen := 2
	keys := make([]string, 0, len(m))
	for key, val := range m {
		keys = append(keys, key)
		keyLen += 4 + len(key)
		if v, ok := val.(json.RawMessage); ok {
			keyLen += len(v)
		} else {
			keyLen += 4
		}
	}
	sort.Strings(keys)
	buf.Grow(keyLen)
	if _, err := buf.WriteRune('{'); err != nil {
		return err
	}
	first := true
	for _, key := range keys {
		elem := m[key]
		if !first {
			if _, err := buf.WriteRune(','); err != nil {
				return err
			}
		}
		first = false
		if b, err := json.Marshal(key); err != nil {
			return err
		} else if _, err := buf.Write(b); err != nil {
			return err
		}
		if _, err := buf.WriteRune(':'); err != nil {
			return err
		}
		if elem == nil {
			if _, err := buf.Write(nullBytes); err != nil {
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
	_, err := buf.WriteRune('}')
	return err
}

func (m *MarshalerMap) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, nullBytes) {
		*m = nil
		return nil
	}
	var rm map[string]json.RawMessage
	if err := json.Unmarshal(b, &rm); err != nil {
		return err
	}
	if *m == nil {
		*m = make(MarshalerMap, len(rm))
	}
	for k, v := range rm {
		(*m)[k] = v
	}
	return nil
}

func (m MarshalerMap) ToLazyArgumentMap() (LazyArgumentMap, error) {
	if m == nil {
		return nil, nil
	}
	result := make(LazyArgumentMap, len(m))
	for k, v := range m {
		if v == nil {
			result[k] = nil
		} else {
			switch v := v.(type) {
			case json.RawMessage:
				result[k] = v
			default:
				if r, err := v.MarshalJSON(); err != nil {
					return result, err
				} else {
					result[k] = r
				}
			}
		}
	}
	return result, nil
}

func (self LazyArgumentMap) ToMarshalerMap() MarshalerMap {
	r := make(MarshalerMap, len(self))
	for k, v := range self {
		r[k] = v
	}
	return r
}

// CallMode Returns the call mode for a call which depends on this source.
func (m MarshalerMap) CallMode() syntax.CallMode {
	return syntax.ModeMapCall
}

// KnownLength returns true if the source is an array with a known length
// or is a map with a known set of keys.
func (m MarshalerMap) KnownLength() bool {
	return true
}

// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
// the length of the array referred to by this source.  Otherwise it will
// return -1.
func (m MarshalerMap) ArrayLength() int {
	return -1
}

// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
// a map[string]Exp with the same keys which any call mapping over this
// source would have.  The values are arbitrary.  Otherwise, it will return
// nil.
func (m MarshalerMap) Keys() map[string]syntax.Exp {
	if len(m) == 0 {
		return nil
	}
	keys := make(map[string]syntax.Exp, len(m))
	for k := range m {
		keys[k] = nil
	}
	return keys
}

func (m MarshalerMap) GoString() string {
	var buf strings.Builder
	if err := buf.WriteByte('{'); err != nil {
		panic(err)
	}
	for i, v := range m {
		if err := buf.WriteByte(' '); err != nil {
			panic(err)
		}
		if _, err := buf.WriteString(i); err != nil {
			panic(err)
		}
		if err := buf.WriteByte(':'); err != nil {
			panic(err)
		}
		if _, err := buf.WriteString(fmt.Sprint(v)); err != nil {
			panic(err)
		}
	}
	if err := buf.WriteByte('}'); err != nil {
		panic(err)
	}
	return buf.String()
}

var (
	jsonMarshalerType = reflect.TypeOf(new(json.Marshaler)).Elem()
	jsonNumberType    = reflect.TypeOf(json.Number(""))
)

// Convenience method to convert an arbitrary object type into
// a LazyArgumentMap.
//
// This is intended primarily for use by authors of native Go stages.
func MakeMarshalerMap(binding interface{}) MarshalerMap {
	if binding == nil {
		return nil
	}
	switch binding := binding.(type) {
	case MarshalerMap:
		return binding
	case LazyArgumentMap:
		m := make(MarshalerMap, len(binding))
		for k, v := range binding {
			m[k] = v
		}
		return m
	case map[string]interface{}:
		m := make(MarshalerMap, len(binding))
		for k, v := range binding {
			switch v := v.(type) {
			case json.Marshaler:
				m[k] = v
			default:
				b, _ := json.Marshal(v)
				m[k] = json.RawMessage(b)
			}
		}
		return m
	case map[string]json.Marshaler:
		return MarshalerMap(binding)
	case map[string]json.RawMessage:
		return MakeMarshalerMap(LazyArgumentMap(binding))
	default:
		v := reflect.ValueOf(binding)
		t := v.Type()
		for t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
			if v.IsNil() {
				return nil
			}
			v = v.Elem()
			t = v.Type()
		}
		if t := v.Type(); t.Kind() == reflect.Map && t.Key().Kind() == reflect.String {
			// For map[string]X just get the key/value pairs out.
			if v.Len() == 0 {
				return nil
			}
			m := make(MarshalerMap)
			for _, key := range v.MapKeys() {
				if vv := v.MapIndex(key); vv.IsValid() {
					in := vv.Interface()

					if mar, ok := in.(json.Marshaler); ok {
						m[key.String()] = mar
					} else {
						b, _ := json.Marshal(in)
						m[key.String()] = json.RawMessage(b)
					}
				}
			}
			return m
		} else if t.Kind() == reflect.Struct &&
			!reflect.PtrTo(t).Implements(jsonMarshalerType) {
			// If the struct has custom marshaling logic then we need to
			// respect that.  Otherwise we can just pull out the public
			// fields.
			return MakeMarshalerMap(argumentMapFromStruct(t, v))
		} else if b, err := json.Marshal(binding); err == nil {
			// Fall back on cross-serializing as json.  This ensures that any
			// nonstandard serialization gets applied.
			var m LazyArgumentMap
			if err := json.Unmarshal(b, &m); err == nil {
				return MakeMarshalerMap(m)
			}
		}
	}
	return nil
}

func isExportedName(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

// Builds a map from a struct type, with keys matching what
// json.Marshal does.  Unlike json.Marshal, does not traverse deeply into
// the struct.
//
// This should not be used for structs which implement json.Marshaler as they
// may encode their keys in arbitrary ways.
//
// t should be v.Type(), and t.Kind() must be reflect.Struct.
func argumentMapFromStruct(t reflect.Type, v reflect.Value) MarshalerMap {
	parseTag := func(tag string) (name string, omitempty bool) {
		if idx := strings.Index(tag, ","); idx != -1 {
			name = tag[:idx]
			tag = tag[idx+1:]
			// Search through comma-separated options
			for tag != "" {
				if idx := strings.Index(tag, ","); idx != -1 {
					if tag[:idx] == "omitempty" {
						return name, true
					}
					tag = tag[idx+1:]
				} else {
					return name, tag == "omitempty"
				}
			}
			return name, false
		} else {
			return tag, false
		}
	}
	isEmpty := func(v reflect.Value) bool {
		switch v.Kind() {
		case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
			return v.Len() == 0
		case reflect.Bool:
			return !v.Bool()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return v.Int() == 0
		case reflect.Uint, reflect.Uint8, reflect.Uint16,
			reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return v.Uint() == 0
		case reflect.Float32, reflect.Float64:
			return v.Float() == 0
		case reflect.Interface, reflect.Ptr:
			return v.IsNil()
		}
		return false
	}
	m := make(MarshalerMap)
	for fnum := 0; fnum < t.NumField(); fnum++ {
		field := t.Field(fnum)
		if isExportedName(field.Name) {
			name, omitEmpty := parseTag(field.Tag.Get("json"))
			if name == "-" {
				continue
			} else if name == "" {
				name = field.Name
			}
			val := v.Field(fnum)
			if !val.CanInterface() {
				continue
			}
			if omitEmpty {
				valType := val.Type()
				if valType == jsonNumberType {
					if s := val.String(); s == "" || s == "0" {
						continue
					}
				} else if isEmpty(val) {
					continue
				}
			}
			v := val.Interface()
			switch v := v.(type) {
			case json.Marshaler:
				m[name] = v
			default:
				if b, err := json.Marshal(v); err == nil {
					m[name] = json.RawMessage(b)
				}
			}
		}
	}
	return m
}

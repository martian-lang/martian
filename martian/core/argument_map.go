//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//

// Data structure for validating and converting arguments and outputs.

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/martian-lang/martian/martian/syntax"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Mapping from argument or output names to values.
//
// Includes convenience methods to convert to or from other structured data
// types.
//
// ArgumentMap always deserializes numbers as json.Number values, in order
// to prevent loss of precision for integer types.
type ArgumentMap map[string]interface{}

func (self *ArgumentMap) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	args := make(map[string]interface{})
	if err := dec.Decode(&args); err != nil {
		return err
	}
	for k, v := range args {
		(*self)[k] = v
	}
	return nil
}

// Returns true if the given value has the correct mro type.
// Non-fatal errors are written to alarms.
func checkType(val json.RawMessage, typename string, arrayDim int,
	alarms *bytes.Buffer) (bool, string) {
	truncateMessage := func(val json.RawMessage, expect string) (bool, string) {
		if len(val) > 35 {
			tr := append(val[:15:15], "..."...)
			val = append(tr, val[len(val)-15:]...)
		}
		return false, fmt.Sprintf("with value %q cannot be parsed as %s.",
			[]byte(val), expect)
	}
	// Null is always legal.
	if bytes.Equal(val, nullBytes) {
		return true, ""
	}
	if arrayDim > 0 {
		var arr []json.RawMessage
		if err := json.Unmarshal(val, &arr); err != nil {
			return truncateMessage(val, "an array")
		}
		for i, v := range arr {
			if ok, msg := checkType(v, typename, arrayDim-1, alarms); !ok {
				return false, fmt.Sprintf("element %d %s", i, msg)
			}
		}
		return true, ""
	} else {
		switch typename {
		case "float":
			var v float64
			if err := json.Unmarshal(val, &v); err != nil {
				return truncateMessage(val, "a floating point number")
			} else {
				return true, ""
			}
		case "int":
			var v int64
			if err := json.Unmarshal(val, &v); err != nil {
				return truncateMessage(val, "an integer")
			} else {
				return true, ""
			}
		case "bool":
			var v bool
			if err := json.Unmarshal(val, &v); err != nil {
				return truncateMessage(val, "boolean")
			} else {
				return true, ""
			}
		case "map":
			var v map[string]json.RawMessage
			if err := json.Unmarshal(val, &v); err != nil {
				return truncateMessage(val, "a map")
			} else {
				return true, ""
			}
		case "path", "file", "string":
			var v string
			if err := json.Unmarshal(val, &v); err != nil {
				return truncateMessage(val, "a string")
			} else {
				return true, ""
			}
		default:
			// User defined file types.  For backwards compatiblity we need
			// to accept everything here.
			var v string
			if err := json.Unmarshal(val, &v); err != nil {
				trunc := val
				if len(val) > 35 {
					trunc = append(val[:15:15], "..."...)
					trunc = append(trunc, val[len(val)-15:]...)
				}
				fmt.Fprintf(alarms,
					"Expected type %s but found %q instead.\n",
					typename, trunc)
			}
			return true, ""
		}
	}
}

// Mapping from argument or output names to values.
//
// LazyArgumentMap does not fully deserialize the arguments.
//
// Includes convenience methods to validate the arguments against parameter
// lists from MRO.
type LazyArgumentMap map[string]json.RawMessage

var nullBytes = []byte("null")

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
func (self LazyArgumentMap) Validate(expected *syntax.Params, isInput bool, optional ...*syntax.Params) (error, string) {
	var result, alarms bytes.Buffer
	tname := func(param syntax.Param) string {
		return param.GetTname() + strings.Repeat("[]", param.GetArrayDim())
	}
	for _, param := range expected.Table {
		if val, ok := self[param.GetId()]; !ok {
			if isInput {
				fmt.Fprintf(&result, "Missing input parameter '%s'\n", param.GetId())
			} else {
				fmt.Fprintf(&result, "Missing output value '%s'\n", param.GetId())
			}
			continue
		} else if len(val) == 0 || bytes.Equal(val, nullBytes) {
			// Allow for null output parameters
			continue
		} else if ok, msg := checkType(val,
			param.GetTname(),
			param.GetArrayDim(),
			&alarms); !ok {
			if isInput {
				fmt.Fprintf(&result,
					"Expected %s input parameter '%s' %s\n",
					tname(param), param.GetId(),
					msg)
			} else {
				fmt.Fprintf(&result,
					"Expected %s output value '%s' %s\n",
					tname(param), param.GetId(),
					msg)
			}
		}
	}
	for key, val := range self {
		if _, ok := expected.Table[key]; !ok {
			isOptional := false
			for _, params := range optional {
				if param, ok := params.Table[key]; ok {
					isOptional = true
					if len(val) > 0 && !bytes.Equal(val, nullBytes) {
						if ok, msg := checkType(val,
							param.GetTname(),
							param.GetArrayDim(),
							&alarms); !ok {
							if isInput {
								fmt.Fprintf(&result,
									"Optional %s input parameter '%s' %s\n",
									tname(param), param.GetId(),
									msg)
							} else {
								fmt.Fprintf(&result,
									"Optional %s output value '%s' %s\n",
									tname(param), param.GetId(),
									msg)
							}
						}
					}
				}
			}
			if !isOptional {
				if isInput {
					fmt.Fprintf(&result, "Unexpected parameter '%s'\n", key)
				} else {
					fmt.Fprintf(&alarms, "Unexpected output '%s'\n", key)
				}
			}
		}
	}
	if result.Len() == 0 {
		return nil, alarms.String()
	} else {
		return fmt.Errorf(result.String()), alarms.String()
	}
}

var (
	jsonMarshalerType   = reflect.TypeOf(new(json.Marshaler)).Elem()
	jsonUnmarshalerType = reflect.TypeOf(new(json.Unmarshaler)).Elem()
	jsonNumberType      = reflect.TypeOf(json.Number(""))
)

// Convenience method to convert an arbitrary object type into
// an ArgumentMap.
//
// This is intended primarily for use by authors of native Go stages.
func MakeArgumentMap(binding interface{}) ArgumentMap {
	if binding == nil {
		return nil
	}
	switch binding := binding.(type) {
	case ArgumentMap:
		return binding
	case LazyArgumentMap:
		m := make(ArgumentMap, len(binding))
		for k, v := range binding {
			m[k] = v
		}
		return m
	case map[string]interface{}:
		return ArgumentMap(binding)
	case map[string]json.RawMessage:
		m := make(ArgumentMap, len(binding))
		for k, v := range binding {
			m[k] = v
		}
		return m
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
			m := make(ArgumentMap)
			for _, key := range v.MapKeys() {
				if vv := v.MapIndex(key); vv.IsValid() {
					m[key.String()] = vv.Interface()
				}
			}
			return m
		} else if t.Kind() == reflect.Struct &&
			!reflect.PtrTo(t).Implements(jsonMarshalerType) {
			// If the struct has custom marshaling logic then we need to
			// respect that.  Otherwise we can just pull out the public
			// fields.
			return argumentMapFromStruct(t, v)
		} else if b, err := json.Marshal(binding); err == nil {
			// Fall back on cross-serializing as json.  This ensures that any
			// nonstandard serialization gets applied.
			m := make(ArgumentMap)
			if err := json.Unmarshal(b, &m); err == nil {
				return m
			}
		}
	}
	return nil
}

// Convenience method to convert an arbitrary object type into
// a LazyArgumentMap.
//
// This is intended primarily for use by authors of native Go stages.
func MakeLazyArgumentMap(binding interface{}) LazyArgumentMap {
	if binding == nil {
		return nil
	}
	switch binding := binding.(type) {
	case LazyArgumentMap:
		return binding
	case ArgumentMap:
		m := make(LazyArgumentMap, len(binding))
		for k, v := range binding {
			m[k], _ = json.Marshal(v)
		}
		return m
	case map[string]interface{}:
		m := make(LazyArgumentMap, len(binding))
		for k, v := range binding {
			m[k], _ = json.Marshal(v)
		}
		return m
	case map[string]json.RawMessage:
		return LazyArgumentMap(binding)
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
			m := make(LazyArgumentMap)
			for _, key := range v.MapKeys() {
				if vv := v.MapIndex(key); vv.IsValid() {
					m[key.String()], _ = json.Marshal(vv.Interface())
				}
			}
			return m
		} else if t.Kind() == reflect.Struct &&
			!reflect.PtrTo(t).Implements(jsonMarshalerType) {
			// If the struct has custom marshaling logic then we need to
			// respect that.  Otherwise we can just pull out the public
			// fields.
			return MakeLazyArgumentMap(argumentMapFromStruct(t, v))
		} else if b, err := json.Marshal(binding); err == nil {
			// Fall back on cross-serializing as json.  This ensures that any
			// nonstandard serialization gets applied.
			m := make(LazyArgumentMap)
			if err := json.Unmarshal(b, &m); err == nil {
				return m
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
func argumentMapFromStruct(t reflect.Type, v reflect.Value) ArgumentMap {
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
	m := make(ArgumentMap)
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
			m[name] = val.Interface()
		}
	}
	return m
}

// Convenience method to convert an ArgumentMap into another kind
// of object.
//
// This is intended primarily for authors of native Golang stages.
func (self ArgumentMap) Decode(target interface{}) error {
	if m, ok := target.(map[string]interface{}); ok {
		for k, v := range self {
			m[k] = v
		}
		return nil
	}
	v := reflect.ValueOf(target)
	t := v.Type()
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		if v.IsNil() {
			if reflect.TypeOf(self).AssignableTo(t) {
				v.Set(reflect.ValueOf(self))
			} else {
				return fmt.Errorf("Nil pointer")
			}
		}
		v = v.Elem()
		t = v.Type()
	}
	if t.Kind() == reflect.Map {
		return self.decodeToMap(t, v)
	} else if t.Kind() == reflect.Struct &&
		!reflect.PtrTo(t).Implements(jsonUnmarshalerType) {
		return self.decodeToStruct(t, v)
	} else {
		if b, err := json.Marshal(self); err != nil {
			return err
		} else {
			return json.Unmarshal(b, target)
		}
	}
}

// Populates a map from argument keys.
func (self ArgumentMap) decodeToMap(t reflect.Type, v reflect.Value) error {
	if t.Key().Kind() != reflect.String {
		return fmt.Errorf("Non-string key type %v", t.Key())
	}
	valType := t.Elem()
	for myKey, myValue := range self {
		val := reflect.ValueOf(myValue)
		if val.Type().AssignableTo(valType) {
			v.SetMapIndex(reflect.ValueOf(myKey), val)
		} else {
			if b, err := json.Marshal(myValue); err != nil {
				return err
			} else {
				targV := reflect.New(valType)
				targ := targV.Interface()
				if err := json.Unmarshal(b, targ); err != nil {
					return err
				}
				v.SetMapIndex(reflect.ValueOf(myKey), targV)
			}
		}
	}
	return nil
}

// Populates a struct's fields from map keys, the same way that json.Marshal
// does.  Unlike json.Marshal, does not traverse deeply into the struct unless
// required in order to conver types.
//
// This should not be used for structs which implement json.Unmarshaler as they
// may encode their keys in arbitrary ways.
//
// t should be v.Type(), and t.Kind() must be reflect.Struct.
func (self ArgumentMap) decodeToStruct(t reflect.Type, v reflect.Value) error {
	parseTag := func(tag string) string {
		if idx := strings.Index(tag, ","); idx != -1 {
			return tag[:idx]
		} else {
			return tag
		}
	}
	for fnum := 0; fnum < t.NumField(); fnum++ {
		field := t.Field(fnum)
		if !isExportedName(field.Name) {
			continue
		}
		name := parseTag(field.Tag.Get("json"))
		if name == "-" {
			continue
		} else if name == "" {
			name = field.Name
		}
		if mapValue, ok := self[name]; ok {
			val := reflect.ValueOf(mapValue)
			if val.Type().AssignableTo(field.Type) {
				v.Field(fnum).Set(val)
			} else {
				if b, err := json.Marshal(mapValue); err != nil {
					return err
				} else {
					targV := reflect.New(field.Type)
					targ := targV.Interface()
					if err := json.Unmarshal(b, targ); err != nil {
						return err
					}
					v.Field(fnum).Set(targV)
				}
			}
		}
	}
	return nil
}

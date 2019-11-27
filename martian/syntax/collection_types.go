// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

// AST entries for array and map types.

package syntax

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type (
	// An array of values with a given type.
	ArrayType struct {
		Elem Type
		Dim  int16
	}

	// A dictionary with string keys, and values of a given type.
	TypedMapType struct {
		Elem Type
	}
)

// baseType returns the inner type after stripping off map or array wrappers.
func baseType(t Type) Type {
	switch t := t.(type) {
	case *ArrayType:
		return baseType(t.Elem)
	case *TypedMapType:
		return baseType(t.Elem)
	}
	return t
}

func (s *ArrayType) GetId() TypeId {
	id := s.Elem.GetId()
	id.ArrayDim += s.Dim
	return id
}
func (s *ArrayType) IsFile() FileKind {
	ft := s.Elem.IsFile()
	if ft == KindIsFile {
		return KindIsDirectory
	}
	return ft
}
func (s *ArrayType) IsAssignableFrom(other Type, lookup *TypeLookup) error {
	if s == other {
		return nil
	}
	switch other := other.(type) {
	case *nullType:
		return nil
	case *ArrayType:
		if err := s.Elem.IsAssignableFrom(other.Elem, lookup); err != nil {
			return &IncompatibleTypeError{
				Message: "incompatible array types",
				Reason:  err,
			}
		} else if other.Dim != s.Dim {
			return &IncompatibleTypeError{
				Message: fmt.Sprintf(
					"array dimension mismatch (%d vs %d)",
					s.Dim, other.Dim),
			}
		}
		return nil
	default:
		return &IncompatibleTypeError{
			Message: fmt.Sprintf(
				"cannot assign %s to an array value",
				other.GetId().str()),
		}
	}
}
func (s *ArrayType) IsValidExpression(exp Exp, pipeline *Pipeline, ast *Ast) error {
	switch exp := exp.(type) {
	case *RefExp:
		if tname, _, err := exp.resolveType(ast, pipeline); err != nil {
			return err
		} else if tname.ArrayDim < 1 {
			return &IncompatibleTypeError{
				Message: "ReferenceError: binding is not an array.",
			}
		} else if tname.ArrayDim != s.Dim {
			return &IncompatibleTypeError{
				Message: fmt.Sprintf(
					"ReferenceError: bound array dimension mismatch (%d, expected %d).",
					tname.ArrayDim, s.Dim),
			}
		} else if t := ast.TypeTable.Get(tname); t == nil {
			return &IncompatibleTypeError{
				Message: "Unknown type " + tname.String(),
			}
		} else if err := s.IsAssignableFrom(t, &ast.TypeTable); err != nil {
			return &IncompatibleTypeError{
				Message: "ReferenceError: incompatible types",
				Reason:  err,
			}
		} else {
			return nil
		}
	case *SplitExp:
		return isValidSplit(s, exp, pipeline, ast)
	case *NullExp:
		return nil
	case *ArrayExp:
		var errs ErrorList
		for i, subexp := range exp.Value {
			var err error
			if s.Dim == 1 {
				err = s.Elem.IsValidExpression(subexp, pipeline, ast)
			} else {
				t := ArrayType{
					Dim:  s.Dim - 1,
					Elem: s.Elem,
				}
				err = t.IsValidExpression(subexp, pipeline, ast)
			}
			if err != nil {
				errs = append(errs, &IncompatibleTypeError{
					Message: fmt.Sprintf("BindingError: array element %d", i),
					Reason:  err,
				})
			}
		}
		return errs.If()
	default:
		return &IncompatibleTypeError{
			Message: fmt.Sprintf(
				"BindingError: cannot assign %s to %s",
				exp.getKind(), s.GetId().str()),
		}
	}
}
func (s *ArrayType) CheckEqual(other Type) error {
	if other, ok := other.(*ArrayType); !ok {
		return &IncompatibleTypeError{
			Message: other.GetId().str() + " is not an array",
		}
	} else if err := s.Elem.CheckEqual(other.Elem); err != nil {
		return &IncompatibleTypeError{
			Message: "array type",
			Reason:  err,
		}
	} else if s.Dim != other.Dim {
		return &IncompatibleTypeError{
			Message: "array dimension mismatch",
		}
	} else {
		return nil
	}
}

func isNullBytes(data json.RawMessage) bool {
	return len(data) == 4 && data[3] == 'l' && data[2] == 'l' && data[1] == 'u' && data[0] == 'n'
}

type unmarshalError struct {
	Message string
	Reason  error
}

func (err *unmarshalError) Error() string {
	return err.Message
}

func (err *unmarshalError) Unwrap() error {
	return err.Reason
}

func (err *unmarshalError) writeTo(w stringWriter) {
	mustWriteString(w, err.Message)
}

func attemptJsonUnmarshal(val json.RawMessage, dest interface{}, expect string) error {
	if err := json.Unmarshal(val, dest); err != nil {
		var msg strings.Builder
		if len(val) > 35 {
			msg.Grow(33 + len(expect) + len("value '' cannot be parsed as "))
			mustWriteString(&msg, "value '")
			mustWrite(&msg, val[:15])
			mustWriteString(&msg, "...")
			mustWrite(&msg, val[len(val)-15:])
		} else if len(val) > 0 {
			msg.Grow(len(val) + len(expect) + len("value '' cannot be parsed as "))
			mustWriteString(&msg, "value '")
			mustWrite(&msg, val)
		} else {
			msg.Grow(len(expect) + len("empty string cannot be parsed as "))
			mustWriteString(&msg, "empty string cannot be parsed as ")
		}
		mustWriteString(&msg, "' cannot be parsed as ")
		mustWriteString(&msg, expect)
		return &unmarshalError{
			Message: msg.String(),
			Reason:  err,
		}
	}
	return nil
}

func (s *ArrayType) CanFilter() bool {
	return s.Elem.CanFilter()
}
func (s *ArrayType) IsValidJson(data json.RawMessage,
	alarms *strings.Builder,
	lookup *TypeLookup) error {
	if isNullBytes(data) {
		return nil
	}
	var arr []json.RawMessage
	if err := attemptJsonUnmarshal(data, &arr, "an array"); err != nil {
		return err
	}
	var subtype Type
	if s.Dim > 1 {
		id := s.GetId()
		id.ArrayDim--
		subtype = lookup.Get(id)
	} else {
		subtype = s.Elem
	}
	var errs ErrorList
	for i, element := range arr {
		if err := subtype.IsValidJson(element, alarms, lookup); err != nil {
			errs = append(errs, &IncompatibleTypeError{
				Message: "element " + strconv.Itoa(i),
				Reason:  err,
			})
		}
	}
	return errs.If()
}

func sameSlice(a, b json.RawMessage) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	return &a[0] == &b[0]
}

func (s *ArrayType) FilterJson(data json.RawMessage, lookup *TypeLookup) (json.RawMessage, bool, error) {
	if isNullBytes(data) || !s.CanFilter() {
		return data, false, nil
	}
	if s.Dim > 0 {
		var arr []json.RawMessage
		if err := json.Unmarshal(data, &arr); err != nil {
			return data, true, err
		}
		if len(arr) == 0 {
			return data, false, nil
		}
		var errs ErrorList
		fatal := false

		var buf bytes.Buffer
		buf.Grow(len(data))
		buf.WriteRune('[')
		if s.Dim == 1 {
			different := false
			for i, m := range arr {
				if i != 0 {
					buf.WriteRune(',')
				}
				fm, f, err := s.Elem.FilterJson(m, lookup)
				if !sameSlice(fm, m) {
					different = true
				}
				buf.Write(fm)
				if err != nil {
					errs = append(errs, err)
					fatal = fatal || f
				}
			}
			if !different {
				return data, fatal, errs.If()
			}
		} else {
			aType := ArrayType{
				Elem: s.Elem,
				Dim:  s.Dim - 1,
			}
			different := false
			for i, m := range arr {
				if i != 0 {
					buf.WriteRune(',')
				}
				fm, f, err := aType.FilterJson(m, lookup)
				if !sameSlice(fm, m) {
					different = true
				}
				arr[i] = fm
				if err != nil {
					errs = append(errs, err)
					fatal = fatal || f
				}
			}
			if !different {
				return data, fatal, errs.If()
			}
		}
		buf.WriteRune(']')
		return buf.Bytes(), fatal, errs.If()
	} else {
		return s.Elem.FilterJson(data, lookup)
	}
}

func (s *TypedMapType) GetId() TypeId {
	id := s.Elem.GetId()
	return TypeId{
		Tname:  id.Tname,
		MapDim: 1 + id.ArrayDim,
	}
}
func (s *TypedMapType) IsFile() FileKind {
	switch s.Elem.IsFile() {
	case KindIsNotFile:
		return KindIsNotFile
	case KindIsDirectory, KindIsFile:
		return KindIsDirectory
	}
	return KindMayContainPaths
}
func (s *TypedMapType) IsAssignableFrom(other Type, lookup *TypeLookup) error {
	if s == other {
		return nil
	}
	switch other := other.(type) {
	case *nullType:
		return nil
	case *TypedMapType:
		if err := s.Elem.IsAssignableFrom(other.Elem, lookup); err != nil {
			return &IncompatibleTypeError{
				Message: "incompatible map types",
				Reason:  err,
			}
		}
		return nil
	case *StructType:
		// A struct with members which are all compatible with the types
		// in this map is allowed.
		var errs ErrorList
		for _, member := range other.Members {
			if err := s.Elem.IsAssignableFrom(
				lookup.Get(member.Tname), lookup); err != nil {
				errs = append(errs, &IncompatibleTypeError{
					Message: fmt.Sprintf("field %s of %s",
						member.Id, other.Id),
					Reason: err,
				})
			}
		}
		return errs.If()
	default:
		return &IncompatibleTypeError{
			Message: fmt.Sprintf(
				"cannot assign %s to a typed map value",
				other.GetId().str()),
		}
	}
}
func (s *TypedMapType) IsValidExpression(exp Exp, pipeline *Pipeline, ast *Ast) error {
	switch exp := exp.(type) {
	case *RefExp:
		if tname, _, err := exp.resolveType(ast, pipeline); err != nil {
			return err
		} else if tname.MapDim == 0 {
			return &IncompatibleTypeError{
				Message: "binding is not a typed map.",
			}
		} else if t := ast.TypeTable.Get(tname); t == nil {
			return &IncompatibleTypeError{
				Message: "Unknown type " + tname.Tname,
			}
		} else if err := s.IsAssignableFrom(t, &ast.TypeTable); err != nil {
			return &IncompatibleTypeError{
				Message: "incompatible types",
				Reason:  err,
			}
		} else {
			return nil
		}
	case *SplitExp:
		return isValidSplit(s, exp, pipeline, ast)
	case *NullExp:
		return nil
	case *MapExp:
		if exp.Kind != KindMap {
			return &IncompatibleTypeError{
				Message: "cannot assign struct literal to map",
			}
		}
		var errs ErrorList
		isDir := (s.IsFile() == KindIsDirectory)
		for key, subexp := range exp.Value {
			if err := s.Elem.IsValidExpression(subexp, pipeline, ast); err != nil {
				errs = append(errs, &IncompatibleTypeError{
					Message: "map key " + key,
					Reason:  err,
				})
			}
			if isDir {
				if err := IsLegalUnixFilename(key); err != nil {
					errs = append(errs, &IncompatibleTypeError{
						Message: "key " + key,
						Reason:  err,
					})
				}
			}
		}
		return errs.If()
	default:
		return &IncompatibleTypeError{
			Message: fmt.Sprintf(
				"cannot assign %s to %s",
				exp.getKind(), s.GetId().str()),
		}
	}
}
func (s *TypedMapType) CheckEqual(other Type) error {
	if other, ok := other.(*TypedMapType); !ok {
		return &IncompatibleTypeError{
			Message: other.GetId().str() + " is not a typed map",
		}
	} else if err := s.Elem.CheckEqual(other.Elem); err != nil {
		return &IncompatibleTypeError{
			Message: "map type",
			Reason:  err,
		}
	} else {
		return nil
	}
}
func (s *TypedMapType) CanFilter() bool {
	return s.Elem.CanFilter()
}
func (s *TypedMapType) IsValidJson(data json.RawMessage,
	alarms *strings.Builder,
	lookup *TypeLookup) error {
	if isNullBytes(data) {
		return nil
	}
	var m map[string]json.RawMessage
	if err := attemptJsonUnmarshal(data, &m, "a map"); err != nil {
		return err
	}
	subtype := s.Elem
	isDir := (s.IsFile() == KindIsDirectory)
	var errs ErrorList
	for k, element := range m {
		if err := subtype.IsValidJson(element, alarms, lookup); err != nil {
			errs = append(errs, &IncompatibleTypeError{
				Message: "key " + k,
				Reason:  err,
			})
		}
		if isDir {
			if err := IsLegalUnixFilename(k); err != nil {
				errs = append(errs, &IncompatibleTypeError{
					Message: "key " + k,
					Reason:  err,
				})
			}
		}
	}
	return errs.If()
}
func (s *TypedMapType) FilterJson(data json.RawMessage, lookup *TypeLookup) (json.RawMessage, bool, error) {
	if isNullBytes(data) || !s.CanFilter() {
		return data, false, nil
	}
	var arr map[string]json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return data, true, err
	}
	if len(arr) == 0 {
		return data, false, nil
	}
	var errs ErrorList
	fatal := false
	first := true
	var buf bytes.Buffer
	buf.Grow(len(data))
	buf.WriteRune('{')
	different := false
	for i, m := range arr {
		if first {
			first = false
		} else {
			buf.WriteRune(',')
		}
		if b, err := json.Marshal(i); err != nil {
			errs = append(errs, err)
			buf.WriteRune('"')
			buf.WriteString(i)
			buf.WriteRune('"')
		} else {
			buf.Write(b)
		}
		buf.WriteRune(':')
		fm, f, err := s.Elem.FilterJson(m, lookup)
		if !sameSlice(fm, m) {
			different = true
		}
		buf.Write(fm)
		if err != nil {
			errs = append(errs, err)
			fatal = fatal || f
		}
	}
	if !different {
		return data, fatal, errs.If()
	}
	buf.WriteRune('}')
	return buf.Bytes(), fatal, errs.If()
}

func (err *IncompatibleTypeError) Error() string {
	if e := err.Reason; e != nil {
		return err.Message + ": " + e.Error()
	} else {
		return err.Message
	}
}

func (err *IncompatibleTypeError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Reason
}

func (err *IncompatibleTypeError) writeTo(w stringWriter) {
	if e := err.Reason; e == nil {
		mustWriteString(w, err.Message)
	} else {
		mustWriteString(w, err.Message)
		mustWriteString(w, ": ")
		if ew, ok := e.(errorWriter); ok {
			ew.writeTo(w)
		} else {
			mustWriteString(w, e.Error())
		}
	}
}

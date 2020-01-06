// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// AST entries for types.

package syntax

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type (
	FileKind int

	Type interface {
		GetId() TypeId
		// Returns whether this type represents a file or directory in the stage
		// outputs.
		IsFile() FileKind

		// For arrays or typed maps, the value type.  Otherwise nil.
		ElementType() Type

		// IsAssignableFrom returns nil if a parameter can accept a
		// value of the given type.
		//
		// Otherwise it will return a *IncompatibleTypeError indicating
		// the reason for the incompatibility.
		IsAssignableFrom(other Type, lookup *TypeLookup) error

		// IsValidExpression returns nil if an expression is valid for
		// assignment to a value of this type.
		IsValidExpression(exp Exp, pipeline *Pipeline, ast *Ast) error

		// CheckEqual returns non-nil if the types are not equal
		CheckEqual(other Type) error

		// CanFilter returns true if the result of FilterJson can be a subset
		// of the input.
		CanFilter() bool

		// CheckJson returns true if the given json message is a superset of
		// what is requried for the type, and a non-nil error if the given json
		// message cannot be parsed as this type.  If it can, but is malformed,
		// non-fatal errors are written to the given strings.Builder.
		IsValidJson(json.RawMessage, *strings.Builder, *TypeLookup) error

		// Returns a version of the input json with any extra struct fields
		// removed.
		FilterJson(json.RawMessage, *TypeLookup) (json.RawMessage, bool, error)
	}

	TypeId struct {
		// The name of the base type.
		Tname string
		// If positive, this is an array Tname[]
		ArrayDim int16
		// If positive, this is a map<Tname[]> where the dimension of the
		// inner array is MapDim-1.
		MapDim int16
	}

	// Used to resolve expressions, but not a legal parameter type.
	// null can be assigned to anyting, but nothing can be assigned to null.
	nullType struct{}

	// IncompatibleTypeError is returned for Type.IsAssignableFrom()
	IncompatibleTypeError struct {
		Message string
		Reason  error
	}
)

const (
	// Types which do not refer to files.
	KindIsNotFile FileKind = iota
	// Types which do not represent files formally, but may contain paths.
	// Such files do not get copied to the final output directory but are still
	// relevent for VDR.
	KindMayContainPaths FileKind = iota
	// Types which refer directly to files.
	KindIsFile FileKind = iota
	// Types which represent directories which contain files.
	KindIsDirectory FileKind = iota
)

func (id TypeId) str() string {
	return id.String()
}

func (id *TypeId) writeTo(w stringWriter) {
	if id.MapDim > 0 {
		mustWriteString(w, `map<`)
	}
	mustWriteString(w, id.Tname)
	if id.MapDim > 0 {
		for i := int16(1); i < id.MapDim; i++ {
			mustWriteString(w, `[]`)
		}
		mustWriteRune(w, '>')
	}
	for i := int16(0); i < id.ArrayDim; i++ {
		mustWriteString(w, `[]`)
	}
}

// MarshalText encodes the type name as UTF-8-encoded text and returns the
// result.
func (id *TypeId) MarshalText() ([]byte, error) {
	if id.ArrayDim == 0 && id.MapDim == 0 {
		return []byte(id.Tname), nil
	}
	var buf bytes.Buffer
	buf.Grow(id.strlen())
	err := id.encodeText(&buf)
	return buf.Bytes(), err
}

func (id *TypeId) UnmarshalText(b []byte) error {
	var newId TypeId
	for len(b) > 2 && b[len(b)-1] == ']' && b[len(b)-2] == '[' {
		newId.ArrayDim++
		b = b[:len(b)-2]
	}
	if len(b) > 5 && b[len(b)-1] == '>' &&
		b[3] == '<' && b[2] == 'p' && b[1] == 'a' && b[0] == 'm' {
		newId.MapDim++
		b = b[:len(b)-1]
		b = b[4:]
		for len(b) > 2 && b[len(b)-1] == ']' && b[len(b)-2] == '[' {
			newId.MapDim++
			b = b[:len(b)-2]
		}
	}
	newId.Tname = string(b)
	*id = newId
	if strings.ContainsAny(newId.Tname, "[]<> ") {
		return fmt.Errorf("invalid type name %s", newId.Tname)
	}
	return nil
}

func (id TypeId) EncodeJSON(buf *bytes.Buffer) error {
	if _, err := buf.WriteRune('"'); err != nil {
		return err
	}
	if err := id.encodeText(buf); err != nil {
		return err
	}
	_, err := buf.WriteRune('"')
	return err
}

func (id *TypeId) encodeText(buf *bytes.Buffer) error {
	if id.MapDim > 0 {
		buf.WriteString(`map<`)
	}
	buf.WriteString(id.Tname)
	if id.MapDim > 0 {
		for i := int16(1); i < id.MapDim; i++ {
			buf.WriteString(`[]`)
		}
		buf.WriteRune('>')
	}
	for i := int16(0); i < id.ArrayDim; i++ {
		buf.WriteString(`[]`)
	}
	return nil
}

func (id TypeId) jsonSizeEstimate() int {
	return 2 + id.strlen()
}

// String returns a string representation of the type ID in a human-readable
// form.
func (id *TypeId) String() string {
	if id.ArrayDim == 0 && id.MapDim == 0 {
		return id.Tname
	} else if id.ArrayDim == 1 && id.MapDim == 0 {
		// short-circuit common case
		return id.Tname + "[]"
	}
	var buf strings.Builder
	buf.Grow(id.strlen())
	if id.MapDim > 0 {
		buf.WriteString(`map<`)
	}
	buf.WriteString(id.Tname)
	if id.MapDim > 0 {
		for i := int16(1); i < id.MapDim; i++ {
			buf.WriteString(`[]`)
		}
		buf.WriteRune('>')
	}
	for i := int16(0); i < id.ArrayDim; i++ {
		buf.WriteString(`[]`)
	}
	return buf.String()
}

// GoString returns a string representation of the type ID in a human-readable
// form.
func (id *TypeId) GoString() string {
	return id.String()
}

func (id *TypeId) strlen() int {
	length := len(id.Tname) + 2*int(id.ArrayDim)
	if id.MapDim > 0 {
		length += 5 + 2*(int(id.MapDim)-1)
	}
	return length
}

func (k *FileKind) MarshalJSON() ([]byte, error) {
	switch *k {
	case KindIsNotFile, KindMayContainPaths:
		return []byte("false"), nil
	case KindIsFile:
		return []byte("true"), nil
	case KindIsDirectory:
		return []byte(`"directory"`), nil
	default:
		return nil, fmt.Errorf("invalid value %v", *k)
	}
}

func (k *FileKind) String() string {
	switch *k {
	case KindIsFile:
		return "file"
	case KindIsDirectory:
		return "directory"
	default:
		return "non-file"
	}
}

func (s *nullType) GetId() TypeId {
	return TypeId{Tname: KindNull}
}
func (s *nullType) IsFile() FileKind { return KindIsNotFile }
func (*nullType) ElementType() Type  { return nil }
func (s *nullType) IsAssignableFrom(other Type, lookup *TypeLookup) error {
	panic("invalid type")
}
func (s *nullType) IsValidExpression(exp Exp, pipeline *Pipeline, ast *Ast) error {
	panic("invalid type")
}
func (s *nullType) CheckEqual(other Type) error {
	if _, ok := other.(*nullType); ok {
		return nil
	}
	return &IncompatibleTypeError{
		Message: other.GetId().str() + " is not null",
	}
}
func (s *nullType) CanFilter() bool {
	return false
}
func (s *nullType) IsValidJson(data json.RawMessage, _ *strings.Builder, _ *TypeLookup) error {
	if !isNullBytes(data) {
		return fmt.Errorf("expected null")
	}
	return nil
}
func (s *nullType) FilterJson(data json.RawMessage, _ *TypeLookup) (json.RawMessage, bool, error) {
	return []byte("null"), false, s.IsValidJson(data, nil, nil)
}

func isValidSplit(s Type, exp *SplitExp, pipeline *Pipeline, ast *Ast) error {
	if exp == nil {
		return fmt.Errorf("cannot split on null")
	} else if exp.Value == nil {
		return nil
	}
	var errs ErrorList
	switch inner := exp.Value.(type) {
	case *ArrayExp:
		for i, subexp := range inner.Value {
			if err := s.IsValidExpression(subexp, pipeline, ast); err != nil {
				errs = append(errs, &IncompatibleTypeError{
					Message: fmt.Sprintf("split element %d", i),
					Reason:  err,
				})
			}
		}
	case *MapExp:
		for i, subexp := range inner.Value {
			if err := s.IsValidExpression(subexp, pipeline, ast); err != nil {
				errs = append(errs, &IncompatibleTypeError{
					Message: fmt.Sprintf("split key %s", i),
					Reason:  err,
				})
			}
		}
	case *RefExp:
		tname, _, err := inner.resolveType(ast, pipeline)
		if err != nil {
			return err
		} else if tname.ArrayDim > 0 {
			tname.ArrayDim--
		} else if tname.MapDim > 0 {
			tname.ArrayDim = tname.MapDim - 1
			tname.MapDim = 0
		} else {
			return &IncompatibleTypeError{
				Message: "binding is not a collection",
			}
		}
		if t := ast.TypeTable.Get(tname); t == nil {
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
	}
	return errs.If()
}

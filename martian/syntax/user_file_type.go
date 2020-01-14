// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// A user-defined file type.
type UserType struct {
	Node AstNode
	Id   string
}

func (*UserType) getDec() {}

func (s *UserType) GetId() TypeId     { return TypeId{Tname: s.Id} }
func (s *UserType) IsFile() FileKind  { return KindIsFile }
func (*UserType) ElementType() Type   { return nil }
func (s *UserType) getNode() *AstNode { return &s.Node }
func (s *UserType) File() *SourceFile { return s.Node.Loc.File }

func (s *UserType) inheritComments() bool     { return false }
func (s *UserType) getSubnodes() []AstNodable { return nil }

func (s *UserType) IsAssignableFrom(other Type, _ *TypeLookup) error {
	if s == other {
		return nil
	}
	switch t := other.(type) {
	case *nullType:
		return nil
	case *BuiltinType:
		// Allow coercion of string or generic file type to specific file type.
		if t.Id == KindFile || t.Id == KindString {
			return nil
		}
		return &IncompatibleTypeError{
			Message: fmt.Sprintf("%s cannot be assigned to %s",
				t.Id, s.Id),
		}
	case *UserType:
		if s.Id == t.Id {
			return nil
		}
		return &IncompatibleTypeError{
			Message: fmt.Sprintf(
				"conversion between user-defined file types %s and %s is not allowed",
				t.Id, s.Id),
		}
	case *ArrayType:
		return &IncompatibleTypeError{
			Message: fmt.Sprintf(
				"cannot assign array %s to singleton %s",
				t.Elem.GetId().str(), s.Id),
		}
	case *TypedMapType:
		return &IncompatibleTypeError{
			Message: fmt.Sprintf(
				"cannot assign map<%s> to singleton %s",
				t.Elem.GetId().str(), s.Id),
		}
	default:
		return &IncompatibleTypeError{
			Message: fmt.Sprintf(
				"%T type %s cannot be assigned to user-defined file type %s",
				t, t.GetId().str(), s.Id),
		}
	}
}
func (s *UserType) IsValidExpression(exp Exp, pipeline *Pipeline, ast *Ast) error {
	switch exp := exp.(type) {
	case *RefExp:
		if tname, _, err := exp.resolveType(ast, pipeline); err != nil {
			return err
		} else if tname.ArrayDim != 0 {
			return &IncompatibleTypeError{
				Message: "ReferenceError: binding is an array",
			}
		} else if tname.MapDim != 0 {
			return &IncompatibleTypeError{
				Message: "ReferenceError: binding is a map",
			}
		} else if t := ast.TypeTable.Get(tname); t == nil {
			return &IncompatibleTypeError{
				Message: "Unknown type " + tname.Tname,
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
	case *StringExp:
		return nil
	default:
		return &IncompatibleTypeError{
			Message: fmt.Sprintf("cannot assign %s to %s", exp.getKind(), s.Id),
		}
	}
}
func (s *UserType) CheckEqual(other Type) error {
	if other, ok := other.(*UserType); !ok {
		return &IncompatibleTypeError{
			Message: other.Id + " is not a user-defined file type",
		}
	} else if s.Id != other.Id {
		return &IncompatibleTypeError{
			Message: other.Id + " != " + s.Id,
		}
	} else {
		return nil
	}
}
func (s *UserType) CanFilter() bool {
	return false
}
func (s *UserType) IsValidJson(data json.RawMessage,
	alarms *strings.Builder,
	lookup *TypeLookup) error {
	if isNullBytes(data) {
		return nil
	}
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' &&
		!bytes.ContainsAny(data, "\"\\") {
		// easy case, no escapes or quotes
		return nil
	}
	// For backwards compatibility we need to accept everything here.
	var st string
	if err := attemptJsonUnmarshal(data, &st, "a string"); err != nil {
		mustWriteString(alarms, "parsing user-defined file type: ")
		mustWriteString(alarms, err.Error())
		mustWriteRune(alarms, '\n')
	}
	return nil
}
func (s *UserType) FilterJson(data json.RawMessage, _ *TypeLookup) (json.RawMessage, bool, error) {
	if isNullBytes(data) {
		return data, false, nil
	}
	var tmp string
	err := json.Unmarshal(data, &tmp)
	// For consistency with IsValidJson, don't treat any errors as fatal.
	return data, false, err
}

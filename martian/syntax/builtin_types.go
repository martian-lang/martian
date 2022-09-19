// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// One of the built-in types.
type BuiltinType struct {
	Id string
}

var (
	builtinString = BuiltinType{KindString}
	builtinInt    = BuiltinType{KindInt}
	builtinFloat  = BuiltinType{KindFloat}
	builtinBool   = BuiltinType{KindBool}
	builtinPath   = BuiltinType{KindPath}
	builtinFile   = BuiltinType{KindFile}
	builtinMap    = BuiltinType{KindMap}
	builtinNull   nullType
)

var builtinTypes = [...]*BuiltinType{
	&builtinString,
	&builtinInt,
	&builtinFloat,
	&builtinBool,
	&builtinPath,
	&builtinFile,
	&builtinMap,
}

func (s *BuiltinType) TypeId() TypeId    { return TypeId{Tname: s.Id} }
func (s *BuiltinType) IsDirectory() bool { return false }
func (s *BuiltinType) IsFile() FileKind {
	switch s.Id {
	case KindPath, KindFile:
		return KindIsFile
	case KindString, KindMap:
		return KindMayContainPaths
	default:
		return KindIsNotFile
	}
}
func (*BuiltinType) ElementType() Type { return nil }
func (s *BuiltinType) IsAssignableFrom(other Type, _ *TypeLookup) error {
	switch other := other.(type) {
	case *nullType:
		return nil
	case *BuiltinType:
		if s == other {
			return nil
		}
		// Allow coercion of string to generic file type
		if other.Id == s.Id ||
			other.Id == KindString && (s.Id == KindFile || s.Id == KindPath) ||
			other.Id == KindInt && s.Id == KindFloat {
			return nil
		}
		return &IncompatibleTypeError{
			Message: fmt.Sprintf("%s cannot be assigned to %s",
				other.Id, s.Id),
		}
	case *UserType:
		if s.Id == KindFile || s.Id == KindString {
			// Allow coercion from specific file types to
			// generic file type.
			return nil
		}
		return &IncompatibleTypeError{
			Message: fmt.Sprintf("%s cannot be assigned to %s",
				other.Id, s.Id),
		}
	case *StructType:
		if s.Id == KindMap {
			return nil
		}
		return &IncompatibleTypeError{
			Message: fmt.Sprintf("struct type %s cannot be assigned to %s",
				other.Id, s.Id),
		}
	case *ArrayType:
		return &IncompatibleTypeError{
			Message: fmt.Sprintf("cannot assign array of %s to singleton %s",
				other.Elem.TypeId().str(), s.Id),
		}
	case *TypedMapType:
		if s.Id == KindMap {
			return nil
		}
		return &IncompatibleTypeError{
			Message: fmt.Sprintf("map<%s> cannot be assigned to %s",
				other.Elem.TypeId().str(), s.Id),
		}
	default:
		return &IncompatibleTypeError{
			Message: fmt.Sprintf("%T type %s cannot be assigned to %s",
				other, other.TypeId().str(), s.Id),
		}
	}
}
func (s *BuiltinType) IsValidExpression(exp Exp, pipeline *Pipeline, ast *Ast) error {
	switch exp := exp.(type) {
	case *RefExp:
		if tname, _, err := exp.resolveType(ast, pipeline); err != nil {
			return err
		} else if tname.ArrayDim > 0 {
			return &wrapError{
				innerError: &IncompatibleTypeError{
					Message: "ReferenceError: binding is an array; expected " + s.Id,
				},
				loc: exp.Node.Loc,
			}
		} else if tname.MapDim != 0 {
			return &wrapError{
				innerError: &IncompatibleTypeError{
					Message: "ReferenceError: binding is a map; expected " + s.Id,
				},
				loc: exp.Node.Loc,
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
	case *DisabledExp:
		return s.IsValidExpression(exp.Value, pipeline, ast)
	case *NullExp:
		return nil
	case *StringExp:
		if s.Id == KindString || s.Id == KindFile || s.Id == KindPath {
			return nil
		} else {
			return &IncompatibleTypeError{
				Message: fmt.Sprintf("cannot assign %s to %s", exp.getKind(), s.Id),
			}
		}
	case *IntExp:
		if s.Id == KindInt || s.Id == KindFloat {
			return nil
		}
		return &IncompatibleTypeError{
			Message: fmt.Sprintf("cannot assign int to %s", s.Id),
		}
	case *FloatExp:
		if s.Id == KindFloat {
			return nil
		} else if s.Id == KindInt {
			if float64(int64(exp.Value)) == exp.Value {
				return nil
			} else {
				return &IncompatibleTypeError{
					Message: fmt.Sprintf("cannot assign %g to an integer", exp.Value),
				}
			}
		}
		return &IncompatibleTypeError{
			Message: fmt.Sprintf("cannot assign float to %s", s.Id),
		}
	case *MapExp:
		if k := exp.getKind(); k != KindMap || s.Id != KindMap {
			return &IncompatibleTypeError{
				Message: fmt.Sprintf("cannot assign %s literal to %s",
					exp.getKind(), s.Id),
			}
		} else if exp.HasRef() {
			refs := exp.FindRefs()
			return &IncompatibleTypeError{
				Message: fmt.Sprintf(
					"%s literal cannot be assigned to untyped map: "+
						"contains reference to %s",
					k,
					refs[0].GoString()),
			}
		}
		return nil
	default:
		if k := exp.getKind(); k != ExpKind(s.Id) {
			return &IncompatibleTypeError{
				Message: fmt.Sprintf("cannot assign %s to %s", k, s.Id),
			}
		}
		return nil
	}
}
func (s *BuiltinType) CheckEqual(other Type) error {
	if other, ok := other.(*BuiltinType); !ok {
		return &IncompatibleTypeError{
			Message: other.TypeId().str() + " is not a " + s.Id,
		}
	} else if other.Id != s.Id {
		return &IncompatibleTypeError{
			Message: other.Id + " is not a " + s.Id,
		}
	} else {
		return nil
	}
}
func (s *BuiltinType) CanFilter() bool {
	return s != nil && s.Id == KindInt
}
func (s *BuiltinType) IsValidJson(data json.RawMessage,
	_ *strings.Builder,
	lookup *TypeLookup) error {
	if isNullBytes(data) {
		return nil
	}
	switch s.Id {
	case KindString, KindPath, KindFile:
		if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' &&
			!bytes.ContainsAny(data, "\"\\") {
			// easy case, no escapes or quotes
			return nil
		}
		var s string
		return attemptJsonUnmarshal(data, &s, "a string")
	case KindInt:
		var i int64
		return attemptJsonUnmarshal(data, &i, "an integer")
	case KindFloat:
		var f float64
		return attemptJsonUnmarshal(data, &f, "a floating point number")
	case KindBool:
		var b bool
		return attemptJsonUnmarshal(data, &b, "a boolean value")
	case KindMap:
		var m map[string]json.RawMessage
		return attemptJsonUnmarshal(data, &m, "a map")
	default:
		panic("invalid builtin type " + s.Id)
	}
}
func (s *BuiltinType) FilterJson(data json.RawMessage, _ *TypeLookup) (json.RawMessage, bool, error) {
	data = bytes.TrimSpace(data)
	if isNullBytes(data) {
		return data, false, nil
	}
	switch s.Id {
	case KindString, KindPath, KindFile:
		var tmp string
		err := json.Unmarshal(data, &tmp)
		return data, err != nil, err
	case KindFloat:
		var tmp float64
		err := json.Unmarshal(data, &tmp)
		return data, err != nil, err
	case KindInt:
		var tmp int64
		if err := json.Unmarshal(data, &tmp); err != nil {
			var tmp float64
			if err := json.Unmarshal(data, &tmp); err != nil {
				return data, true, err
			}
			if i := int64(tmp); float64(i) != tmp {
				return data, true, err
			} else if b, jerr := json.Marshal(&i); jerr != nil {
				return data, true, err
			} else {
				return b, false, err
			}
		}
		return data, false, nil
	case KindBool:
		var tmp bool
		err := json.Unmarshal(data, &tmp)
		return data, err != nil, err
	case KindMap:
		var tmp map[string]json.RawMessage
		err := json.Unmarshal(data, &tmp)
		return data, err != nil, err
	default:
		panic("invalid builtin type " + s.Id)
	}
}

func (s *BuiltinType) String() string {
	return s.TypeId().str()
}

// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

// AST entry for struct types.

package syntax

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type (
	StructMember struct {
		Node  AstNode
		Tname TypeId
		// The key for this value in json.
		Id string
		// The name of the file or directory in the top-level outputs
		// directory to use for this output.
		//
		// If this is not set, the default is the Id, unless the type
		// is a user-defined file type, in which case it is Id.TypeName.
		OutName string
		// The name by which this value is labeled when printing outputs
		// to the console.
		Help      string
		isComplex bool
		isFile    FileKind
	}

	StructType struct {
		Node    AstNode
		Id      string
		Members []*StructMember
		Table   map[string]*StructMember
		isFile  FileKind
	}
)

func structFromCallable(callable Callable) *StructType {
	st := StructType{
		Node: *callable.getNode(),
		Id:   callable.GetId(),
	}
	if p := callable.GetOutParams(); p != nil && len(p.List) > 0 {
		st.Members = make([]*StructMember, len(p.List))
		for i, m := range p.List {
			st.Members[i] = &m.StructMember
		}
	}
	return &st
}
func (m *StructMember) setIsFile(k FileKind)    { m.isFile = k }
func (m *StructMember) IsFile() FileKind        { return m.isFile }
func (m *StructMember) getNode() *AstNode       { return &m.Node }
func (m *StructMember) File() *SourceFile       { return m.Node.Loc.File }
func (*StructMember) inheritComments() bool     { return false }
func (*StructMember) getSubnodes() []AstNodable { return nil }

// Gets the name used to refer to this parameter in outputs.
func (s *StructMember) GetOutName() string {
	return s.OutName
}

// Returns the default base filename for this output parameter, or an
// empty string if the parameter is not a file type.
func (s *StructMember) GetOutFilename() string {
	if fk := s.isFile; fk != KindIsFile && fk != KindIsDirectory {
		return ""
	} else if s.OutName != "" {
		return s.OutName
	} else if s.isComplex ||
		s.Tname.Tname == KindFile ||
		s.Tname.Tname == KindPath {
		return s.Id
	} else {
		return s.Id + "." + s.Tname.Tname
	}
}

// GetDisplayName returns the name by which this field is described when
// printing outputs.
func (s *StructMember) GetDisplayName() string {
	if s.Help != "" {
		return s.Help
	}
	return s.Id
}

func (*StructType) getDec()             {}
func (s *StructType) GetId() TypeId     { return TypeId{Tname: s.Id} }
func (s *StructType) IsFile() FileKind  { return s.isFile }
func (s *StructType) getNode() *AstNode { return &s.Node }
func (s *StructType) File() *SourceFile { return s.Node.Loc.File }

func (*StructType) inheritComments() bool { return false }
func (s *StructType) getSubnodes() []AstNodable {
	members := make([]AstNodable, 0, len(s.Members))
	for _, m := range s.Members {
		members = append(members, m)
	}
	return members
}

func (s *StructType) IsAssignableFrom(other Type, typeTable *TypeLookup) error {
	if s == other {
		return nil
	}
	switch t := other.(type) {
	case *nullType:
		return nil
	case *StructType:
		if t.Table == nil {
			panic("IsAssignableFrom called on AST before compilation")
		}
		var errs ErrorList
		for _, member := range s.Members {
			o := t.Table[member.Id]
			const msg = "struct %s cannot be used as %s: "
			if o == nil {
				errs = append(errs, &IncompatibleTypeError{
					Message: fmt.Sprintf(
						msg+"no member %s",
						t.Id, s.Id, member.Id),
				})
			} else if member.Tname.ArrayDim != o.Tname.ArrayDim {
				errs = append(errs, &IncompatibleTypeError{
					Message: fmt.Sprintf(
						msg+"member %s: differing array dimensions %d vs %d",
						t.Id, s.Id, member.Id, o.Tname.ArrayDim, member.Tname.ArrayDim),
				})
			} else if member.Tname.MapDim != o.Tname.MapDim {
				if o.Tname.MapDim == 0 {
					errs = append(errs, &IncompatibleTypeError{
						Message: fmt.Sprintf(
							msg+"member %s: not a map",
							t.Id, s.Id, member.Id),
					})
				} else if member.Tname.MapDim == 0 {
					errs = append(errs, &IncompatibleTypeError{
						Message: fmt.Sprintf(
							msg+"member %s: unexpected map",
							t.Id, s.Id, member.Id),
					})
				} else {
					errs = append(errs, &IncompatibleTypeError{
						Message: fmt.Sprintf(
							msg+"member %s: differing inner array dimensions %d vs %d",
							t.Id, s.Id, member.Id, o.Tname.MapDim-1, member.Tname.MapDim-1),
					})
				}
			} else if member.Tname != o.Tname {
				mt := typeTable.Get(member.Tname)
				ot := typeTable.Get(o.Tname)
				if err := mt.IsAssignableFrom(ot, typeTable); err != nil {
					errs = append(errs, &IncompatibleTypeError{
						Message: fmt.Sprintf(
							msg+"member %s",
							t.Id, s.Id, member.Id),
						Reason: err,
					})
				}
			}
		}
		return errs.If()
	default:
		return &IncompatibleTypeError{
			Message: fmt.Sprintf(
				"cannot assign non-struct type %s to struct %s",
				other.GetId().str(), s.Id),
		}
	}
}
func (s *StructType) IsValidExpression(exp Exp, pipeline *Pipeline, ast *Ast) error {
	switch exp := exp.(type) {
	case *RefExp:
		if tname, _, err := exp.resolveType(ast, pipeline); err != nil {
			return err
		} else if tname.ArrayDim != 0 {
			return &IncompatibleTypeError{
				Message: "binding is an array",
			}
		} else if tname.MapDim != 0 {
			return &IncompatibleTypeError{
				Message: "binding is a map",
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
		var errs ErrorList
		for _, member := range s.Members {
			if v, ok := exp.Value[member.Id]; !ok {
				errs = append(errs, &IncompatibleTypeError{
					Message: "missing value for struct field " + member.Id,
				})
			} else if err := ast.TypeTable.Get(member.Tname).IsValidExpression(
				v, pipeline, ast); err != nil {
				errs = append(errs, &IncompatibleTypeError{
					Message: "field " + member.Id,
					Reason:  err,
				})
			}
		}
		if len(exp.Value) > len(s.Members) {
			for key := range exp.Value {
				if _, ok := s.Table[key]; !ok {
					errs = append(errs, &IncompatibleTypeError{
						Message: "unexpected field " + key,
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

func (s *StructType) CheckEqual(other Type) error {
	if other, ok := other.(*StructType); !ok {
		return &IncompatibleTypeError{
			Message: other.GetId().str() + " is not a struct type",
		}
	} else {
		var errs ErrorList
		for _, member := range s.Members {
			if om := other.Table[member.Id]; om == nil {
				errs = append(errs, &IncompatibleTypeError{
					Message: "missing field: " + member.Id,
				})
			} else if om.Tname.Tname != member.Tname.Tname {
				errs = append(errs, &IncompatibleTypeError{
					Message: fmt.Sprint("field ", member.Id,
						": differing types"),
				})
			} else if member.Tname.ArrayDim != om.Tname.ArrayDim {
				errs = append(errs, &IncompatibleTypeError{
					Message: fmt.Sprint("field ", member.Id,
						": differing array dimension"),
				})
			} else if member.Tname.MapDim != om.Tname.MapDim {
				if member.Tname.MapDim == 0 || om.Tname.MapDim == 0 {
					errs = append(errs, &IncompatibleTypeError{
						Message: fmt.Sprint("field ", member.Id,
							": not a typed map"),
					})
				} else {
					errs = append(errs, &IncompatibleTypeError{
						Message: fmt.Sprint("field ", member.Id,
							": differing inner array dimension"),
					})
				}
			} else if om.Help != member.Help {
				errs = append(errs, &IncompatibleTypeError{
					Message: fmt.Sprint("field ", member.Id,
						": differing output display names"),
				})
			} else if om.OutName != member.OutName {
				errs = append(errs, &IncompatibleTypeError{
					Message: fmt.Sprint("field ", member.Id,
						": differing explicit output names"),
				})
			}
		}
		return errs.If()
	}
}
func (s *StructType) CanFilter() bool {
	return true
}
func (s *StructType) IsValidJson(data json.RawMessage,
	alarms *strings.Builder,
	lookup *TypeLookup) error {
	if isNullBytes(data) {
		return nil
	}
	var m map[string]json.RawMessage
	if err := attemptJsonUnmarshal(data, &m, "a map"); err != nil {
		return err
	}
	var errs ErrorList
	for _, member := range s.Members {
		t := lookup.Get(member.Tname)
		if t == nil {
			panic("unknown type " + member.Tname.String())
		}
		if element, ok := m[member.Id]; !ok {
			errs = append(errs, &IncompatibleTypeError{
				Message: "missing key: " + member.Id,
			})
		} else if err := t.IsValidJson(element, alarms, lookup); err != nil {
			errs = append(errs, &IncompatibleTypeError{
				Message: "key " + member.Id,
				Reason:  err,
			})
		}
	}
	return errs.If()
}
func (s *StructType) FilterJson(data json.RawMessage, lookup *TypeLookup) (json.RawMessage, bool, error) {
	if isNullBytes(data) {
		return data, false, nil
	}
	var arr map[string]json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return data, true, err
	}
	var errs ErrorList
	fatal := false
	var buf bytes.Buffer
	buf.Grow(len(data))
	if _, err := buf.WriteRune('{'); err != nil {
		panic(err)
	}
	different := len(arr) != len(s.Members)
	for i, m := range s.Members {
		if i != 0 {
			if _, err := buf.WriteRune(','); err != nil {
				panic(err)
			}
		}
		if b, err := json.Marshal(m.Id); err != nil {
			fatal = true
			errs = append(errs, err)
			if _, err := buf.WriteRune('"'); err != nil {
				panic(err)
			}
			if _, err := buf.WriteString(m.Id); err != nil {
				panic(err)
			}
			if _, err := buf.WriteRune('"'); err != nil {
				panic(err)
			}
		} else if _, err := buf.Write(b); err != nil {
			panic(err)
		}
		if _, err := buf.WriteRune(':'); err != nil {
			panic(err)
		}
		if b, ok := arr[m.Id]; !ok {
			errs = append(errs, &IncompatibleTypeError{
				Message: "missing field: " + m.Id,
			})
			fatal = true
			if _, err := buf.WriteString("null"); err != nil {
				panic(err)
			}
		} else if t := lookup.Get(m.Tname); t == nil {
			errs = append(errs, &IncompatibleTypeError{
				Message: "unknown type " + m.Tname.String() +
					" for field " + m.Id,
			})
			fatal = true
			if _, err := buf.WriteString("null"); err != nil {
				panic(err)
			}
		} else if t.CanFilter() {
			fm, f, err := t.FilterJson(b, lookup)
			if !different && !sameSlice(b, fm) {
				different = true
			}
			if err != nil {
				errs = append(errs, err)
				fatal = fatal || f
			}
			if _, err := buf.Write(fm); err != nil {
				panic(err)
			}
		} else {
			if _, err := buf.Write(b); err != nil {
				panic(err)
			}
		}
	}
	if !different {
		return data, fatal, errs.If()
	}
	if _, err := buf.WriteRune('}'); err != nil {
		panic(err)
	}
	return buf.Bytes(), fatal, errs.If()
}

type StructFieldError struct {
	Message    string
	InnerError error
}

func (err *StructFieldError) Error() string {
	if e := err.InnerError; e == nil {
		return err.Message
	} else {
		return err.Message + ": " + e.Error()
	}
}

func (err *StructFieldError) Unwrap() error {
	if err == nil {
		return err
	}
	return err.InnerError
}

func (err *StructFieldError) writeTo(w stringWriter) {
	if e := err.InnerError; e == nil {
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

// Evaluate the type for a field, possibly recursively.
//
// For example given,
//
//    struct Foo(
//        int bar,
//    )
//
//    struct Baz(
//        Foo boz
//    )
//
// then if called for the Baz type, field "boz" would give "Foo", "boz,bar"
// would give int, and so on.  If called for an array or map of Baz, the
// projection evaluates to an array or map of the appropriate type.
func fieldType(id TypeId, lookup *TypeLookup, field string) (TypeId, error) {
	if field == "" {
		return id, nil
	}
	if id.ArrayDim != 0 {
		inner, err := fieldType(TypeId{
			Tname:  id.Tname,
			MapDim: id.MapDim,
		}, lookup, field)
		inner.ArrayDim += id.ArrayDim
		return inner, err
	} else if id.MapDim != 0 {
		inner, err := fieldType(TypeId{
			Tname: id.Tname,
		}, lookup, field)
		if err != nil {
			return inner, err
		}
		if inner.MapDim != 0 {
			return inner, fmt.Errorf("invalid projection through nested maps")
		}
		inner.MapDim = id.MapDim + inner.ArrayDim
		inner.ArrayDim = 0
		return inner, err
	}
	t := lookup.Get(id)
	if t == nil {
		return id, fmt.Errorf("unknown type %s", id.String())
	}
	if t, ok := t.(*StructType); !ok {
		return id, fmt.Errorf("type %s is not a struct", id.String())
	} else {
		fieldParts := strings.SplitN(field, ".", 2)
		if m := t.Table[fieldParts[0]]; m == nil {
			return id, &StructFieldError{
				Message: "no field " + fieldParts[0] + " in struct " + id.Tname + " (evaluating " + field + ")",
			}
		} else if len(fieldParts) == 1 {
			return m.Tname, nil
		} else if inner, err := fieldType(m.Tname, lookup, fieldParts[1]); err != nil {
			return inner, &StructFieldError{
				Message:    "field " + fieldParts[0],
				InnerError: err,
			}
		} else {
			return inner, nil
		}
	}
}

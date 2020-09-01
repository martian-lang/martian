// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Value expressions, e.g. things which can be assigned to bindings.

package syntax

import "fmt"

// Kinds of value or reference expressions.  These include all of
// the builtin types as well as "array" and "null", and for references
// "self" and "call".
const (
	// Represents an array of expressions.
	KindArray  = ExpKind("array")
	KindSplit  = "split"
	KindMerge  = "merge"
	KindMap    = "map"
	KindFloat  = "float"
	KindInt    = "int"
	KindString = "string"
	KindBool   = "bool"
	KindNull   = "null"
	KindStruct = "struct"

	// A reference to the pipeline's inputs.
	KindSelf = "self"

	// A reference to another call in the pipeline.
	KindCall = "call"

	// Any file type, include the builtin "file" type or a user-defined
	// file type.
	KindFile = "file"

	// A file path.
	KindPath = "path"
)

type (
	// Represents the type of expression.  For most literal values, this
	// is the same as the type.
	ExpKind string

	// Interface Exp represents a value (either literal or reference)
	// which can be assigned to a binding (e.g. in a call or return
	// statement).
	Exp interface {
		fmt.GoStringer
		AstNodable
		getKind() ExpKind
		format(w stringWriter, prefix string)
		equal(other Exp) error
		// HasRef returns true if this expression or any sub-expression is a
		// reference.
		HasRef() bool
		// HasSplit returns true if this expression or any sub-expression is
		// a split.
		HasSplit() bool

		// Recursively searches the expression for references.
		FindRefs() []*RefExp

		// Recursively searches the expression for references, propagating
		// type information and appending the results to the given list.
		FindTypedRefs(list []*BoundReference, t Type, lookup *TypeLookup) ([]*BoundReference, error)

		// Evaluates a binding path through a literal expression.
		//
		// This handles projection - given an expression
		//
		//   {
		//     a: {b: "foo"},
		//     c: {
		//       "d":{e: "bar"},
		//       "f": {e: "baz"},
		//     },
		//     d: [{e:"bar"}, {e:"baz"}],
		//     f: STAGE.out,
		//   }
		//
		// we'd get
		//
		//   exp.BindingPath("a.b") -> "foo"
		//   exp.BindingPath("c.e") -> {"d":"bar", "f":"baz"}
		//   exp.BindingPath("d.e") -> ["bar", "baz"]
		//   exp.BindingPath("f.bar") -> STAGE.out.bar
		BindingPath(bindPath string,
			forks map[*CallStm]CollectionIndex) (Exp, error)
		resolveRefs(self, siblings map[string]*ResolvedBinding,
			lookup *TypeLookup) (Exp, error)
		filter(t Type, lookup *TypeLookup) (Exp, error)

		// Returns a json representation of the concrete value of a
		// literal expression.  For references, returns
		//
		//   {"__reference__": "Id.OutputId"}
		MarshalJSON() ([]byte, error)

		JsonWriter
		jsonSizeEstimator
	}

	// The ValExp interface is implemented by literal expressions.  These may
	// be collections, which may contain RefExp values
	ValExp interface {
		Exp
		val() interface{}
	}

	// Save boilerplate by using this to define the common methods.
	valExp struct {
		Node AstNode
	}

	// An ArrayExp represents a literal array of expressions.
	ArrayExp struct {
		valExp
		Value []Exp
	}

	// A MapExp represents a literal map of expressions.
	MapExp struct {
		valExp
		// Will be KindMap or KindStruct.
		Kind  ExpKind
		Value map[string]Exp
	}

	// A StringExp represents a string or file type literal.
	StringExp struct {
		valExp
		Value string
	}

	// A BoolExp represents a boolean value.
	BoolExp struct {
		valExp
		Value bool
	}

	// An IntExp represents an integer literal.
	IntExp struct {
		valExp
		Value int64
	}

	// A FloatExp represents a floating point literal.
	FloatExp struct {
		valExp
		Value float64
	}

	// A NullExp represents a literal null.
	NullExp struct {
		valExp
	}
)

func (e *valExp) getNode() *AstNode       { return &e.Node }
func (s *valExp) File() *SourceFile       { return s.Node.Loc.File }
func (s *valExp) Line() int               { return s.Node.Loc.Line }
func (*valExp) getSubnodes() []AstNodable { return nil }
func (s *valExp) inheritComments() bool   { return false }
func (*valExp) HasRef() bool              { return false }
func (*valExp) HasSplit() bool            { return false }
func (*valExp) FindRefs() []*RefExp       { return nil }

func (s *ArrayExp) getKind() ExpKind  { return KindArray }
func (s *MapExp) getKind() ExpKind    { return s.Kind }
func (s *StringExp) getKind() ExpKind { return KindString }
func (s *BoolExp) getKind() ExpKind   { return KindBool }
func (s *IntExp) getKind() ExpKind    { return KindInt }
func (s *FloatExp) getKind() ExpKind  { return KindFloat }
func (s *NullExp) getKind() ExpKind   { return KindNull }

func (s *ArrayExp) val() interface{}  { return s.Value }
func (s *MapExp) val() interface{}    { return s.Value }
func (s *StringExp) val() interface{} { return s.Value }
func (s *BoolExp) val() interface{}   { return s.Value }
func (s *IntExp) val() interface{}    { return s.Value }
func (s *FloatExp) val() interface{}  { return s.Value }
func (s *NullExp) val() interface{}   { return nil }

func (s *ArrayExp) getSubnodes() []AstNodable {
	subs := make([]AstNodable, 0, len(s.Value))
	for _, n := range s.Value {
		subs = append(subs, n)
	}
	return subs
}

func (s *MapExp) getSubnodes() []AstNodable {
	subs := make([]AstNodable, 0, len(s.Value))
	for _, n := range s.Value {
		subs = append(subs, n)
	}
	return subs
}

func (e *ArrayExp) HasRef() bool {
	if e == nil {
		return false
	}
	for _, exp := range e.Value {
		if exp.HasRef() {
			return true
		}
	}
	return false
}

func (e *ArrayExp) HasSplit() bool {
	if e == nil {
		return false
	}
	for _, exp := range e.Value {
		if exp.HasSplit() {
			return true
		}
	}
	return false
}

func (e *MapExp) HasRef() bool {
	if e == nil {
		return false
	}
	for _, exp := range e.Value {
		if exp.HasRef() {
			return true
		}
	}
	return false
}

func (e *MapExp) HasSplit() bool {
	if e == nil {
		return false
	}
	for _, exp := range e.Value {
		if exp.HasSplit() {
			return true
		}
	}
	return false
}

func (e *ArrayExp) FindRefs() []*RefExp {
	var result []*RefExp
	for _, v := range e.Value {
		r := v.FindRefs()
		if len(r) > 0 {
			if len(result) == 0 {
				result = r
			} else {
				result = append(result, r...)
			}
		}
	}
	return result
}

func (e *MapExp) FindRefs() []*RefExp {
	var result []*RefExp
	for _, v := range e.Value {
		r := v.FindRefs()
		if len(r) > 0 {
			if len(result) == 0 {
				result = r
			} else {
				result = append(result, r...)
			}
		}
	}
	return result
}
func (e *RefExp) FindRefs() []*RefExp {
	refs := []*RefExp{e}
	for _, i := range e.Forks {
		if m := i.IndexSource(); m != nil {
			if s, ok := m.(Exp); ok {
				refs = append(refs, s.FindRefs()...)
			}
		}
	}
	return refs
}

func (s *StringExp) Type() (Type, int) { return &builtinString, 0 }
func (s *BoolExp) Type() (Type, int)   { return &builtinBool, 0 }
func (s *IntExp) Type() (Type, int)    { return &builtinInt, 0 }
func (s *FloatExp) Type() (Type, int)  { return &builtinFloat, 0 }
func (s *NullExp) Type() (Type, int)   { return &builtinNull, 0 }

type skipWalkError struct{}

func (skipWalkError) Error() string {
	return "skipped"
}

// SkipExp is returned by ExpVisitor functions to indicate that WalkExp should
// not decend into subexpressions.
var SkipExp error = skipWalkError{}

// An ExpVisitor function is a callback to be passed to WalkExp.
//
// The bindingPath is the string which would be used to bind the given struct
// element in mro syntax.
//
// For more information, see WalkExp.
type ExpVisitor func(exp Exp, bindingPath string) error

// WalkExp calls the given visitor function for the given expression and every
// subexpression (e.g. array, map, or struct element).  If the function returns
// SkipExp, the walk will not decend into subexpressions of that expression.
// If it returns any other error, the walk will be aborted and that error will
// be returned.
func WalkExp(exp Exp, visitor ExpVisitor) error {
	return walkExp(exp, visitor, "")
}

func walkExp(exp Exp, visitor ExpVisitor, path string) error {
	if err := visitor(exp, path); err != nil {
		if err == SkipExp {
			return nil
		}
		return err
	}
	switch exp := exp.(type) {
	case *SplitExp:
		return walkExp(exp.Value, visitor, path)
	case *MergeExp:
		return walkExp(exp.Value, visitor, path)
	case *DisabledExp:
		if err := visitor(exp.Disabled, path); err != nil &&
			err != SkipExp {
			return err
		}
		return walkExp(exp.Value, visitor, path)
	case *ArrayExp:
		for _, val := range exp.Value {
			if err := walkExp(val, visitor, path); err != nil {
				return err
			}
		}
	case *MapExp:
		for k, val := range exp.Value {
			p := path
			if exp.Kind == KindStruct {
				if p == "" {
					p = k
				} else if k != "" {
					p = p + "." + k
				}
			}
			if err := walkExp(val, visitor, p); err != nil {
				return err
			}
		}
	}
	return nil
}

// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

// Value expressions, e.g. things which can be assigned to bindings.

package syntax

// Kinds of value or reference expressions.  These include all of
// the builtin types as well as "array" and "null", and for references
// "self" and "call".
const (
	// Represents an array of expressions.
	KindArray  = ExpKind("array")
	KindSweep  = "sweep"
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
		AstNodable
		getKind() ExpKind
		format(w stringWriter, prefix string)
		equal(other Exp) bool
		// HasRef returns true if this expression or any sub-expression is a
		// reference.
		HasRef() bool
		// HasSweep returns true if this expression or any sub-expression is
		// a sweep.
		HasSweep() bool

		// Recursively searches the expression for references.
		FindRefs() []*RefExp

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
		BindingPath(bindPath string) (Exp, error)
		filter(t Type, lookup *TypeLookup) (Exp, error)

		// Returns a json representation of the concrete value of a
		// literal expression.  For references, returns
		//
		//   {"__reference__": "Id.OutputId"}
		MarshalJSON() ([]byte, error)
		GoString() string

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

	// A SweepExp represents a sweep over expressions.
	SweepExp struct {
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
		Kind  ExpKind
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

	// A RefExp represents a value that is a reference to a pipeline input or
	// a call output.
	RefExp struct {
		Node AstNode
		Kind ExpKind

		// For KindSelf, the name of the input parameter.  For KindCall,
		// the call's Id.
		Id string

		// For KindCall, the Id of the output parameter of the bound call.
		OutputId string
	}
)

func (e *valExp) getNode() *AstNode       { return &e.Node }
func (s *valExp) File() *SourceFile       { return s.Node.Loc.File }
func (s *valExp) inheritComments() bool   { return false }
func (*valExp) getSubnodes() []AstNodable { return nil }
func (*valExp) HasRef() bool              { return false }
func (*valExp) HasSweep() bool            { return false }
func (*valExp) FindRefs() []*RefExp       { return nil }

func (s *ArrayExp) getKind() ExpKind  { return KindArray }
func (s *SweepExp) getKind() ExpKind  { return KindSweep }
func (s *MapExp) getKind() ExpKind    { return s.Kind }
func (s *StringExp) getKind() ExpKind { return s.Kind }
func (s *BoolExp) getKind() ExpKind   { return KindBool }
func (s *IntExp) getKind() ExpKind    { return KindInt }
func (s *FloatExp) getKind() ExpKind  { return KindFloat }
func (s *NullExp) getKind() ExpKind   { return KindNull }

func (s *ArrayExp) val() interface{}  { return s.Value }
func (s *SweepExp) val() interface{}  { return s.Value }
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
func (s *SweepExp) getSubnodes() []AstNodable {
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

func (e *ArrayExp) HasSweep() bool {
	if e == nil {
		return false
	}
	for _, exp := range e.Value {
		if exp.HasSweep() {
			return true
		}
	}
	return false
}

func (e *SweepExp) HasRef() bool {
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

func (e *SweepExp) HasSweep() bool {
	return true
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

func (e *MapExp) HasSweep() bool {
	if e == nil {
		return false
	}
	for _, exp := range e.Value {
		if exp.HasSweep() {
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
func (e *SweepExp) FindRefs() []*RefExp {
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
	return []*RefExp{e}
}

func (s *StringExp) Type() (Type, int) { return &builtinString, 0 }
func (s *BoolExp) Type() (Type, int)   { return &builtinBool, 0 }
func (s *IntExp) Type() (Type, int)    { return &builtinInt, 0 }
func (s *FloatExp) Type() (Type, int)  { return &builtinFloat, 0 }
func (s *NullExp) Type() (Type, int)   { return &builtinNull, 0 }

func (s *RefExp) getNode() *AstNode { return &s.Node }
func (s *RefExp) File() *SourceFile { return s.Node.Loc.File }
func (s *RefExp) getKind() ExpKind  { return s.Kind }

func (s *RefExp) inheritComments() bool { return false }
func (s *RefExp) getSubnodes() []AstNodable {
	return nil
}

func (*RefExp) HasRef() bool {
	return true
}
func (*RefExp) HasSweep() bool {
	return false
}

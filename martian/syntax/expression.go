// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

// Value expressions, e.g. things which can be assigned to bindings.

package syntax

// Kinds of value or reference expressions.  These include all of
// the builtin types as well as "array" and "null", and for references
// "self" and "call".
const (
	// Represents an array of expressions.
	KindArray  = ExpKind("array")
	KindSplit  = "split"
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
		// HasSplit returns true if this expression or any sub-expression is
		// a split.
		HasSplit() bool

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
		BindingPath(bindPath string,
			fork map[MapCallSource]CollectionIndex,
			index []CollectionIndex) (Exp, error)
		resolveRefs(self, siblings map[string]*ResolvedBinding,
			lookup *TypeLookup) (Exp, error)
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

	// A SplitExp represents an expression which is either a typed map or array,
	// where a call is made repeatedly, once for each element in the collection.
	// Only top-level expressions can be split.
	SplitExp struct {
		valExp
		// Either a MapExp, ArrayExp, or RefExp.  In a fully-resolved pipeline,
		// it could also be another nested SplitExp.
		Value Exp
		// The call that this split was made on.  This becomes relevent when
		// resolving a map call of a pipeline - calls within that pipeline
		// may split further, but potentially over a different set of keys.
		Call *CallStm

		// The dimensionality source
		Source MapCallSource
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
)

func (e *valExp) getNode() *AstNode       { return &e.Node }
func (s *valExp) File() *SourceFile       { return s.Node.Loc.File }
func (s *valExp) inheritComments() bool   { return false }
func (*valExp) getSubnodes() []AstNodable { return nil }
func (*valExp) HasRef() bool              { return false }
func (*valExp) HasSplit() bool            { return false }
func (*valExp) FindRefs() []*RefExp       { return nil }

func (s *ArrayExp) getKind() ExpKind  { return KindArray }
func (s *SplitExp) getKind() ExpKind  { return KindSplit }
func (s *MapExp) getKind() ExpKind    { return s.Kind }
func (s *StringExp) getKind() ExpKind { return s.Kind }
func (s *BoolExp) getKind() ExpKind   { return KindBool }
func (s *IntExp) getKind() ExpKind    { return KindInt }
func (s *FloatExp) getKind() ExpKind  { return KindFloat }
func (s *NullExp) getKind() ExpKind   { return KindNull }

func (s *ArrayExp) val() interface{}  { return s.Value }
func (s *SplitExp) val() interface{}  { return s.Value }
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

func (s *SplitExp) getSubnodes() []AstNodable {
	return []AstNodable{s.Value}
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

func (e *SplitExp) HasRef() bool {
	if e == nil || e.Value == nil {
		return false
	}
	return e.Value.HasRef()
}

func (e *SplitExp) HasSplit() bool {
	return true
}

func (e *SplitExp) CallMode() CallMode {
	if e == nil || e.Source == nil {
		return ModeUnknownMapCall
	}
	return e.Source.CallMode()
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

func (e *SplitExp) FindRefs() []*RefExp {
	refs := e.Value.FindRefs()
	if s, ok := e.Source.(Exp); ok && s != nil {
		refs = append(refs, s.FindRefs()...)
	}
	return refs
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
	for _, m := range e.MergeOver {
		if s, ok := m.(Exp); ok {
			refs = append(refs, s.FindRefs()...)
		}
	}
	for m := range e.ForkIndex {
		if s, ok := m.(Exp); ok {
			refs = append(refs, s.FindRefs()...)
		}
	}
	return refs
}

// IsEmpty returns true if the split expression is over an empty array or map.
func (e *SplitExp) IsEmpty() bool {
	switch exp := e.Value.(type) {
	case *ArrayExp:
		if len(exp.Value) == 0 {
			return true
		}
	case *MapExp:
		if len(exp.Value) == 0 {
			return true
		}
	case *NullExp:
		return true
	case *SplitExp:
		return exp.IsEmpty()
	}
	return false
}

func (s *StringExp) Type() (Type, int) { return &builtinString, 0 }
func (s *BoolExp) Type() (Type, int)   { return &builtinBool, 0 }
func (s *IntExp) Type() (Type, int)    { return &builtinInt, 0 }
func (s *FloatExp) Type() (Type, int)  { return &builtinFloat, 0 }
func (s *NullExp) Type() (Type, int)   { return &builtinNull, 0 }

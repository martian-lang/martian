// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

// Value expressions, e.g. things which can be assigned to bindings.

package syntax

// Kinds of value or reference expressions.  These include all of
// the builtin types as well as "array" and "null", and for references
// "self" and "call".
const (
	// Represents an array of expressions.
	KindArray  = ExpKind("array")
	KindMap    = "map"
	KindFloat  = "float"
	KindInt    = "int"
	KindString = "string"
	KindBool   = "bool"
	KindNull   = "null"

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
		// Whitelist exp types to internal implementations.
		getExp()
		AstNodable
		getKind() ExpKind
		resolveType(*Ast, *Pipeline) ([]string, int, error)
		format(w stringWriter, prefix string)
		equal(other Exp) bool

		// Returns a representation of the concrete value of the
		// literal expression.  Returns nil in the case of reference
		// expressions.
		ToInterface() interface{}
	}

	// A ValExp represents a literal value, or an array of Exp objects.
	ValExp struct {
		Node  AstNode
		Kind  ExpKind
		Value interface{}
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

func (s *ValExp) getNode() *AstNode { return &s.Node }
func (s *ValExp) File() *SourceFile { return s.Node.Loc.File }
func (s *ValExp) getKind() ExpKind  { return s.Kind }

func (s *ValExp) inheritComments() bool { return false }
func (s *ValExp) getSubnodes() []AstNodable {
	if s.Kind == KindArray {
		if arr, ok := s.Value.([]Exp); !ok {
			return nil
		} else {
			subs := make([]AstNodable, 0, len(arr))
			for _, n := range arr {
				subs = append(subs, n)
			}
			return subs
		}
	} else {
		return nil
	}
}

func (*ValExp) getExp() {}

func (valExp *ValExp) ToInterface() interface{} {
	// Convert tree of Exps into a tree of interface{}s.
	if valExp.Kind == KindArray {
		varray := []interface{}{}
		for _, exp := range valExp.Value.([]Exp) {
			varray = append(varray, exp.ToInterface())
		}
		return varray
	} else if valExp.Kind == KindMap {
		vmap := map[string]interface{}{}
		// Type assertion fails if map is empty
		valExpMap, ok := valExp.Value.(map[string]Exp)
		if ok {
			for k, exp := range valExpMap {
				vmap[k] = exp.ToInterface()
			}
		}
		return vmap
	} else {
		return valExp.Value
	}
}

func (s *RefExp) getNode() *AstNode { return &s.Node }
func (s *RefExp) File() *SourceFile { return s.Node.Loc.File }
func (s *RefExp) getKind() ExpKind  { return s.Kind }

func (s *RefExp) inheritComments() bool { return false }
func (s *RefExp) getSubnodes() []AstNodable {
	return nil
}

func (*RefExp) getExp() {}

func (self *RefExp) ToInterface() interface{} {
	return nil
}

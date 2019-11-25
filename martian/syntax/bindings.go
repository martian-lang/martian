// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// Definitions for calls and bindings.

package syntax

type (
	// A binding defines the assignment of a value expression to a
	// callable's input parameter in a call, or to a pipeline's output
	// parameter in a return statement.
	BindStm struct {
		Node  AstNode
		Id    string
		Exp   Exp
		Tname TypeId

		// If true, the expression is an array and the pipeline will
		// fork into versions for each value in the array.
		Sweep bool
	}

	// An ordered set of BindStm objects.
	BindStms struct {
		Node  AstNode
		List  []*BindStm `json:"-"`
		Table map[string]*BindStm
	}
)

func (s *BindStm) getNode() *AstNode  { return &s.Node }
func (s *BindStm) File() *SourceFile  { return s.Node.Loc.File }
func (s *BindStms) getNode() *AstNode { return &s.Node }
func (s *BindStms) File() *SourceFile { return s.Node.Loc.File }

func (s *BindStm) inheritComments() bool { return false }
func (s *BindStm) getSubnodes() []AstNodable {
	return []AstNodable{s.Exp}
}

func (s *BindStms) inheritComments() bool { return true }
func (s *BindStms) getSubnodes() []AstNodable {
	subs := make([]AstNodable, 0, len(s.List))
	for _, n := range s.List {
		subs = append(subs, n)
	}
	return subs
}

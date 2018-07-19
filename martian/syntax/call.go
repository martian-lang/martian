// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// Definitions for calls and bindings.

package syntax

type (
	// A CallStm contains information about the invocation of a callable
	// object.  These declarations may exist in pipeline or as the top
	// level call.
	CallStm struct {
		Node      AstNode
		Modifiers *Modifiers

		// The name of this call, which can be bound in references.
		Id string

		// The name of the callable object being called.  This will
		// be the same as Id unless the call is aliased.
		DecId string

		// The set of bindings for the input arguments of the callable.
		Bindings *BindStms
	}

	// A binding defines the assignment of a value expression to a
	// callable's input parameter in a call, or to a pipeline's output
	// parameter in a return statement.
	BindStm struct {
		Node  AstNode
		Id    string
		Exp   Exp
		Sweep bool
		Tname string
	}

	// An ordered set of BindStm objects.
	BindStms struct {
		Node  AstNode
		List  []*BindStm `json:"-"`
		Table map[string]*BindStm
	}

	// A set of modifiers on a call.
	Modifiers struct {
		// If true, this call should be run locally, even in cluster mode.
		Local bool

		// If true, this is a preflight stage.  It must run before all
		// non-preflight stages, and cannot have any outputs. Local
		// preflight stages have their standard output echoed to the
		// top-level standard output.
		Preflight bool

		// If true, this stage's output files should be cleaned out after
		// all dependent stages have completed.
		Volatile bool
		Bindings *BindStms
	}
)

func (s *CallStm) getNode() *AstNode { return &s.Node }
func (s *CallStm) File() *SourceFile { return s.Node.Loc.File }

func (s *CallStm) inheritComments() bool { return false }
func (s *CallStm) getSubnodes() []AstNodable {
	if s.Modifiers != nil && s.Modifiers.Bindings != nil {
		return []AstNodable{s.Bindings, s.Modifiers.Bindings}
	} else {
		return []AstNodable{s.Bindings}
	}
}

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

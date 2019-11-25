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

	// A set of modifiers on a call.
	Modifiers struct {
		// The set of bindings, if any, for call modifiers which are
		// determined at run time (e.g. disabled).  There may also
		// be bindings for modifiers which are known at compile time,
		// because they may have comments associated with them.
		Bindings *BindStms

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

//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// MRO abstract syntax tree.
//

// Package syntax defines the the MRO pipeline declaration language.
//
// This includes the grammar and AST definition, as well as the parsers,
// preprocessors, and formatters for it.
package syntax // import "github.com/martian-lang/martian/martian/syntax"

type (
	AstNode struct {
		Loc SourceLoc

		// comments which are in the scope for the node, appearing
		// before the node, but not attached to the node.
		scopeComments []*commentBlock

		Comments []string `json:"comments,omitempty"`
	}

	SourceLoc struct {
		Line int
		File *SourceFile
	}

	SourceFile struct {
		FileName     string
		FullPath     string
		IncludedFrom []*SourceLoc
	}

	AstNodable interface {
		nodeContainer
		// Interface whitelist for Dec, Param, Exp, and Stm implementors.
		// Patterned after code in Go's ast.go.
		getNode() *AstNode
		File() *SourceFile
	}

	nodeContainer interface {
		getSubnodes() []AstNodable
		// If true, indicates that the first subnode of this node should
		// get the comments which were attached to this node.  This is true
		// for container nodes such as Params and BindStms.
		inheritComments() bool
	}

	Dec interface {
		AstNodable
		getDec()
	}

	// Include directive.
	Include struct {
		Node  AstNode
		Value string
	}

	// Comments are also not, strictly speaking, part of the AST, but for
	// formatting code we need to keep track of them.
	commentBlock struct {
		Loc   SourceLoc
		Value string
	}

	Ast struct {
		// All user-defined file types found in the source.
		UserTypes []*UserType

		// All struct types found in the source.
		StructTypes []*StructType

		// All valid types, both user-defined and builtin.
		TypeTable TypeLookup

		// The source file object for each named include.
		Files     map[string]*SourceFile
		Stages    []*Stage
		Pipelines []*Pipeline
		Callables *Callables
		Call      *CallStm
		Errors    []error
		Includes  []*Include
		comments  []*commentBlock
	}
)

func NewAst(decs []Dec, call *CallStm, srcFile *SourceFile) *Ast {
	self := &Ast{
		Callables: new(Callables),
		Call:      call,
		Files: map[string]*SourceFile{
			srcFile.FullPath: srcFile,
		},
	}

	for _, dec := range decs {
		switch dec := dec.(type) {
		case *UserType:
			self.UserTypes = append(self.UserTypes, dec)
		case *StructType:
			self.StructTypes = append(self.StructTypes, dec)
		case *Stage:
			self.Stages = append(self.Stages, dec)
			self.Callables.List = append(self.Callables.List, dec)
		case *Pipeline:
			self.Pipelines = append(self.Pipelines, dec)
			self.Callables.List = append(self.Callables.List, dec)
		}
	}
	return self
}

func NewAstNode(loc int, file *SourceFile) AstNode {
	return AstNode{
		Loc: SourceLoc{
			Line: loc,
			File: file,
		},
	}
}

// Gets the name of the file that defines the node.
func DefiningFile(node AstNodable) string {
	return node.getNode().Loc.File.FullPath
}

func (s *Ast) inheritComments() bool { return false }
func (s *Ast) getSubnodes() []AstNodable {
	subs := make([]AstNodable, 0,
		1+len(s.UserTypes)+
			len(s.StructTypes)+
			len(s.Callables.List)+
			len(s.Includes))
	for _, n := range s.Includes {
		subs = append(subs, n)
	}
	for _, n := range s.UserTypes {
		subs = append(subs, n)
	}
	for _, n := range s.StructTypes {
		subs = append(subs, n)
	}
	for _, n := range s.Callables.List {
		subs = append(subs, n)
	}
	if s.Call != nil {
		subs = append(subs, s.Call)
	}
	return subs
}

func (s *AstNode) getNode() *AstNode         { return s }
func (s *AstNode) getSubnodes() []AstNodable { return nil }
func (s *AstNode) inheritComments() bool     { return false }
func (s *AstNode) File() *SourceFile         { return s.Loc.File }

func (s *Include) getNode() *AstNode         { return &s.Node }
func (s *Include) getSubnodes() []AstNodable { return nil }
func (s *Include) inheritComments() bool     { return false }
func (s *Include) File() *SourceFile         { return s.Node.Loc.File }

func (ast *Ast) merge(other *Ast) error {
	ast.UserTypes = append(other.UserTypes, ast.UserTypes...)
	ast.StructTypes = append(other.StructTypes, ast.StructTypes...)
	ast.Stages = append(other.Stages, ast.Stages...)
	ast.Pipelines = append(other.Pipelines, ast.Pipelines...)
	if ast.Call == nil {
		ast.Call = other.Call
	} else if other.Call != nil {
		return &DuplicateCallError{
			First:  ast.Call,
			Second: ast.Call,
		}
	}
	for k, v := range other.Files {
		ast.Files[k] = v
	}
	ast.Callables.List = append(other.Callables.List, ast.Callables.List...)
	ast.Errors = append(other.Errors, ast.Errors...)
	ast.Includes = append(ast.Includes, other.Includes...)
	ast.comments = append(other.comments, ast.comments...)
	return nil
}

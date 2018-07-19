// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// AST entries for types.

package syntax

type (
	Type interface {
		GetId() string
		IsFile() bool
	}

	// One of the built-in types.
	BuiltinType struct {
		Id string
	}

	// A user-defined file type.
	UserType struct {
		Node AstNode
		Id   string
	}
)

var builtinTypes = [...]*BuiltinType{
	{KindString},
	{KindInt},
	{KindFloat},
	{KindBool},
	{KindPath},
	{KindFile},
	{KindMap},
}

func (*UserType) getDec() {}

func (s *BuiltinType) GetId() string { return s.Id }
func (s *BuiltinType) IsFile() bool {
	switch s.Id {
	case KindPath, KindFile:
		return true
	default:
		return false
	}
}

func (s *UserType) GetId() string     { return s.Id }
func (s *UserType) IsFile() bool      { return true }
func (s *UserType) getNode() *AstNode { return &s.Node }
func (s *UserType) File() *SourceFile { return s.Node.Loc.File }

func (s *UserType) inheritComments() bool     { return false }
func (s *UserType) getSubnodes() []AstNodable { return nil }

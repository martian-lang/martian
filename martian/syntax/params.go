// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Stage and pipeline types.

package syntax

type (
	// Interface Params provides a common interface for input and output
	// parameter sets.
	Params interface {
		// Returns the parameter with the given ID.
		//
		// The second return will be false if the id is not present or
		// if the set has not been compiled.
		GetParam(string) (Param, bool)
		getWidths() (int, int, int, int)
	}

	// An ordered set of parameters.
	InParams struct {
		// Lookup table of params by Id.  Populated during compile.
		Table map[string]*InParam
		List  []*InParam `json:"-"`
	}

	// An ordered set of parameters.
	OutParams struct {
		// Lookup table of params by Id.  Populated during compile.
		Table map[string]*OutParam
		List  []*OutParam `json:"-"`
	}

	StructMemberLike interface {
		NamedNode
		GetTname() TypeId
		GetArrayDim() int
		GetHelp() string
		GetOutName() string
		IsFile() FileKind
	}

	Param interface {
		StructMemberLike
		getMode() string
		setIsFile(FileKind)
	}

	InParam struct {
		Id     string
		Help   string
		Node   AstNode
		Tname  TypeId
		Isfile FileKind
	}

	OutParam struct {
		StructMember
	}

	// To simplify implementation of the parser, this stores the stage's
	// ChunkIns and ChunkOuts.
	paramsTuple struct {
		Ins     *InParams
		Outs    *OutParams
		Present bool
	}

	RetainParams struct {
		Node   AstNode
		Params []*RetainParam
	}

	RetainParam struct {
		Id   string
		Node AstNode
	}
)

func (s *InParams) GetParam(id string) (Param, bool) {
	p, ok := s.Table[id]
	return p, ok
}
func (s *OutParams) GetParam(id string) (Param, bool) {
	p, ok := s.Table[id]
	return p, ok
}

func (p *RetainParams) getNode() *AstNode     { return &p.Node }
func (s *RetainParams) File() *SourceFile     { return s.Node.Loc.File }
func (s *RetainParams) Line() int             { return s.Node.Loc.Line }
func (s *RetainParams) inheritComments() bool { return true }
func (s *RetainParams) getSubnodes() []AstNodable {
	params := make([]AstNodable, 0, len(s.Params))
	for _, p := range s.Params {
		params = append(params, p)
	}
	return params
}

func (s *RetainParam) getNode() *AstNode         { return &s.Node }
func (s *RetainParam) File() *SourceFile         { return s.Node.Loc.File }
func (s *RetainParam) Line() int                 { return s.Node.Loc.Line }
func (s *RetainParam) getSubnodes() []AstNodable { return nil }
func (s *RetainParam) inheritComments() bool     { return false }

func (s *InParam) getNode() *AstNode    { return &s.Node }
func (s *InParam) File() *SourceFile    { return s.Node.Loc.File }
func (s *InParam) Line() int            { return s.Node.Loc.Line }
func (s *InParam) getMode() string      { return "in" }
func (s *InParam) GetTname() TypeId     { return s.Tname }
func (s *InParam) GetArrayDim() int     { return int(s.Tname.ArrayDim) }
func (s *InParam) GetId() string        { return s.Id }
func (s *InParam) GetHelp() string      { return s.Help }
func (s *InParam) GetOutName() string   { return "" }
func (s *InParam) IsFile() FileKind     { return s.Isfile }
func (s *InParam) setIsFile(b FileKind) { s.Isfile = b }

func (s *InParam) inheritComments() bool { return false }
func (s *InParam) getSubnodes() []AstNodable {
	return nil
}

func (s *OutParam) getNode() *AstNode { return &s.Node }
func (s *OutParam) File() *SourceFile { return s.Node.Loc.File }
func (s *OutParam) Line() int         { return s.Node.Loc.Line }
func (s *OutParam) getMode() string   { return "out" }
func (s *OutParam) GetId() string     { return s.Id }

func (s *OutParam) inheritComments() bool { return false }
func (s *OutParam) getSubnodes() []AstNodable {
	return nil
}

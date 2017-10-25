//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO abstract syntax tree.
//

// Package syntax defines the the MRO pipeline declaration language.
//
// This includes the grammar and AST definition, as well as the parsers,
// preprocessors, and formatters for it.
package syntax

type StageLanguage string

type (
	AstNode struct {
		Loc          int
		Fname        string
		IncludeStack []string
	}

	AstNodable interface {
		getNode() *AstNode
	}

	Type interface {
		GetId() string
	}

	BuiltinType struct {
		Id string
	}

	UserType struct {
		Node AstNode
		Id   string
	}

	Dec interface {
		getDec()
	}

	Callable interface {
		getNode() *AstNode
		GetId() string
		GetInParams() *Params
		GetOutParams() *Params
		Type() string
		format(printer *printer)
	}

	Stage struct {
		Node        AstNode
		Id          string
		InParams    *Params
		OutParams   *Params
		Src         *SrcParam
		SplitParams *Params
		Split       bool
	}

	Pipeline struct {
		Node      AstNode
		Id        string
		InParams  *Params
		OutParams *Params
		Calls     []*CallStm
		Callables *Callables `json:"-"`
		Ret       *ReturnStm
	}

	Params struct {
		List  []Param
		Table map[string]Param
	}

	Callables struct {
		List  []Callable `json:"-"`
		Table map[string]Callable
	}

	Param interface {
		getNode() *AstNode
		getMode() string
		GetTname() string
		GetArrayDim() int
		GetId() string
		GetHelp() string
		GetOutName() string
		IsFile() bool
		setIsFile(bool)
	}

	InParam struct {
		Node     AstNode
		Tname    string
		ArrayDim int
		Id       string
		Help     string
		Isfile   bool
	}

	OutParam struct {
		Node     AstNode
		Tname    string
		ArrayDim int
		Id       string
		Help     string
		OutName  string
		Isfile   bool
	}

	SrcParam struct {
		Node AstNode
		Lang StageLanguage
		Path string
		Args []string
	}

	BindStm struct {
		Node  AstNode
		Id    string
		Exp   Exp
		Sweep bool
		Tname string
	}

	BindStms struct {
		Node  AstNode
		List  []*BindStm `json:"-"`
		Table map[string]*BindStm
	}

	Modifiers struct {
		Local     bool
		Preflight bool
		Volatile  bool
	}

	CallStm struct {
		Node      AstNode
		Modifiers *Modifiers
		Id        string
		Bindings  *BindStms
	}

	ReturnStm struct {
		Node     AstNode
		Bindings *BindStms
	}

	Exp interface {
		getExp()
		getNode() *AstNode
		getKind() string
		resolveType(*Ast, Callable) ([]string, int, error)
		format(prefix string) string
		ToInterface() interface{}
	}

	ValExp struct {
		Node  AstNode
		Kind  string
		Value interface{}
	}

	RefExp struct {
		Node     AstNode
		Kind     string
		Id       string
		OutputId string
	}

	// Preprocessor variables aren't, strictly speaking, part of the
	// AST.  They're stripped before parsing, except when formatting code.
	preprocessorDirective struct {
		Node  AstNode
		Value string
	}

	// Comments are also not, strictly speaking, part of the AST, but for
	// formatting code we need to keep track of them.
	commentBlock struct {
		Loc   int
		Value string
	}

	Ast struct {
		UserTypes     []*UserType
		UserTypeTable map[string]*UserType
		TypeTable     map[string]Type
		Stages        []*Stage
		Pipelines     []*Pipeline
		Callables     *Callables
		Call          *CallStm
		Errors        []error
		preprocess    []*preprocessorDirective
		comments      []*commentBlock
	}
)

func NewAst(decs []Dec, call *CallStm) *Ast {
	self := &Ast{}
	self.UserTypes = []*UserType{}
	self.UserTypeTable = map[string]*UserType{}
	self.TypeTable = map[string]Type{}
	self.Stages = []*Stage{}
	self.Pipelines = []*Pipeline{}
	self.Callables = &Callables{[]Callable{}, map[string]Callable{}}
	self.Call = call
	self.Errors = []error{}

	for _, dec := range decs {
		switch dec := dec.(type) {
		case *UserType:
			self.UserTypes = append(self.UserTypes, dec)
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

func NewAstNode(loc int, locmap []FileLoc) AstNode {
	// Process the accumulated comments/whitespace.

	if len(locmap) > 0 {
		// If there's no newline at the end of the source and the error is in the
		// node at the end of the file, the loc can be one larger than the size
		// of the locmap. So cap it so we don't have an array out of bounds.
		if loc >= len(locmap) {
			loc = len(locmap) - 1
		}
		return AstNode{locmap[loc].loc, locmap[loc].fname, locmap[loc].includedFrom}
	} else {
		// locmap will be empty when yaccParse is called from mrf
		return AstNode{loc, "", nil}
	}
}

// Interface whitelist for Dec, Param, Exp, and Stm implementors.
// Patterned after code in Go's ast.go.
func (*UserType) getDec() {}
func (*Stage) getDec()    {}
func (*Pipeline) getDec() {}
func (*ValExp) getExp()   {}
func (*RefExp) getExp()   {}

func (s *BuiltinType) GetId() string  { return s.Id }
func (s *UserType) GetId() string     { return s.Id }
func (s *UserType) getNode() *AstNode { return &s.Node }

func (s *Stage) GetId() string         { return s.Id }
func (s *Stage) getNode() *AstNode     { return &s.Node }
func (s *Stage) GetInParams() *Params  { return s.InParams }
func (s *Stage) GetOutParams() *Params { return s.OutParams }
func (s *Stage) Type() string          { return "stage" }

func (s *Pipeline) GetId() string         { return s.Id }
func (s *Pipeline) getNode() *AstNode     { return &s.Node }
func (s *Pipeline) GetInParams() *Params  { return s.InParams }
func (s *Pipeline) GetOutParams() *Params { return s.OutParams }
func (s *Pipeline) Type() string          { return "pipeline" }

func (s *CallStm) getNode() *AstNode { return &s.Node }

func (s *InParam) getNode() *AstNode  { return &s.Node }
func (s *InParam) getMode() string    { return "in" }
func (s *InParam) GetTname() string   { return s.Tname }
func (s *InParam) GetArrayDim() int   { return s.ArrayDim }
func (s *InParam) GetId() string      { return s.Id }
func (s *InParam) GetHelp() string    { return s.Help }
func (s *InParam) GetOutName() string { return "" }
func (s *InParam) IsFile() bool       { return s.Isfile }
func (s *InParam) setIsFile(b bool)   { s.Isfile = b }

func (s *OutParam) getNode() *AstNode  { return &s.Node }
func (s *OutParam) getMode() string    { return "out" }
func (s *OutParam) GetTname() string   { return s.Tname }
func (s *OutParam) GetArrayDim() int   { return s.ArrayDim }
func (s *OutParam) GetId() string      { return s.Id }
func (s *OutParam) GetHelp() string    { return s.Help }
func (s *OutParam) GetOutName() string { return s.OutName }
func (s *OutParam) IsFile() bool       { return s.Isfile }
func (s *OutParam) setIsFile(b bool)   { s.Isfile = b }

func (s *ReturnStm) getNode() *AstNode { return &s.Node }
func (s *BindStm) getNode() *AstNode   { return &s.Node }
func (s *BindStms) getNode() *AstNode  { return &s.Node }

func (s *ValExp) getNode() *AstNode { return &s.Node }
func (s *ValExp) getKind() string   { return s.Kind }

func (s *RefExp) getNode() *AstNode { return &s.Node }
func (s *RefExp) getKind() string   { return s.Kind }

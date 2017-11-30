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
		Comments     []string
	}

	AstNodable interface {
		nodeContainer
		getNode() *AstNode
	}

	nodeContainer interface {
		getSubnodes() []AstNodable
		// If true, indicates that the first subnode of this node should
		// get the comments which were attached to this node.  This is true
		// for container nodes such as Params and BindStms.
		inheritComments() bool
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
		AstNodable
		getDec()
	}

	Callable interface {
		AstNodable
		GetId() string
		GetInParams() *Params
		GetOutParams() *Params
		Type() string
		format(printer *printer)
	}

	Resources struct {
		Node        AstNode
		ThreadNode  *AstNode
		MemNode     *AstNode
		SpecialNode *AstNode

		Threads int
		MemGB   int
		Special string
	}

	paramsTuple struct {
		Present bool
		Ins     *Params
		Outs    *Params
	}

	Stage struct {
		Node      AstNode
		Id        string
		InParams  *Params
		OutParams *Params
		Src       *SrcParam
		ChunkIns  *Params
		ChunkOuts *Params
		Split     bool
		Resources *Resources
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
		AstNodable
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
		Bindings  *BindStms
	}

	CallStm struct {
		Node      AstNode
		Modifiers *Modifiers
		Id        string
		DecId     string
		Bindings  *BindStms
	}

	ReturnStm struct {
		Node     AstNode
		Bindings *BindStms
	}

	ExpKind string

	Exp interface {
		getExp()
		AstNodable
		getKind() ExpKind
		resolveType(*Ast, Callable) ([]string, int, error)
		format(prefix string) string
		ToInterface() interface{}
	}

	ValExp struct {
		Node  AstNode
		Kind  ExpKind
		Value interface{}
	}

	RefExp struct {
		Node     AstNode
		Kind     ExpKind
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

const (
	KindArray  ExpKind = "array"
	KindMap            = "map"
	KindFloat          = "float"
	KindInt            = "int"
	KindString         = "string"
	KindBool           = "bool"
	KindNull           = "null"
	KindSelf           = "self" // reference
	KindCall           = "call" // reference
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
		return AstNode{locmap[loc].loc, locmap[loc].fname, locmap[loc].includedFrom, nil}
	} else {
		// locmap will be empty when yaccParse is called from mrf
		return AstNode{loc, "", nil, nil}
	}
}

func (s *Ast) inheritComments() bool { return false }
func (s *Ast) getSubnodes() []AstNodable {
	subs := make([]AstNodable, 0,
		1+len(s.UserTypes)+
			len(s.Callables.List)+
			len(s.preprocess))
	for _, n := range s.preprocess {
		subs = append(subs, n)
	}
	for _, n := range s.UserTypes {
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

func (s *preprocessorDirective) getNode() *AstNode         { return &s.Node }
func (s *preprocessorDirective) getSubnodes() []AstNodable { return nil }
func (s *preprocessorDirective) inheritComments() bool     { return false }

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

func (s *UserType) inheritComments() bool     { return false }
func (s *UserType) getSubnodes() []AstNodable { return nil }

func (s *Stage) GetId() string         { return s.Id }
func (s *Stage) getNode() *AstNode     { return &s.Node }
func (s *Stage) GetInParams() *Params  { return s.InParams }
func (s *Stage) GetOutParams() *Params { return s.OutParams }
func (s *Stage) Type() string          { return "stage" }

func (s *Stage) inheritComments() bool { return false }
func (s *Stage) getSubnodes() []AstNodable {
	subs := make([]AstNodable, 0, 2+
		len(s.InParams.List)+len(s.OutParams.List)+
		len(s.ChunkIns.List)+len(s.ChunkOuts.List))
	for _, n := range s.InParams.List {
		subs = append(subs, n)
	}
	for _, n := range s.OutParams.List {
		subs = append(subs, n)
	}
	subs = append(subs, s.Src)
	for _, n := range s.ChunkIns.List {
		subs = append(subs, n)
	}
	for _, n := range s.ChunkOuts.List {
		subs = append(subs, n)
	}
	if s.Resources != nil {
		subs = append(subs, s.Resources)
	}
	return subs
}

func (s *Resources) getNode() *AstNode     { return &s.Node }
func (s *Resources) inheritComments() bool { return false }
func (s *Resources) getSubnodes() []AstNodable {
	subs := make([]AstNodable, 0, 3)
	if s.ThreadNode != nil {
		subs = append(subs, s.ThreadNode)
	}
	if s.MemNode != nil {
		subs = append(subs, s.MemNode)
	}
	if s.SpecialNode != nil {
		subs = append(subs, s.SpecialNode)
	}
	return subs
}

func (s *Pipeline) GetId() string         { return s.Id }
func (s *Pipeline) getNode() *AstNode     { return &s.Node }
func (s *Pipeline) GetInParams() *Params  { return s.InParams }
func (s *Pipeline) GetOutParams() *Params { return s.OutParams }
func (s *Pipeline) Type() string          { return "pipeline" }

func (s *Pipeline) inheritComments() bool { return false }
func (s *Pipeline) getSubnodes() []AstNodable {
	subs := make([]AstNodable, 0, 1+
		len(s.InParams.List)+len(s.OutParams.List)+len(s.Calls))
	for _, n := range s.InParams.List {
		subs = append(subs, n)
	}
	for _, n := range s.OutParams.List {
		subs = append(subs, n)
	}
	for _, n := range s.Calls {
		subs = append(subs, n)
	}
	subs = append(subs, s.Ret)
	return subs
}

func (s *CallStm) getNode() *AstNode { return &s.Node }

func (s *CallStm) inheritComments() bool { return false }
func (s *CallStm) getSubnodes() []AstNodable {
	if s.Modifiers != nil && s.Modifiers.Bindings != nil {
		return []AstNodable{s.Bindings, s.Modifiers.Bindings}
	} else {
		return []AstNodable{s.Bindings}
	}
}

func (s *InParam) getNode() *AstNode  { return &s.Node }
func (s *InParam) getMode() string    { return "in" }
func (s *InParam) GetTname() string   { return s.Tname }
func (s *InParam) GetArrayDim() int   { return s.ArrayDim }
func (s *InParam) GetId() string      { return s.Id }
func (s *InParam) GetHelp() string    { return s.Help }
func (s *InParam) GetOutName() string { return "" }
func (s *InParam) IsFile() bool       { return s.Isfile }
func (s *InParam) setIsFile(b bool)   { s.Isfile = b }

func (s *InParam) inheritComments() bool { return false }
func (s *InParam) getSubnodes() []AstNodable {
	return nil
}

func (s *OutParam) getNode() *AstNode  { return &s.Node }
func (s *OutParam) getMode() string    { return "out" }
func (s *OutParam) GetTname() string   { return s.Tname }
func (s *OutParam) GetArrayDim() int   { return s.ArrayDim }
func (s *OutParam) GetId() string      { return s.Id }
func (s *OutParam) GetHelp() string    { return s.Help }
func (s *OutParam) GetOutName() string { return s.OutName }
func (s *OutParam) IsFile() bool       { return s.Isfile }
func (s *OutParam) setIsFile(b bool)   { s.Isfile = b }

func (s *OutParam) inheritComments() bool { return false }
func (s *OutParam) getSubnodes() []AstNodable {
	return nil
}

func (s *SrcParam) getNode() *AstNode         { return &s.Node }
func (s *SrcParam) inheritComments() bool     { return false }
func (s *SrcParam) getSubnodes() []AstNodable { return nil }

func (s *ReturnStm) getNode() *AstNode { return &s.Node }
func (s *BindStm) getNode() *AstNode   { return &s.Node }
func (s *BindStms) getNode() *AstNode  { return &s.Node }

func (s *ReturnStm) inheritComments() bool { return false }
func (s *ReturnStm) getSubnodes() []AstNodable {
	return []AstNodable{s.Bindings}
}

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

func (s *ValExp) getNode() *AstNode { return &s.Node }
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

func (s *RefExp) getNode() *AstNode { return &s.Node }
func (s *RefExp) getKind() ExpKind  { return s.Kind }

func (s *RefExp) inheritComments() bool { return false }
func (s *RefExp) getSubnodes() []AstNodable {
	return nil
}

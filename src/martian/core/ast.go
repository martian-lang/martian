//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO abstract syntax tree.
//
package core

import (
	"regexp"
	"strings"
)

type (
	AstNode struct {
		Loc      int
		Fname    string
		Comments string `json:"-"`
	}

	AstNodable interface {
		getNode() *AstNode
	}

	Type interface {
		getId() string
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
		getId() string
		getInParams() *Params
		getOutParams() *Params
		format() string
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
		getTname() string
		getArrayDim() int
		getId() string
		getHelp() string
		getOutName() string
		getIsFile() bool
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
		Lang string
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

	Ast struct {
		UserTypes     []*UserType
		UserTypeTable map[string]*UserType
		TypeTable     map[string]Type
		Stages        []*Stage
		Pipelines     []*Pipeline
		Callables     *Callables
		Call          *CallStm
		Errors        []error
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

func NewAstNode(lval *mmSymType) AstNode {
	// Process the accumulated comments/whitespace.

	// Compress consecutive newlines into one.
	re := regexp.MustCompile("\n{2,}")
	comments := re.ReplaceAllString(lval.comments, "\n")

	// Remove whitespace at either end.
	comments = strings.TrimSpace(comments)

	// Add newline to the end.
	comments += "\n"

	// Reset lexer's comment/whitespace accumulator.
	lval.comments = ""

	locmap := lval.locmap
	loc := lval.loc

	if len(locmap) > 0 {
		// If there's no newline at the end of the source and the error is in the
		// node at the end of the file, the loc can be one larger than the size
		// of the locmap. So cap it so we don't have an array out of bounds.
		if loc >= len(locmap) {
			loc = len(locmap) - 1
		}
		return AstNode{locmap[loc].loc, locmap[loc].fname, comments}
	} else {
		// locmap will be empty when yaccParse is called from mrf
		return AstNode{1, "", comments}
	}
}

// Interface whitelist for Dec, Param, Exp, and Stm implementors.
// Patterned after code in Go's ast.go.
func (*UserType) getDec() {}
func (*Stage) getDec()    {}
func (*Pipeline) getDec() {}
func (*ValExp) getExp()   {}
func (*RefExp) getExp()   {}

func (s *BuiltinType) getId() string  { return s.Id }
func (s *UserType) getId() string     { return s.Id }
func (s *UserType) getNode() *AstNode { return &s.Node }

func (s *Stage) getId() string         { return s.Id }
func (s *Stage) getNode() *AstNode     { return &s.Node }
func (s *Stage) getInParams() *Params  { return s.InParams }
func (s *Stage) getOutParams() *Params { return s.OutParams }

func (s *Pipeline) getId() string         { return s.Id }
func (s *Pipeline) getNode() *AstNode     { return &s.Node }
func (s *Pipeline) getInParams() *Params  { return s.InParams }
func (s *Pipeline) getOutParams() *Params { return s.OutParams }

func (s *CallStm) getNode() *AstNode { return &s.Node }

func (s *InParam) getNode() *AstNode  { return &s.Node }
func (s *InParam) getMode() string    { return "in" }
func (s *InParam) getTname() string   { return s.Tname }
func (s *InParam) getArrayDim() int   { return s.ArrayDim }
func (s *InParam) getId() string      { return s.Id }
func (s *InParam) getHelp() string    { return s.Help }
func (s *InParam) getOutName() string { return "" }
func (s *InParam) getIsFile() bool    { return s.Isfile }
func (s *InParam) setIsFile(b bool)   { s.Isfile = b }

func (s *OutParam) getNode() *AstNode  { return &s.Node }
func (s *OutParam) getMode() string    { return "out" }
func (s *OutParam) getTname() string   { return s.Tname }
func (s *OutParam) getArrayDim() int   { return s.ArrayDim }
func (s *OutParam) getId() string      { return s.Id }
func (s *OutParam) getHelp() string    { return s.Help }
func (s *OutParam) getOutName() string { return s.OutName }
func (s *OutParam) getIsFile() bool    { return s.Isfile }
func (s *OutParam) setIsFile(b bool)   { s.Isfile = b }

func (s *ReturnStm) getNode() *AstNode { return &s.Node }
func (s *BindStm) getNode() *AstNode   { return &s.Node }
func (s *BindStms) getNode() *AstNode  { return &s.Node }

func (s *ValExp) getNode() *AstNode { return &s.Node }
func (s *ValExp) getKind() string   { return s.Kind }

func (s *RefExp) getNode() *AstNode { return &s.Node }
func (s *RefExp) getKind() string   { return s.Kind }

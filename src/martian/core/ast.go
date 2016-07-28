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
		Comments string
	}

	Locatable interface {
		getLoc() int
	}

	Filetype struct {
		Node AstNode
		Id   string
	}

	Dec interface {
		getDec()
	}

	Callable interface {
		getNode() *AstNode
		getLoc() int
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
		Callables *Callables
		Ret       *ReturnStm
	}

	Params struct {
		List  []Param
		Table map[string]Param
	}

	Callables struct {
		List  []Callable
		Table map[string]Callable
	}

	Param interface {
		getNode() *AstNode
		getLoc() int
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
		List  []*BindStm
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
		format() string
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
		Locmap        []FileLoc
		TypeTable     map[string]bool
		Filetypes     []*Filetype
		FiletypeTable map[string]bool
		Stages        []*Stage
		Pipelines     []*Pipeline
		Callables     *Callables
		Call          *CallStm
	}
)

func NewAst(decs []Dec, call *CallStm) *Ast {
	self := &Ast{}
	self.Locmap = []FileLoc{}
	self.TypeTable = map[string]bool{}
	self.Filetypes = []*Filetype{}
	self.FiletypeTable = map[string]bool{}
	self.Stages = []*Stage{}
	self.Pipelines = []*Pipeline{}
	self.Callables = &Callables{[]Callable{}, map[string]Callable{}}
	self.Call = call

	for _, dec := range decs {
		switch dec := dec.(type) {
		case *Filetype:
			self.Filetypes = append(self.Filetypes, dec)
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

	return AstNode{lval.loc, comments}
}

// Interface whitelist for Dec, Param, Exp, and Stm implementors.
// Patterned after code in Go's ast.go.
func (*Filetype) getDec() {}
func (*Stage) getDec()    {}
func (*Pipeline) getDec() {}
func (*ValExp) getExp()   {}
func (*RefExp) getExp()   {}

func (s *Filetype) getNode() *AstNode { return &s.Node }
func (s *Filetype) getLoc() int       { return s.Node.Loc }

func (s *Stage) getId() string         { return s.Id }
func (s *Stage) getNode() *AstNode     { return &s.Node }
func (s *Stage) getLoc() int           { return s.Node.Loc }
func (s *Stage) getInParams() *Params  { return s.InParams }
func (s *Stage) getOutParams() *Params { return s.OutParams }

func (s *Pipeline) getId() string         { return s.Id }
func (s *Pipeline) getNode() *AstNode     { return &s.Node }
func (s *Pipeline) getLoc() int           { return s.Node.Loc }
func (s *Pipeline) getInParams() *Params  { return s.InParams }
func (s *Pipeline) getOutParams() *Params { return s.OutParams }

func (s *CallStm) getLoc() int { return s.Node.Loc }

func (s *InParam) getNode() *AstNode  { return &s.Node }
func (s *InParam) getMode() string    { return "in" }
func (s *InParam) getTname() string   { return s.Tname }
func (s *InParam) getArrayDim() int   { return s.ArrayDim }
func (s *InParam) getId() string      { return s.Id }
func (s *InParam) getHelp() string    { return s.Help }
func (s *InParam) getLoc() int        { return s.Node.Loc }
func (s *InParam) getOutName() string { return "" }
func (s *InParam) getIsFile() bool    { return s.Isfile }
func (s *InParam) setIsFile(b bool)   { s.Isfile = b }

func (s *OutParam) getNode() *AstNode  { return &s.Node }
func (s *OutParam) getMode() string    { return "out" }
func (s *OutParam) getTname() string   { return s.Tname }
func (s *OutParam) getArrayDim() int   { return s.ArrayDim }
func (s *OutParam) getId() string      { return s.Id }
func (s *OutParam) getHelp() string    { return s.Help }
func (s *OutParam) getLoc() int        { return s.Node.Loc }
func (s *OutParam) getOutName() string { return s.OutName }
func (s *OutParam) getIsFile() bool    { return s.Isfile }
func (s *OutParam) setIsFile(b bool)   { s.Isfile = b }

func (s *ReturnStm) getLoc() int { return s.Node.Loc }
func (s *BindStm) getLoc() int   { return s.Node.Loc }
func (s *BindStms) getLoc() int  { return s.Node.Loc }

func (s *ValExp) getNode() *AstNode { return &s.Node }
func (s *ValExp) getKind() string   { return s.Kind }
func (s *ValExp) getLoc() int       { return s.Node.Loc }

func (s *RefExp) getNode() *AstNode { return &s.Node }
func (s *RefExp) getKind() string   { return s.Kind }
func (s *RefExp) getLoc() int       { return s.Node.Loc }

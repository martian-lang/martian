//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
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
		loc      int
		comments string
	}

	Locatable interface {
		getLoc() int
	}

	Filetype struct {
		node AstNode
		id   string
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
		node        AstNode
		id          string
		inParams    *Params
		outParams   *Params
		src         *SrcParam
		splitParams *Params
	}

	Pipeline struct {
		node      AstNode
		id        string
		inParams  *Params
		outParams *Params
		calls     []*CallStm
		callables *Callables
		ret       *ReturnStm
	}

	Params struct {
		list  []Param
		table map[string]Param
	}

	Callables struct {
		list  []Callable
		table map[string]Callable
	}

	Param interface {
		getNode() *AstNode
		getLoc() int
		getMode() string
		getTname() string
		getIsArray() bool
		getId() string
		getHelp() string
		getIsFile() bool
		setIsFile(bool)
	}

	InParam struct {
		node    AstNode
		tname   string
		isarray bool
		id      string
		help    string
		isfile  bool
	}

	OutParam struct {
		node    AstNode
		tname   string
		isarray bool
		id      string
		help    string
		isfile  bool
	}

	SrcParam struct {
		node AstNode
		lang string
		path string
	}

	BindStm struct {
		node  AstNode
		id    string
		exp   Exp
		sweep bool
		tname string
	}

	BindStms struct {
		list  []*BindStm
		table map[string]*BindStm
	}

	CallStm struct {
		node     AstNode
		volatile bool
		id       string
		bindings *BindStms
	}

	ReturnStm struct {
		node     AstNode
		bindings *BindStms
	}

	Exp interface {
		getExp()
		getNode() *AstNode
		getKind() string
		resolveType(*Ast, Callable) ([]string, bool, error)
		format() string
	}

	ValExp struct {
		node  AstNode
		kind  string
		value interface{}
	}

	RefExp struct {
		node     AstNode
		kind     string
		id       string
		outputId string
	}

	Ast struct {
		locmap        []FileLoc
		typeTable     map[string]bool
		filetypes     []*Filetype
		filetypeTable map[string]bool
		stages        []*Stage
		pipelines     []*Pipeline
		callables     *Callables
		call          *CallStm
	}
)

func NewAst(decs []Dec, call *CallStm) *Ast {
	self := &Ast{}
	self.locmap = []FileLoc{}
	self.typeTable = map[string]bool{}
	self.filetypes = []*Filetype{}
	self.filetypeTable = map[string]bool{}
	self.stages = []*Stage{}
	self.pipelines = []*Pipeline{}
	self.callables = &Callables{[]Callable{}, map[string]Callable{}}
	self.call = call

	for _, dec := range decs {
		switch dec := dec.(type) {
		case *Filetype:
			self.filetypes = append(self.filetypes, dec)
		case *Stage:
			self.stages = append(self.stages, dec)
			self.callables.list = append(self.callables.list, dec)
		case *Pipeline:
			self.pipelines = append(self.pipelines, dec)
			self.callables.list = append(self.callables.list, dec)
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

func (s *Filetype) getNode() *AstNode { return &s.node }
func (s *Filetype) getLoc() int       { return s.node.loc }

func (s *Stage) getId() string         { return s.id }
func (s *Stage) getNode() *AstNode     { return &s.node }
func (s *Stage) getLoc() int           { return s.node.loc }
func (s *Stage) getInParams() *Params  { return s.inParams }
func (s *Stage) getOutParams() *Params { return s.outParams }

func (s *Pipeline) getId() string         { return s.id }
func (s *Pipeline) getNode() *AstNode     { return &s.node }
func (s *Pipeline) getLoc() int           { return s.node.loc }
func (s *Pipeline) getInParams() *Params  { return s.inParams }
func (s *Pipeline) getOutParams() *Params { return s.outParams }

func (s *CallStm) getLoc() int { return s.node.loc }

func (s *InParam) getNode() *AstNode { return &s.node }
func (s *InParam) getMode() string   { return "in" }
func (s *InParam) getTname() string  { return s.tname }
func (s *InParam) getIsArray() bool  { return s.isarray }
func (s *InParam) getId() string     { return s.id }
func (s *InParam) getHelp() string   { return s.help }
func (s *InParam) getLoc() int       { return s.node.loc }
func (s *InParam) getIsFile() bool   { return s.isfile }
func (s *InParam) setIsFile(b bool)  { s.isfile = b }

func (s *OutParam) getNode() *AstNode { return &s.node }
func (s *OutParam) getMode() string   { return "out" }
func (s *OutParam) getTname() string  { return s.tname }
func (s *OutParam) getIsArray() bool  { return s.isarray }
func (s *OutParam) getId() string     { return s.id }
func (s *OutParam) getHelp() string   { return s.help }
func (s *OutParam) getLoc() int       { return s.node.loc }
func (s *OutParam) getIsFile() bool   { return s.isfile }
func (s *OutParam) setIsFile(b bool)  { s.isfile = b }

func (s *ReturnStm) getLoc() int { return s.node.loc }
func (s *BindStm) getLoc() int   { return s.node.loc }

func (s *ValExp) getNode() *AstNode { return &s.node }
func (s *ValExp) getKind() string   { return s.kind }
func (s *ValExp) getLoc() int       { return s.node.loc }

func (s *RefExp) getNode() *AstNode { return &s.node }
func (s *RefExp) getKind() string   { return s.kind }
func (s *RefExp) getLoc() int       { return s.node.loc }

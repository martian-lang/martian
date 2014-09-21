//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario Abstract Syntax Tree
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
		Loc() int
	}

	Filetype struct {
		node AstNode
		Id   string
	}

	Dec interface {
		dec()
	}

	Callable interface {
		Node() *AstNode
		Loc() int
		GetId() string
		InParams() *Params
		OutParams() *Params
		format() string
	}

	Stage struct {
		node        AstNode
		Id          string
		inParams    *Params
		outParams   *Params
		src         *SrcParam
		splitParams *Params
	}

	Pipeline struct {
		node      AstNode
		Id        string
		inParams  *Params
		outParams *Params
		Calls     []*CallStm
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
		Node() *AstNode
		Loc() int
		Mode() string
		Tname() string
		Id() string
		Help() string
		IsFile() bool
		SetIsFile(bool)
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
		Exp   Exp
		sweep bool
		Tname string
	}

	BindStms struct {
		List  []*BindStm
		table map[string]*BindStm
	}

	CallStm struct {
		node     AstNode
		volatile bool
		Id       string
		Bindings *BindStms
	}

	ReturnStm struct {
		node     AstNode
		bindings *BindStms
	}

	Exp interface {
		exp()
		Node() *AstNode
		GetKind() string
		ResolveType(*Ast, *Pipeline) (string, error)
		format() string
	}

	ValExp struct {
		node  AstNode
		Kind  string
		Value interface{}
	}

	RefExp struct {
		node     AstNode
		Kind     string
		Id       string
		outputId string
	}

	Ast struct {
		locmap        []FileLoc
		typeTable     map[string]bool
		filetypes     []*Filetype
		FiletypeTable map[string]bool
		Stages        []*Stage
		Pipelines     []*Pipeline
		callables     *Callables
		call          *CallStm
	}
)

func NewAstNode(lval *mmSymType) AstNode {
	re := regexp.MustCompile("\n{2,}")
	comments := re.ReplaceAllString(lval.comments, "\n")
	comments = strings.TrimSpace(comments)
	comments += "\n"
	node := AstNode{lval.loc, comments}
	lval.comments = ""
	return node
}

// Interface whitelist for Dec, Param, Exp, and Stm implementors.
// Patterned after code in Go's ast.go.
func (*Filetype) dec() {}
func (*Stage) dec()    {}
func (*Pipeline) dec() {}
func (*ValExp) exp()   {}
func (*RefExp) exp()   {}

func (s *Filetype) Node() *AstNode { return &s.node }
func (s *Filetype) Loc() int       { return s.node.loc }

func (s *Stage) GetId() string      { return s.Id }
func (s *Stage) Node() *AstNode     { return &s.node }
func (s *Stage) Loc() int           { return s.node.loc }
func (s *Stage) InParams() *Params  { return s.inParams }
func (s *Stage) OutParams() *Params { return s.outParams }

func (s *Pipeline) GetId() string      { return s.Id }
func (s *Pipeline) Node() *AstNode     { return &s.node }
func (s *Pipeline) Loc() int           { return s.node.loc }
func (s *Pipeline) InParams() *Params  { return s.inParams }
func (s *Pipeline) OutParams() *Params { return s.outParams }

func (s *CallStm) Loc() int { return s.node.loc }

func (s *InParam) Node() *AstNode   { return &s.node }
func (s *InParam) Mode() string     { return "in" }
func (s *InParam) Tname() string    { return s.tname }
func (s *InParam) Id() string       { return s.id }
func (s *InParam) Help() string     { return s.help }
func (s *InParam) Loc() int         { return s.node.loc }
func (s *InParam) IsFile() bool     { return s.isfile }
func (s *InParam) SetIsFile(b bool) { s.isfile = b }

func (s *OutParam) Node() *AstNode   { return &s.node }
func (s *OutParam) Mode() string     { return "out" }
func (s *OutParam) Tname() string    { return s.tname }
func (s *OutParam) Id() string       { return s.id }
func (s *OutParam) Help() string     { return s.help }
func (s *OutParam) Loc() int         { return s.node.loc }
func (s *OutParam) IsFile() bool     { return s.isfile }
func (s *OutParam) SetIsFile(b bool) { s.isfile = b }

func (s *ReturnStm) Loc() int { return s.node.loc }
func (s *BindStm) Loc() int   { return s.node.loc }

func (s *ValExp) Node() *AstNode  { return &s.node }
func (s *ValExp) GetKind() string { return s.Kind }
func (s *ValExp) Loc() int        { return s.node.loc }

func (s *RefExp) Node() *AstNode  { return &s.node }
func (s *RefExp) GetKind() string { return s.Kind }
func (s *RefExp) Loc() int        { return s.node.loc }

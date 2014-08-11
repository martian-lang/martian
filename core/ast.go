//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Margo
//
package core

type (
	AstNode struct {
		loc int
	}

	Locatable interface {
		Loc() int
	}

	Filetype struct {
		node AstNode
		id   string
	}

	Dec interface {
		dec()
	}

	Callable interface {
		Node() AstNode
		Loc() int
		Id() string
		InParams() *Params
		OutParams() *Params
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
		Node() AstNode
		Loc() int
		Mode() string
		Tname() string
		Id() string
		Help() string
		IsFile() bool
		SetIsFile(bool)
	}

	InParam struct {
		node   AstNode
		tname  string
		id     string
		help   string
		isfile bool
	}

	OutParam struct {
		node   AstNode
		tname  string
		id     string
		help   string
		isfile bool
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
		exp()
		Kind() string
		ResolveType(*Ast, *Pipeline) (string, error)
	}

	ValExp struct {
		node AstNode
		// union-style multi-value store
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

// Interface whitelist for Dec, Param, Exp, and Stm implementors.
// Patterned after code in Go's ast.go.
func (*Filetype) dec() {}
func (*Stage) dec()    {}
func (*Pipeline) dec() {}
func (*ValExp) exp()   {}
func (*RefExp) exp()   {}

func (s *Filetype) Id() string    { return s.id }
func (s *Filetype) Node() AstNode { return s.node }
func (s *Filetype) Loc() int      { return s.node.loc }

func (s *Stage) Id() string         { return s.id }
func (s *Stage) Node() AstNode      { return s.node }
func (s *Stage) Loc() int           { return s.node.loc }
func (s *Stage) InParams() *Params  { return s.inParams }
func (s *Stage) OutParams() *Params { return s.outParams }

func (s *Pipeline) Id() string         { return s.id }
func (s *Pipeline) Node() AstNode      { return s.node }
func (s *Pipeline) Loc() int           { return s.node.loc }
func (s *Pipeline) InParams() *Params  { return s.inParams }
func (s *Pipeline) OutParams() *Params { return s.outParams }

func (s *CallStm) Id() string { return s.id }
func (s *CallStm) Loc() int   { return s.node.loc }

func (s *InParam) Node() AstNode    { return s.node }
func (s *InParam) Mode() string     { return "in" }
func (s *InParam) Tname() string    { return s.tname }
func (s *InParam) Id() string       { return s.id }
func (s *InParam) Help() string     { return s.help }
func (s *InParam) Loc() int         { return s.node.loc }
func (s *InParam) IsFile() bool     { return s.isfile }
func (s *InParam) SetIsFile(b bool) { s.isfile = b }

func (s *OutParam) Node() AstNode    { return s.node }
func (s *OutParam) Mode() string     { return "out" }
func (s *OutParam) Tname() string    { return s.tname }
func (s *OutParam) Id() string       { return s.id }
func (s *OutParam) Help() string     { return s.help }
func (s *OutParam) Loc() int         { return s.node.loc }
func (s *OutParam) IsFile() bool     { return s.isfile }
func (s *OutParam) SetIsFile(b bool) { s.isfile = b }

func (s *ReturnStm) Loc() int { return s.node.loc }
func (s *BindStm) Loc() int   { return s.node.loc }

func (s *ValExp) Kind() string { return s.kind }
func (s *ValExp) Loc() int     { return s.node.loc }

func (s *RefExp) Kind() string { return s.kind }
func (s *RefExp) Loc() int     { return s.node.loc }

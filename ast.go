package main

type (
	Node struct {
		loc int
	}

	Dec interface {
		dec()
		ID()   string
		Node() Node
	}

	FileTypeDec struct {
		node Node
		id   string
	}

	StageDec struct {
		node     Node
		id       string
		params   []Param
		splitter []Param
	}

	PipelineDec struct {
		node   Node
		id     string
		params []Param
		calls  []*CallStm
		ret    *ReturnStm
	}

	Param interface {
		param()
	}

	InParam struct {
		node  Node
		tname string
		id    string
		help  string
	}

	OutParam struct {
		node  Node
		tname string
		id    string
		help  string
	}

	SourceParam struct {
		node Node
		lang string
		path string
	}

	Stm interface {
		stm()
	}

	BindStm struct {
		node  Node
		id    string
		exp   Exp
		sweep bool
	}

	CallStm struct {
		node     Node
		volatile bool
		id       string
		bindings []*BindStm
	}

	ReturnStm struct {
		node     Node
		bindings []*BindStm
	}

	Exp interface {
		exp()
	}

	ValExp struct {
		node Node
		// union-style multi-value store
		kind string
		fval float64
		ival int64
		sval string
		bval bool
		null bool
	}

	RefExp struct {
		node     Node
		kind     string
		id       string
		outputId string
	}

	Ast struct {
		Decs []Dec
		call *CallStm
	}
)

// Interface whitelist for Dec, Param, Exp, and Stm implementors.
// Patterned after code in Go's ast.go.
func (*FileTypeDec) dec()   {}
func (*StageDec) dec()      {}
func (*PipelineDec) dec()   {}
func (*InParam) param()     {}
func (*OutParam) param()    {}
func (*SourceParam) param() {}
func (*ValExp) exp()        {}
func (*RefExp) exp()        {}
func (*BindStm) stm()       {}
func (*CallStm) stm()       {}
func (*ReturnStm) stm()     {}

func (s *FileTypeDec) ID() string	{ return s.id }
func (s *StageDec) ID() string		{ return s.id }
func (s *PipelineDec) ID() string	{ return s.id }

func (s *FileTypeDec) Node() Node	{ return s.node }
func (s *StageDec) Node() Node		{ return s.node }
func (s *PipelineDec) Node() Node	{ return s.node }

// This global is where we build the AST. It will get passed out
// by the main parsing function.
var ast Ast

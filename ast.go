package main

type (
	Node struct {
		loc int
	}

	Filetype struct {
		node Node
		id   string
	}

	Dec interface {
		dec()
		Node() Node
		ID() string
	}

	Idable interface {
		Node() Node
		ID() string
	}

	Callable interface {
		callable()
		Node() Node
		ID() string
		Params() []Param
	}

	Stage struct {
		node      Node
		id        string
		inparams  []InParam
		outparams []OutParam
		src       Src
		splitter  []Param
	}

	Pipeline struct {
		node      Node
		id        string
		inparams  []InParam
		outparams []OutParam
		calls     []*CallStm
		ret       *ReturnStm
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

	Src struct {
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
		filetypes []*Filetype
		stages    []*Stage
		pipelines []*Pipeline
		callables []Callable
		call      *CallStm
	}
)

// Interface whitelist for Dec, Param, Exp, and Stm implementors.
// Patterned after code in Go's ast.go.
func (*Filetype) dec()   {}
func (*Stage) dec()      {}
func (*Pipeline) dec()   {}
func (*InParam) param()  {}
func (*OutParam) param() {}
func (*Src) param()      {}
func (*ValExp) exp()     {}
func (*RefExp) exp()     {}
func (*BindStm) stm()    {}
func (*CallStm) stm()    {}
func (*ReturnStm) stm()  {}

func (s *Filetype) ID() string { return s.id }
func (s *Filetype) Node() Node { return s.node }

func (s *Stage) callable()       {}
func (s *Stage) ID() string      { return s.id }
func (s *Stage) Node() Node      { return s.node }
func (s *Stage) Params() []Param { return s.params }

func (s *Pipeline) callable()       {}
func (s *Pipeline) ID() string      { return s.id }
func (s *Pipeline) Node() Node      { return s.node }
func (s *Pipeline) Params() []Param { return s.params }

// This global is where we build the AST. It will get passed out
// by the main parsing function.
var ast Ast

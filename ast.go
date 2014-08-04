package main

type (
	Node struct {
		lineno int
	}

	Dec interface {
		dec()
	}

	FileTypeDec struct {
		Node
		id string
	}

	StageDec struct {
		Node
		id       string
		params   []Param
		splitter []Param
	}

	PipelineDec struct {
		Node
		id     string
		params []Param
		calls  []*CallStm
		ret    *ReturnStm
	}

	Param interface {
		param()
	}

	InParam struct {
		Node
		tname string
		id    string
		help  string
	}

	OutParam struct {
		Node
		tname string
		id    string
		help  string
	}

	SourceParam struct {
		Node
		lang string
		path string
	}

	Stm interface {
		stm()
	}

	BindStm struct {
		Node
		id    string
		exp   Exp
		sweep bool
	}

	CallStm struct {
		Node
		volatile bool
		id       string
		bindings []*BindStm
	}

	ReturnStm struct {
		Node
		bindings []*BindStm
	}

	Exp interface {
		exp()
	}

	ValExp struct {
		Node
		// union-style multi-value store
		kind string
		fval float64
		ival int64
		sval string
		bval bool
		null bool
	}

	RefExp struct {
		Node
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

// This global is where we build the AST. It will get passed out
// by the main parsing function.
var ast Ast

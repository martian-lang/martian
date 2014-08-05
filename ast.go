package main

type (
	Node struct {
		loc int
	}

    Locatable interface {
        Loc() int
    }

	Filetype struct {
		node Node
		id   string
	}

	Dec interface {
		dec()
	}

	Callable interface {
		callable()
		Node() Node
        Loc() int
		Id() string
	}

	Stage struct {
		node        Node
		id          string
		inParams    *ParamScope
		outParams   *ParamScope
		src         *Src
		splitParams *ParamScope
	}

	Pipeline struct {
		node      Node
		id        string
		inParams  *ParamScope
		outParams *ParamScope
		calls     []*CallStm
		ret       *ReturnStm
	}

    ParamScope struct {
        params []Param
        table  map[string]Param
    }

    CallScope struct {
        callables []Callable
        table     map[string]Callable
    }

    Param interface {
        param()
        Node() Node
        Loc() int
        Mode() string
        Tname() string
        Id() string
        Help() string
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
        locmap    []FileLoc
		typeTable map[string]bool
        filetypes []*Filetype
		stages    []*Stage
		pipelines []*Pipeline
        callScope *CallScope
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

func (s *Filetype) Id() string { return s.id }
func (s *Filetype) Node() Node { return s.node }
func (s *Filetype) Loc() int   { return s.node.loc }

func (s *Stage) callable()        {}
func (s *Stage) Id() string       { return s.id }
func (s *Stage) Node() Node       { return s.node }
func (s *Stage) Loc() int         { return s.node.loc }

func (s *Pipeline) callable()     {}
func (s *Pipeline) Id() string    { return s.id }
func (s *Pipeline) Node() Node    { return s.node }
func (s *Pipeline) Loc() int      { return s.node.loc }

func (s *InParam) Node() Node     { return s.node }
func (s *InParam) Mode() string   { return "in" }
func (s *InParam) Tname() string  { return s.tname }
func (s *InParam) Id() string     { return s.id }
func (s *InParam) Help() string   { return s.help }
func (s *InParam) Loc() int       { return s.node.loc }

func (s *OutParam) Node() Node    { return s.node }
func (s *OutParam) Mode() string  { return "out" }
func (s *OutParam) Tname() string { return s.tname }
func (s *OutParam) Id() string    { return s.id }
func (s *OutParam) Help() string  { return s.help }
func (s *OutParam) Loc() int      { return s.node.loc }

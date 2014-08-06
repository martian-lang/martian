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
        Node() Node
        Loc() int
        Id() string
        InParams() *Params
        OutParams() *Params
    }

    Stage struct {
        node        Node
        id          string
        inParams    *Params
        outParams   *Params
        src         *Src
        splitParams *Params
    }

    Pipeline struct {
        node      Node
        id        string
        inParams  *Params
        outParams *Params
        calls     []*Call
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

    Binding struct {
        node  Node
        id    string
        exp   Exp
        sweep bool
        tname string
    }

    Bindings struct {
        list  []*Binding
        table map[string]*Binding
    }

    Call struct {
        node     Node
        volatile bool
        id       string
        bindings *Bindings
    }

    ReturnStm struct {
        node     Node
        bindings *Bindings
    }

    Exp interface {
        exp()
        Kind() string
        ResolveType(*Ast, *Pipeline) (string, error)
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
        callables *Callables
        call      *Call
    }
)

// Interface whitelist for Dec, Param, Exp, and Stm implementors.
// Patterned after code in Go's ast.go.
func (*Filetype) dec()   {}
func (*Stage) dec()      {}
func (*Pipeline) dec()   {}
func (*ValExp) exp()     {}
func (*RefExp) exp()     {}

func (s *Filetype) Id() string    { return s.id }
func (s *Filetype) Node() Node    { return s.node }
func (s *Filetype) Loc() int      { return s.node.loc }

func (s *Stage) Id() string       { return s.id }
func (s *Stage) Node() Node       { return s.node }
func (s *Stage) Loc() int         { return s.node.loc }
func (s *Stage) InParams() *Params { return s.inParams }
func (s *Stage) OutParams() *Params { return s.outParams }

func (s *Pipeline) Id() string    { return s.id }
func (s *Pipeline) Node() Node    { return s.node }
func (s *Pipeline) Loc() int      { return s.node.loc }
func (s *Pipeline) InParams() *Params { return s.inParams }
func (s *Pipeline) OutParams() *Params { return s.outParams }

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

func (s *ReturnStm) Loc() int     { return s.node.loc }
func (s *Call) Loc() int          { return s.node.loc }
func (s *Binding) Loc() int       { return s.node.loc }

func (s *ValExp) Kind() string    { return s.kind }
func (s *ValExp) Loc() int        { return s.node.loc }

func (s *RefExp) Kind() string    { return s.kind }
func (s *RefExp) Loc() int { return s.node.loc }
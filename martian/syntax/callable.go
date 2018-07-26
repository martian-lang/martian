// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// Stage and pipeline types.

package syntax

type (
	// A Callable object is a stage or pipeline which can be called.
	Callable interface {
		AstNodable
		GetId() string
		GetInParams() *InParams
		GetOutParams() *OutParams
		Type() string
		format(printer *printer)
		EquivalentTo(other Callable,
			myCallables, otherCallables *Callables) bool
	}

	// An ordered set of Callable objects.
	Callables struct {
		List []Callable `json:"-"`

		// Lookup table of callables by Id.  Populated during compile.
		Table map[string]Callable
	}

	// An ordered set of parameters.
	InParams struct {
		List []*InParam

		// Lookup table of params by Id.  Populated during compile.
		Table map[string]*InParam
	}

	// An ordered set of parameters.
	OutParams struct {
		List []*OutParam

		// Lookup table of params by Id.  Populated during compile.
		Table map[string]*OutParam
	}

	Param interface {
		AstNodable
		getMode() string
		GetTname() string
		GetArrayDim() int
		GetId() string
		GetHelp() string
		GetOutName() string
		IsFile() bool
		setIsFile(bool)
	}

	InParam struct {
		Node     AstNode
		Tname    string
		Id       string
		Help     string
		ArrayDim int16
		Isfile   bool
	}

	OutParam struct {
		Node     AstNode
		Tname    string
		Id       string
		Help     string
		OutName  string
		ArrayDim int16
		Isfile   bool
	}

	Stage struct {
		Node      AstNode
		Id        string
		InParams  *InParams
		OutParams *OutParams
		Retain    *RetainParams
		Src       *SrcParam
		ChunkIns  *InParams
		ChunkOuts *OutParams
		Resources *Resources
		Split     bool
	}

	// To simplify implementation of the parser, this stores the stage's
	// ChunkIns and ChunkOuts.
	paramsTuple struct {
		Present bool
		Ins     *InParams
		Outs    *OutParams
	}

	RetainParams struct {
		Node   AstNode
		Params []*RetainParam
	}

	RetainParam struct {
		Node AstNode
		Id   string
	}

	// The name of the stage language.  Must be one of
	// py, exec, or comp.
	//
	// py stages are run via the python adapter.
	//
	// comp stages are expected to be compiled binaries which
	// are executed by mrjob.
	//
	// exec stages are run directly by mrp and must take care of
	// everything themselves.  This mode is not recommended and
	// exists mainly for backwards compatibility.
	StageLanguage string

	// Stage executable declaration.
	SrcParam struct {
		Node AstNode
		Lang StageLanguage
		Path string
		Args []string
	}

	// Stage resouce definitions.
	Resources struct {
		Node         AstNode
		ThreadNode   *AstNode
		MemNode      *AstNode
		SpecialNode  *AstNode
		VolatileNode *AstNode

		Special        string
		Threads        int16
		MemGB          int16
		StrictVolatile bool
	}

	Pipeline struct {
		Node      AstNode
		Id        string
		InParams  *InParams
		OutParams *OutParams
		Calls     []*CallStm
		Callables *Callables `json:"-"`
		Ret       *ReturnStm
		Retain    *PipelineRetains
	}

	// Specifies the set of references which may or may not also be
	// returned but which should not be removed by VDR under any
	// circumstances.
	PipelineRetains struct {
		Node AstNode
		Refs []*RefExp
	}

	// The set of bindings for the return values of a pipeline.
	ReturnStm struct {
		Node     AstNode
		Bindings *BindStms
	}
)

func (p *PipelineRetains) getNode() *AstNode     { return &p.Node }
func (s *PipelineRetains) File() *SourceFile     { return s.Node.Loc.File }
func (s *PipelineRetains) inheritComments() bool { return true }
func (s *PipelineRetains) getSubnodes() []AstNodable {
	params := make([]AstNodable, 0, len(s.Refs))
	for _, p := range s.Refs {
		params = append(params, p)
	}
	return params
}

func (p *RetainParams) getNode() *AstNode     { return &p.Node }
func (s *RetainParams) File() *SourceFile     { return s.Node.Loc.File }
func (s *RetainParams) inheritComments() bool { return true }
func (s *RetainParams) getSubnodes() []AstNodable {
	params := make([]AstNodable, 0, len(s.Params))
	for _, p := range s.Params {
		params = append(params, p)
	}
	return params
}

func (s *ReturnStm) inheritComments() bool { return false }
func (s *ReturnStm) getSubnodes() []AstNodable {
	return []AstNodable{s.Bindings}
}

func (s *RetainParam) getNode() *AstNode         { return &s.Node }
func (s *RetainParam) File() *SourceFile         { return s.Node.Loc.File }
func (s *RetainParam) getSubnodes() []AstNodable { return nil }
func (s *RetainParam) inheritComments() bool     { return false }

func (*Stage) getDec()                    {}
func (*Pipeline) getDec()                 {}
func (s *Stage) GetId() string            { return s.Id }
func (s *Stage) getNode() *AstNode        { return &s.Node }
func (s *Stage) File() *SourceFile        { return s.Node.Loc.File }
func (s *Stage) GetInParams() *InParams   { return s.InParams }
func (s *Stage) GetOutParams() *OutParams { return s.OutParams }
func (s *Stage) Type() string             { return "stage" }

func (s *Stage) inheritComments() bool { return false }
func (s *Stage) getSubnodes() []AstNodable {
	subs := make([]AstNodable, 0, 2+
		len(s.InParams.List)+len(s.OutParams.List)+
		len(s.ChunkIns.List)+len(s.ChunkOuts.List))
	for _, n := range s.InParams.List {
		subs = append(subs, n)
	}
	for _, n := range s.OutParams.List {
		subs = append(subs, n)
	}
	subs = append(subs, s.Src)
	for _, n := range s.ChunkIns.List {
		subs = append(subs, n)
	}
	for _, n := range s.ChunkOuts.List {
		subs = append(subs, n)
	}
	if s.Resources != nil {
		subs = append(subs, s.Resources)
	}
	if s.Retain != nil {
		subs = append(subs, s.Retain)
	}
	return subs
}

func (s *Resources) getNode() *AstNode     { return &s.Node }
func (s *Resources) File() *SourceFile     { return s.Node.Loc.File }
func (s *Resources) inheritComments() bool { return false }
func (s *Resources) getSubnodes() []AstNodable {
	subs := make([]AstNodable, 0, 3)
	if s.ThreadNode != nil {
		subs = append(subs, s.ThreadNode)
	}
	if s.MemNode != nil {
		subs = append(subs, s.MemNode)
	}
	if s.SpecialNode != nil {
		subs = append(subs, s.SpecialNode)
	}
	if s.VolatileNode != nil {
		subs = append(subs, s.VolatileNode)
	}
	return subs
}

func (s *Pipeline) GetId() string            { return s.Id }
func (s *Pipeline) getNode() *AstNode        { return &s.Node }
func (s *Pipeline) File() *SourceFile        { return s.Node.Loc.File }
func (s *Pipeline) GetInParams() *InParams   { return s.InParams }
func (s *Pipeline) GetOutParams() *OutParams { return s.OutParams }
func (s *Pipeline) Type() string             { return "pipeline" }

func (s *Pipeline) inheritComments() bool { return false }
func (s *Pipeline) getSubnodes() []AstNodable {
	subs := make([]AstNodable, 0, 1+
		len(s.InParams.List)+len(s.OutParams.List)+len(s.Calls))
	for _, n := range s.InParams.List {
		subs = append(subs, n)
	}
	for _, n := range s.OutParams.List {
		subs = append(subs, n)
	}
	for _, n := range s.Calls {
		subs = append(subs, n)
	}
	subs = append(subs, s.Ret)
	if s.Retain != nil {
		subs = append(subs, s.Retain)
	}
	return subs
}

func (s *InParam) getNode() *AstNode  { return &s.Node }
func (s *InParam) File() *SourceFile  { return s.Node.Loc.File }
func (s *InParam) getMode() string    { return "in" }
func (s *InParam) GetTname() string   { return s.Tname }
func (s *InParam) GetArrayDim() int   { return int(s.ArrayDim) }
func (s *InParam) GetId() string      { return s.Id }
func (s *InParam) GetHelp() string    { return s.Help }
func (s *InParam) GetOutName() string { return "" }
func (s *InParam) IsFile() bool       { return s.Isfile }
func (s *InParam) setIsFile(b bool)   { s.Isfile = b }

func (s *InParam) inheritComments() bool { return false }
func (s *InParam) getSubnodes() []AstNodable {
	return nil
}

func (s *OutParam) getNode() *AstNode  { return &s.Node }
func (s *OutParam) File() *SourceFile  { return s.Node.Loc.File }
func (s *OutParam) getMode() string    { return "out" }
func (s *OutParam) GetTname() string   { return s.Tname }
func (s *OutParam) GetArrayDim() int   { return int(s.ArrayDim) }
func (s *OutParam) GetId() string      { return s.Id }
func (s *OutParam) GetHelp() string    { return s.Help }
func (s *OutParam) GetOutName() string { return s.OutName }
func (s *OutParam) IsFile() bool       { return s.Isfile }
func (s *OutParam) setIsFile(b bool)   { s.Isfile = b }

func (s *OutParam) inheritComments() bool { return false }
func (s *OutParam) getSubnodes() []AstNodable {
	return nil
}

func (s *SrcParam) getNode() *AstNode         { return &s.Node }
func (s *SrcParam) File() *SourceFile         { return s.Node.Loc.File }
func (s *SrcParam) inheritComments() bool     { return false }
func (s *SrcParam) getSubnodes() []AstNodable { return nil }

func (s *ReturnStm) getNode() *AstNode { return &s.Node }
func (s *ReturnStm) File() *SourceFile { return s.Node.Loc.File }

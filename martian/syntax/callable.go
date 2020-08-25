// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Stage and pipeline types.

package syntax

type (
	// A Callable object is a stage or pipeline which can be called.
	Callable interface {
		NamedNode
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
		cmd  string
		Type StageCodeType
		Path string
		Args []string
	}

	// Stage resource definitions.
	Resources struct {
		Node         AstNode
		ThreadNode   *AstNode
		MemNode      *AstNode
		VMemNode     *AstNode
		SpecialNode  *AstNode
		VolatileNode *AstNode

		Special        string
		Threads        float32
		MemGB          float32
		VMemGB         float32
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
func (s *PipelineRetains) Line() int             { return s.Node.Loc.Line }
func (s *PipelineRetains) inheritComments() bool { return true }
func (s *PipelineRetains) getSubnodes() []AstNodable {
	params := make([]AstNodable, 0, len(s.Refs))
	for _, p := range s.Refs {
		params = append(params, p)
	}
	return params
}

func (s *ReturnStm) inheritComments() bool { return false }
func (s *ReturnStm) getSubnodes() []AstNodable {
	return []AstNodable{s.Bindings}
}

func (*Stage) getDec()    {}
func (*Pipeline) getDec() {}

// GetId returns the name of the stage.
func (s *Stage) GetId() string            { return s.Id }
func (s *Stage) getNode() *AstNode        { return &s.Node }
func (s *Stage) File() *SourceFile        { return s.Node.Loc.File }
func (s *Stage) Line() int                { return s.Node.Loc.Line }
func (s *Stage) GetInParams() *InParams   { return s.InParams }
func (s *Stage) GetOutParams() *OutParams { return s.OutParams }

// Type returns "stage".
func (s *Stage) Type() string { return KindStage.str() }

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
func (s *Resources) Line() int             { return s.Node.Loc.Line }
func (s *Resources) inheritComments() bool { return false }
func (s *Resources) getSubnodes() []AstNodable {
	subnodes := [...]*AstNode{
		s.MemNode,
		s.SpecialNode,
		s.ThreadNode,
		s.VMemNode,
		s.VolatileNode,
	}
	n := 0
	for _, node := range subnodes {
		if node != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	subs := make([]AstNodable, 0, n)
	for _, node := range subnodes {
		if node != nil {
			subs = append(subs, node)
		}
	}
	return subs
}

// GetId returns the name of the pipeline.
func (s *Pipeline) GetId() string {
	if s == nil {
		return ""
	}
	return s.Id
}
func (s *Pipeline) getNode() *AstNode        { return &s.Node }
func (s *Pipeline) File() *SourceFile        { return s.Node.Loc.File }
func (s *Pipeline) Line() int                { return s.Node.Loc.Line }
func (s *Pipeline) GetInParams() *InParams   { return s.InParams }
func (s *Pipeline) GetOutParams() *OutParams { return s.OutParams }

// Type returns "pipeline".
func (s *Pipeline) Type() string { return KindPipeline.str() }

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

func (s *Pipeline) findCall(id string) *CallStm {
	for _, call := range s.Calls {
		if call.Id == id {
			return call
		}
	}
	return nil
}

func (s *SrcParam) getNode() *AstNode         { return &s.Node }
func (s *SrcParam) File() *SourceFile         { return s.Node.Loc.File }
func (s *SrcParam) Line() int                 { return s.Node.Loc.Line }
func (s *SrcParam) inheritComments() bool     { return false }
func (s *SrcParam) getSubnodes() []AstNodable { return nil }

func (s *ReturnStm) getNode() *AstNode { return &s.Node }
func (s *ReturnStm) File() *SourceFile { return s.Node.Loc.File }
func (s *ReturnStm) Line() int         { return s.Node.Loc.Line }

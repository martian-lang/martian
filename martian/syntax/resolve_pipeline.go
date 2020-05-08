// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Code for resolving the pipeline graph.

package syntax

// CallGraphPipeline represents a pipeline in a call graph.
type CallGraphPipeline struct {
	CallGraphStage
	Children []CallGraphNode `json:"children"`
	pipeline *Pipeline
	Retain   []*RefExp `json:"retained,omitempty"`
}

// Kind returns KindPipeline.
func (c *CallGraphPipeline) Kind() CallGraphNodeType {
	return KindPipeline
}

// Returns the nodes of any stages or subpipelines called by this pipeline.
func (c *CallGraphPipeline) GetChildren() []CallGraphNode {
	return c.Children
}

// The ast node for this pipeline.
func (c *CallGraphPipeline) Callable() Callable {
	return c.pipeline
}

// If the top-level call is a stage, not a pipeline, create a "fake" pipeline
// which wraps that stage.
func wrapStageAsPipeline(call *CallStm, stage *Stage) *Pipeline {
	returns := &BindStms{
		List:  make([]*BindStm, 0, len(stage.OutParams.List)),
		Table: make(map[string]*BindStm, len(stage.OutParams.List)),
	}
	for _, param := range stage.OutParams.List {
		binding := &BindStm{
			Id:    param.Id,
			Tname: param.Tname,
			Exp: &RefExp{
				Kind:     KindCall,
				Id:       stage.Id,
				OutputId: param.Id,
			},
		}
		returns.List = append(returns.List, binding)
		returns.Table[param.Id] = binding
	}
	return &Pipeline{
		Node:      stage.Node,
		Id:        stage.Id,
		InParams:  stage.InParams,
		OutParams: stage.OutParams,
		Calls:     []*CallStm{call},
		Callables: &Callables{
			List: []Callable{stage},
			Table: map[string]Callable{
				stage.Id: stage,
			},
		},
		Ret: &ReturnStm{Bindings: returns},
	}
}

func (pipe *CallGraphPipeline) makeChildNodes(prefix string, ast *Ast) error {
	if len(pipe.pipeline.Calls) > 0 {
		pipe.Children = make([]CallGraphNode, len(pipe.pipeline.Calls))
		var errs ErrorList
		for i, c := range pipe.pipeline.Calls {
			n, err := ast.makeCallGraphNodes(prefix, c, pipe, false)
			if err != nil {
				errs = append(errs, &wrapError{
					innerError: err,
					loc:        pipe.call.Node.Loc,
				})
			}
			pipe.Children[i] = n
		}
		if pipe.Children[0] != nil {
			cid := pipe.Children[0].GetFqid()
			// slice a child fqid, share the memory for it.
			pipe.Fqid = cid[:len(cid)-1-len(pipe.Children[0].Call().Id)]
		}
		if err := errs.If(); err != nil {
			return err
		} else if pipe.Children[0] == nil {
			panic("nil child")
		}
	} else {
		pipe.Fqid = makeFqid(prefix, pipe.call, pipe.Parent)
	}
	return nil
}

func hasMerge(exp Exp, source MapCallSource) bool {
	switch exp := exp.(type) {
	case *RefExp:
		for _, src := range exp.MergeOver {
			if src == source {
				return true
			}
		}
	case *ArrayExp:
		for _, v := range exp.Value {
			if hasMerge(v, source) {
				return true
			}
		}
	case *MapExp:
		for _, v := range exp.Value {
			if hasMerge(v, source) {
				return true
			}
		}
	case *SplitExp:
		if exp.Source == source {
			return false
		}
		merge := hasMerge(exp.Value, source)
		return merge
	case *DisabledExp:
		return hasMerge(exp.Value, source) ||
			hasMerge(exp.Disabled, source)
	}
	return false
}

// Find all splits tied to the given call and promote them.
// So for example {a: split [1,2], b: 2, c: {d: split [4,5]}}
// turns into [{a:1,b:2,c:{d:4}},{a:2,b:2,c:{d:5}}].  References can't
// be promoted in this way.
func unsplit(exp Exp, fork map[MapCallSource]CollectionIndex, call ForkRootList) (Exp, error) {
	if len(call) == 0 {
		return exp.BindingPath("", fork, nil)
	}
	// Deal with the simple cases, which do not need a length or key.
	switch exp := exp.(type) {
	case *SplitExp:
		if exp.Source == call[len(call)-1].MapSource() {
			return unsplit(exp.Value, fork, call[:len(call)-1])
		} else if exp.Source == call[0].MapSource() {
			return unsplit(exp.Value, fork, call[1:])
		} else if len(call) > 2 {
			for i, c := range call[1 : len(call)-2] {
				if exp.Source == c.MapSource() {
					return unsplit(exp.Value, fork, append(call[:i+1:i+1], call[i+2:]...))
				}
			}
		}
		return exp, &bindingError{Msg: "unmatched split call"}
	case *RefExp:
		exp.Simplify()
		if _, ok := fork[call[0].MapSource()]; !ok {
			fork[call[0].MapSource()] = unknownIndex{src: call[0].MapSource()}
			defer delete(fork, call[0].MapSource())
		}
		return exp.BindingPath("", fork, nil)
	case *NullExp:
		return exp, nil
	case *DisabledExp:
		disable, err := unsplit(exp.Disabled, fork, call)
		if err != nil {
			return exp, err
		}
		if disable, ok := disable.(*BoolExp); ok {
			if disable.Value {
				return &NullExp{
					valExp: valExp{Node: disable.Node},
				}, nil
			} else {
				return unsplit(exp.Value, fork, call)
			}
		}
		if inner, err := unsplit(exp.Value, fork, call); err != nil {
			return exp, err
		} else {
			return exp.makeDisabledExp(disable, inner)
		}
	}
	if !call[0].MapSource().KnownLength() {
		return exp, nil
	}
	switch call[0].MapSource().CallMode() {
	case ModeNullMapCall:
		return &NullExp{valExp: valExp{Node: *exp.getNode()}}, nil
	case ModeArrayCall:
		arr := ArrayExp{
			valExp: valExp{Node: *exp.getNode()},
			Value:  make([]Exp, call[0].MapSource().ArrayLength()),
		}
		var errs ErrorList
		for i := range arr.Value {
			ii := arrayIndex(i)
			fork[call[0].MapSource()] = &ii
			e, err := unsplit(exp, fork, call[1:])
			if err != nil {
				errs = append(errs, err)
			}
			arr.Value[i] = e
		}
		delete(fork, call[0].MapSource())
		return &arr, errs.If()
	case ModeMapCall:
		m := MapExp{
			valExp: valExp{Node: *exp.getNode()},
			Value:  make(map[string]Exp, len(call[0].MapSource().Keys())),
			Kind:   KindMap,
		}
		var errs ErrorList
		for i := range call[0].MapSource().Keys() {
			fork[call[0].MapSource()] = mapKeyIndex(i)
			e, err := unsplit(exp, fork, call[1:])
			if err != nil {
				errs = append(errs, err)
			}
			m.Value[i] = e
		}
		delete(fork, call[0].MapSource())
		return &m, errs.If()
	default:
		panic("invalid map call type")
	}
}

func (node *CallGraphPipeline) resolve(siblings map[string]*ResolvedBinding,
	mapped ForkRootList, lookup *TypeLookup) error {
	if err := node.resolveInputs(siblings, mapped, lookup); err != nil {
		return err
	}
	if node.isAlwaysDisabled() {
		// No point in going further.  Trim away the child nodes and return
		// null.
		node.Children = nil
		node.Outputs = &ResolvedBinding{
			Exp: &NullExp{
				valExp: valExp{Node: node.pipeline.Node},
			},
			Type: &builtinNull,
		}
		node.Disable = alwaysDisable(node.Disable)
		return nil
	}
	if node.call.CallMode() != ModeSingleCall {
		if node.Source == nil {
			panic("nil source for mapped call " + node.Fqid)
		}
		mapped = append(mapped, &node.CallGraphStage)
	}
	var childMap map[string]*ResolvedBinding
	if len(node.Children) > 0 {
		var errs ErrorList
		rootSet := make(map[CallGraphNode]struct{}, len(mapped)+1)
		for _, n := range mapped {
			rootSet[n] = struct{}{}
		}
		childMap = make(map[string]*ResolvedBinding, len(node.Children))
		for _, child := range node.Children {
			if err := child.resolve(childMap, mapped, lookup); err != nil {
				errs = append(errs, &wrapError{
					innerError: err,
					loc:        node.Call().Node.Loc,
				})
			}
			childMap[child.Call().Id] = child.ResolvedOutputs()
			for _, f := range child.ForkRoots() {
				if _, ok := rootSet[f]; !ok {
					rootSet[f] = struct{}{}
					mapped = append(mapped, f)
				}
			}
		}
		// Stages which forked based on upstream splits should not propagate
		// their splits to the outputs, so we remove them here.
		for _, child := range node.Children {
			if stage, ok := child.(*CallGraphStage); ok {
				if err := stage.unsplit(); err != nil {
					errs = append(errs, err)
				}
			}
		}
		if err := errs.If(); err != nil {
			return err
		}
	}
	errs := node.resolvePipelineOuts(childMap, lookup)
	if r := node.pipeline.Retain; r != nil && len(r.Refs) > 0 {
		node.Retain = make([]*RefExp, 0, len(r.Refs))
		for _, ref := range r.Refs {
			resolved, err := ref.resolveRefs(node.Inputs, childMap,
				lookup, true)
			if err != nil {
				errs = append(errs, err)
			}
			if resolved != nil {
				node.Retain = append(node.Retain, resolved.FindRefs()...)
			}
		}
	}
	return errs.If()
}

func (node *CallGraphPipeline) makeOutExp(
	childMap map[string]*ResolvedBinding,
	lookup *TypeLookup) (Exp, error) {
	outs, err := node.pipeline.Ret.Bindings.resolve(node.Inputs, childMap,
		lookup, false)
	if err != nil {
		return nil, &bindingError{
			Msg: node.Fqid + " outputs",
			Err: err,
		}
	}
	var errs ErrorList
	value := make(map[string]Exp, len(outs))
	for k, out := range outs {
		outVal := updateMapSources(node.call, node.Source, out.Exp)
		value[k] = outVal
	}

	return &MapExp{
		valExp: valExp{Node: node.pipeline.Ret.Node},
		Kind:   KindStruct,
		Value:  value,
	}, errs.If()
}

func (node *CallGraphPipeline) resolvePipelineOuts(
	childMap map[string]*ResolvedBinding,
	lookup *TypeLookup) ErrorList {
	var errs ErrorList
	if len(node.pipeline.Ret.Bindings.List) > 0 {
		exp, err := node.makeOutExp(childMap, lookup)
		if err != nil {
			errs = append(errs, err)
		}

		tid := TypeId{Tname: node.pipeline.Id}

		if len(node.Forks) > 0 {
			m := make(map[MapCallSource]CollectionIndex, 1)
			for i := len(node.Forks) - 1; i >= 0; i-- {
				fs := node.Forks[i]
				if fs.MapSource() == nil {
					panic(fs.GetFqid() + " has no source")
				}

				switch fs.MapSource().CallMode() {
				case ModeArrayCall:
					tid.ArrayDim++
				case ModeMapCall:
					tid.MapDim = tid.ArrayDim + 1
					tid.ArrayDim = 0
				}
				if !hasMerge(exp, fs.MapSource()) {
					e, err := unsplit(exp, m, node.Forks[i:i+1])
					if err != nil {
						errs = append(errs, err)
					}
					exp = e

					if i == 0 {
						node.Forks = node.Forks[1:]
						i--
					} else {
						node.Forks = append(node.Forks[:i], node.Forks[i+1:]...)
						i--
					}
				}
			}
		}
		if len(node.Disable) > 0 && (node.Parent == nil ||
			len(node.Disable) > len(node.Parent.Disable)) {
			var d *DisabledExp
			exp, err = d.makeDisabledExp(node.Disable[len(node.Disable)-1], exp)
			if err != nil {
				errs = append(errs, err)
			}
		}
		node.Outputs = &ResolvedBinding{
			Exp:  exp,
			Type: lookup.Get(tid),
		}
	} else {
		node.Outputs = &ResolvedBinding{
			Exp:  &NullExp{valExp: valExp{Node: node.call.Node}},
			Type: &builtinNull,
		}
	}
	return errs
}

func (node *CallGraphPipeline) unsplit() error {
	var errs ErrorList
	for _, c := range node.Children {
		if err := c.unsplit(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(node.Forks) > 0 {
		node.resolvePipelineForks(node.Forks)
		e, err := unsplit(node.Outputs.Exp,
			make(map[MapCallSource]CollectionIndex,
				len(node.Forks)), node.Forks)
		if err != nil {
			errs = append(errs, err)
		}
		node.Outputs.Exp = e
	}
	mapped := node.Forks
	node.Forks = nil
	node.resolvePipelineForks(mapped)
	return errs.If()
}

func (node *CallGraphPipeline) resolvePipelineForks(mapped ForkRootList) {
	if len(mapped) == 0 {
		node.Forks = nil
		return
	}
	splits := make(map[*CallStm]struct{}, len(mapped))
	findSplitCalls(node.Outputs.Exp, splits, true)
	for _, d := range node.Disable {
		findSplitCalls(d, splits, true)
	}
	for _, m := range mapped {
		if _, ok := splits[m.Call()]; !ok {
			if !m.MapSource().KnownLength() && hasMerge(node.Outputs.Exp, m.MapSource()) {
				splits[m.Call()] = struct{}{}
			}
		}
	}
	for _, n := range node.Forks {
		delete(splits, n.Call())
	}
	if len(splits) > 0 {
		if cap(node.Forks) < len(node.Forks)+len(splits) {
			f := make(ForkRootList, len(node.Forks), len(node.Forks)+len(splits))
			copy(f, node.Forks)
			node.Forks = f
		}
		for _, src := range mapped {
			if _, ok := splits[src.Call()]; ok {
				node.Forks = append(node.Forks, src)
			}
		}
	}
}

// Retained values, exempt from VDR.
func (node *CallGraphPipeline) Retained() []*RefExp {
	return node.Retain
}

func (node *CallGraphPipeline) NodeClosure() map[string]CallGraphNode {
	m := make(map[string]CallGraphNode, 16+4*len(node.Children))
	node.nodeClosure(m)
	return m
}
func (node *CallGraphPipeline) nodeClosure(m map[string]CallGraphNode) {
	m[node.Fqid] = node
	for _, c := range node.Children {
		c.nodeClosure(m)
	}
}

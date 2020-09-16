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
				Id:       call.Id,
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
		if node.split == nil {
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
				if err := stage.unsplit(lookup); err != nil {
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
				lookup)
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
		lookup)
	if err != nil {
		return nil, &wrapError{
			innerError: &bindingError{
				Msg: node.Fqid + " outputs",
				Err: err,
			},
			loc: node.pipeline.Ret.Node.Loc,
		}
	}
	value := make(map[string]Exp, len(outs))
	for k, out := range outs {
		value[k] = out.Exp
	}

	return &MapExp{
		valExp: valExp{Node: node.pipeline.Ret.Node},
		Kind:   KindStruct,
		Value:  value,
	}, nil
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

		if len(node.Disable) > 0 && (node.Parent == nil ||
			len(node.Disable) > len(node.Parent.Disable)) {
			var d *DisabledExp
			exp, err = d.makeDisabledExp(node.Disable[len(node.Disable)-1], exp)
			if err != nil {
				errs = append(errs, err)
			}
		}
		tid := TypeId{Tname: node.pipeline.Id}
		if node.split != nil {
			switch node.split.Source.CallMode() {
			case ModeArrayCall:
				tid.ArrayDim++
			case ModeMapCall:
				tid.MapDim++
			default:
				errs = append(errs, &wrapError{
					innerError: &bindingError{
						Msg: "invalid mapping mode: " +
							node.split.Source.CallMode().String() + " for " +
							node.split.GoString(),
					},
					loc: node.split.Node.Loc,
				})
			}
			exp = &MergeExp{
				Call:      &node.CallGraphStage,
				MergeOver: node.split.Source,
				Value:     exp,
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

func (node *CallGraphPipeline) unsplit(lookup *TypeLookup) error {
	var errs ErrorList
	for _, c := range node.Children {
		if err := c.unsplit(lookup); err != nil {
			errs = append(errs, err)
		}
	}
	e := node.Outputs.Exp
	t := node.Outputs.Type
	for i := len(node.Forks) - 1; i >= 0; i-- {
		if f := node.Forks[i]; f != &node.CallGraphStage {
			if f.isAlwaysDisabled() {
				e = &NullExp{
					valExp: valExp{Node: *e.getNode()},
				}
				t = nullType{}
			} else {
				e = &MergeExp{
					Call:      f,
					MergeOver: f.split.Source,
					Value:     e,
				}
				if nt, err := lookup.AddDim(t, f.CallMode()); err != nil {
					errs = append(errs, err)
				} else if nt == nil {
					panic("nil type is not a valid expansion for " + t.TypeId().str())
				} else {
					t = nt
				}
			}
		}
	}
	e, err := e.BindingPath("", nil, lookup)
	if err != nil {
		errs = append(errs, &bindingError{
			Msg: node.Fqid + " outputs",
			Err: err,
		})
	}
	node.Outputs.Exp, node.Outputs.Type, node.Forks = unmergeExp(e, t, lookup, node.Forks[:0])
	var splitCalls map[*CallStm]struct{}
	for k, binding := range node.Inputs {
		// Ensure inputs can be scanned for refs, and also that their
		// types are cached.  Otherwise, at runtime mrp may end up trying to
		// cache the types concurrently.
		if refs, err := binding.FindRefs(lookup); err != nil {
			errs = append(errs, &bindingError{
				Msg: node.Fqid + " input " + k,
				Err: err,
			})
		} else if len(refs) > 0 {
			if splitCalls == nil {
				splitCalls = make(map[*CallStm]struct{}, len(node.Forks)+1)
			}
			for _, ref := range refs {
				for c, i := range ref.Exp.Forks {
					if i.IndexSource() != nil {
						splitCalls[c] = struct{}{}
					}
				}
			}
			for _, f := range node.Forks {
				delete(splitCalls, f.call)
			}
			for n := node; n != nil; n = n.Parent {
				if _, ok := splitCalls[n.call]; ok {
					m := &MergeExp{
						Call:      &n.CallGraphStage,
						MergeOver: n.split.Source,
						Value:     binding.Exp,
					}
					s := &SplitExp{
						valExp: valExp{Node: *binding.Exp.getNode()},
						Call:   n.call,
						Value:  m,
						Source: n.split.Source,
					}
					if exp, err := s.BindingPath("", nil, lookup); err != nil {
						errs = append(errs, &bindingError{
							Msg: "making pipeline input splits for " + node.Fqid,
							Err: err,
						})
					} else {
						binding.Exp = exp
					}
				}
			}
			for k := range splitCalls {
				delete(splitCalls, k)
			}
		}
	}
	if _, err := node.Outputs.FindRefs(lookup); err != nil {
		errs = append(errs, &bindingError{
			Msg: node.Fqid + " outputs",
			Err: err,
		})
	}
	return errs.If()
}

// Remove merges.  For pipelines, we merge all of the output forks, in the hopes
// that we can do them statically, but if any are left over then we need to get
// rid of them.
func unmergeExp(exp Exp, t Type, lookup *TypeLookup, forks ForkRootList) (Exp, Type, ForkRootList) {
	switch exp := exp.(type) {
	case *MergeExp:
		switch exp.MergeOver.CallMode() {
		case ModeArrayCall:
			t = lookup.GetArray(t, -1)
		case ModeMapCall:
			t = t.(*TypedMapType).Elem
		}
		return unmergeExp(exp.Value, t, lookup, append(forks, exp.Call))
	}
	return exp, t, forks
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

// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// Code for resolving the pipeline graph.

package syntax

import (
	"fmt"
	"sort"
	"strings"
)

type (
	// ResolvedBinding contains information about how a binding gets resolved
	// in a call graph.
	ResolvedBinding struct {
		// If the binding resolved to a literal expression in mro, this returns
		// that expression.  Any parts of the expression are bound to the outputs
		// of a stage, the ID in the resulting *RefExp will be the Fqid of the
		// stage, not the base ID of the stage as it would be for a *RefExp found
		// in an AST.
		Exp Exp `json:"expression"`
		// The type for the binding.  For inputs, this is the expected input
		// type, not the output type of the bound node.
		Type Type `json:"type"`
	}

	CallGraphNodeType int

	// CallGraphNode describes a node in the graph of top-level calls.
	CallGraphNode interface {
		// The pipeline, if any, which contains this node.
		GetParent() *CallGraphPipeline
		// If this is a pipeline, the stages or subpipelines called
		// by this stage.
		GetChildren() []CallGraphNode
		// The fully-qualified ID suffix for this node. That is
		// e.g. ...id(grandparent).id(parent).id(this).
		//
		// Unlike the FQName() used in core.Node, this one lacks the
		// ID.PipestanceId. prefix, but is otherwise the same.
		GetFqid() string
		// The ast node for this callable object.
		Callable() Callable
		// The call, either in the context of the parent or the top-level call.
		Call() *CallStm

		// Kind returns either KindPipeline or KindStage depending on the node
		// type.
		Kind() CallGraphNodeType

		// The resolved input bindings for this node.  References are followed
		// until either a literal expression is found or a stage node is
		// encountered.
		ResolvedInputs() map[string]*ResolvedBinding
		// The resolved input bindings for this node.  References are followed
		// until either a literal expression is found or a stage node is
		// encountered.  For stage nodes, this will simply be the stage FQID
		// and output name.
		ResolvedOutputs() *ResolvedBinding

		// DisableBindings returns the set of expressions which could disable a
		// stage or pipeline.
		//
		// A stage or pipeline is disabled if it is called like
		//
		//   call STAGE (
		//       ...
		//   ) using (
		//       disabled = FOO.value,
		//   )
		//
		// or if its parent pipeline is disabled.  If an expression disabling
		// this node or any of its parent pipelines evaluates to a constant
		// true, then only that constant expression will be returned.
		// Otherwise, the set of reference expressions which could disable this
		// node or any of its parents is returned.
		Disabled() []Exp

		// Get the set of all nodes represented by this node or any of its
		// children.
		NodeClosure() map[string]CallGraphNode

		// Retained values.
		Retained() []*RefExp

		nodeClosure(map[string]CallGraphNode)

		resolve(map[string]*ResolvedBinding, *TypeLookup) error
	}

	// CallGraphStage represents a stage in a call graph.
	CallGraphStage struct {
		Parent  *CallGraphPipeline `json:"-"`
		Fqid    string             `json:"fqid"`
		call    *CallStm
		stage   *Stage
		Inputs  map[string]*ResolvedBinding `json:"inputs"`
		Outputs *ResolvedBinding            `json:"outputs"`
		Disable []Exp                       `json:"disabled,omitempty"`
	}

	// CallGraphPipeline represents a pipeline in a call graph.
	CallGraphPipeline struct {
		CallGraphStage
		Children []CallGraphNode `json:"children"`
		pipeline *Pipeline
		Retain   []*RefExp `json:"retained,omitempty"`
	}
)

const (
	KindStage    CallGraphNodeType = iota
	KindPipeline CallGraphNodeType = iota
)

func (k *CallGraphNodeType) String() string {
	switch *k {
	case KindStage:
		return "stage"
	case KindPipeline:
		return "pipeline"
	default:
		panic("unknown node type")
	}
}

func (k CallGraphNodeType) str() string {
	return k.String()
}

func (k *CallGraphNodeType) GoString() string {
	return k.String()
}

func (k *CallGraphNodeType) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}

func (k *CallGraphNodeType) UnmarshalText(b []byte) error {
	if len(b) > len("pipeline") {
		return fmt.Errorf("invalid call graph type")
	}
	switch string(b) {
	case KindStage.str():
		*k = KindStage
		return nil
	case KindPipeline.str():
		*k = KindPipeline
		return nil
	default:
		return fmt.Errorf("invalid call graph type")
	}
}

// Kind returns KindStage.
func (c *CallGraphStage) Kind() CallGraphNodeType {
	return KindStage
}

// Kind returns KindPipeline.
func (c *CallGraphPipeline) Kind() CallGraphNodeType {
	return KindPipeline
}

// GetParent returns the pipeline, if any, which contains this node.
func (c *CallGraphStage) GetParent() *CallGraphPipeline {
	return c.Parent
}

// GetChildren returns nil.  Stages don't have children.
func (c *CallGraphStage) GetChildren() []CallGraphNode {
	return nil
}

// DisableBindings returns the set of expressions which could disable a
// stage or pipeline.
//
// A stage or pipeline is disabled if it is called like
//
//   call STAGE (
//       ...
//   ) using (
//       disabled = FOO.value,
//   )
//
// or if its parent pipeline is disabled.
//
// If an expression disabling this node or any of its parent pipelines evaluates
// to a constant true, then only that constant expression will be returned.
// Otherwise, the set of reference expressions which could disable this node or
// any of its parents is returned.  Constants evaluating to false are omitted.
func (c *CallGraphStage) Disabled() []Exp {
	return c.Disable
}

// Returns the nodes of any stages or subpipelines called by this pipeline.
func (c *CallGraphPipeline) GetChildren() []CallGraphNode {
	return c.Children
}

// The fully-qualified ID suffix for this node. That is
// e.g. ...id(grandparent).id(parent).id(this).
//
// Unlike the FQName() used in core.Node, this one lacks the
// ID.PipestanceId. prefix, but is otherwise the same.
func (c *CallGraphStage) GetFqid() string {
	return c.Fqid
}

// The ast node for this stage.
func (c *CallGraphStage) Callable() Callable {
	return c.stage
}

// The ast node for this pipeline.
func (c *CallGraphPipeline) Callable() Callable {
	return c.pipeline
}

// The call, either in the context of the parent or the top-level call.
func (c *CallGraphStage) Call() *CallStm {
	return c.call
}

// The resolved input bindings for this node.  References are followed
// until either a literal expression is found or a stage node is
// encountered.
func (c *CallGraphStage) ResolvedInputs() map[string]*ResolvedBinding {
	if c == nil {
		return nil
	}
	return c.Inputs
}

// ResolvedOutputs returns the resolved input bindings for this node.
//
// References are followed until either a literal expression is found or a
// stage node is encountered.  For stage nodes, this will simply be the stage
// FQID and output name.
func (c *CallGraphStage) ResolvedOutputs() *ResolvedBinding {
	return c.Outputs
}

// Retained values, exempt from VDR.
func (c *CallGraphStage) Retained() []*RefExp {
	if r := c.stage.Retain; r != nil && len(r.Params) > 0 {
		result := make([]*RefExp, 0, len(r.Params))
		for _, rp := range r.Params {
			result = append(result, &RefExp{
				Node:     rp.Node,
				Id:       c.Fqid,
				OutputId: rp.Id,
			})
		}
		return result
	}
	return nil
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

func (ast *Ast) makeCallGraphNodes(prefix string,
	call *CallStm, parent *CallGraphPipeline,
	forcePipeline bool) (CallGraphNode, error) {
	callable := ast.Callables.Table[call.DecId]
	if callable == nil {
		return nil, &wrapError{
			innerError: fmt.Errorf("no callable object named %s", call.DecId),
			loc:        call.Node.Loc,
		}
	}
	switch callable := callable.(type) {
	case *Stage:
		if forcePipeline {
			pipe := CallGraphPipeline{
				CallGraphStage: CallGraphStage{
					Parent: parent,
					call:   call,
				},
				pipeline: wrapStageAsPipeline(call, callable),
			}
			return &pipe, pipe.makeChildNodes(prefix, ast)
		}
		fqid := makeFqid(prefix, call, parent)
		st := CallGraphStage{
			Parent: parent,
			Fqid:   fqid,
			call:   call,
			stage:  callable,
		}
		return &st, nil
	case *Pipeline:
		pipe := CallGraphPipeline{
			CallGraphStage: CallGraphStage{
				Parent: parent,
				call:   call,
			},
			pipeline: callable,
		}
		return &pipe, pipe.makeChildNodes(prefix, ast)
	default:
		panic(fmt.Sprintf("invalid callable type %T", callable))
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
		} else if err := errs.If(); err != nil {
			return err
		} else {
			panic("nil child")
		}
	} else {
		pipe.Fqid = makeFqid(prefix, pipe.call, pipe.Parent)
	}
	return nil
}

func makeFqid(prefix string, call *CallStm, parent *CallGraphPipeline) string {
	fqid := call.Id
	if parent != nil {
		fqidlen := len(call.Id) + len(prefix)
		fqidParts := []string{call.Id}
		for p := parent; p != nil; p = p.GetParent() {
			fqidlen += 1 + len(p.Call().Id)
			fqidParts = append(fqidParts, p.Call().Id)
		}
		var buf strings.Builder
		buf.Grow(fqidlen)
		if _, err := buf.WriteString(prefix); err != nil {
			panic(err)
		}
		for i := len(fqidParts) - 1; i >= 0; i-- {
			if _, err := buf.WriteString(fqidParts[i]); err != nil {
				panic(err)
			}
			if i != 0 {
				if _, err := buf.WriteRune('.'); err != nil {
					panic(err)
				}
			}
		}
		fqid = buf.String()
	}
	return fqid
}

type bindingError struct {
	Msg string
	Err error
}

func (err *bindingError) Error() string {
	if e := err.Err; e == nil {
		return err.Msg
	} else {
		var buf strings.Builder
		buf.Grow(len(err.Msg) + 25)
		err.writeTo(&buf)
		return buf.String()
	}
}

func (err *bindingError) Unwrap() error {
	if err == nil {
		return err
	}
	return err.Err
}

func (err *bindingError) writeTo(w stringWriter) {
	if e := err.Err; e == nil {
		mustWriteString(w, err.Msg)
	} else {
		mustWriteString(w, err.Msg)
		mustWriteString(w, ": ")
		if ew, ok := e.(errorWriter); ok {
			ew.writeTo(w)
		} else {
			mustWriteString(w, e.Error())
		}
	}
}

// MakeCallGraph returns a node in the graph of stages and pipelines, with
// inputs and outputs fully resolved to either literal expressions or references
// to stage outputs.
//
// The prefix string is prepended to the fully-qualified IDs of all of the nodes.
func (ast *Ast) MakeCallGraph(prefix string, call *CallStm) (CallGraphNode, error) {
	node, err := ast.makeCallGraphNodes(prefix, call, nil, false)
	if err != nil {
		return node, err
	}
	return node, node.resolve(nil, &ast.TypeTable)
}

// MakePipelineCallGraph returns a node in the graph of stages and pipelines,
// with inputs and outputs fully resolved to either literal expressions or
// references to stage outputs.
//
// Unlike MakeCallGraph, MakePipelineCallGraph will create a "fake" wrapper
// pipeline object around calls which are directly to stages, in order to ensure
// that the top-level call is a pipeline call.
func (ast *Ast) MakePipelineCallGraph(prefix string, call *CallStm) (*CallGraphPipeline, error) {
	node, err := ast.makeCallGraphNodes(prefix, call, nil, true)
	if err != nil {
		return nil, err
	}
	return node.(*CallGraphPipeline), node.resolve(nil, &ast.TypeTable)
}

func (bindings *BindStms) resolve(self, calls map[string]*ResolvedBinding,
	lookup *TypeLookup) (map[string]*ResolvedBinding, error) {
	if len(bindings.List) == 0 {
		return nil, nil
	}
	result := make(map[string]*ResolvedBinding, len(bindings.List))
	var errs ErrorList
	for _, binding := range bindings.List {
		r, err := resolveExp(binding.Exp, binding.Tname, self, calls, lookup)
		if err != nil {
			errs = append(errs, &bindingError{
				Msg: "BindingError: input parameter " + binding.Id,
				Err: err,
			})
		}
		result[binding.Id] = r
	}
	return result, errs.If()
}

func resolveExp(exp Exp, tname TypeId, self, siblings map[string]*ResolvedBinding, lookup *TypeLookup) (*ResolvedBinding, error) {
	t := lookup.Get(tname)
	if t == nil {
		return nil, fmt.Errorf("unknown type " + tname.String())
	}
	exp, err := resolveRefs(exp, self, siblings, lookup)
	if err != nil {
		return &ResolvedBinding{
			Exp:  exp,
			Type: t,
		}, err
	}
	exp, err = exp.filter(t, lookup)
	if err != nil {
		return &ResolvedBinding{
			Exp:  exp,
			Type: t,
		}, err
	}
	return &ResolvedBinding{
		Exp:  exp,
		Type: t,
	}, err
}

func resolveRefs(exp Exp,
	self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) (Exp, error) {
	if !exp.HasRef() {
		return exp, nil
	}
	switch exp := exp.(type) {
	case *RefExp:
		var res *ResolvedBinding
		if exp.Kind == KindSelf {
			res = self[exp.Id]
			if res == nil {
				return exp, &bindingError{
					Msg: "unknown parameter " + exp.Id,
				}
			}
		} else {
			res = siblings[exp.Id]
			if res == nil {
				return exp, &bindingError{
					Msg: "unknown call name " + exp.Id,
				}
			}
		}
		return res.Exp.BindingPath(exp.OutputId)
	case *ArrayExp:
		var errs ErrorList
		result := ArrayExp{
			valExp: valExp{Node: exp.Node},
			Value:  make([]Exp, len(exp.Value)),
		}
		for i, subexp := range exp.Value {
			e, err := resolveRefs(subexp, self, siblings, lookup)
			if err != nil {
				errs = append(errs, &bindingError{
					Msg: fmt.Sprintf("element %d", i),
					Err: err,
				})
			}
			result.Value[i] = e
		}
		return &result, errs.If()
	case *SweepExp:
		var errs ErrorList
		result := SweepExp{
			valExp: valExp{Node: exp.Node},
			Value:  make([]Exp, len(exp.Value)),
		}
		for i, subexp := range exp.Value {
			e, err := resolveRefs(subexp, self, siblings, lookup)
			if err != nil {
				errs = append(errs, &bindingError{
					Msg: fmt.Sprintf("element %d", i),
					Err: err,
				})
			}
			result.Value[i] = e
		}
		return &result, errs.If()
	case *MapExp:
		var errs ErrorList
		result := MapExp{
			valExp: valExp{Node: exp.Node},
			Kind:   exp.Kind,
			Value:  make(map[string]Exp, len(exp.Value)),
		}
		for i, subexp := range exp.Value {
			e, err := resolveRefs(subexp, self, siblings, lookup)
			if err != nil {
				errs = append(errs, &bindingError{
					Msg: "key " + i,
					Err: err,
				})
			}
			result.Value[i] = e
		}
		return &result, errs.If()
	default:
		panic(fmt.Sprintf("unexpected ref in %T", exp))
	}
}

func (node *CallGraphStage) resolveInputs(siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) error {
	var errs ErrorList
	var parentInputs map[string]*ResolvedBinding
	var disable []Exp
	if parent := node.Parent; parent != nil {
		parentInputs = parent.Inputs
		disable = parent.Disable
	}
	ins, err := node.call.Bindings.resolve(parentInputs, siblings, lookup)
	if err != nil {
		errs = append(errs, err)
	}
	node.Inputs = ins
	node.Disable, err = node.resolveDisable(disable, parentInputs,
		siblings, lookup)
	if err != nil {
		errs = append(errs, err)
	}
	return errs.If()
}

func (node *CallGraphStage) resolveDisable(disable []Exp,
	parentInputs, siblings map[string]*ResolvedBinding, lookup *TypeLookup) ([]Exp, error) {
	mod := node.call.Modifiers
	if mod == nil {
		return disable, nil
	}
	bind := mod.Bindings
	if bind == nil || (len(disable) >= 1 &&
		disable[0].getKind() == KindBool) {
		return disable, nil
	}
	d := bind.Table[disabled]
	if d == nil {
		return disable, nil
	}
	r, err := resolveExp(d.Exp, d.Tname, parentInputs, siblings, lookup)
	if err != nil {
		return disable, &bindingError{
			Msg: "BindingError: disabled control binding",
			Err: err,
		}
	}
	switch r := r.Exp.(type) {
	case *RefExp:
		for _, e := range disable {
			if e == r {
				return disable, nil
			}
		}
		v := make([]Exp, len(disable), len(disable)+1)
		copy(v, disable)
		return append(v, r), nil
	case *NullExp:
		return disable, nil
	case *BoolExp:
		if r.Value {
			return []Exp{r}, nil
		}
		return disable, nil
	case *SweepExp:
		if len(r.Value) == 0 {
			return disable, nil
		}
		allFalse := true
		allTrue := true
		for _, e := range r.Value {
			switch e := e.(type) {
			case *RefExp, *NullExp:
				allTrue = false
				allFalse = false
			case *BoolExp:
				if e.Value {
					allFalse = false
				} else {
					allTrue = false
				}
			default:
				return disable, &wrapError{
					innerError: &bindingError{
						Msg: "BindingError: disabled control binding was not boolean",
					},
					loc: e.getNode().Loc,
				}
			}
		}
		if allFalse {
			return disable, nil
		}
		if allTrue {
			return []Exp{r.Value[0]}, nil
		}
		v := make([]Exp, len(disable), len(disable)+1)
		copy(v, disable)
		return append(v, r), nil
	default:
		return disable, &wrapError{
			innerError: &bindingError{
				Msg: "BindingError: disabled control binding was not boolean",
			},
			loc: r.getNode().Loc,
		}
	}
}

func (node *CallGraphStage) resolve(siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) error {
	err := node.resolveInputs(siblings, lookup)

	if len(node.stage.OutParams.List) > 0 {
		if len(node.Disable) >= 1 && node.Disable[0].getKind() == KindBool {
			// constantly-disabled stages always output null.  No need to
			// propagate references.
			node.Outputs = &ResolvedBinding{
				Exp: &NullExp{
					valExp: valExp{Node: node.stage.Node},
				},
				Type: lookup.Get(TypeId{Tname: node.stage.Id}),
			}
		} else {
			node.Outputs = &ResolvedBinding{
				Exp: &RefExp{
					Node: node.stage.Node,
					Kind: KindCall,
					Id:   node.Fqid,
				},
				Type: lookup.Get(TypeId{Tname: node.stage.Id}),
			}
		}
	}
	return err
}

func (node *CallGraphPipeline) resolve(siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) error {
	if err := node.resolveInputs(siblings, lookup); err != nil {
		return err
	}
	var childMap map[string]*ResolvedBinding
	var errs ErrorList
	if len(node.Children) > 0 {
		childMap = make(map[string]*ResolvedBinding, len(node.Children))
		for _, child := range node.Children {
			if err := child.resolve(childMap, lookup); err != nil {
				errs = append(errs, err)
			}
			childMap[child.Call().Id] = child.ResolvedOutputs()
		}
		if err := errs.If(); err != nil {
			return err
		}
	}
	if len(node.Disable) >= 1 && node.Disable[0].getKind() == KindBool {
		// constantly-disabled stages always output null.  No need to
		// propagate references.
		node.Outputs = &ResolvedBinding{
			Exp: &NullExp{
				valExp: valExp{Node: node.pipeline.Node},
			},
			Type: lookup.Get(TypeId{Tname: node.pipeline.Id}),
		}
	} else {
		outs, err := node.pipeline.Ret.Bindings.resolve(node.Inputs, childMap, lookup)
		if err != nil {
			errs = append(errs, err)
		}
		if len(outs) > 0 {
			exp := MapExp{
				valExp: valExp{Node: node.pipeline.Ret.Node},
				Kind:   KindStruct,
				Value:  make(map[string]Exp, len(outs)),
			}
			for k, out := range outs {
				exp.Value[k] = out.Exp
			}
			node.Outputs = &ResolvedBinding{
				Exp:  &exp,
				Type: lookup.Get(TypeId{Tname: node.pipeline.Id}),
			}
		}
	}
	if r := node.pipeline.Retain; r != nil && len(r.Refs) > 0 {
		node.Retain = make([]*RefExp, 0, len(r.Refs))
		for _, ref := range r.Refs {
			resolved, err := resolveRefs(ref, node.Inputs, childMap, lookup)
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

// Retained values, exempt from VDR.
func (node *CallGraphPipeline) Retained() []*RefExp {
	return node.Retain
}

func (node *CallGraphStage) NodeClosure() map[string]CallGraphNode {
	return map[string]CallGraphNode{
		node.Fqid: node,
	}
}

func (node *CallGraphStage) nodeClosure(m map[string]CallGraphNode) {
	m[node.Fqid] = node
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

// BoundReference contains information about a reference with type information.
type BoundReference struct {
	Exp  *RefExp
	Type Type
}

// Finds all of the expressions in this binding which are reference expressions,
// with types attached.
//
// This is distsinct from Exp.FindRefs() in that it propagates type information,
// which is relevent if any type conversions are taking place.
func (b *ResolvedBinding) FindRefs(lookup *TypeLookup) ([]*BoundReference, error) {
	if !b.Exp.HasRef() {
		return nil, nil
	}
	switch exp := b.Exp.(type) {
	case *RefExp:
		return []*BoundReference{&BoundReference{
			Exp:  exp,
			Type: b.Type,
		}}, nil
	case *ArrayExp:
		t := b.Type.GetId()
		if t.ArrayDim == 0 {
			return nil, &wrapError{
				innerError: &bindingError{
					Msg: "unexpected array",
				},
				loc: exp.Node.Loc,
			}
		}
		t.ArrayDim--
		nt := lookup.Get(t)
		if nt == nil {
			panic("invalid type " + t.String())
		}
		var errs ErrorList
		result := make([]*BoundReference, 0, len(exp.Value))
		for _, e := range exp.Value {
			if !e.HasRef() {
				continue
			}
			rb := ResolvedBinding{
				Exp:  e,
				Type: nt,
			}
			if refs, err := rb.FindRefs(lookup); err != nil {
				errs = append(errs, err)
			} else if len(refs) > 0 {
				result = append(result, refs...)
			}
		}
		return result, errs.If()
	case *SweepExp:
		var errs ErrorList
		result := make([]*BoundReference, 0, len(exp.Value))
		for _, e := range exp.Value {
			if !e.HasRef() {
				continue
			}
			rb := ResolvedBinding{
				Exp:  e,
				Type: b.Type,
			}
			if refs, err := rb.FindRefs(lookup); err != nil {
				errs = append(errs, err)
			} else if len(refs) > 0 {
				result = append(result, refs...)
			}
		}
		return result, errs.If()
	case *MapExp:
		switch t := b.Type.(type) {
		case *TypedMapType:
			var errs ErrorList
			result := make([]*BoundReference, 0, len(exp.Value))
			keys := make([]string, 0, len(exp.Value))
			for key := range exp.Value {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				e := exp.Value[key]
				if e == nil || !e.HasRef() {
					continue
				}
				rb := ResolvedBinding{
					Exp:  e,
					Type: t.Elem,
				}
				if refs, err := rb.FindRefs(lookup); err != nil {
					errs = append(errs, err)
				} else if len(refs) > 0 {
					result = append(result, refs...)
				}
			}
			return result, errs.If()
		case *StructType:
			var errs ErrorList
			result := make([]*BoundReference, 0, len(t.Members))
			for _, member := range t.Members {
				if v, ok := exp.Value[member.Id]; !ok {
					errs = append(errs, &bindingError{
						Msg: "missing " + member.Id,
					})
				} else if v.HasRef() {
					rb := ResolvedBinding{
						Exp:  v,
						Type: lookup.Get(member.Tname),
					}
					if refs, err := rb.FindRefs(lookup); err != nil {
						errs = append(errs, err)
					} else if len(refs) > 0 {
						result = append(result, refs...)
					}
				}
			}
			return result, errs.If()
		default:
			return nil, &wrapError{
				innerError: &bindingError{
					Msg: "unexpected " + string(exp.Kind),
				},
				loc: exp.Node.Loc,
			}
		}
	default:
		panic(fmt.Sprintf("invalid reference type %T", exp))
	}
}

// Returns a map containing the stages referenced by expression
// (recursively) and the specific referenecd binding paths.
func FindRefs(exp Exp) map[string][]string {
	if !exp.HasRef() {
		return nil
	}
	refs := exp.FindRefs()
	collate := make(map[string]map[string]struct{})
	for _, ref := range refs {
		if r := collate[ref.Id]; r == nil {
			collate[ref.Id] = map[string]struct{}{ref.OutputId: struct{}{}}
		} else {
			r[ref.OutputId] = struct{}{}
		}
	}
	result := make(map[string][]string, len(collate))
	for key, vals := range collate {
		vlist := make([]string, 0, len(vals))
		for v := range vals {
			vlist = append(vlist, v)
		}
		sort.Strings(vlist)
		result[key] = vlist
	}
	return result
}

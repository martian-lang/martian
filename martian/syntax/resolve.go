// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

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

	// Map of bindings, used for input arguments.  Keys are sorted in JSON
	// output.
	ResolvedBindingMap map[string]*ResolvedBinding

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
		ResolvedInputs() ResolvedBindingMap
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

		// Get the set of mapped call nodes, which may include this node or any
		// of its ancestor nodes, for which this node must also be split.
		ForkRoots() ForkRootList

		MapSource() MapCallSource

		nodeClosure(map[string]CallGraphNode)

		resolve(map[string]*ResolvedBinding, ForkRootList, *TypeLookup) error

		unsplit() error
	}

	// ForkRootList selects dimensions over which a call node may fork.
	ForkRootList []*CallGraphStage

	// CallGraphStage represents a stage in a call graph.
	CallGraphStage struct {
		Parent  *CallGraphPipeline `json:"-"`
		Fqid    string             `json:"fqid"`
		call    *CallStm
		stage   *Stage
		Inputs  ResolvedBindingMap `json:"inputs"`
		Outputs *ResolvedBinding   `json:"outputs"`
		Disable []Exp              `json:"disabled,omitempty"`
		Forks   ForkRootList       `json:"fork_roots,omitempty"`
		Source  MapCallSource      `json:"-"`
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

// Get the set of mapped calls, which may include this node or any of
// its ancestor nodes, for which this node must also be split.
func (c *CallGraphStage) ForkRoots() ForkRootList {
	return c.Forks
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
func (c *CallGraphStage) ResolvedInputs() ResolvedBindingMap {
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

func (s *CallGraphStage) MapSource() MapCallSource {
	return s.Source
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
	} else if prefix != "" {
		fqid = prefix + fqid
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
	if err := node.resolve(nil, nil, &ast.TypeTable); err != nil {
		return node, err
	}
	return node, node.unsplit()
}

// MakePipelineCallGraph returns a node in the graph of stages and pipelines,
// with inputs and outputs fully resolved to either literal expressions or
// references to stage outputs.
//
// Unlike MakeCallGraph, MakePipelineCallGraph will create a "fake" wrapper
// pipeline object around calls which are directly to stages, in order to ensure
// that the top-level call is a pipeline call.
func (ast *Ast) MakePipelineCallGraph(
	prefix string,
	call *CallStm) (*CallGraphPipeline, error) {
	node, err := ast.makeCallGraphNodes(prefix, call, nil, true)
	if err != nil {
		return nil, err
	}
	if err := node.resolve(nil, nil, &ast.TypeTable); err != nil {
		return node.(*CallGraphPipeline), err
	}
	return node.(*CallGraphPipeline), node.unsplit()
}

func (bindings *BindStms) resolve(self, calls map[string]*ResolvedBinding,
	lookup *TypeLookup, keepSplit bool) (map[string]*ResolvedBinding, error) {
	if len(bindings.List) == 0 {
		return nil, nil
	}
	result := make(map[string]*ResolvedBinding, len(bindings.List))
	var errs ErrorList
	for _, binding := range bindings.List {
		if binding.Id == "*" {
			// Compilation should have expanded this one out.
			continue
		}
		tid := binding.Tname
		r, err := resolveExp(binding.Exp, tid, self, calls, lookup, keepSplit)
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

func resolveExp(exp Exp, tname TypeId, self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup, keepSplit bool) (*ResolvedBinding, error) {
	t := lookup.Get(tname)
	if t == nil {
		return nil, fmt.Errorf("unknown type " + tname.String())
	}
	rexp, err := exp.resolveRefs(self, siblings, lookup, keepSplit)
	if err != nil {
		return &ResolvedBinding{
			Exp:  exp,
			Type: t,
		}, err
	}
	fexp, err := rexp.filter(t, lookup)
	if err != nil {
		return &ResolvedBinding{
			Exp:  fexp,
			Type: t,
		}, err
	}
	return &ResolvedBinding{
		Exp:  fexp,
		Type: t,
	}, err
}

func alwaysDisable(disable []Exp) []Exp {
	if len(disable) >= 1 {
		if v, ok := disable[0].(*BoolExp); ok && v.Value {
			return disable[:1]
		}
	}
	return []Exp{&trueExp}
}

func (node *CallGraphStage) resolveInputs(siblings map[string]*ResolvedBinding,
	mapped ForkRootList,
	lookup *TypeLookup) error {
	var errs ErrorList
	var parentInputs map[string]*ResolvedBinding
	var disable []Exp
	if parent := node.Parent; parent != nil {
		parentInputs = parent.Inputs
		disable = parent.Disable
	}
	ins, err := node.call.Bindings.resolve(parentInputs, siblings,
		lookup, true)
	if err != nil {
		errs = append(errs, &bindingError{
			Msg: node.Fqid,
			Err: err,
		})
	}
	if node.isEmptyMapping() {
		node.Disable = alwaysDisable(disable)
	} else {
		node.Disable, err = node.resolveDisable(disable, parentInputs,
			siblings, lookup)
		if err != nil {
			errs = append(errs, &wrapError{
				innerError: err,
				loc:        node.Call().Node.Loc,
			})
		}
	}
	node.Source, err = unifyMapSources(node.call, ins, node.Disable)
	if err != nil {
		errs = append(errs, &bindingError{
			Msg: node.Fqid,
			Err: err,
		})
	}
	node.Inputs = ins
	node.resolveForks(mapped, node)
	if node.Source != nil {
		match := false
		for _, f := range node.Forks {
			if f.MapSource() == node.Source {
				match = true
				break
			}
		}
		if !match {
			for _, f := range mapped {
				if f.MapSource() == node.Source {
					match = true
					node.Forks = append(node.Forks, f)
					break
				}
			}
		}
		if !match {
			node.Forks = append(node.Forks, node)
		}
	}
	return errs.If()
}

func unifyMapSources(call *CallStm, ins map[string]*ResolvedBinding, disable []Exp) (MapCallSource, error) {
	splits := make(map[*SplitExp]struct{})
	for _, b := range ins {
		findSplitsForCall(b.Exp, call, splits)
	}
	for _, b := range disable {
		findSplitsForCall(b, call, splits)
	}
	// Sort the splits before merging, to make the resolution repeatable.
	splitList := make([]*SplitExp, 0, len(splits))
	for sp := range splits {
		splitList = append(splitList, sp)
	}
	sort.Slice(splitList, func(i, j int) bool {
		if f1 := splitList[i].Node.Loc.File; f1 != nil {
			if f2 := splitList[j].Node.Loc.File; f2 != nil {
				if f1.FileName < f2.FileName {
					return true
				} else if f2.FileName < f1.FileName {
					return false
				}
			} else {
				return true
			}
		} else if f2 := splitList[j].Node.Loc.File; f2 != nil {
			return false
		}
		return splitList[i].Node.Loc.Line < splitList[j].Node.Loc.Line
	})
	var root MapCallSource
	var errs ErrorList
	var ref MapCallSource
	for _, sp := range splitList {
		if ref == nil || !ref.KnownLength() {
			if r, ok := sp.Source.(refMapResolver); ok && (ref == nil || sp.Source.KnownLength()) {
				rr := r.resolveMapSource(sp.CallMode())
				if rr.CallMode() == ModeSingleCall {
					ref = &ReferenceMappingSource{
						Ref:  rr.(*RefExp),
						Mode: sp.Source.CallMode(),
					}
				} else {
					ref = rr
				}
			}
			if r, ok := sp.Value.(*RefExp); ok && (ref == nil || r.KnownLength()) {
				rr := r.resolveMapSource(sp.CallMode())
				if rr.CallMode() == ModeSingleCall {
					ref = &ReferenceMappingSource{
						Ref:  rr.(*RefExp),
						Mode: sp.Source.CallMode(),
					}
				} else {
					ref = rr
				}
			}
		}
		if f, err := mergeSplitSource(root, sp); err != nil {
			errs = append(errs, err)
		} else {
			root = f
		}
	}
	if err := errs.If(); err != nil {
		return root, err
	}
	if root != nil {
		if ref != nil && (ref.KnownLength() || !root.KnownLength()) {
			root = ref
		}
		for _, sp := range splitList {
			sp.Source = root
		}
	}
	return root, nil
}

func mergeSplitSource(root MapCallSource, sp *SplitExp) (MapCallSource, error) {
	switch src := sp.Value.(type) {
	case MapCallSource:
		if f, err := MergeMapCallSources(root, src); err != nil {
			return root, err
		} else {
			root = f
		}
	}
	if f, err := MergeMapCallSources(root, sp.Source); err != nil {
		return root, err
	} else {
		root = f
	}
	return root, nil
}

func updateMapSources(call *CallStm, root MapCallSource, exp Exp) Exp {
	switch exp := exp.(type) {
	case *SplitExp:
		if exp.Call == call {
			ee := *exp
			ee.Source = root
			ee.Value = updateMapSources(call, root, exp.Value)
			return &ee
		}
		if e := updateMapSources(call, root, exp.Value); e != exp.Value {
			ee := *exp
			ee.Value = e
			return &ee
		}
	case *ArrayExp:
		if hasSplit(exp, call) != nil {
			arr := *exp
			arr.Value = make([]Exp, len(exp.Value))
			for i, v := range exp.Value {
				arr.Value[i] = updateMapSources(call, root, v)
			}
			return &arr
		}
	case *MapExp:
		if hasSplit(exp, call) != nil {
			arr := *exp
			arr.Value = make(map[string]Exp, len(exp.Value))
			for i, v := range exp.Value {
				arr.Value[i] = updateMapSources(call, root, v)
			}
			return &arr
		}
	case *DisabledExp:
		disabled := updateMapSources(call, root, exp.Disabled)
		inner := updateMapSources(call, root, exp.Value)
		e, err := exp.makeDisabledExp(disabled, inner)
		if err != nil {
			panic(err)
		}
		return e
	}
	return exp
}

func (node *CallGraphStage) resolveForks(mapped ForkRootList, localRoot *CallGraphStage) {
	if len(mapped) == 0 {
		return
	}
	splits := make(map[*CallStm]struct{}, len(mapped))
	for _, input := range node.Inputs {
		findSplitCalls(input.Exp, splits, false)
	}
	for _, d := range node.Disable {
		findSplitCalls(d, splits, false)
	}
	if len(splits) > 0 {
		if localRoot == nil {
			node.Forks = make(ForkRootList, 0, len(splits))
		} else {
			node.Forks = make(ForkRootList, 0, len(splits)+1)
		}
		for _, src := range mapped {
			if src == localRoot {
				continue
			}
			if _, ok := splits[src.Call()]; ok {
				node.Forks = append(node.Forks, src)
			}
		}
	}
}

var trueExp = BoolExp{Value: true}

func (node *CallGraphStage) resolveDisable(disable []Exp,
	parentInputs, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup) ([]Exp, error) {
	mod := node.call.Modifiers
	if mod == nil {
		return disable, nil
	}
	for len(disable) >= 1 &&
		disable[0].getKind() == KindBool {
		if disable[0].(*BoolExp).Value {
			return disable[:1], nil
		} else {
			disable = disable[1:]
		}
	}
	if node.call.KnownLength() {
		// Map over an empty set is equivilent to disabled.
		switch node.call.CallMode() {
		case ModeArrayCall:
			if node.call.ArrayLength() == 0 {
				return []Exp{&trueExp}, nil
			}
		case ModeMapCall:
			if len(node.call.Keys()) == 0 {
				return []Exp{&trueExp}, nil
			}
		case ModeNullMapCall:
			return []Exp{&trueExp}, nil
		}
	}
	bind := mod.Bindings
	if bind == nil {
		return disable, nil
	}
	d := bind.Table[disabled]
	if d == nil {
		return disable, nil
	}
	r, err := resolveExp(d.Exp, d.Tname, parentInputs, siblings, lookup, true)
	if err != nil {
		return disable, &bindingError{
			Msg: "BindingError: disabled control binding",
			Err: err,
		}
	}
	return resolveDisableExp(r.Exp, disable)
}

func resolveDisableExp(r Exp, disable []Exp) ([]Exp, error) {
	switch r := r.(type) {
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
		return disable, &wrapError{
			innerError: &bindingError{
				Msg: "BindingError: disabled cannot be bound to a null value.",
			},
			loc: r.getNode().Loc,
		}
	case *BoolExp:
		if r.Value {
			return []Exp{r}, nil
		}
		return disable, nil
	case *SplitExp:
		switch v := r.Value.(type) {
		case *ArrayExp:
			return resolveDisableArray(r, v.Value, disable)
		case *MapExp:
			if v.Kind != KindMap {
				return disable, &wrapError{
					innerError: &bindingError{
						Msg: "BindingError: cannot split " + string(v.Kind),
					},
					loc: r.getNode().Loc,
				}
			}
			return resolveDisableMap(r, v.Value, disable)
		}
		return resolveDisableExp(r.Value, disable)
	case *DisabledExp:
		for _, e := range disable {
			if r.Disabled.equal(e) {
				return resolveDisableExp(r.Value, disable)
			}
		}
		return disable, &wrapError{
			innerError: &bindingError{
				Msg: "BindingError: disabled was bound to a value that may be disabled at runtime",
			},
			loc: r.getNode().Loc,
		}
	default:
		return disable, &wrapError{
			innerError: &bindingError{
				Msg: "BindingError: disabled control binding was not boolean",
			},
			loc: r.getNode().Loc,
		}
	}
}

func resolveDisableArray(r Exp, v, disable []Exp) ([]Exp, error) {
	if len(v) == 0 {
		return disable, nil
	} else if len(v) == 1 {
		return resolveDisableExp(v[0], disable)
	}
	allFalse := true
	allTrue := true
	for _, e := range v {
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
		return []Exp{v[0]}, nil
	}
	result := make([]Exp, len(disable), len(disable)+1)
	copy(result, disable)
	return append(result, r), nil
}

func resolveDisableMap(r Exp, v map[string]Exp, disable []Exp) ([]Exp, error) {
	if len(v) == 0 {
		return disable, nil
	} else if len(v) == 1 {
		for _, e := range v {
			return resolveDisableExp(e, disable)
		}
	}
	allFalse := true
	allTrue := true
	for _, e := range v {
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
		for _, e := range v {
			return []Exp{e}, nil
		}
	}
	result := make([]Exp, len(disable), len(disable)+1)
	copy(result, disable)
	return append(result, r), nil
}

func (node *CallGraphStage) resolve(siblings map[string]*ResolvedBinding,
	mapped ForkRootList, lookup *TypeLookup) error {
	err := node.resolveInputs(siblings, mapped, lookup)

	if len(node.stage.OutParams.List) > 0 {
		if node.isAlwaysDisabled() {
			// constantly-disabled stages always output null.  No need to
			// propagate references.
			node.Outputs = &ResolvedBinding{
				Exp: &NullExp{
					valExp: valExp{Node: node.stage.Node},
				},
				Type: lookup.Get(TypeId{Tname: node.stage.Id}),
			}
			node.Disable = alwaysDisable(node.Disable)
		} else {
			tid := TypeId{Tname: node.stage.Id}
			ref := RefExp{
				Node: node.stage.Node,
				Kind: KindCall,
				Id:   node.Fqid,
			}
			var exp Exp = &ref
			for i := len(node.Forks) - 1; i >= 0; i-- {
				fs := node.Forks[i]
				ref.MergeOver = append(ref.MergeOver, fs.MapSource())
				switch fs.MapSource().CallMode() {
				case ModeArrayCall:
					tid.ArrayDim++
				case ModeMapCall:
					tid.MapDim = tid.ArrayDim + 1
					tid.ArrayDim = 0
				case ModeNullMapCall:
					exp = &NullExp{
						valExp: valExp{Node: node.stage.Node},
					}
				}
				if _, ok := exp.(*NullExp); !ok && fs.MapSource() != node.MapSource() {
					// Forking because the parent pipeline forked, so other
					// calls in the same pipeline should also fork.
					// These splits will be removed by unsplit later.
					exp = &SplitExp{
						valExp: valExp{Node: fs.Call().Node},
						Call:   fs.Call(),
						Source: fs.MapSource(),
						Value:  exp,
					}
				}
			}
			node.Outputs = &ResolvedBinding{
				Exp:  exp,
				Type: lookup.Get(tid),
			}
		}
	}
	return err
}

func (node *CallGraphStage) unsplit() error {
	if node.Outputs == nil {
		return nil
	}
	for s, ok := node.Outputs.Exp.(*SplitExp); ok; s, ok = node.Outputs.Exp.(*SplitExp) {
		node.Outputs.Exp = s.Value
	}
	return nil
}

func findSplitCalls(exp Exp, result map[*CallStm]struct{}, onlyUnknown bool) {
	switch exp := exp.(type) {
	case *SplitExp:
		if !onlyUnknown || !exp.Source.KnownLength() {
			result[exp.Call] = struct{}{}
		}
		findSplitCalls(exp.Value, result, onlyUnknown)
	case *ArrayExp:
		for _, v := range exp.Value {
			findSplitCalls(v, result, onlyUnknown)
		}
	case *MapExp:
		for _, v := range exp.Value {
			findSplitCalls(v, result, onlyUnknown)
		}
	case *DisabledExp:
		findSplitCalls(exp.Value, result, onlyUnknown)
	}
}

func findSplitsForCall(exp Exp, call *CallStm, result map[*SplitExp]struct{}) {
	switch exp := exp.(type) {
	case *SplitExp:
		if exp.Call == call {
			result[exp] = struct{}{}
		}
		findSplitsForCall(exp.Value, call, result)
	case *ArrayExp:
		for _, v := range exp.Value {
			findSplitsForCall(v, call, result)
		}
	case *MapExp:
		for _, v := range exp.Value {
			findSplitsForCall(v, call, result)
		}
	case *DisabledExp:
		findSplitsForCall(exp.Value, call, result)
	}
}

// Returns true if the node never runs.  Only returns a correct result if
// inputs have been resolved already.
//
// It is not legal to specify a constantly-disabled call in MRO directly,
// however it can happen if the call to a pipeline includes a literal value
// which results in one of the calls by that pipeline being disabled.
func (node *CallGraphStage) isAlwaysDisabled() bool {
	if len(node.Disable) >= 1 && node.Disable[0].getKind() == KindBool {
		// false is trimmed out of the disable list, so it must be true.
		return true
	}
	return node.isEmptyMapping()
}

func (node *CallGraphStage) isEmptyMapping() bool {
	if m := node.call.CallMode(); m == ModeSingleCall {
		return false
	} else if m == ModeNullMapCall {
		return true
	}
	for _, input := range node.Inputs {
		if exp, ok := input.Exp.(*SplitExp); ok {
			if exp.IsEmpty() {
				return true
			}
		}
	}
	return false
}

func hasSplit(exp Exp, call MapCallSource) Exp {
	switch exp := exp.(type) {
	case *SplitExp:
		if exp.Call == call {
			return exp.InnerValue()
		}
		return hasSplit(exp.Value, call)
	case *ArrayExp:
		var result Exp
		for _, v := range exp.Value {
			if v := hasSplit(v, call); v != nil {
				switch v := v.(type) {
				case *RefExp:
					result = v
				case *NullExp:
					if result == nil {
						result = v
					}
				default:
					return v
				}
			}
		}
		return result
	case *MapExp:
		var result Exp
		for _, v := range exp.Value {
			if v := hasSplit(v, call); v != nil {
				switch v := v.(type) {
				case *RefExp:
					result = v
				case *NullExp:
					if result == nil {
						result = v
					}
				default:
					return v
				}
			}
		}
		return result
	case *DisabledExp:
		return hasSplit(exp.Value, call)
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

// Get the value of the split, bypassing any intermediate splits.
func (exp *SplitExp) InnerValue() Exp {
	if v, ok := exp.Value.(*SplitExp); ok {
		return v.InnerValue()
	}
	return exp.Value
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
			Type: lookup.Get(TypeId{Tname: node.pipeline.Id}),
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

// CallMode Returns the call mode for a call which depends on this source.
func (node *CallGraphStage) CallMode() CallMode {
	return node.call.CallMode()
}

// KnownLength returns true if the source is an array with a known length
// or is a map with a known set of keys.
func (node *CallGraphStage) KnownLength() bool {
	if node.call.KnownLength() {
		return true
	}
	if node.Source != nil {
		return node.Source.KnownLength()
	}
	return true
}

// If KnownLength is true and CallMode is ModeArrayCall, ArrayLength returns
// the length of the array referred to by this source.  Otherwise it will
// return -1.
func (node *CallGraphStage) ArrayLength() int {
	if node.Source != nil {
		return node.Source.ArrayLength()
	}
	return node.call.ArrayLength()
}

// If KnownLength is true and CallMode is ModeMapCall, MapKeys will return
// a map[string]Exp with the same keys which any call mapping over this
// source would have.  The values are arbitrary.  Otherwise, it will return
// nil.
func (node *CallGraphStage) Keys() map[string]Exp {
	if node.Source != nil {
		return node.Source.Keys()
	}
	return node.call.Keys()
}

func (node *CallGraphStage) GoString() string {
	return node.Fqid
}

// BoundReference contains information about a reference with type information.
type BoundReference struct {
	Exp  *RefExp
	Type Type
}

func bindingType(p string, t Type, lookup *TypeLookup) (Type, error) {
	if p == "" {
		return t, nil
	}
	i := strings.IndexRune(p, '.')
	switch t := t.(type) {
	case *TypedMapType:
		r, err := bindingType(p, t.Elem, lookup)
		if r != nil {
			return lookup.GetMap(r), err
		}
		return r, err
	case *ArrayType:
		r, err := bindingType(p, t.Elem, lookup)
		if r != nil {
			return lookup.GetArray(r, t.Dim), err
		}
		return r, err
	case *StructType:
		element := p
		rest := ""
		if i > 0 {
			element = p[:i]
			rest = p[i+1:]
		}
		member := t.Table[element]
		if member == nil {
			return t, &bindingError{
				Msg: "no member " + element + " in " + t.Id,
			}
		}
		return bindingType(rest, lookup.Get(member.Tname), lookup)
	}
	return t, &bindingError{
		Msg: "can't resolve path through " + t.GetId().str(),
	}
}

func (b *ResolvedBinding) BindingPath(p string,
	fork map[MapCallSource]CollectionIndex, index []CollectionIndex,
	lookup *TypeLookup) (*ResolvedBinding, error) {
	t, err := bindingType(p, b.Type, lookup)
	if err != nil {
		return b, err
	}
	e, err := b.Exp.BindingPath(p, fork, index)
	if err != nil || (e == b.Exp && t == b.Type) {
		return b, err
	}
	return &ResolvedBinding{
		Exp:  e,
		Type: t,
	}, nil
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
		return []*BoundReference{{
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
				errs = append(errs, &bindingError{
					Msg: "in array",
					Err: err,
				})
			} else if len(refs) > 0 {
				result = append(result, refs...)
			}
		}
		return result, errs.If()
	case *DisabledExp:
		rb := *b
		rb.Exp = exp.Value
		refs, err := rb.FindRefs(lookup)
		if err != nil {
			return refs, err
		}
		return append(refs, &BoundReference{
			Exp:  exp.Disabled,
			Type: &builtinBool,
		}), nil
	case *SplitExp:
		t := b.Type.GetId()
		var innerType Type
		switch exp.InnerValue().(type) {
		case *MapExp:
			innerType = lookup.GetMap(b.Type)
		case *ArrayExp:
			innerType = lookup.GetArray(b.Type, 1)
		case *RefExp:
			if t.ArrayDim > 0 {
				t.ArrayDim--
			} else if t.MapDim > 0 {
				t.ArrayDim = t.MapDim - 1
				t.MapDim = 0
			}
			innerType = lookup.Get(t)
		case *NullExp:
			innerType = &builtinNull
		default:
			return nil, &wrapError{
				innerError: &bindingError{
					Msg: "split was not over an array, map, or ref",
				},
				loc: exp.Node.Loc,
			}
		}
		rb := ResolvedBinding{
			Exp:  exp.Value,
			Type: innerType,
		}
		result, err := rb.FindRefs(lookup)
		if err != nil {
			err = &bindingError{
				Msg: "in split",
				Err: err,
			}
		}
		return result, err
	case *MapExp:
		switch t := b.Type.(type) {
		case *TypedMapType:
			if exp.Kind == KindStruct {
				// To avoid special handling of references, pipeline output
				// bindings for mapped calls of pipelines will be structs of
				// maps rather than maps of structs.  But, that means we have
				// to have special handling here instead.
				if t, ok := t.Elem.(*StructType); ok {
					return findStructRefs(lookup, t, exp, false, true)
				}
			}
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
					errs = append(errs, &bindingError{
						Msg: "map key " + key,
						Err: err,
					})
				} else if len(refs) > 0 {
					result = append(result, refs...)
				}
			}
			return result, errs.If()
		case *StructType:
			return findStructRefs(lookup, t, exp, false, false)
		case *ArrayType:
			// To avoid special handling of references, pipeline output
			// bindings for mapped calls of pipelines will be structs of
			// arrays rather than arrays of structs.  But, that means we have
			// to have special handling here instead.
			if t, ok := t.Elem.(*StructType); ok {
				return findStructRefs(lookup, t, exp, true, false)
			}
			return nil, &wrapError{
				innerError: &bindingError{
					Msg: "unexpected " + string(exp.Kind) +
						" (expected " + t.GetId().str() + ")",
				},
				loc: exp.Node.Loc,
			}
		default:
			return nil, &wrapError{
				innerError: &bindingError{
					Msg: "unexpected " + string(exp.Kind) +
						" (expected " + t.GetId().str() + ")",
				},
				loc: exp.Node.Loc,
			}
		}
	default:
		panic(fmt.Sprintf("invalid reference type %T", exp))
	}
}

func findStructRefs(lookup *TypeLookup, t *StructType, exp *MapExp, arr, typedMap bool) ([]*BoundReference, error) {
	var errs ErrorList
	result := make([]*BoundReference, 0, len(t.Members))
	for _, member := range t.Members {
		if v, ok := exp.Value[member.Id]; !ok {
			errs = append(errs, &bindingError{
				Msg: "missing " + member.Id,
			})
		} else if v.HasRef() {
			tn := member.Tname
			if arr {
				tn.ArrayDim++
			}
			if typedMap {
				if tn.MapDim != 0 {
					errs = append(errs, &bindingError{
						Msg: "can't nest map for field " + member.Id + " in a map call",
					})
				} else {
					tn.MapDim = tn.ArrayDim + 1
					tn.ArrayDim = 0
				}
			}
			rb := ResolvedBinding{
				Exp:  v,
				Type: lookup.Get(tn),
			}
			if refs, err := rb.FindRefs(lookup); err != nil {
				errs = append(errs, &bindingError{
					Msg: "struct field " + member.Id,
					Err: err,
				})
			} else if len(refs) > 0 {
				result = append(result, refs...)
			}
		}
	}
	return result, errs.If()
}

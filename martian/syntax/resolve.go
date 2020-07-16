// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Code for resolving the pipeline graph.

package syntax

import (
	"fmt"
)

type (
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

		Split() *SplitExp

		nodeClosure(map[string]CallGraphNode)

		resolve(map[string]*ResolvedBinding, ForkRootList, *TypeLookup) error

		unsplit(*TypeLookup) error
	}

	// ForkRootList selects dimensions over which a call node may fork.
	ForkRootList []*CallGraphStage
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
		// Most filesystems have a file name length limit of 255 characters.
		// The journal files are written out as e.g.
		// FQID.fork0.chnk123.u0123456789.errors
		// Make it an error to have too deep a nesting for this to work.
		if len(fqid) > 222+len(prefix) {
			return &st, fmt.Errorf(
				"length of id string %s (%d) exceeds %d characters",
				fqid, len(fqid), 222+len(prefix))
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
	return node, node.unsplit(&ast.TypeTable)
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
	return node.(*CallGraphPipeline), node.unsplit(&ast.TypeTable)
}

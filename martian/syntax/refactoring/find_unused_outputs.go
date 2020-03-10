// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"sort"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
)

type StageOutput struct {
	Stage  *syntax.Stage
	Output *syntax.OutParam
}

func removeOutputId(set StringSet, ref string) {
	if i := strings.IndexByte(ref, '.'); i < 0 {
		delete(set, ref)
	} else {
		delete(set, ref[:i])
	}
}

func removeNodeOutputs(exp syntax.Exp,
	nodes map[string]syntax.CallGraphNode,
	outputs map[decId]StringSet) {
	for _, ref := range exp.FindRefs() {
		pnode := nodes[ref.Id]
		if ref.OutputId == "" {
			delete(outputs, makeDecId(pnode.Callable()))
		} else {
			removeOutputId(
				outputs[makeDecId(pnode.Callable())],
				ref.OutputId)
		}
	}
}

// Find stage outputs which are not used.
//
// Unlinke RemoveUnusedOutputs, which finds unbound pipeline outputs, this will
// first resolve the pipeline graphs (so that each stage input or pipeline
// output is either a constant or a reference to another stage output) and then
// return the set of stage outputs which are not bound (possibly indirectly)
// to either a top-level pipeline output or to another stage's inputs.
func FindUnusedStageOutputs(topCalls StringSet, asts []*syntax.Ast) ([]*StageOutput, error) {
	type pipePair struct {
		pipe *syntax.Pipeline
		ast  *syntax.Ast
	}
	topPipes := make(map[decId]pipePair, len(topCalls))

	for _, ast := range asts {
		for _, pipe := range ast.Pipelines {
			if topCalls.Contains(pipe.Id) {
				topPipes[makeDecId(pipe)] = pipePair{
					pipe: pipe,
					ast:  ast,
				}
			}
		}
	}
	nodeSets := make([]map[string]syntax.CallGraphNode, 0, len(topPipes))
	for _, pipe := range topPipes {
		graph, err := pipe.ast.MakePipelineCallGraph("",
			syntax.GenerateAbstractCall(pipe.pipe, &pipe.ast.TypeTable))
		if err != nil {
			return nil, err
		}
		nodeSets = append(nodeSets, graph.NodeClosure())
	}

	outputs := make(map[decId]StringSet, len(asts[0].Pipelines)+1)

	// Collect the outputs of all non-top stage nodes.
	for _, nodes := range nodeSets {
		for _, node := range nodes {
			if node.Kind() == syntax.KindStage &&
				!topCalls.Contains(node.Callable().GetId()) {
				if outp := node.Callable().GetOutParams(); outp != nil {
					dec := makeDecId(node.Callable())
					if _, ok := outputs[dec]; !ok {
						outs := make(StringSet, len(outp.List))
						for _, param := range outp.List {
							if !HasKeepComment(param) {
								outs.Add(param.Id)
							}
						}
						// Keep retained outputs.
						if stage, ok := node.Callable().(*syntax.Stage); ok &&
							stage != nil && stage.Retain != nil {
							for _, ret := range node.Callable().(*syntax.Stage).Retain.Params {
								delete(outs, ret.Id)
							}
						}
						outputs[dec] = outs
					}
				}
			}
		}
	}

	// Remove the outputs which were used
	for _, nodes := range nodeSets {
		for _, node := range nodes {
			if node.Kind() == syntax.KindStage {
				for _, input := range node.ResolvedInputs() {
					removeNodeOutputs(input.Exp, nodes, outputs)
				}
				for _, disable := range node.Disabled() {
					removeNodeOutputs(disable, nodes, outputs)
				}
			}
			for _, ref := range node.Retained() {
				pnode := nodes[ref.Id]
				if ref.OutputId == "" {
					delete(outputs, makeDecId(pnode.Callable()))
				} else {
					removeOutputId(
						outputs[makeDecId(pnode.Callable())],
						ref.OutputId)
				}
			}
			// Also remove outputs which feed top-level outputs.
			if topCalls.Contains(node.Callable().GetId()) {
				removeNodeOutputs(node.ResolvedOutputs().Exp, nodes, outputs)
			}
		}
	}
	result := make([]*StageOutput, 0, 2*len(outputs))
	for dec, outs := range outputs {
		if len(outs) > 0 {
			stage := getStage(dec, asts)
			if stage != nil {
				for out := range outs {
					result = append(result, &StageOutput{
						Stage:  stage,
						Output: stage.GetOutParams().Table[out],
					})
				}
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		ri, rj := result[i].Output, result[j].Output
		if f, g := syntax.DefiningFile(ri), syntax.DefiningFile(rj); f < g {
			return true
		} else if f > g {
			return false
		}
		if f, g := ri.Line(), rj.Line(); f < g {
			return true
		} else if f > g {
			return false
		}
		return ri.Id < rj.Id
	})
	return result, nil
}

func getStage(id decId, asts []*syntax.Ast) *syntax.Stage {
	for _, ast := range asts {
		if ast != nil {
			for _, stage := range ast.Stages {
				if stage.Id == id.Name && syntax.DefiningFile(stage) == id.File {
					return stage
				}
			}
		}
	}
	return nil
}

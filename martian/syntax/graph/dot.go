// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// This file contains methods for redering pipelines in graphviz dot format.

// Package graph contains formatting methods for martian pipline call graphs.
package graph

import (
	"fmt"
	"hash/crc32"
	"io"
	"sort"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
)

// RenderDot writes the given call graph in graphviz dot format.
//
// indentStart and indent control the indenting of the output, similar to
// json.MarshalIndent.
//
// graphAttrs are attribute lines to add to the top-level graph, e.g.
//
//   packmode = clust
func RenderDot(pipeline *syntax.CallGraphPipeline, w io.StringWriter,
	indentStart, indent string, graphAttrs ...string) error {
	if _, err := w.WriteString(indentStart); err != nil {
		return err
	}
	if _, err := w.WriteString("digraph "); err != nil {
		return err
	}
	if _, err := w.WriteString(pipeline.Fqid); err != nil {
		return err
	}
	if _, err := w.WriteString(" {\n"); err != nil {
		return err
	}
	if _, err := w.WriteString(indentStart); err != nil {
		return err
	}
	if _, err := w.WriteString(indent); err != nil {
		return err
	}
	if _, err := w.WriteString(`id = "`); err != nil {
		return err
	}
	if _, err := w.WriteString(pipeline.Fqid); err != nil {
		return err
	}
	if _, err := w.WriteString("\";\n"); err != nil {
		return err
	}
	for _, attr := range graphAttrs {
		if _, err := w.WriteString(indentStart); err != nil {
			return err
		}
		if _, err := w.WriteString(indent); err != nil {
			return err
		}
		if _, err := w.WriteString(attr); err != nil {
			return err
		}
		if _, err := w.WriteString(";\n"); err != nil {
			return err
		}
	}
	if err := renderPipelineDot(pipeline, w, indentStart, indent); err != nil {
		return err
	}

	if pipeline.Outputs != nil && pipeline.Outputs.Exp != nil && pipeline.Outputs.Exp.HasRef() {
		if _, err := w.WriteString(indentStart); err != nil {
			return err
		}
		if _, err := w.WriteString(indent); err != nil {
			return err
		}
		if _, err := w.WriteString("\"Output\" [style=dotted];\n"); err != nil {
			return err
		}
	}
	// Print output edges.
	if err := renderEdges(makePipelineEdgeBindings(pipeline), w,
		"Output", "style=dashed",
		indentStart, indent); err != nil {
		return err
	}
	if _, err := w.WriteString(indentStart); err != nil {
		return err
	}
	if _, err := w.WriteString("}\n"); err != nil {
		return err
	}
	return nil
}

// map from node -> to path -> from path.
type edgeBindingSet map[string]map[string]map[string]struct{}

func (set edgeBindingSet) Add(node, to, from string) {
	ns := set[node]
	if ns == nil {
		ns = make(map[string]map[string]struct{})
		set[node] = ns
	}
	ts := ns[to]
	if ts == nil {
		ts = make(map[string]struct{})
		ns[to] = ts
	}
	ts[from] = struct{}{}
}

func addEdgeBindings(exp syntax.Exp, path string, set edgeBindingSet) {
	if err := syntax.WalkExp(exp, func(exp syntax.Exp, p string) error {
		if !exp.HasRef() {
			return syntax.SkipExp
		}
		switch exp := exp.(type) {
		case *syntax.DisabledExp:
			if path != "" {
				if p == "" {
					p = path
				} else {
					p = path + "." + p
				}
			}
			// Skip disable, but keep value.
			addEdgeBindings(exp.Value, p, set)
			return syntax.SkipExp
		case *syntax.RefExp:
			if path != "" {
				if p == "" {
					p = path
				} else {
					p = path + "." + p
				}
			}
			set.Add(exp.Id, p, exp.OutputId)
		}
		return nil
	}); err != nil {
		panic(err)
	}
}

func makePipelineEdgeBindings(pipeline *syntax.CallGraphPipeline) edgeBindingSet {
	if pipeline.Outputs == nil || pipeline.Outputs.Exp == nil ||
		!pipeline.Outputs.Exp.HasRef() {
		return nil
	}
	set := make(edgeBindingSet, len(pipeline.Callable().GetOutParams().List))
	addEdgeBindings(pipeline.Outputs.Exp, "", set)
	return set
}

func makeStageEdgeBindings(stage *syntax.CallGraphStage) edgeBindingSet {
	set := make(edgeBindingSet, len(stage.Inputs))
	for k, inp := range stage.Inputs {
		addEdgeBindings(inp.Exp, k, set)
	}
	for _, inp := range stage.Disabled() {
		addEdgeBindings(inp, "(disabled)", set)
	}
	return set
}

func renderEdges(refs edgeBindingSet, w io.StringWriter,
	target, style string,
	indentStart, indent string) error {
	if len(refs) == 0 {
		return nil
	}
	shortTarget := target
	if i := strings.LastIndexByte(target, '.'); i > 0 {
		shortTarget = target[i+1:]
	}
	refNodes := make([]string, 0, len(refs))
	for node := range refs {
		refNodes = append(refNodes, node)
	}
	sort.Strings(refNodes)
	for _, node := range refNodes {
		if _, err := w.WriteString(indentStart); err != nil {
			return err
		}
		if _, err := w.WriteString(indent); err != nil {
			return err
		}
		if _, err := w.WriteString(`"`); err != nil {
			return err
		}
		if _, err := w.WriteString(node); err != nil {
			return err
		}
		if _, err := w.WriteString(
			"\" -> \""); err != nil {
			return err
		}
		if _, err := w.WriteString(target); err != nil {
			return err
		}
		if _, err := w.WriteString(
			"\" ["); err != nil {
			return err
		}
		if style != "" {
			if _, err := w.WriteString(style); err != nil {
				return err
			}
			if _, err := w.WriteString("; "); err != nil {
				return err
			}
		}
		if _, err := w.WriteString("url=\"#"); err != nil {
			return err
		}
		if _, err := w.WriteString(target); err != nil {
			return err
		}
		if _, err := w.WriteString("\"; tooltip=\""); err != nil {
			return err
		}
		shortNode := node
		if i := strings.LastIndexByte(node, '.'); i > 0 {
			shortNode = node[i+1:]
		}
		nodeRefs := refs[node]
		tos := make([]string, 0, len(nodeRefs))
		for to := range nodeRefs {
			tos = append(tos, to)
		}
		sort.Strings(tos)
		for i, to := range tos {
			if i != 0 {
				if _, err := w.WriteString(";\n"); err != nil {
					return err
				}
			}
			if _, err := w.WriteString(shortNode); err != nil {
				return err
			}
			idSet := nodeRefs[to]
			ids := make([]string, 0, len(idSet))
			for id := range idSet {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			if len(ids) > 1 || len(ids) == 1 && ids[0] != "" {
				if _, err := w.WriteString("."); err != nil {
					return err
				}
			}
			if len(ids) > 1 {
				if _, err := w.WriteString("{"); err != nil {
					return err
				}
			}
			for i, id := range ids {
				if i != 0 {
					if _, err := w.WriteString(","); err != nil {
						return err
					}
				}
				if _, err := w.WriteString(id); err != nil {
					return err
				}
			}
			if len(ids) > 1 {
				if _, err := w.WriteString("}"); err != nil {
					return err
				}
			}
			if _, err := w.WriteString(" -> "); err != nil {
				return err
			}
			if _, err := w.WriteString(shortTarget); err != nil {
				return err
			}
			if _, err := w.WriteString("."); err != nil {
				return err
			}
			if _, err := w.WriteString(to); err != nil {
				return err
			}
		}
		if len(tos) == 1 && tos[0] == "(disabled)" {
			if _, err := w.WriteString("\",style=dashed];\n"); err != nil {
				return err
			}
		} else if _, err := w.WriteString("\"];\n"); err != nil {
			return err
		}
	}
	return nil
}

func constantDisabled(node syntax.CallGraphNode) bool {
	d := node.Disabled()
	if len(d) == 0 {
		return false
	}
	if v, ok := d[0].(*syntax.BoolExp); ok {
		return v.Value
	}
	return false
}

func renderPipelineDot(pipeline *syntax.CallGraphPipeline, w io.StringWriter,
	indentStart, indent string) error {
	for _, call := range pipeline.Children {
		if _, err := w.WriteString(indentStart); err != nil {
			return err
		}
		if _, err := w.WriteString(indent); err != nil {
			return err
		}
		switch call := call.(type) {
		case *syntax.CallGraphPipeline:
			if _, err := w.WriteString(`subgraph "cluster`); err != nil {
				return err
			}
			if _, err := w.WriteString(call.Fqid); err != nil {
				return err
			}
			if _, err := w.WriteString("\" {\n"); err != nil {
				return err
			}
			if _, err := w.WriteString(indentStart); err != nil {
				return err
			}
			if _, err := w.WriteString(indent); err != nil {
				return err
			}
			if _, err := w.WriteString(indent); err != nil {
				return err
			}
			if _, err := w.WriteString("label = \""); err != nil {
				return err
			}
			if _, err := w.WriteString(call.Call().Id); err != nil {
				return err
			}
			if _, err := w.WriteString("\";\n"); err != nil {
				return err
			}
			if constantDisabled(call) {
				if _, err := w.WriteString(indentStart); err != nil {
					return err
				}
				if _, err := w.WriteString(indent); err != nil {
					return err
				}
				if _, err := w.WriteString(indent); err != nil {
					return err
				}
				if _, err := w.WriteString("style=invis;\n"); err != nil {
					return err
				}
			}
			if _, err := w.WriteString(indentStart); err != nil {
				return err
			}
			if _, err := w.WriteString(indent); err != nil {
				return err
			}
			if _, err := w.WriteString(indent); err != nil {
				return err
			}
			if _, err := w.WriteString(`id = "`); err != nil {
				return err
			}
			if _, err := w.WriteString(call.Fqid); err != nil {
				return err
			}
			if _, err := w.WriteString("\";\n"); err != nil {
				return err
			}
			if err := renderPipelineDot(call, w,
				indentStart+indent, indent); err != nil {
				return err
			}
			if _, err := w.WriteString(indentStart); err != nil {
				return err
			}
			if _, err := w.WriteString(indent); err != nil {
				return err
			}
			if _, err := w.WriteString("};\n"); err != nil {
				return err
			}
		case *syntax.CallGraphStage:
			if err := renderStageDot(call, w, indentStart, indent); err != nil {
				return err
			}
		default:
			panic(fmt.Sprintf("unknown type %T for %s",
				call, call.GetFqid()))
		}
	}
	return nil
}

func renderStageDot(stage *syntax.CallGraphStage, w io.StringWriter,
	indentStart, indent string) error {
	if _, err := w.WriteString(`"`); err != nil {
		return err
	}
	if _, err := w.WriteString(stage.Fqid); err != nil {
		return err
	}
	if _, err := w.WriteString(`" [label="`); err != nil {
		return err
	}
	if _, err := w.WriteString(stage.Call().Id); err != nil {
		return err
	}
	if _, err := w.WriteString(`",id="`); err != nil {
		return err
	}
	if _, err := w.WriteString(stage.Fqid); err != nil {
		return err
	}
	if _, err := w.WriteString(`",`); err != nil {
		return err
	}
	color := fmt.Sprintf(`color="%f,0.85,0.65"`,
		float64(crc32.ChecksumIEEE([]byte(stage.Callable().GetId())))/
			float64(1<<32-1))
	if _, err := w.WriteString(color); err != nil {
		return err
	}
	if constantDisabled(stage) {
		if _, err := w.WriteString(",style=invis];\n"); err != nil {
			return err
		}
	} else if stage.Call().Modifiers.Preflight {
		if _, err := w.WriteString(",style=dashed];\n"); err != nil {
			return err
		}
	} else if _, err := w.WriteString(",style=rounded];\n"); err != nil {
		return err
	}

	return renderEdges(makeStageEdgeBindings(stage), w,
		stage.Fqid, color, indentStart, indent)
}

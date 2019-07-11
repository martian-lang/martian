// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

// This file contains methods for redering pipelines in graphviz dot format.

package syntax

import (
	"fmt"
	"hash/crc32"
	"sort"
	"strings"
)

// Formats a pipeline in graphviz dot format.
func (pipeline *Pipeline) RenderDot(name string, callables map[string]Callable,
	indentStart, indent string) string {
	var builder strings.Builder
	builder.WriteString(indentStart)
	builder.WriteString("digraph ")
	builder.WriteString(name)
	builder.WriteString(" {\n")
	builder.WriteString(indentStart)
	builder.WriteString(indent)
	builder.WriteString(`id = "`)
	builder.WriteString(name)
	builder.WriteString("\";\n")
	outs := pipeline.renderDot(&builder, name, indentStart, indent, callables, nil)
	for _, node := range bindingNodeSet(outs) {
		builder.WriteString(indentStart)
		builder.WriteString(indent)
		builder.WriteRune('"')
		builder.WriteString(node)
		builder.WriteString("\" -> Outputs [style=dashed];\n")
	}
	builder.WriteString(indentStart)
	builder.WriteRune('}')
	return builder.String()
}

// renderDot renders the pipeline to dot format for the stuff inside (but not
// including) the {}
func (pipeline *Pipeline) renderDot(builder *strings.Builder, fqname,
	indentStart, indent string, callables map[string]Callable,
	inputBindings map[string][]string) map[string][]string {

	bindings := map[string]map[string][]string{
		"self": inputBindings,
	}
	for _, call := range pipeline.Calls {
		callable := callables[call.DecId]
		builder.WriteString(indentStart)
		builder.WriteString(indent)
		switch c := callable.(type) {
		case *Pipeline:
			n := fqname + "." + call.Id
			builder.WriteString(`subgraph "cluster`)
			builder.WriteString(n)
			builder.WriteString("\" {\n")
			builder.WriteString(indentStart)
			builder.WriteString(indent)
			builder.WriteString(indent)
			builder.WriteString("label = \"")
			builder.WriteString(call.Id)
			builder.WriteString("\";\n")
			builder.WriteString(indentStart)
			builder.WriteString(indent)
			builder.WriteString(indent)
			builder.WriteString(`id = "`)
			builder.WriteString(n)
			builder.WriteString("\";\n")
			bindings[call.Id] = c.renderDot(builder,
				n, indentStart+indent, indent,
				callables, resolvePipelineBindings(call.Bindings, bindings))
			builder.WriteString("};\n")
		case *Stage:
			builder.WriteRune('"')
			builder.WriteString(fqname)
			builder.WriteRune('.')
			builder.WriteString(call.Id)
			builder.WriteString(`" [label="`)
			builder.WriteString(call.Id)
			builder.WriteString(`",id="`)
			builder.WriteString(fqname)
			builder.WriteRune('.')
			builder.WriteString(call.Id)
			builder.WriteString(`",color=`)
			color := fmt.Sprintf(`"%f,0.85,0.65"`,
				float64(crc32.ChecksumIEEE([]byte(call.Id)))/
					float64(1<<32-1))

			builder.WriteString(color)
			if call.Modifiers.Preflight {
				builder.WriteString(",style=dashed];\n")
			} else {
				builder.WriteString(",style=rounded];\n")
			}
			if c.OutParams != nil {
				m := make(map[string][]string, len(c.OutParams.List))
				n := []string{fqname + "." + call.Id}
				for _, p := range c.OutParams.List {
					m[p.Id] = n
				}
				bindings[call.Id] = m
			}
			for _, prenode := range bindingNodeSet(
				resolvePipelineBindings(call.Bindings, bindings)) {
				builder.WriteString(indentStart)
				builder.WriteString(indent)
				builder.WriteRune('"')
				builder.WriteString(prenode)
				builder.WriteString(`" -> "`)
				builder.WriteString(fqname)
				builder.WriteRune('.')
				builder.WriteString(call.Id)
				builder.WriteString(`" [color=`)
				builder.WriteString(color)
				builder.WriteString("];\n")
			}
		default:
			panic(fmt.Sprintf("unknown callable type %T for %s.%s",
				callable, fqname, call.DecId))
		}
	}
	builder.WriteString(indentStart)
	var outs map[string][]string
	if pipeline.Ret != nil {
		outs = resolvePipelineBindings(pipeline.Ret.Bindings, bindings)
	}
	return outs
}

func bindingNodeSet(all map[string][]string) []string {
	if len(all) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(all))
	for _, vs := range all {
		for _, v := range vs {
			set[v] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for n := range set {
		result = append(result, n)
	}
	sort.Strings(result)
	return result
}

func resolvePipelineBindings(bindings *BindStms,
	resolved map[string]map[string][]string) map[string][]string {
	if bindings == nil {
		return nil
	}
	if len(bindings.List) == 0 {
		return nil
	}
	result := make(map[string][]string, len(bindings.List))
	for _, binding := range bindings.List {
		if binding == nil {
			continue
		}
		result[binding.Id] = resolveBindingExpression(binding.Exp, resolved)
	}
	return result
}

func resolveBindingExpression(exp Exp, resolved map[string]map[string][]string) []string {
	switch e := exp.(type) {
	case *RefExp:
		if e.Kind == KindSelf {
			return resolved["self"][e.Id]
		} else {
			return resolved[e.Id][e.OutputId]
		}
	case *ArrayExp:
		var result []string
		for _, subExp := range e.Value {
			result = append(result, resolveBindingExpression(subExp, resolved)...)
		}
		return result
	case *MapExp:
		var result []string
		for _, subExp := range e.Value {
			result = append(result, resolveBindingExpression(subExp, resolved)...)
		}
		return result
	}
	return nil
}

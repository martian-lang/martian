//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

package main

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"

	"github.com/martian-lang/martian/martian/syntax"
)

func writeStageStructs(buffer *bytes.Buffer,
	lookup *syntax.TypeLookup, stage syntax.Callable, onlyIns bool) {
	prefix := GoName(stage.GetId())

	if !onlyIns {
		buffer.WriteString("//\n// ")
		buffer.WriteString(stage.GetId())
		buffer.WriteString("\n//\n\n")
	}

	writeStageArgs(buffer, lookup, prefix, stage)
	if !onlyIns {
		writeStageOuts(buffer, lookup, prefix, stage)

		if stage, ok := stage.(*syntax.Stage); ok && stage.Split {
			writeStageChunkDef(buffer, lookup, prefix, stage)
			fmt.Fprintf(buffer, `
// A structure to decode args to the join method for %s
type %sJoinArgs struct {
	core.JobResources
	%sArgs
}
`, stage.Id, prefix, prefix)
			writeStageChunkOuts(buffer, lookup, prefix, stage)
		}
	}
}

func writeStruct(buffer *bytes.Buffer, lookup *syntax.TypeLookup, s *syntax.StructType) {
	prefix := GoName(s.Id)

	if len(s.Node.Comments) > 0 {
		for _, c := range s.Node.Comments {
			buffer.WriteString("// ")
			buffer.WriteString(strings.TrimSpace(strings.TrimLeft(c, "#")))
			buffer.WriteRune('\n')
		}
	} else {
		fmt.Fprintf(buffer,
			"// A structure to encode and decode the %s struct.\n",
			s.Id)
	}
	fmt.Fprintf(buffer,
		"type %s struct {\n",
		prefix)
	for _, param := range s.Members {
		writeParam(buffer, lookup, param)
	}
	buffer.WriteString("}\n\n")
}

// Convert mro stage and variable names into appropriate exported go names.
func GoName(stageName string) string {
	parts := strings.Split(stageName, "_")
	var result bytes.Buffer
	for _, p := range parts {
		for i, r := range p {
			if i == 0 {
				result.WriteRune(unicode.ToUpper(r))
			} else if unicode.IsUpper(r) {
				result.WriteString(strings.ToLower(p[i:]))
				break
			} else {
				result.WriteString(p[i:])
				break
			}
		}
	}
	return result.String()
}

func writeParam(buffer *bytes.Buffer, lookup *syntax.TypeLookup, param syntax.StructMemberLike) {
	var comments []string
	switch p := param.(type) {
	case *syntax.InParam:
		comments = p.Node.Comments
	case *syntax.OutParam:
		comments = p.Node.Comments
	case *syntax.StructMember:
		comments = p.Node.Comments
	default:
		return // Other param types aren't supported here.
	}
	// Keep track of when we need a blank comment line between
	// blocks of comments from different sources.
	spacer := false
	if h := param.GetHelp(); h != "" {
		fmt.Fprintf(buffer,
			"\t// %s\n",
			h)
		spacer = true
	}
	if o := param.GetOutName(); o != "" {
		if spacer {
			buffer.WriteString("\t//\n")
		}
		fmt.Fprintf(buffer,
			"\t// %s\n",
			o)
		spacer = true
	}
	if len(comments) > 0 && spacer {
		buffer.WriteString("\t//\n")
	}
	for _, c := range comments {
		fmt.Fprintf(buffer,
			"\t//%s\n",
			c[1:])
		spacer = true
	}
	if param.IsFile() == syntax.KindIsFile {
		if spacer {
			buffer.WriteString("\t//\n")
		}
		switch t := param.GetTname().Tname; t {
		case syntax.KindFile:
			buffer.WriteString("\t// file")
		case syntax.KindPath:
			buffer.WriteString("\t// path")
		default:
			fmt.Fprintf(buffer,
				"\t// %s file", t)
		}
		if param.GetArrayDim() > 0 {
			buffer.WriteString("s\n")
		} else {
			buffer.WriteRune('\n')
		}
	}
	tid := param.GetTname()
	buffer.WriteRune('\t')
	buffer.WriteString(GoName(param.GetId()))
	buffer.WriteRune(' ')
	for i := tid.ArrayDim; i > 0; i-- {
		buffer.WriteString("[]")
	}
	if tid.MapDim > 0 {
		buffer.WriteString("map[string]")
		for i := tid.MapDim; i > 1; i-- {
			buffer.WriteString("[]")
		}
	}
	switch tid.Tname {
	case syntax.KindInt, syntax.KindBool:
		buffer.WriteString(tid.Tname)
	case syntax.KindFloat:
		buffer.WriteString("float64")
	case syntax.KindMap:
		buffer.WriteString("map[string]json.RawMessage")
	case syntax.KindString, syntax.KindFile, syntax.KindPath:
		buffer.WriteString("string")
	default:
		if _, ok := lookup.Get(syntax.TypeId{
			Tname: tid.Tname}).(*syntax.StructType); ok {
			// Struct type
			buffer.WriteRune('*')
			buffer.WriteString(GoName(tid.Tname))
		} else {
			buffer.WriteString("string")
		}
	}
	fmt.Fprintf(buffer,
		" `json:\"%s\"`\n",
		param.GetId())
}

func writeStageArgs(buffer *bytes.Buffer, lookup *syntax.TypeLookup,
	prefix string, stage syntax.Callable) {
	// Args
	fmt.Fprintf(buffer,
		"// A structure to encode and decode args to the %s %s.\n",
		stage.GetId(), stage.Type())
	fmt.Fprintf(buffer,
		"type %sArgs struct {\n",
		prefix)
	for _, param := range stage.GetInParams().List {
		writeParam(buffer, lookup, param)
	}
	buffer.WriteString("}\n\n")
}

func writeStageOuts(buffer *bytes.Buffer, lookup *syntax.TypeLookup,
	prefix string, stage syntax.Callable) {
	// Args
	fmt.Fprintf(buffer,
		"// A structure to encode and decode outs from the %s %s.\n",
		stage.GetId(), stage.Type())
	fmt.Fprintf(buffer,
		"type %sOuts struct {\n",
		prefix)
	for _, param := range stage.GetOutParams().List {
		writeParam(buffer, lookup, param)
	}
	buffer.WriteString("}\n\n")
}

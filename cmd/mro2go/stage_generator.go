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

func writeStageStructs(buffer *bytes.Buffer, stage *syntax.Stage) {
	prefix := GoName(stage.Id)

	buffer.WriteString("//\n// ")
	buffer.WriteString(stage.Id)
	buffer.WriteString("\n//\n\n")

	writeStageArgs(buffer, prefix, stage)
	writeStageOuts(buffer, prefix, stage)

	if stage.Split {
		writeStageChunkDef(buffer, prefix, stage)
		fmt.Fprintf(buffer, `
// A structure to decode args to the join method for %s
type %sJoinArgs struct {
	core.JobResources
	%sArgs
}
`, stage.Id, prefix, prefix)
		writeStageChunkOuts(buffer, prefix, stage)
	}
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

func writeParam(buffer *bytes.Buffer, param syntax.Param) {
	var comments []string
	switch p := param.(type) {
	case *syntax.InParam:
		comments = p.Node.Comments
	case *syntax.OutParam:
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
		fmt.Fprintf(buffer,
			"\t// %s file",
			param.GetTname().Tname)
		if param.GetArrayDim() > 0 {
			buffer.WriteString("s\n")
		} else {
			buffer.WriteRune('\n')
		}
	}
	var goType string
	tid := param.GetTname()
	if _, ok := param.(*syntax.OutParam); ok &&
		param.IsFile() == syntax.KindIsFile && tid.ArrayDim > 0 &&
		tid.MapDim == 0 {
		// HACK: Currently, Martian puts filenames into out parameters
		// of file type, even for arrays.  That can't be fixed, yet, because
		// there are released pipelines which depend on that broken behavior.
		buffer.WriteString("\t// In _outs file written by Martian, will be of type string.\n")
		buffer.WriteString("\t// Must be changed to type []string by successful stages.\n")
		goType = "interface{}"

		fmt.Fprintf(buffer,
			"\t%s %s `json:\"%s\"`\n",
			GoName(param.GetId()),
			goType,
			param.GetId())
	} else {
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
		default:
			if param.IsFile() == syntax.KindIsDirectory {
				// Struct type
				buffer.WriteRune('*')
				buffer.WriteString(tid.Tname)
			} else {
				buffer.WriteString("string")
			}
		}
		fmt.Fprintf(buffer,
			" `json:\"%s\"`\n",
			param.GetId())
	}
}

func writeStageArgs(buffer *bytes.Buffer, prefix string, stage *syntax.Stage) {
	// Args
	fmt.Fprintf(buffer,
		"// A structure to encode and decode args to the %s stage.\n",
		stage.Id)
	fmt.Fprintf(buffer,
		"type %sArgs struct {\n",
		prefix)
	for _, param := range stage.InParams.List {
		writeParam(buffer, param)
	}
	buffer.WriteString("}\n\n")
}

func writeStageOuts(buffer *bytes.Buffer, prefix string, stage *syntax.Stage) {
	// Args
	fmt.Fprintf(buffer,
		"// A structure to encode and decode outs from the %s stage.\n",
		stage.Id)
	fmt.Fprintf(buffer,
		"type %sOuts struct {\n",
		prefix)
	for _, param := range stage.OutParams.List {
		writeParam(buffer, param)
	}
	buffer.WriteString("}\n\n")
}

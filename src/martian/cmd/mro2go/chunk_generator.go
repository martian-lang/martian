//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//

package main

import (
	"bytes"
	"fmt"
	"martian/syntax"
)

func writeStageChunkDef(buffer *bytes.Buffer, prefix string, stage *syntax.Stage) {
	if len(stage.ChunkIns.List) > 0 {
		// chunk def
		fmt.Fprintf(buffer, `
// A structure to encode chunk definitions for %s.
// Defines the resources and chunk-specific arguments.
type %sChunkDef struct {`,
			stage.Id, prefix)
		buffer.WriteString("\n\t*core.JobResources `json:\",omitempty\"`\n")
		for _, param := range stage.ChunkIns.List {
			writeParam(buffer, param)
		}
		fmt.Fprintf(buffer, `}

func (self *%sChunkDef) ToChunkDef() *core.ChunkDef {
	return &core.ChunkDef{
		Resources: self.JobResources,
		Args: core.ArgumentMap{
`, prefix)
		for _, param := range stage.ChunkIns.List {
			fmt.Fprintf(buffer,
				"\t\t\t\"%s\": self.%s,\n",
				param.GetId(),
				GoName(param.GetId()))
		}
		fmt.Fprintf(buffer, `		},
	}
}

// A structure to decode args to the chunks for %s
type %sChunkArgs struct {
	%sChunkDef
	%sArgs
}
`, stage.Id, prefix, prefix, prefix)
	} else {
		fmt.Fprintf(buffer, `
// %s chunks have no extra inputs.
type %sChunkDef core.JobResources

func (self *%sChunkDef) ToChunkDef() *core.ChunkDef {
	return &core.ChunkDef{
		Resources: self.JobResources,
		Args: make(core.ArgumentMap),
	}
}

// %s chunks have no extra inputs.
type %sChunkArgs %sArgs
`, stage.Id, prefix, prefix, stage.Id, prefix, prefix)
	}
}

func writeKeyMarshaller(buffer *bytes.Buffer, param syntax.Param, i int) {
	fmt.Fprintf(buffer, `
	if b, err := json.Marshal(&self.%s); err != nil {
		return nil, err
	} else {`, GoName(param.GetId()))
	if i == 0 {
		fmt.Fprintf(buffer, `
		buf.WriteString("\"%s\":")`, param.GetId())
	} else {
		fmt.Fprintf(buffer, `
		buf.WriteString(",\"%s\":")`, param.GetId())
	}
	buffer.WriteString(`
		buf.Write(b)
	}`)
}

func writeStageChunkOuts(buffer *bytes.Buffer, prefix string, stage *syntax.Stage) {
	if len(stage.ChunkOuts.List) > 0 && len(stage.OutParams.List) > 0 {
		fmt.Fprintf(buffer, `
// A structure to encode outs from the chunks for %s.
type %sChunkOuts struct {
	%sOuts
`,
			stage.Id, prefix, prefix)
		for _, param := range stage.ChunkOuts.List {
			writeParam(buffer, param)
		}
		fmt.Fprintf(buffer, `}

func (self *%sChunkOuts) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteRune('{')`, prefix)
		for i, param := range stage.OutParams.List {
			writeKeyMarshaller(buffer, param, i)
		}
		for _, param := range stage.ChunkOuts.List {
			writeKeyMarshaller(buffer, param, 1)
		}
		buffer.WriteString(`
	buf.WriteRune('}')
	return buf.Bytes(), nil
}
`)
	} else if len(stage.OutParams.List) > 0 {
		fmt.Fprintf(buffer, `
// %s chunks have no extra outputs
type %sChunkOuts %sOuts
`, stage.Id, prefix, prefix)
	} else {
		fmt.Fprintf(buffer, `
// A structure to encode outs from the chunks for %s.
type %sChunkOuts struct {
`,
			stage.Id, prefix)
		for _, param := range stage.ChunkOuts.List {
			writeParam(buffer, param)
		}
		buffer.WriteString("}\n")
	}
}

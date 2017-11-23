//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO canonical formatting. Inspired by gofmt.
//

package syntax

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
)

const (
	INDENT  string = "    "
	NEWLINE string = "\n"
)

func max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

type printer struct {
	buf      bytes.Buffer
	comments []*commentBlock
}

func (self *printer) printComments(loc int, prefix string) {
	for len(self.comments) > 0 && self.comments[0].Loc <= loc {
		self.buf.WriteString(prefix)
		self.buf.WriteString(self.comments[0].Value)
		self.buf.WriteString(NEWLINE)
		self.comments = self.comments[1:]
	}
}

func (self *printer) WriteString(s string) {
	self.buf.WriteString(s)
}

func (self *printer) Write(b []byte) (int, error) {
	return self.buf.Write(b)
}

func (self *printer) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&self.buf, format, args...)
}

func (self *printer) DumpComments() {
	for _, comment := range self.comments {
		self.buf.WriteString(comment.Value)
		self.buf.WriteString(NEWLINE)
	}
	self.comments = nil
}

func (self *printer) String() string {
	return self.buf.String()
}

//
// Expression
//
func (self *ValExp) format(prefix string) string {
	if self.Value == nil {
		return "null"
	}
	if self.Kind == KindInt {
		return fmt.Sprintf("%d", self.Value)
	}
	if self.Kind == KindFloat {
		return fmt.Sprintf("%g", self.Value)
	}
	if self.Kind == KindString {
		return fmt.Sprintf("\"%s\"", self.Value)
	}
	if self.Kind == KindMap {
		bytes, err := json.MarshalIndent(self.ToInterface(), prefix, INDENT)
		if err != nil {
			panic(err)
		}
		return string(bytes)
	}
	if self.Kind == KindArray {
		return self.formatArray(prefix)
	}
	return fmt.Sprintf("%v", self.Value)
}

func (self *ValExp) formatArray(prefix string) string {
	values := self.Value.([]Exp)
	if len(values) == 0 {
		return "[]"
	} else if len(values) == 1 {
		// Place single-element arrays on a single line.
		return fmt.Sprintf("[%s]",
			values[0].format(prefix))
	} else {
		var buf bytes.Buffer
		buf.WriteString("[\n")
		vindent := prefix + INDENT
		for i, val := range values {
			if i > 0 {
				buf.WriteString(",\n")
			}
			buf.WriteString(vindent)
			buf.WriteString(val.format(vindent))
		}
		buf.WriteRune('\n')
		buf.WriteString(prefix)
		buf.WriteRune(']')
		return buf.String()
	}
}

func (self *RefExp) format(prefix string) string {
	if self.Kind == KindCall {
		fsrc := self.Id
		if self.OutputId != "default" {
			fsrc += "." + self.OutputId
		}
		return fsrc
	}
	return "self." + self.Id
}

//
// Binding
//
func (self *BindStm) format(printer *printer, prefix string, idWidth int) {
	printer.printComments(self.getNode().Loc, prefix+INDENT)
	printer.printComments(self.Exp.getNode().Loc, prefix+INDENT)
	idPad := ""
	if len(self.Id) < idWidth {
		idPad = strings.Repeat(" ", idWidth-len(self.Id))
	}
	fmtExp := self.Exp.format(prefix + INDENT)
	if self.Sweep {
		fmtExp = fmt.Sprintf("sweep(%s)", strings.Trim(fmtExp, "[]"))
	}
	printer.Printf("%s%s%s%s = %s,\n", prefix, INDENT,
		self.Id, idPad, fmtExp)
}

func (self *BindStms) format(printer *printer, prefix string) {
	printer.printComments(self.getNode().Loc, prefix)
	idWidth := 0
	for _, bindstm := range self.List {
		if len(bindstm.Id) < 30 {
			idWidth = max(idWidth, len(bindstm.Id))
		}
	}
	for _, bindstm := range self.List {
		bindstm.format(printer, prefix, idWidth)
	}
}

//
// Parameter
//
func paramFormat(printer *printer, param Param, modeWidth int, typeWidth int, idWidth int, helpWidth int) {
	printer.printComments(param.getNode().Loc, INDENT)
	id := param.GetId()
	if id == "default" {
		id = ""
	}

	// Generate column alignment paddings.
	modePad := strings.Repeat(" ", modeWidth-len(param.getMode()))
	typePad := strings.Repeat(" ", typeWidth-len(param.GetTname())-2*param.GetArrayDim())
	idPad := ""
	if idWidth > len(id) {
		idPad = strings.Repeat(" ", idWidth-len(id))
	}
	helpPad := ""
	if helpWidth > len(param.GetHelp()) {
		helpPad = strings.Repeat(" ", helpWidth-len(param.GetHelp()))
	}

	// Common columns up to type name.
	printer.Printf("%s%s%s %s", INDENT,
		param.getMode(), modePad, param.GetTname())

	// If type is annotated as array, add brackets and shrink padding.
	for i := 0; i < param.GetArrayDim(); i++ {
		printer.WriteString("[]")
	}

	// Add id if not default.
	if id != "" {
		printer.Printf("%s %s", typePad, id)
	}

	// Add help string if it exists.
	if len(param.GetHelp()) > 0 {
		if id == "" {
			printer.Printf("%s ", typePad)
		}
		printer.Printf("%s  \"%s\"", idPad, param.GetHelp())
	}

	// Add outname string if it exists.
	if len(param.GetOutName()) > 0 {
		if param.GetHelp() == "" {
			printer.Printf("%s  ", idPad)
		}
		printer.Printf("%s  \"%s\"", helpPad, param.GetOutName())
	}
	printer.WriteString(",\n")
}

func (self *Params) getWidths() (int, int, int, int) {
	modeWidth := 0
	typeWidth := 0
	idWidth := 0
	helpWidth := 0
	for _, param := range self.List {
		modeWidth = max(modeWidth, len(param.getMode()))
		typeWidth = max(typeWidth, len(param.GetTname())+2*param.GetArrayDim())
		if len(param.GetId()) < 35 {
			idWidth = max(idWidth, len(param.GetId()))
		}
		if len(param.GetHelp()) < 25 {
			helpWidth = max(helpWidth, len(param.GetHelp()))
		}
	}
	return modeWidth, typeWidth, idWidth, helpWidth
}

func measureParamsWidths(paramsList ...*Params) (int, int, int, int) {
	modeWidth := 0
	typeWidth := 0
	idWidth := 0
	helpWidth := 0
	for _, params := range paramsList {
		mw, tw, iw, hw := params.getWidths()
		modeWidth = max(modeWidth, mw)
		typeWidth = max(typeWidth, tw)
		idWidth = max(idWidth, iw)
		helpWidth = max(helpWidth, hw)
	}
	return modeWidth, typeWidth, idWidth, helpWidth
}

func (self *Params) format(printer *printer, modeWidth int, typeWidth int, idWidth int, helpWidth int) {
	for _, param := range self.List {
		paramFormat(printer, param, modeWidth, typeWidth, idWidth, helpWidth)
	}
}

//
// Pipeline, Call, Return
//
func (self *Pipeline) format(printer *printer) {
	printer.printComments(self.Node.Loc, "")

	modeWidth, typeWidth, idWidth, helpWidth := measureParamsWidths(
		self.InParams, self.OutParams,
	)

	printer.Printf("pipeline %s(\n", self.Id)
	self.InParams.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	self.OutParams.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	printer.WriteString(")\n{")
	for _, callstm := range self.Calls {
		printer.WriteString(NEWLINE)
		callstm.format(printer, INDENT)
	}
	printer.WriteString(NEWLINE)
	self.Ret.format(printer)
	printer.WriteString("}\n")
}

func (self *CallStm) format(printer *printer, prefix string) {
	printer.printComments(self.Node.Loc, prefix)
	printer.WriteString(prefix)
	printer.WriteString("call ")
	if self.Modifiers.Bindings == nil {
		if self.Modifiers.Local {
			printer.WriteString("local ")
		}
		if self.Modifiers.Preflight {
			printer.WriteString("preflight ")
		}
		if self.Modifiers.Volatile {
			printer.WriteString("volatile ")
		}
	}
	printer.WriteString(self.DecId)
	if self.Id != self.DecId {
		printer.WriteString(" as ")
		printer.WriteString(self.Id)
	}
	printer.WriteString("(\n")
	self.Bindings.format(printer, prefix)
	printer.WriteString(prefix)
	if self.Modifiers.Bindings != nil && len(self.Modifiers.Bindings.List) > 0 {
		printer.WriteString(") using (\n")
		// Convert unbound-form mods to bound form.
		// Because we remove elements from the binding table if they're
		// static, we can't just use the table to see if they're needed.
		var foundMods Modifiers
		for _, binding := range self.Modifiers.Bindings.List {
			switch binding.Id {
			case local:
				foundMods.Local = true
			case preflight:
				foundMods.Preflight = true
			case volatile:
				foundMods.Volatile = true
			}
		}
		if self.Modifiers.Local && !foundMods.Local {
			self.Modifiers.Bindings.List = append(self.Modifiers.Bindings.List,
				&BindStm{
					Node: self.Modifiers.Bindings.Node,
					Id:   "local",
					Exp:  &ValExp{self.Modifiers.Bindings.Node, KindBool, true},
				})
		}
		if self.Modifiers.Preflight && !foundMods.Preflight {
			self.Modifiers.Bindings.List = append(self.Modifiers.Bindings.List,
				&BindStm{
					Node: self.Modifiers.Bindings.Node,
					Id:   "preflight",
					Exp:  &ValExp{self.Modifiers.Bindings.Node, KindBool, true},
				})
		}
		if self.Modifiers.Volatile && !foundMods.Volatile {
			self.Modifiers.Bindings.List = append(self.Modifiers.Bindings.List,
				&BindStm{
					Node: self.Modifiers.Bindings.Node,
					Id:   "volatile",
					Exp:  &ValExp{self.Modifiers.Bindings.Node, KindBool, true},
				})
		}
		sort.Slice(self.Modifiers.Bindings.List, func(i, j int) bool {
			return self.Modifiers.Bindings.List[i].Id < self.Modifiers.Bindings.List[j].Id
		})
		self.Modifiers.Bindings.format(printer, prefix)
		printer.WriteString(prefix)
	}
	printer.WriteString(")\n")
}

func (self *ReturnStm) format(printer *printer) {
	printer.printComments(self.Node.Loc, INDENT)
	printer.WriteString(INDENT)
	printer.WriteString("return (\n")
	self.Bindings.format(printer, INDENT)
	printer.WriteString(INDENT)
	printer.WriteString(")\n")
}

//
// Stage
//
func (self *Stage) format(printer *printer) {
	printer.printComments(self.Node.Loc, "")

	modeWidth, typeWidth, idWidth, helpWidth := measureParamsWidths(
		self.InParams, self.OutParams, self.ChunkIns, self.ChunkOuts,
	)
	modeWidth = max(modeWidth, len("src"))

	printer.Printf("stage %s(\n", self.Id)
	self.InParams.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	self.OutParams.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	self.Src.format(printer, modeWidth, typeWidth, idWidth)
	if idWidth > 30 || helpWidth > 20 {
		_, _, idWidth, helpWidth = measureParamsWidths(
			self.ChunkIns, self.ChunkOuts)
	}
	if self.Split {
		printer.WriteString(") split using (\n")
		self.ChunkIns.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
		self.ChunkOuts.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	}
	if self.Resources != nil {
		self.Resources.format(printer)
	}
	printer.WriteString(")\n")
}

func (self *Resources) format(printer *printer) {
	printer.printComments(self.Node.Loc, INDENT)
	printer.WriteString(") using (\n")
	// Pad depending on which arguments are present.
	// mem_gb  = x,
	// special = y
	// threads = y,
	var memPad string
	if self.SpecialNode != nil || self.ThreadNode != nil {
		memPad = " "
	}
	if self.MemNode != nil {
		printer.printComments(self.MemNode.Loc, INDENT)
		printer.WriteString(INDENT)
		printer.Printf("mem_gb%s = %d,\n", memPad, self.MemGB)
	}
	if self.SpecialNode != nil {
		printer.printComments(self.SpecialNode.Loc, INDENT)
		printer.WriteString(INDENT)
		printer.Printf("special = \"%s\",\n", self.Special)
	}
	if self.ThreadNode != nil {
		printer.printComments(self.ThreadNode.Loc, INDENT)
		printer.WriteString(INDENT)
		printer.Printf("threads = %d,\n", self.Threads)
	}
}

func (self *SrcParam) format(printer *printer, modeWidth int, typeWidth int, idWidth int) {
	printer.printComments(self.Node.Loc, INDENT)
	langPad := strings.Repeat(" ", typeWidth-len(string(self.Lang)))
	modePad := strings.Repeat(" ", modeWidth-len("src"))
	printer.Printf("%ssrc%s %v%s \"%s\",\n", INDENT,
		modePad, self.Lang, langPad, self.Path)
}

//
// Callable
//
func (self *Callables) format(printer *printer) {
	for _, callable := range self.List {
		printer.WriteString(NEWLINE)
		callable.format(printer)
	}
}

//
// Filetype
//
func (self *UserType) format(printer *printer) {
	printer.printComments(self.Node.Loc, "")
	printer.Printf("filetype %s;\n", self.Id)
}

//
// AST
//
func (self *Ast) format() string {
	printer := printer{comments: self.comments}
	for _, directive := range self.preprocess {
		printer.printComments(directive.Node.Loc, "")
		printer.WriteString(directive.Value)
		printer.WriteString(NEWLINE)
	}
	if len(self.preprocess) > 0 && len(self.UserTypes) > 0 {
		printer.WriteString(NEWLINE)
	}

	// filetype declarations.
	for _, filetype := range self.UserTypes {
		filetype.format(&printer)
	}

	// callables.
	self.Callables.format(&printer)

	// call.
	if self.Call != nil {
		if len(self.Callables.List) > 0 {
			printer.WriteString(NEWLINE)
		}
		self.Call.format(&printer, "")
	}

	printer.DumpComments()
	return printer.String()
}

//
// Exported API
//
func FormatFile(filename string) (string, error) {
	// Read MRO source file.
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return Format(string(data), filename)
}

func Format(src, filename string) (string, error) {
	// Parse and generate the AST.
	global, mmli := yaccParse(src, []FileLoc{})
	if mmli != nil { // mmli is an mmLexInfo struct
		return "", &ParseError{mmli.token, filename, mmli.loc, nil}
	}

	// Format the source.
	return global.format(), nil
}

func JsonDumpAsts(asts []*Ast) string {
	type JsonDump struct {
		UserTypes map[string]*UserType
		Stages    map[string]*Stage
		Pipelines map[string]*Pipeline
	}

	jd := JsonDump{
		UserTypes: map[string]*UserType{},
		Stages:    map[string]*Stage{},
		Pipelines: map[string]*Pipeline{},
	}

	for _, ast := range asts {
		for _, t := range ast.UserTypes {
			jd.UserTypes[t.Id] = t
		}
		for _, stage := range ast.Stages {
			jd.Stages[stage.Id] = stage
		}
		for _, pipeline := range ast.Pipelines {
			jd.Pipelines[pipeline.Id] = pipeline
		}
	}
	if jsonBytes, err := json.MarshalIndent(jd, "", "    "); err == nil {
		return string(jsonBytes)
	} else {
		return fmt.Sprintf("{ error: \"%s\" }", err.Error())
	}
}

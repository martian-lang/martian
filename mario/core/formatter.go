//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// MRO canonical formatting. Inspired by gofmt.
//
package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

var INDENT string = "    "
var NEWLINE string = "\n"

func max(x int, y int) int {
	if x > y {
		return x
	}
	return y
}

//
// Expression
//
func (self *ValExp) format() string {
	if self.value == nil {
		return "null"
	}
	if self.kind == "float" {
		// %v prints using exponential notation, which we don't want.
		// Also, strip trailing zeroes.
		re := regexp.MustCompile("0+$")
		return re.ReplaceAllString(fmt.Sprintf("%f", self.value), "")
	}
	if self.kind == "string" {
		return fmt.Sprintf("\"%s\"", self.value)
	}
	if self.kind == "map" || self.kind == "array" {
		bytes, _ := json.Marshal(expToInterface(self))
		return string(bytes)
	}
	return fmt.Sprintf("%v", self.value)
}

func (self *RefExp) format() string {
	if self.kind == "call" {
		fsrc := self.id
		if self.outputId != "default" {
			fsrc += "." + self.outputId
		}
		return fsrc
	}
	return "self." + self.id
}

//
// Binding
//
func (self *BindStm) format(idWidth int) string {
	idPad := strings.Repeat(" ", idWidth-len(self.id))
	return fmt.Sprintf("%s%s%s%s%s = %s,  ", self.exp.getNode().comments,
		INDENT, INDENT, self.id, idPad, self.exp.format())
}

func (self *BindStms) format() string {
	idWidth := 0
	for _, bindstm := range self.list {
		idWidth = max(idWidth, len(bindstm.id))
	}
	fsrc := ""
	for _, bindstm := range self.list {
		fsrc += bindstm.format(idWidth)
	}
	return fsrc
}

//
// Parameter
//
func paramFormat(param Param, modeWidth int, typeWidth int, idWidth int) string {
	id := param.getId()
	if id == "default" {
		id = ""
	}

	// Generate column alignment paddings.
	modePad := strings.Repeat(" ", modeWidth-len(param.getMode()))
	typePad := strings.Repeat(" ", typeWidth-len(param.getTname()))
	idPad := strings.Repeat(" ", idWidth-len(id))

	// Common columns up to type name.
	fsrc := fmt.Sprintf("%s%s%s%s %s", param.getNode().comments, INDENT,
		param.getMode(), modePad, param.getTname())

	// Add id if not default.
	if id != "" {
		fsrc += fmt.Sprintf("%s %s", typePad, id)
	}

	// Add help string if it exists.
	if len(param.getHelp()) > 0 {
		if id == "" {
			fsrc += fmt.Sprintf("%s ", typePad)
		}
		fsrc += fmt.Sprintf("%s  \"%s\"", idPad, param.getHelp())
	}
	return fsrc + ",  "
}

func (self *Params) getWidths() (int, int, int) {
	modeWidth := 0
	typeWidth := 0
	idWidth := 0
	for _, param := range self.list {
		modeWidth = max(modeWidth, len(param.getMode()))
		typeWidth = max(typeWidth, len(param.getTname()))
		idWidth = max(idWidth, len(param.getId()))
	}
	return modeWidth, typeWidth, idWidth
}

func measureParamsWidths(paramsList []*Params) (int, int, int) {
	modeWidth := 0
	typeWidth := 0
	idWidth := 0
	for _, params := range paramsList {
		mw, tw, iw := params.getWidths()
		modeWidth = max(modeWidth, mw)
		typeWidth = max(typeWidth, tw)
		idWidth = max(idWidth, iw)
	}
	return modeWidth, typeWidth, idWidth
}

func (self *Params) format(modeWidth int, typeWidth int, idWidth int) string {
	fsrc := ""
	for _, param := range self.list {
		fsrc += paramFormat(param, modeWidth, typeWidth, idWidth)
	}
	return fsrc
}

//
// Pipeline, Call, Return
//
func (self *Pipeline) format() string {
	modeWidth, typeWidth, idWidth := measureParamsWidths([]*Params{
		self.inParams, self.outParams,
	})

	// Steal the first param's comment.
	fsrc := self.inParams.list[0].getNode().comments
	self.inParams.list[0].getNode().comments = NEWLINE

	fsrc += NEWLINE
	fsrc += fmt.Sprintf("pipeline %s(", self.id)
	fsrc += self.inParams.format(modeWidth, typeWidth, idWidth)
	fsrc += self.outParams.format(modeWidth, typeWidth, idWidth)
	fsrc += self.node.comments
	fsrc += ")"
	fsrc += NEWLINE
	fsrc += "{"
	for _, callstm := range self.calls {
		fsrc += callstm.format()
	}
	fsrc += self.ret.format()
	fsrc += "}"
	return fsrc
}

func (self *CallStm) format() string {
	fsrc := self.bindings.list[0].exp.getNode().comments
	self.bindings.list[0].exp.getNode().comments = ""
	volatile := ""
	if self.volatile {
		volatile = "volatile "
	}
	fsrc += fmt.Sprintf("%scall %s%s(%s", INDENT, volatile, self.id, NEWLINE)
	fsrc += self.bindings.format()
	fsrc += self.node.comments
	fsrc += fmt.Sprintf("%s)", INDENT)
	fsrc += NEWLINE
	return fsrc
}

func (self *ReturnStm) format() string {
	fsrc := self.node.comments
	fsrc += fmt.Sprintf("%sreturn (", INDENT)
	fsrc += self.bindings.format()
	fsrc += NEWLINE
	fsrc += fmt.Sprintf("%s)", INDENT)
	fsrc += self.node.comments
	return fsrc
}

//
// Stage
//
func (self *Stage) format() string {
	modeWidth, typeWidth, idWidth := measureParamsWidths([]*Params{
		self.inParams, self.outParams, self.splitParams,
	})

	// Steal comment from first in param.
	fsrc := self.inParams.list[0].getNode().comments
	self.inParams.list[0].getNode().comments = NEWLINE

	fsrc += fmt.Sprintf("stage %s(", self.id)
	fsrc += self.inParams.format(modeWidth, typeWidth, idWidth)
	fsrc += self.outParams.format(modeWidth, typeWidth, idWidth)
	fsrc += self.src.format(modeWidth, typeWidth, idWidth)
	fsrc += self.node.comments
	fsrc += ")"
	if len(self.splitParams.list) > 0 {
		fsrc += " split using ("
		fsrc += self.splitParams.format(modeWidth, typeWidth, idWidth)
		fsrc += NEWLINE + ")"
	}
	return fsrc
}

func (self *SrcParam) format(modeWidth int, typeWidth int, idWidth int) string {
	langPad := strings.Repeat(" ", typeWidth-len(self.lang))
	return fmt.Sprintf("%s%ssrc %s%s \"%s\", ", self.node.comments, INDENT,
		self.lang, langPad, self.path)
}

//
// Callable
//
func (self *Callables) format() string {
	fsrc := ""
	for _, callable := range self.list {
		fsrc += callable.format()
		fsrc += NEWLINE
	}
	return fsrc
}

//
// Filetype
//
func (self *Filetype) format() string {
	fsrc := self.node.comments
	fsrc += fmt.Sprintf("filetype %s;  ", self.id)
	return fsrc
}

//
// AST
//
func (self *Ast) format() string {
	fsrc := ""

	// filetype declarations.
	for _, filetype := range self.filetypes {
		fsrc += filetype.format()
	}
	if len(self.filetypes) > 0 {
		fsrc += NEWLINE
	}

	// callables.
	fsrc += self.callables.format()

	return fsrc
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
	src := string(data)

	// Comment out @include lines since we're not preprocessing.
	re := regexp.MustCompile("@include\\s+\"")
	src = re.ReplaceAllString(src, "#@include \"")

	// Parse and generate the AST.
	global, mmli := yaccParse(src)
	if mmli != nil { // mmli is an mmLexInfo struct
		return "", &ParseError{mmli.token, filename, mmli.loc}
	}

	// Format the source.
	fmtsrc := global.format()

	// Uncomment the @include lines.
	fmtsrc = strings.Replace(fmtsrc, "#@include \"", "@include \"", -1)
	return fmtsrc, nil
}

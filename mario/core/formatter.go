//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario Formatting
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
	if self.Value == nil {
		return "null"
	}
	if self.Kind == "string" {
		return fmt.Sprintf("\"%s\"", self.Value)
	}
	if self.Kind == "map" || self.Kind == "array" {
		bytes, _ := json.Marshal(expToInterface(self))
		return string(bytes)
	}
	return fmt.Sprintf("%v", self.Value)
}

func (self *RefExp) format() string {
	if self.Kind == "call" {
		fsrc := self.Id
		if self.outputId != "default" {
			fsrc += "." + self.outputId
		}
		return fsrc
	}
	return "self." + self.Id
}

//
// Binding
//
func (self *BindStm) format(idWidth int) string {
	idPad := strings.Repeat(" ", idWidth-len(self.id))
	return fmt.Sprintf("%s%s%s%s%s = %s,  ", self.Exp.Node().comments,
		INDENT, INDENT, self.id, idPad, self.Exp.format())
}

func (self *BindStms) format() string {
	idWidth := 0
	for _, bindstm := range self.List {
		idWidth = max(idWidth, len(bindstm.id))
	}
	fsrc := ""
	for _, bindstm := range self.List {
		fsrc += bindstm.format(idWidth)
	}
	return fsrc
}

//
// Parameter
//
func paramFormat(self Param, modeWidth int, typeWidth int, idWidth int) string {
	// Column align a parameter expression.
	modePad := strings.Repeat(" ", modeWidth-len(self.Mode()))
	typePad := strings.Repeat(" ", typeWidth-len(self.Tname()))
	id := self.Id()
	if id == "default" {
		id = ""
		typePad = ""
	}
	fsrc := fmt.Sprintf("%s%s %s%s %s%s %s", self.Node().comments, INDENT,
		self.Mode(), modePad, self.Tname(), typePad, id)
	if len(self.Help()) > 0 {
		idPad := strings.Repeat(" ", idWidth-len(id))
		fsrc += fmt.Sprintf("%s  \"%s\"", idPad, self.Help())
	}
	return fsrc + ",  "
}

func (self *Params) getWidths() (int, int, int) {
	modeWidth := 0
	typeWidth := 0
	idWidth := 0
	for _, param := range self.list {
		modeWidth = max(modeWidth, len(param.Mode()))
		typeWidth = max(typeWidth, len(param.Tname()))
		idWidth = max(idWidth, len(param.Id()))
	}
	return modeWidth, typeWidth, idWidth
}

func measureCallable(callable Callable) (int, int, int) {
	modeWidthIn, typeWidthIn, idWidthIn := callable.InParams().getWidths()
	modeWidthOut, typeWidthOut, idWidthOut := callable.OutParams().getWidths()
	return max(modeWidthIn, modeWidthOut), max(typeWidthIn, typeWidthOut),
		max(idWidthIn, idWidthOut)
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
	modeWidth, typeWidth, idWidth := measureCallable(self)

	// Steal the first param's comment.
	fsrc := self.inParams.list[0].Node().comments
	self.inParams.list[0].Node().comments = NEWLINE

	fsrc += NEWLINE
	fsrc += fmt.Sprintf("pipeline %s(", self.Id)
	fsrc += self.inParams.format(modeWidth, typeWidth, idWidth)
	fsrc += self.outParams.format(modeWidth, typeWidth, idWidth)
	fsrc += self.node.comments
	fsrc += ")"
	fsrc += NEWLINE
	fsrc += "{"
	for _, callstm := range self.Calls {
		fsrc += callstm.format()
	}
	fsrc += self.ret.format()
	fsrc += "}"
	return fsrc
}

func (self *CallStm) format() string {
	fsrc := self.Bindings.List[0].Exp.Node().comments
	self.Bindings.List[0].Exp.Node().comments = ""
	volatile := ""
	if self.volatile {
		volatile = "volatile "
	}
	fsrc += fmt.Sprintf("%scall %s%s(%s", INDENT, volatile, self.Id, NEWLINE)
	fsrc += self.Bindings.format()
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
	modeWidth, typeWidth, idWidth := measureCallable(self)

	// Steal comment from first in param.
	fsrc := self.inParams.list[0].Node().comments
	self.inParams.list[0].Node().comments = NEWLINE

	fsrc += fmt.Sprintf("stage %s(", self.Id)
	fsrc += self.inParams.format(modeWidth, typeWidth, idWidth)
	fsrc += self.outParams.format(modeWidth, typeWidth, idWidth)
	fsrc += self.src.format()
	fsrc += self.node.comments
	fsrc += ")"
	return fsrc
}

func (self *SrcParam) format() string {
	fsrc := self.node.comments
	fsrc += fmt.Sprintf("%s src %s \"%s\", ", INDENT, self.lang, self.path)
	return fsrc
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
	fsrc += fmt.Sprintf("filetype %s;  ", self.Id)
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

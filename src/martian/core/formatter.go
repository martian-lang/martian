//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
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

//
// Expression
//
func (self *ValExp) format() string {
	if self.Value == nil {
		return "null"
	}
	if self.Kind == "float" {
		// Use %f because %v prints using exp notation, which we don't want.
		// Also, strip trailing zeroes.
		re := regexp.MustCompile("0+$")
		str := re.ReplaceAllString(fmt.Sprintf("%f", self.Value), "")
		// Put back a zero if it's all zeroes after decimal point.
		re = regexp.MustCompile("\\.$")
		return re.ReplaceAllString(str, ".0")
		return str
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
func (self *BindStm) format(idWidth int) string {
	idPad := strings.Repeat(" ", idWidth-len(self.Id))
	return fmt.Sprintf("%s%s%s%s%s = %s,", self.Exp.getNode().Comments,
		INDENT, INDENT, self.Id, idPad, self.Exp.format())
}

func (self *BindStms) format() string {
	idWidth := 0
	for _, bindstm := range self.List {
		idWidth = max(idWidth, len(bindstm.Id))
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
	fsrc := fmt.Sprintf("%s%s%s%s %s", param.getNode().Comments, INDENT,
		param.getMode(), modePad, param.getTname())

	// If type is annotated as array, add brackets and shrink padding.
	for i := 0; i < param.getArrayDim(); i++ {
		fsrc += "[]"
		typePad = typePad[2:]
	}

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
	return fsrc + ","
}

func (self *Params) getWidths() (int, int, int) {
	modeWidth := 0
	typeWidth := 0
	idWidth := 0
	for _, param := range self.List {
		modeWidth = max(modeWidth, len(param.getMode()))
		typeWidth = max(typeWidth, len(param.getTname())+2*param.getArrayDim())
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
	for _, param := range self.List {
		fsrc += paramFormat(param, modeWidth, typeWidth, idWidth)
	}
	return fsrc
}

//
// Pipeline, Call, Return
//
func (self *Pipeline) format() string {
	modeWidth, typeWidth, idWidth := measureParamsWidths([]*Params{
		self.InParams, self.OutParams,
	})

	// Steal the first param's comment.
	fsrc := ""
	if len(self.InParams.List) > 0 {
		fsrc += self.InParams.List[0].getNode().Comments
		self.InParams.List[0].getNode().Comments = NEWLINE
	}

	fsrc += NEWLINE
	fsrc += fmt.Sprintf("pipeline %s(", self.Id)
	fsrc += self.InParams.format(modeWidth, typeWidth, idWidth)
	fsrc += self.OutParams.format(modeWidth, typeWidth, idWidth)
	fsrc += self.Node.Comments
	fsrc += ")"
	fsrc += NEWLINE
	fsrc += "{"
	for _, callstm := range self.Calls {
		fsrc += callstm.format(INDENT)
	}
	fsrc += self.Ret.format()
	fsrc += "}"
	return fsrc
}

func (self *CallStm) format(prefix string) string {
	fsrc := ""
	if len(self.Bindings.List) > 0 {
		fsrc += self.Bindings.List[0].Exp.getNode().Comments
		self.Bindings.List[0].Exp.getNode().Comments = ""
	}
	volatile := ""
	local := ""
	preflight := ""
	if self.Modifiers.Local {
		local = "local "
	}
	if self.Modifiers.Preflight {
		preflight = "preflight "
	}
	if self.Modifiers.Volatile {
		volatile = "volatile "
	}
	fsrc += fmt.Sprintf("%scall %s%s%s%s(%s", prefix, local, preflight, volatile, self.Id, NEWLINE)
	fsrc += self.Bindings.format()
	fsrc += self.Node.Comments
	fsrc += fmt.Sprintf("%s)", prefix)
	fsrc += NEWLINE
	return fsrc
}

func (self *ReturnStm) format() string {
	fsrc := self.Node.Comments
	fsrc += fmt.Sprintf("%sreturn (", INDENT)
	fsrc += self.Bindings.format()
	fsrc += NEWLINE
	fsrc += fmt.Sprintf("%s)", INDENT)
	fsrc += self.Node.Comments
	return fsrc
}

//
// Stage
//
func (self *Stage) format() string {
	modeWidth, typeWidth, idWidth := measureParamsWidths([]*Params{
		self.InParams, self.OutParams, self.SplitParams,
	})

	// Steal comment from first in param.
	fsrc := ""
	if len(self.InParams.List) > 0 {
		fsrc += self.InParams.List[0].getNode().Comments
		self.InParams.List[0].getNode().Comments = NEWLINE
	}

	fsrc += fmt.Sprintf("stage %s(", self.Id)
	fsrc += self.InParams.format(modeWidth, typeWidth, idWidth)
	fsrc += self.OutParams.format(modeWidth, typeWidth, idWidth)
	fsrc += self.Src.format(modeWidth, typeWidth, idWidth)
	fsrc += self.Node.Comments
	fsrc += ")"
	if self.Split {
		fsrc += " split using ("
		fsrc += self.SplitParams.format(modeWidth, typeWidth, idWidth)
		fsrc += NEWLINE + ")"
	}
	return fsrc
}

func (self *SrcParam) format(modeWidth int, typeWidth int, idWidth int) string {
	langPad := strings.Repeat(" ", typeWidth-len(self.Lang))
	return fmt.Sprintf("%s%ssrc %s%s \"%s\",", self.Node.Comments, INDENT,
		self.Lang, langPad, self.Path)
}

//
// Callable
//
func (self *Callables) format() string {
	fsrc := ""
	for _, callable := range self.List {
		fsrc += callable.format()
		fsrc += NEWLINE
	}
	return fsrc
}

//
// Filetype
//
func (self *Filetype) format() string {
	fsrc := self.Node.Comments
	fsrc += fmt.Sprintf("filetype %s;", self.Id)
	return fsrc
}

//
// AST
//
func (self *Ast) format() string {
	fsrc := ""

	// filetype declarations.
	for _, filetype := range self.Filetypes {
		fsrc += filetype.format()
	}
	if len(self.Filetypes) > 0 {
		fsrc += NEWLINE
	}

	// call.
	if self.Call != nil {
		fsrc += self.Call.format("")
	}

	// callables.
	fsrc += self.Callables.format()

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

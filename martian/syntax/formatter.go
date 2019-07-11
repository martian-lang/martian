//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO canonical formatting. Inspired by gofmt.
//

package syntax

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

type stringWriter interface {
	io.ByteWriter
	io.Writer
	WriteRune(rune) (int, error)
	WriteString(string) (int, error)
}

type printer struct {
	buf         strings.Builder
	comments    map[string][]*commentBlock
	lastComment SourceLoc
}

func (self *printer) printComments(node *AstNode, prefix string) {
	if self.lastComment.File != nil && node.Loc.File != nil &&
		self.lastComment.File.FullPath != node.Loc.File.FullPath {
		for _, comment := range self.comments[self.lastComment.File.FullPath] {
			self.buf.WriteString(comment.Value)
			self.buf.WriteString(NEWLINE)
		}
		delete(self.comments, self.lastComment.File.FullPath)
		self.buf.WriteString("#\n# @include \"")
		self.buf.WriteString(node.Loc.File.FileName)
		self.buf.WriteString("\"\n#\n\n")
	}
	for _, c := range node.scopeComments {
		if self.lastComment.Line != 0 && self.lastComment.Line == c.Loc.Line-2 {
			self.buf.WriteString(NEWLINE)
		}

		self.lastComment = c.Loc
		self.buf.WriteString(prefix)
		self.buf.WriteString(c.Value)
		self.buf.WriteString(NEWLINE)
	}
	if len(node.scopeComments) > 0 {
		self.buf.WriteString(NEWLINE)
	}
	for _, c := range node.Comments {
		self.buf.WriteString(prefix)
		self.buf.WriteString(c)
		self.buf.WriteString(NEWLINE)
	}
	self.lastComment = node.Loc
}

func (self *printer) WriteString(s string) (int, error) {
	return self.buf.WriteString(s)
}

func (self *printer) mustWriteString(s string) {
	if _, err := self.WriteString(s); err != nil {
		panic(err)
	}
}

func (self *printer) Write(b []byte) (int, error) {
	return self.buf.Write(b)
}

func (self *printer) WriteByte(b byte) error {
	return self.buf.WriteByte(b)
}

func (self *printer) WriteRune(r rune) (int, error) {
	return self.buf.WriteRune(r)
}

func (self *printer) mustWriteRune(r rune) {
	if _, err := self.WriteRune(r); err != nil {
		panic(err)
	}
}

func (self *printer) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&self.buf, format, args...)
}

func (self *printer) DumpComments() {
	for _, fcomments := range self.comments {
		for _, comment := range fcomments {
			self.buf.WriteString(comment.Value)
			self.buf.WriteString(NEWLINE)
		}
	}
	self.comments = nil
}

func (self *printer) String() string {
	return self.buf.String()
}

// Format returns mro source code for the AST.
func (self *Ast) Format() string {
	includesProcessed := len(self.Files) > 1
	return self.format(!includesProcessed)
}

//
// AST
//
func (self *Ast) format(writeIncludes bool) string {
	needSpacer := false
	printer := printer{
		comments: make(map[string][]*commentBlock, len(self.Files)),
	}
	if len(self.Files) > 0 {
		// Set the printer's last comment location to the top of the
		// top-level file, so that the top-level include is reported
		// correctly.
		var topFile *SourceFile
		for _, f := range self.Files {
			topFile = f
			break
		}
		for topFile != nil && len(topFile.IncludedFrom) > 0 {
			topFile = topFile.IncludedFrom[0].File
		}
		printer.lastComment = SourceLoc{
			Line: 0,
			File: topFile,
		}
	}

	for _, comment := range self.comments {
		printer.comments[comment.Loc.File.FullPath] = append(
			printer.comments[comment.Loc.File.FullPath],
			comment)
	}
	if writeIncludes {
		for _, directive := range self.Includes {
			printer.printComments(&directive.Node, "")
			printer.mustWriteString("@include \"")
			printer.mustWriteString(directive.Value)
			printer.mustWriteRune('"')
			printer.mustWriteString(NEWLINE)
			needSpacer = true
		}
	}

	// filetype declarations.
	if needSpacer && len(self.UserTypes) > 0 {
		printer.mustWriteString(NEWLINE)
	}
	for _, filetype := range self.UserTypes {
		filetype.format(&printer)
		needSpacer = true
	}
	if needSpacer && len(self.StructTypes) > 0 {
		printer.mustWriteString(NEWLINE)
	}
	for i, structType := range self.StructTypes {
		if i != 0 {
			printer.mustWriteString(NEWLINE)
		}
		structType.format(&printer)
		needSpacer = true
	}

	// callables.
	if needSpacer && self.Callables != nil && len(self.Callables.List) > 0 {
		printer.mustWriteString(NEWLINE)
	}
	self.Callables.format(&printer)

	// call.
	if self.Call != nil {
		if self.Callables != nil && len(self.Callables.List) > 0 || needSpacer {
			printer.mustWriteString(NEWLINE)
		}
		self.Call.format(&printer, "")
	}

	// Any comments which went at the ends of a file, after any nodes.
	printer.DumpComments()
	return printer.String()
}

//
// Exported API
//

func FormatFile(filename string, fixIncludes bool, mropath []string) (string, error) {
	var parser Parser
	return parser.FormatFile(filename, fixIncludes, mropath)
}

func (parser *Parser) FormatFile(filename string, fixIncludes bool, mropath []string) (string, error) {
	// Read MRO source file.
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return parser.FormatSrcBytes(data, filename, fixIncludes, mropath)
}

func Format(src string, filename string, fixIncludes bool, mropath []string) (string, error) {
	return FormatSrcBytes([]byte(src), filename, fixIncludes, mropath)
}

func FormatSrcBytes(src []byte, filename string, fixIncludes bool, mropath []string) (string, error) {
	var parser Parser
	return parser.FormatSrcBytes(src, filename, fixIncludes, mropath)
}

func (parser *Parser) FormatSrcBytes(src []byte, filename string, fixIncludes bool, mropath []string) (string, error) {
	global, mmli := parser.UncheckedParse(src, filename)
	if mmli != nil { // mmli is an mmLexInfo struct
		return "", mmli
	}
	var err error
	if fixIncludes {
		err = parser.FixIncludes(global, mropath)
	}

	// Format the source.
	return global.format(true), err
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

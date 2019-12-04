//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian lexical scanner.
//

package syntax

import (
	"bytes"
	"errors"
	"strings"
)

type mmLexInfo struct {
	src      []byte // All the data we're scanning
	pos      int    // Position of the scan head
	loc      int    // Keep track of the line number
	previous []byte //
	token    []byte // Cache the last token for error messaging
	err      string // errors reported by the parser
	global   *Ast
	exp      ValExp // If parsing an expression, rather than an AST.
	srcfile  *SourceFile
	comments []*commentBlock
	// for many byte->string conversions, the same string is expected
	// to show up frequently.  For example the stage name will usually
	// appear at least 3 times: when it's declared, when it's called, and
	// when its output is referenced.  So we coalesce those allocations.
	intern *stringIntern
}

var newlineBytes = []byte("\n")

func (self *mmLexInfo) Loc() SourceLoc {
	return SourceLoc{
		Line: self.loc,
		File: self.srcfile,
	}
}

func (self *mmLexInfo) Lex(lval *mmSymType) int {
	// Loop until we return a token or run out of data.
	for {
		// Stop if we run out of data.
		if self.pos >= len(self.src) {
			return 0
		}
		// Slice the data using pos as a cursor.
		head := self.src[self.pos:]

		// Iterate through the regexps until one matches the head.
		tokid, val := nextToken(head)
		// Advance the cursor pos.
		self.pos += len(val)

		// If whitespace or comment, advance line count by counting newlines.
		if tokid == SKIP {
			self.loc += bytes.Count(val, newlineBytes)
			continue
		} else if tokid == COMMENT {
			self.comments = append(self.comments, &commentBlock{
				self.Loc(),
				string(bytes.TrimSpace(val)),
			})
			self.loc++
			continue
		}

		// If got parseable token, pass it and line number to parser.
		self.previous = self.token
		self.token = val
		lval.val = self.token
		lval.loc = self.loc // give grammar rules access to loc

		// give NewAstNode access to file to generate file-local locations
		lval.srcfile = self.srcfile
		lval.global = self.global
		lval.intern = self.intern

		return tokid
	}
}

func (self *mmLexInfo) getLine() (int, []byte) {
	if self.pos >= len(self.src) {
		return 0, nil
	}
	lineStart := bytes.LastIndexByte(self.src[:self.pos], '\n')
	lineEnd := bytes.IndexByte(self.src[self.pos:], '\n')
	if lineStart == -1 {
		lineStart = 0
	}
	if lineEnd == -1 {
		return lineStart, self.src[lineStart:]
	} else {
		return lineStart, self.src[lineStart : self.pos+lineEnd]
	}
}

// mmLexInfo must have a method Error(string) for the yacc parser, but we
// want a version with Error() string to satisfy the error interface.
type mmLexError struct {
	info mmLexInfo
}

func (err *mmLexError) writeTo(w stringWriter) {
	mustWriteString(w, "MRO ParseError: unexpected token '")
	mustWrite(w, err.info.token)
	if err.info.err != "" {
		mustWriteString(w, "' (")
		mustWriteString(w, err.info.err)
		mustWriteRune(w, ')')
	} else {
		mustWriteRune(w, '\'')
	}
	if len(err.info.previous) > 0 {
		mustWriteString(w, " after '")
		mustWrite(w, err.info.previous)
		mustWriteRune(w, '\'')
	}
	if lineStart, line := err.info.getLine(); len(line) > 0 {
		mustWriteString(w, "\n")
		mustWrite(w, line)
		mustWriteRune(w, '\n')
		for i := 0; i < err.info.pos-len(err.info.token)-lineStart; i++ {
			mustWriteRune(w, ' ')
		}
		mustWriteString(w, "^\n    at ")
	} else {
		mustWriteString(w, " at ")
	}
	loc := err.info.Loc()
	loc.writeTo(w, "        ")
}

func (self *mmLexError) Error() string {
	var buff strings.Builder
	buff.Grow(200)
	self.writeTo(&buff)
	return buff.String()
}

func (self *mmLexInfo) Error(e string) {
	// Unfortunately goyacc doesn't expose a stable API to get at the expected
	// token other than this.
	if i := strings.Index(e, ", expecting "); i > 0 {
		self.err = e[i+2:]
	} else {
		self.err = ""
	}
}

func init() {
	// There does not seem to be a way to make goyacc not initialize this to false.
	mmErrorVerbose = true
}

func yaccParseAny(src []byte, file *SourceFile, intern *stringIntern) (int, mmLexError) {
	lexinfo := mmLexError{
		info: mmLexInfo{
			src:     src,
			pos:     0,
			loc:     1,
			srcfile: file,
			intern:  intern,
		},
	}
	result := mmParse(&lexinfo.info)
	if result == 0 {
		lexinfo.info.global.comments = lexinfo.info.comments
		lexinfo.info.global.comments = compileComments(
			lexinfo.info.global.comments, lexinfo.info.global)
	}
	return result, lexinfo
}

// yaccParse parses an Ast from a byte array.
func yaccParse(src []byte, file *SourceFile, intern *stringIntern) (*Ast, error) {
	if result, info := yaccParseAny(src, file, intern); result != 0 {
		return nil, &info // return lex on error to provide loc and token info
	} else if info.info.exp != nil {
		return info.info.global, errors.New(
			"Expected: includes or stage or pipeline or call.")
	} else {
		return info.info.global, nil // success
	}
}

// parseExp parses a ValExp from a byte array.
func parseExp(src []byte, file *SourceFile, intern *stringIntern) (ValExp, error) {
	if result, info := yaccParseAny(src, file, intern); result != 0 {
		return nil, &info // return lex on error to provide loc and token info
	} else if info.info.exp == nil {
		return nil, errors.New("Expected: expression, got mro instead")
	} else {
		return info.info.exp, nil
	}
}

func attachComments(comments []*commentBlock, node *AstNode) []*commentBlock {
	scopeComments := make([]*commentBlock, 0, len(comments))
	nodeComments := make([]*commentBlock, 0, len(comments))
	loc := node.Loc
	for len(comments) > 0 && comments[0].Loc.Line <= loc.Line {
		if len(nodeComments) > 0 &&
			nodeComments[len(nodeComments)-1].Loc.Line <
				comments[0].Loc.Line-1 {
			// If a line was skipped, move those comments to  discard the comments before
			// the skipped line, only associating the ones which
			// didn't skip a line.
			scopeComments = append(scopeComments, nodeComments...)
			nodeComments = nil
		}
		nodeComments = append(nodeComments, comments[0])
		comments = comments[1:]
	}
	if len(nodeComments) > 0 &&
		nodeComments[len(nodeComments)-1].Loc.Line < node.Loc.Line-1 {
		// If there was a blank non-comment line between the last comment
		// block and this node, stick on scopeComments
		scopeComments = append(scopeComments, nodeComments...)
		nodeComments = nil
	}
	node.scopeComments = scopeComments
	node.Comments = make([]string, 0, len(nodeComments))
	for _, c := range nodeComments {
		node.Comments = append(node.Comments, c.Value)
	}
	return comments
}

func compileComments(comments []*commentBlock, node nodeContainer) []*commentBlock {
	nodes := node.getSubnodes()
	for _, n := range nodes {
		comments = attachComments(comments, n.getNode())
		comments = compileComments(comments, n)
	}
	if len(nodes) > 0 && node.inheritComments() {
		nodes[0].getNode().scopeComments = append(
			node.(AstNodable).getNode().scopeComments,
			nodes[0].getNode().scopeComments...)
		nodes[0].getNode().Comments = append(
			node.(AstNodable).getNode().Comments,
			nodes[0].getNode().Comments...)
	}
	return comments
}

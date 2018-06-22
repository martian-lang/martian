//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian lexical scanner. Simple regexp-based implementation.
//

package syntax

import (
	"bytes"
	"regexp"
	"strings"
)

// re matches text to produce token.
type rule struct {
	pattern string
	re      *regexp.Regexp
	tokid   int
}

// Pre-compile regexps for token matching.
func newRule(pattern string, tokid int) *rule {
	return &rule{
		pattern,
		regexp.MustCompile("^" + pattern),
		tokid,
	}
}

var rules = []*rule{
	// Order matters.
	newRule("\\s+", SKIP),      // whitespace
	newRule("#.*\\n", COMMENT), // Python-style comments
	newRule(`@include`, INCLUDE_DIRECTIVE),
	newRule("=", EQUALS),
	newRule("\\(", LPAREN),
	newRule("\\)", RPAREN),
	newRule("{", LBRACE),
	newRule("}", RBRACE),
	newRule("\\[", LBRACKET),
	newRule("\\]", RBRACKET),
	newRule(":", COLON),
	newRule(";", SEMICOLON),
	newRule(",", COMMA),
	newRule("\\.", DOT),
	newRule("\"[^\\\"]*\"", LITSTRING), // double-quoted strings. escapes not supported
	newRule("filetype\\b", FILETYPE),
	newRule("stage\\b", STAGE),
	newRule("pipeline\\b", PIPELINE),
	newRule("call\\b", CALL),
	newRule(local+"\\b", LOCAL),
	newRule(preflight+"\\b", PREFLIGHT),
	newRule(volatile+"\\b", VOLATILE),
	newRule(disabled+"\\b", DISABLED),
	newRule(strict+"\\b", STRICT),
	newRule("threads\\b", THREADS),
	newRule("mem_?gb\\b", MEM_GB),
	newRule("special\\b", SPECIAL),
	newRule("retain\\b", RETAIN),
	newRule("sweep\\b", SWEEP),
	newRule("split\\b", SPLIT),
	newRule("using\\b", USING),
	newRule("self\\b", SELF),
	newRule("return\\b", RETURN),
	newRule("in\\b", IN),
	newRule("out\\b", OUT),
	newRule("src\\b", SRC),
	newRule("as\\b", AS),
	newRule("py\\b", PY),
	newRule("go\\b", GO),
	newRule("sh\\b", SH),
	newRule("exec\\b", EXEC),
	newRule("comp\\b", COMPILED),
	newRule("map\\b", MAP),
	newRule("int\\b", INT),
	newRule("string\\b", STRING),
	newRule("float\\b", FLOAT),
	newRule("path\\b", PATH),
	newRule("bool\\b", BOOL),
	newRule("true\\b", TRUE),
	newRule("false\\b", FALSE),
	newRule("null\\b", NULL),
	newRule("default\\b", DEFAULT),
	newRule("_?[a-zA-Z][a-zA-z0-9_]*\\b", ID),
	newRule("-?[0-9]+\\.[0-9]+\\b", NUM_FLOAT),                     // support exponential
	newRule("-?[0-9]+(:?\\.[0-9]+)?[eE][-+]?[0-9]+\\b", NUM_FLOAT), // support exponential
	newRule("-?[0-9]+\\b", NUM_INT),
	newRule(".", INVALID),
}

type mmLexInfo struct {
	src      []byte // All the data we're scanning
	pos      int    // Position of the scan head
	loc      int    // Keep track of the line number
	previous []byte //
	token    []byte // Cache the last token for error messaging
	global   *Ast
	srcfile  *SourceFile
	comments []*commentBlock
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
		var val []byte
		var r *rule
		for _, r = range rules {
			val = r.re.Find(head)
			if len(val) > 0 {
				// Advance the cursor pos.
				self.pos += len(val)
				break
			}
		}

		// If whitespace or comment, advance line count by counting newlines.
		if r.tokid == SKIP {
			self.loc += bytes.Count(val, newlineBytes)
			continue
		} else if r.tokid == COMMENT {
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
		lval.val = string(self.token)
		lval.loc = self.loc // give grammar rules access to loc

		// give NewAstNode access to file to generate file-local locations
		lval.srcfile = self.srcfile
		lval.global = self.global

		return r.tokid
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
	w.WriteString("MRO ParseError: unexpected token '")
	w.Write(err.info.token)
	if len(err.info.previous) > 0 {
		w.WriteString("' after '")
		w.Write(err.info.previous)
	}
	if lineStart, line := err.info.getLine(); len(line) > 0 {
		w.WriteString("'\n")
		w.Write(line)
		w.WriteRune('\n')
		for i := 0; i < err.info.pos-len(err.info.token)-lineStart; i++ {
			w.WriteRune(' ')
		}
		w.WriteString("^\n    at ")
	} else {
		w.WriteString("' at ")
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

func (self *mmLexInfo) Error(string) {}

func yaccParse(src []byte, file *SourceFile) (*Ast, error) {
	lexinfo := mmLexError{
		info: mmLexInfo{
			src:     src,
			pos:     0,
			loc:     1,
			srcfile: file,
		},
	}
	if mmParse(&lexinfo.info) != 0 {
		return nil, &lexinfo // return lex on error to provide loc and token info
	}
	lexinfo.info.global.comments = lexinfo.info.comments
	lexinfo.info.global.comments = compileComments(
		lexinfo.info.global.comments, lexinfo.info.global)
	return lexinfo.info.global, nil // success
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

func unquote(qs string) string {
	return strings.Replace(qs, "\"", "", -1)
}

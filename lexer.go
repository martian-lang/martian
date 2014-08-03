package main

import (
	"fmt"
	"regexp"
	"strings"
)

type Rule struct {
	re    *regexp.Regexp
	token int
}

func NewRule(pattern string, token int) *Rule {
	// Pre-compile regexps for token matching
	re, _ := regexp.Compile("^" + pattern)
	return &Rule{re, token}
}

var rules = []*Rule{
	// Order matters.
	NewRule("\\s+", SKIP),   // whitespace
	NewRule("#.*\\n", SKIP), // Python-style comments
	NewRule("=", EQUALS),
	NewRule("\\(", LPAREN),
	NewRule("\\)", RPAREN),
	NewRule("{", LBRACE),
	NewRule("}", RBRACE),
	NewRule("\\[", LBRACKET),
	NewRule("\\]", RBRACKET),
	NewRule(";", SEMICOLON),
	NewRule(",", COMMA),
	NewRule("\\.", DOT),
	NewRule("\"[^\\\"]*\"", LITSTRING), // double-quoted strings. escapes not supported
	NewRule("filetype\\b", FILETYPE),
	NewRule("stage\\b", STAGE),
	NewRule("pipeline\\b", PIPELINE),
	NewRule("call\\b", CALL),
	NewRule("volatile\\b", VOLATILE),
	NewRule("sweep\\b", SWEEP),
	NewRule("split\\b", SPLIT),
	NewRule("using\\b", USING),
	NewRule("self\\b", SELF),
	NewRule("return\\b", RETURN),
	NewRule("in\\b", IN),
	NewRule("out\\b", OUT),
	NewRule("src\\b", SRC),
	NewRule("py\\b", PY),
	NewRule("go\\b", GO),
	NewRule("sh\\b", SH),
	NewRule("exec\\b", EXEC),
	NewRule("int\\b", INT),
	NewRule("string\\b", STRING),
	NewRule("float\\b", FLOAT),
	NewRule("path\\b", PATH),
	NewRule("file\\b", FILE),
	NewRule("bool\\b", BOOL),
	NewRule("true\\b", TRUE),
	NewRule("false\\b", FALSE),
	NewRule("null\\b", NULL),
	NewRule("default\\b", DEFAULT),
	NewRule("[a-zA-Z_][a-zA-z0-9_]*", ID),
	NewRule("-?[0-9]+\\.[0-9]+([eE][-+]?[0-9]+)?\\b", NUM_FLOAT), // support exponential
	NewRule("-?[0-9]+\\b", NUM_INT),
	NewRule(".", INVALID),
}

type SyntaxError struct {
	Lineno int
	Line   string
	Token  string
	Err    error
}

func (self *SyntaxError) Error() string {
	return fmt.Sprintf("MRO syntax error: unexpected token '%s' on line %d:\n\n%s", self.Token, self.Lineno, self.Line)
}

type mmLex struct {
	source string       // All the data we're scanning
	pos    int          // Position of the scan head
	lineno int          // Keep track of the line number
	last   string       // Cache the last token for error messaging
	err    *SyntaxError // Constructed syntax error object
}

func (self *mmLex) Lex(lval *mmSymType) int {
	// Loop until we return a token or run out of data.
	for {
		// Stop if we run out of data.
		if self.pos >= len(self.source) {
			return 0
		}
		// Slice the data using pos as a cursor.
		head := self.source[self.pos:]

		// Iterate through the regexps until one matches the head.
		var val string
		var rule *Rule
		for _, rule = range rules {
			val = rule.re.FindString(head)
			if len(val) > 0 {
				break
			}
		}

		// Advance the cursor pos.
		self.pos += len(val)

		// If whitespace or comment, advance line count by counting newlines.
		if rule.token == SKIP {
			self.lineno += strings.Count(val, "\n")
			continue
		}

		// If got parseable token, pass it and line number to parser.
		// fmt.Println(rule.token, val, self.lineno)
		lval.val = val
		self.last = val
		lval.lineno = self.lineno
		return rule.token
	}
}

func (self *mmLex) Error(s string) {
	// Capture the error line by searching back and forth for newlines.
	spos := strings.LastIndex(self.source[0:self.pos], "\n") + 1
	epos := strings.Index(self.source[self.pos:], "\n") + self.pos + 1
	self.err = &SyntaxError{
		Lineno: self.lineno,
		Line:   self.source[spos:epos],
		Token:  self.last,
	}
}

func Parse(src string) (*Ptree, error) {
	lex := mmLex{src, 0, 1, "", nil}
	if mmParse(&lex) == 0 {
		return &ptree, nil
	}
	return nil, lex.err
}

package main

import (
	"fmt"
	"io/ioutil"
	"regexp"
)

const (
	SKIP = iota
	FILETYPE
	STAGE
	PIPELINE
	CALL
	VOLATILE
	SWEEP
	SPLIT
	USING
	SELF
	RETURN
	EQUALS
	LPAREN
	RPAREN
	LBRACE
	RBRACE
	LBRACKET
	RBRACKET
	SEMICOLON
	COMMA
	DOT
	IN
	OUT
	SRC
	PY
	INT
	STRING
	FLOAT
	PATH
	FILE
	BOOL
	TRUE
	FALSE
	NULL
	DEFAULT
	LITSTRING
	ID
	NUM_FLOAT
	NUM_INT
)

type Rule struct {
	re    *regexp.Regexp
	token int
}

func NewRule(pattern string, token int) *Rule {
	re, _ := regexp.Compile("^" + pattern)
	return &Rule{re, token}
}

var rules = []*Rule{
	NewRule("\\s+", SKIP),
	NewRule("#.*\\n", SKIP),
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
	NewRule("\"[^\\\"]*\"", LITSTRING),
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
	NewRule("-?[0-9]+\\.[0-9]+([eE][-+]?[0-9]+)?\\b", NUM_FLOAT),
	NewRule("-?[0-9]+\\b", NUM_INT),
}

type MarioLex struct {
	source string
	pos    int
}

func (self *MarioLex) Lex() int {
	if self.pos >= len(self.source) {
		return 1
	}
	head := self.source[self.pos:]

	var val string
	var rule *Rule
	for _, rule = range rules {
		val = rule.re.FindString(head)
		if len(val) > 0 {
			break
		}
	}
	self.pos += len(val)
	if rule.token != SKIP {
		fmt.Println(rule.token, val)
	}
	return 0
}

func main() {
	data, _ := ioutil.ReadFile("stages.mro")
	lexer := MarioLex{
		source: string(data),
		pos:    0,
	}
	for {
		retval := lexer.Lex()
		if retval != 0 {
			break
		}
	}
}

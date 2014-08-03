%{

package main

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

%}

%union{
	val string
}

%type <val> file dec_list dec param_list param help type src_lang split_exp call_stm_list call_stm return_stm bind_stm_list bind_stm exp_list exp value_exp

%token <val> SKIP FILETYPE STAGE PIPELINE CALL VOLATILE SWEEP SPLIT USING SELF RETURN EQUALS LPAREN RPAREN LBRACE RBRACE LBRACKET RBRACKET SEMICOLON COMMA DOT IN OUT SRC PY INT STRING FLOAT PATH FILE BOOL TRUE FALSE NULL DEFAULT LITSTRING ID NUM_FLOAT NUM_INT INVALID

%%
file
	: dec_list
	;

dec_list
	: dec_list dec
	| dec
	;

dec
	: FILETYPE file_id SEMICOLON
	| STAGE ID LPAREN param_list RPAREN 
	| STAGE ID LPAREN param_list RPAREN split_exp
	| PIPELINE ID LPAREN param_list RPAREN LBRACE call_stm_list return_stm RBRACE
	;

file_id
	: ID DOT ID
	| ID
	;

param_list
	: param_list param
	| param
	;

param
	: IN type ID help 
	| OUT type help 
	| OUT type ID help 
	| SRC src_lang LITSTRING COMMA
	;
help
    : LITSTRING COMMA
    | COMMA
    ;

type
    : INT
    | STRING
    | PATH
    | FLOAT
    | BOOL
    | ID
    | ID DOT ID
    ;

src_lang
    : PY
    ;

split_exp
	: SPLIT USING LPAREN param_list RPAREN
	;

return_stm
    : RETURN LPAREN bind_stm_list RPAREN
    ;

call_stm_list
    : call_stm_list call_stm
    | call_stm
    ;

call_stm
    : CALL ID LPAREN bind_stm_list RPAREN
    | CALL VOLATILE ID LPAREN bind_stm_list RPAREN
    ;

bind_stm_list
    : bind_stm_list bind_stm
    | bind_stm
    ;

bind_stm
    : ID EQUALS exp COMMA
    | ID EQUALS SWEEP LPAREN exp RPAREN COMMA
    ;

exp_list
    : exp_list COMMA exp
    | exp
    ; 

exp
    : LBRACKET exp_list RBRACKET
    | LBRACKET RBRACKET
    | PATH LPAREN LITSTRING RPAREN
    | FILE LPAREN LITSTRING RPAREN
    | NUM_FLOAT
    | NUM_INT
    | LITSTRING
    | TRUE
    | FALSE
    | NULL
    | value_exp
    ;

value_exp
    : ID DOT ID
    | ID
    | SELF DOT ID
    ;
%%

//const SKIP = 0

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
	NewRule(".", INVALID),
}

type MarioLex struct {
	source string
	pos    int
	lineno int
	last   string
}

func (self *MarioLex) Lex(lval *MarioSymType) int {
	// 
	for {
		if self.pos >= len(self.source) {
			return 0
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
		if rule.token == SKIP {
			self.lineno += strings.Count(val, "\n")
			continue
		}

		fmt.Println(rule.token, val, self.lineno)
		lval.val = val
		self.last = val
		return rule.token
	}
}

func (self *MarioLex) Error(s string) {
	fmt.Printf("Unexpected token '%s' on line %d\n", self.last, self.lineno)
}

func main() {

	//MarioDebug = 1000
	data, _ := ioutil.ReadFile("stages.mro")
	MarioParse(&MarioLex{
		source: string(data),
		pos:    0,
		lineno: 1,
	})
	/*
	for {
		retval := lexer.Lex()
		if retval != 0 {
			break
		}
	}
	*/
}

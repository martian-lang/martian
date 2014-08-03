%{

package main

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)


type Node interface {
}

type FileTypeDec struct {
	Node
	filetype string
}
func NewFileTypeDec(filetype string) *FileTypeDec {
	return &FileTypeDec{filetype: filetype}
}

type StageDec struct {
	Node
	name string
}
func NewStageDec(name string) *StageDec {
	return &StageDec{name: name}
}

type PipelineDec struct {
	Node
	name string
}
func NewPipelineDec(name string) *PipelineDec {
	return &PipelineDec{name: name}
}

type File struct {
	decList []Node
}

var ast File
%}

%union{
	val string
	node Node
	list []Node
}

%type <val> file_id
%type <node> file dec param help type src_lang
%type <list> dec_list param_list
%type <node> split_exp call_stm_list call_stm return_stm bind_stm_list 
%type <node> bind_stm exp_list exp value_exp

%token <val> ID
%token FILETYPE SEMICOLON
%token SKIP FILETYPE STAGE PIPELINE CALL VOLATILE SWEEP SPLIT 
%token USING SELF RETURN EQUALS LPAREN RPAREN LBRACE RBRACE 
%token LBRACKET RBRACKET SEMICOLON COMMA DOT IN OUT SRC PY 
%token INT STRING FLOAT PATH FILE BOOL TRUE FALSE NULL DEFAULT 
%token LITSTRING NUM_FLOAT NUM_INT INVALID
%%
file
	: dec_list
		{{ ast = File{ decList: $1 } }}
		/*
	| call_stm
		{{ ast = File{ decList: []*Dec{} } }}
		*/
	;

dec_list
	: dec_list dec
		{{ $$ = append($1, $2) }}
	| dec
		{{ $$ = []Node{$1} }}
	;

dec
	: FILETYPE file_id SEMICOLON
		{{ $$ = NewFileTypeDec($2) }}
	| STAGE ID LPAREN param_list RPAREN 
		{{ $$ = NewStageDec($2) }}
	| STAGE ID LPAREN param_list RPAREN split_exp
		{{ $$ = NewStageDec($2) }}
	| PIPELINE ID LPAREN param_list RPAREN LBRACE call_stm_list return_stm RBRACE
		{{ $$ = NewPipelineDec($2) }}
	;

file_id
	: ID DOT ID
		{{ $$ = $1 + "." + $3 }}
	| ID
		{{ $$ = $1 }}
	;

param_list
	: param_list param
		{{ $$ = nil }}
	| param
		{{ $$ = nil }}
	;

param
	: IN type ID help
		{{ $$ = nil }}
	| OUT type help 
		{{ $$ = nil }}
	| OUT type ID help 
		{{ $$ = nil }}
	| SRC src_lang LITSTRING COMMA
		{{ $$ = nil }}
	;
help
    : LITSTRING COMMA
		{{ $$ = nil }}
    | COMMA
		{{ $$ = nil }}
    ;

type
    : INT
		{{ $$ = nil }}
    | STRING
		{{ $$ = nil }}
    | PATH
		{{ $$ = nil }}
    | FLOAT
		{{ $$ = nil }}
    | BOOL
		{{ $$ = nil }}
    | ID
		{{ $$ = nil }}
    | ID DOT ID
		{{ $$ = nil }}
    ;

src_lang
    : PY
		{{ $$ = nil }}
    ;

split_exp
	: SPLIT USING LPAREN param_list RPAREN
		{{ $$ = nil }}
	;

return_stm
    : RETURN LPAREN bind_stm_list RPAREN
		{{ $$ = nil }}
    ;

call_stm_list
    : call_stm_list call_stm
		{{ $$ = nil }}
    | call_stm
		{{ $$ = nil }}
    ;

call_stm
    : CALL ID LPAREN bind_stm_list RPAREN
		{{ $$ = nil }}
    | CALL VOLATILE ID LPAREN bind_stm_list RPAREN
		{{ $$ = nil }}
    ;

bind_stm_list
    : bind_stm_list bind_stm
		{{ $$ = nil }}
    | bind_stm
		{{ $$ = nil }}
    ;

bind_stm
    : ID EQUALS exp COMMA
		{{ $$ = nil }}
    | ID EQUALS SWEEP LPAREN exp RPAREN COMMA
		{{ $$ = nil }}
    ;

exp_list
    : exp_list COMMA exp
		{{ $$ = nil }}
    | exp
		{{ $$ = nil }}
    ; 

exp
    : LBRACKET exp_list RBRACKET
		{{ $$ = nil }}
    | LBRACKET RBRACKET
		{{ $$ = nil }}
    | PATH LPAREN LITSTRING RPAREN
		{{ $$ = nil }}
    | FILE LPAREN LITSTRING RPAREN
		{{ $$ = nil }}
    | NUM_FLOAT
		{{ $$ = nil }}
    | NUM_INT
		{{ $$ = nil }}
    | LITSTRING
		{{ $$ = nil }}
    | TRUE
		{{ $$ = nil }}
    | FALSE
		{{ $$ = nil }}
    | NULL
		{{ $$ = nil }}
    | value_exp
		{{ $$ = nil }}
    ;

value_exp
    : ID DOT ID
		{{ $$ = nil }}
    | ID
		{{ $$ = nil }}
    | SELF DOT ID
		{{ $$ = nil }}
    ;
%%

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
	data, _ := ioutil.ReadFile("stages.mro")
	MarioParse(&MarioLex{
		source: string(data),
		pos:    0,
		lineno: 1,
	})
	fmt.Println(len(ast.decList))
	for _, dec := range ast.decList {
		{
			v, ok := dec.(*FileTypeDec)
			if ok {
				fmt.Println(v.filetype)
			}
		}
		{
			v, ok := dec.(*StageDec)
			if ok {
				fmt.Println(v.name)
			}
		}
		{
			v, ok := dec.(*PipelineDec)
			if ok {
				fmt.Println(v.name)
			}
		}	}
}

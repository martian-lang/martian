%{

package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type (
	Node struct {
		lineno int
	}

	Dec interface {
		dec()
	}

	FileTypeDec struct {
		Node
		id string
	}

	StageDec struct {
		Node
		id       string
		params   []Param
		splitter []Param
	}

	PipelineDec struct {
		Node
		id     string
		params []Param
		calls  []*CallStm
		ret    *ReturnStm
	}

	Param interface {
		param()
	}

	InParam struct {
		Node
		tname string
		id    string
		help  string
	}

	OutParam struct {
		Node
		tname string
		id    string
		help  string
	}

	SourceParam struct {
		Node
		lang string
		path string
	}

	Stm interface {
		stm()
	}

	BindStm struct {
		Node
		id     string
		exp    Exp
		sweep  bool
	}

	CallStm struct {
		Node
		volatile bool
		id       string
		bindings []*BindStm
	}

	ReturnStm struct {
		Node
		bindings []*BindStm
	}

	Exp interface {
		exp()
	}

	ValExp struct {
		Node
		// union-style multi-value store
		kind string
		fval float64
		ival int64
		sval string
		bval bool
		null bool
	}

	RefExp struct {
		Node
		kind     string
		id       string
		outputId string
	}

	Ptree struct {
		Decs []Dec
		call *CallStm
	}
)
// Interface whitelist for Dec, Param, Exp, and Stm implementors. 
// Patterned after code in Go's ast.go.
func (*FileTypeDec) dec()   {}
func (*StageDec) dec()      {}
func (*PipelineDec) dec()   {}
func (*InParam) param()     {}
func (*OutParam) param()    {}
func (*SourceParam) param() {}
func (*ValExp) exp()        {}
func (*RefExp) exp()        {}
func (*BindStm) stm()       {}
func (*CallStm) stm()       {}
func (*ReturnStm) stm()     {}

// This global is where we build the AST. It will get passed out
// by the main parsing function.
var ptree Ptree
%}

%union{
	lineno   int
	val      string
	dec      Dec
	decs     []Dec
	param    Param
	params   []Param
	exp      Exp
	exps     []Exp
	stm      Stm
	stms     []Stm
	call     *CallStm
	calls    []*CallStm
	binding  *BindStm
	bindings []*BindStm
	retstm   *ReturnStm
}

%type <val>      file_id type help type src_lang
%type <dec>      dec 
%type <decs>     dec_list 
%type <param>    param
%type <params>   param_list split_exp
%type <exp>      exp ref_exp
%type <exps>     exp_list
%type <call>     call_stm 
%type <calls>    call_stm_list 
%type <binding>  bind_stm
%type <bindings> bind_stm_list
%type <retstm>   return_stm

%token SKIP INVALID 
%token SEMICOLON LBRACKET RBRACKET LPAREN RPAREN LBRACE RBRACE COMMA EQUALS
%token FILETYPE STAGE PIPELINE CALL VOLATILE SWEEP SPLIT USING SELF RETURN
%token IN OUT SRC
%token <val> ID LITSTRING NUM_FLOAT NUM_INT DOT
%token <val> PY GO SH EXEC
%token <val> INT STRING FLOAT PATH FILE BOOL TRUE FALSE NULL DEFAULT
%%
file
	: dec_list
		{{ ptree = Ptree{$1, nil} }}
	| call_stm
		{{ ptree = Ptree{nil, $1} }}
	;

dec_list
	: dec_list dec
		{{ $$ = append($1, $2) }}
	| dec
		{{ $$ = []Dec{$1} }}
	;

dec
	: FILETYPE file_id SEMICOLON
		{{ $$ = &FileTypeDec{Node{mmlval.lineno}, $2} }}
	| STAGE ID LPAREN param_list RPAREN 
		{{ $$ = &StageDec{Node{mmlval.lineno}, $2, $4, nil} }}
	| STAGE ID LPAREN param_list RPAREN split_exp
		{{ $$ = &StageDec{Node{mmlval.lineno}, $2, $4, $6} }}
	| PIPELINE ID LPAREN param_list RPAREN LBRACE call_stm_list return_stm RBRACE
		{{ $$ = &PipelineDec{Node{mmlval.lineno}, $2, $4, $7, $8} }}
	;

file_id
	: ID DOT ID
		{{ $$ = $1 + $2 + $3 }}
	| ID
	;

param_list
	: param_list param
		{{ $$ = append($1, $2) }}
	| param
		{{ $$ = []Param{$1} }}
	;

param
	: IN type ID help
		{{ $$ = &InParam{Node{mmlval.lineno}, $2, $3, $4} }}
	| OUT type help 
		{{ $$ = &OutParam{Node{mmlval.lineno}, $2, "default", $3} }}
	| OUT type ID help 
		{{ $$ = &OutParam{Node{mmlval.lineno}, $2, $3, $4} }}
	| SRC src_lang LITSTRING COMMA
		{{ $$ = &SourceParam{Node{mmlval.lineno}, $2, $3} }}
	;
help
    : LITSTRING COMMA
    	{{ $$ = $1 }}
    | COMMA
    	{{ $$ = "" }}
    ;

type
    : INT
    | STRING
    | PATH
    | FLOAT
    | BOOL
    | ID
    | ID DOT ID
		{{ $$ = $1 + "." + $3 }}
    ;

src_lang
    : PY
    //| GO
    //| SH
    //| EXEC
    ;

split_exp
	: SPLIT USING LPAREN param_list RPAREN
		{{ $$ = $4 }}
	;

return_stm
    : RETURN LPAREN bind_stm_list RPAREN
		{{ $$ = &ReturnStm{Node{mmlval.lineno}, $3 } }}
    ;

call_stm_list
    : call_stm_list call_stm
		{{ $$ = append($1, $2) }}
    | call_stm
		{{ $$ = []*CallStm{$1} }}
    ;

call_stm
    : CALL ID LPAREN bind_stm_list RPAREN
		{{ $$ = &CallStm{Node{mmlval.lineno}, false, $2, $4 } }}
    | CALL VOLATILE ID LPAREN bind_stm_list RPAREN
		{{ $$ = &CallStm{Node{mmlval.lineno}, true, $3, $5 } }}
    ;

bind_stm_list
    : bind_stm_list bind_stm
		{{ $$ = append($1, $2) }}
    | bind_stm
		{{ $$ = []*BindStm{$1} }}
    ;

bind_stm
    : ID EQUALS exp COMMA
		{{ $$ = &BindStm{Node{mmlval.lineno}, $1, $3, false} }}
    | ID EQUALS SWEEP LPAREN exp RPAREN COMMA
		{{ $$ = &BindStm{Node{mmlval.lineno}, $1, $5, true} }}
    ;

exp_list
    : exp_list COMMA exp
		{{ $$ = append($1, $3) }}
    | exp
		{{ $$ = []Exp{$1} }}
    ; 

exp
    : LBRACKET exp_list RBRACKET
		{{ $$ = nil }}
    | LBRACKET RBRACKET
		{{ $$ = nil }}
    | PATH LPAREN LITSTRING RPAREN
		{{ $$ = &ValExp{Node:Node{mmlval.lineno}, kind: $1, sval: strings.Replace($3, "\"", "", -1) } }}
    | FILE LPAREN LITSTRING RPAREN
		{{ $$ = &ValExp{Node:Node{mmlval.lineno}, kind: $1, sval: strings.Replace($3, "\"", "", -1) } }}
    | NUM_FLOAT
		{{  // Lexer guarantees parseable float strings.
			f, _ := strconv.ParseFloat($1, 64)
			$$ = &ValExp{Node:Node{mmlval.lineno}, kind: "float", fval: f } 
		}}
    | NUM_INT
		{{  // Lexer guarantees parseable int strings.
			i, _ := strconv.ParseInt($1, 0, 64)
			$$ = &ValExp{Node:Node{mmlval.lineno}, kind: "int", ival: i } 
		}}
    | LITSTRING
		{{ $$ = &ValExp{Node:Node{mmlval.lineno}, kind: "string", sval: strings.Replace($1, "\"", "", -1)} }}
    | TRUE
		{{ $$ = &ValExp{Node:Node{mmlval.lineno}, kind: "bool", bval: true} }}
    | FALSE
		{{ $$ = &ValExp{Node:Node{mmlval.lineno}, kind: "bool", bval: false} }}
    | NULL
		{{ $$ = &ValExp{Node:Node{mmlval.lineno}, kind: "null", null: true} }}
    | ref_exp
    	{{ $$ = $1 }}
    ;

ref_exp
    : ID DOT ID
		{{ $$ = &RefExp{Node{mmlval.lineno}, "call", $1, $3} }}
    | ID
		{{ $$ = &RefExp{Node{mmlval.lineno}, "call", $1, "default"} }}
    | SELF DOT ID
		{{ $$ = &RefExp{Node{mmlval.lineno}, "self", $3, ""} }}
    ;
%%

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
	NewRule("\\s+", 		SKIP),  	// whitespace 
	NewRule("#.*\\n", 		SKIP),		// Python-style comments
	NewRule("=", 			EQUALS),
	NewRule("\\(", 			LPAREN),
	NewRule("\\)", 			RPAREN),
	NewRule("{", 			LBRACE),
	NewRule("}", 			RBRACE),
	NewRule("\\[", 			LBRACKET),
	NewRule("\\]", 			RBRACKET),
	NewRule(";", 			SEMICOLON),
	NewRule(",", 			COMMA),
	NewRule("\\.", 			DOT),
	NewRule("\"[^\\\"]*\"", LITSTRING),	// double-quoted strings. escapes not supported
	NewRule("filetype\\b", 	FILETYPE),
	NewRule("stage\\b", 	STAGE),
	NewRule("pipeline\\b", 	PIPELINE),
	NewRule("call\\b",		CALL),
	NewRule("volatile\\b", 	VOLATILE),
	NewRule("sweep\\b", 	SWEEP),
	NewRule("split\\b", 	SPLIT),
	NewRule("using\\b", 	USING),
	NewRule("self\\b", 		SELF),
	NewRule("return\\b", 	RETURN),
	NewRule("in\\b", 		IN),
	NewRule("out\\b", 		OUT),
	NewRule("src\\b", 		SRC),
	NewRule("py\\b", 		PY),
	NewRule("go\\b", 		GO),
	NewRule("sh\\b", 		SH),
	NewRule("exec\\b", 		EXEC),
	NewRule("int\\b", 		INT),
	NewRule("string\\b", 	STRING),
	NewRule("float\\b", 	FLOAT),
	NewRule("path\\b", 		PATH),
	NewRule("file\\b", 		FILE),
	NewRule("bool\\b", 		BOOL),
	NewRule("true\\b", 		TRUE),
	NewRule("false\\b", 	FALSE),
	NewRule("null\\b", 		NULL),
	NewRule("default\\b", 	DEFAULT),
	NewRule("[a-zA-Z_][a-zA-z0-9_]*", ID),
	NewRule("-?[0-9]+\\.[0-9]+([eE][-+]?[0-9]+)?\\b", NUM_FLOAT), // support exponential
	NewRule("-?[0-9]+\\b", 	NUM_INT),
	NewRule(".", 			INVALID),
}

type SyntaxError struct {
	Lineno int
	Line string
	Token string
	Err error
}
func (self *SyntaxError) Error() string {
	return fmt.Sprintf("MRO syntax error: unexpected token '%s' on line %d:\n\n%s", self.Token, self.Lineno, self.Line)
}

type mmLex struct {
	source  string  	 // All the data we're scanning
	pos     int     	 // Position of the scan head
	lineno  int     	 // Keep track of the line number
	last    string  	 // Cache the last token for error messaging
	err     *SyntaxError // Constructed syntax error object
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

		// If it was whitespace or a comment, advance the line counter
		// by counting newlines.
		if rule.token == SKIP {
			self.lineno += strings.Count(val, "\n")
			continue
		}

		// If we got a parseable token, pass it and the line number
		// to the parser.
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
		Line: self.source[spos:epos], 
		Token: self.last,
	}
}

func Parse(src string) (*Ptree, error) {
	lex := mmLex{src, 0, 1, "", nil}
	if mmParse(&lex) == 0 {
		return &ptree, nil
	}
	return nil, lex.err
}

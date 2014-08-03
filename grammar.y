%{
package main

import (
	"strconv"
	"strings"
)
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
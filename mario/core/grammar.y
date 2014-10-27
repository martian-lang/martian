%{
//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO grammar.
//
package core

import (
    "strconv"
    "strings"
)

func unquote(qs string) string {
    return strings.Replace(qs, "\"", "", -1)
}
%}

%union{
    global    *Ast
    loc       int
    val       string
    comments  string
    dec       Dec
    decs      []Dec
    inparam   *InParam
    outparam  *OutParam
    params    *Params
    src       *SrcParam
    exp       Exp
    exps      []Exp
    kvpairs   map[string]Exp
    call      *CallStm
    calls     []*CallStm
    binding   *BindStm
    bindings  *BindStms
    retstm    *ReturnStm
}

%type <val>       file_id type help type src_lang
%type <dec>       dec 
%type <decs>      dec_list
%type <inparam>   in_param
%type <outparam>  out_param
%type <params>    in_param_list out_param_list split_param_list
%type <src>       src_stm
%type <exp>       exp ref_exp
%type <exps>      exp_list
%type <kvpairs>   kvpair_list
%type <call>      call_stm 
%type <calls>     call_stm_list 
%type <binding>   bind_stm
%type <bindings>  bind_stm_list
%type <retstm>    return_stm

%token SKIP INVALID 
%token SEMICOLON COLON COMMA EQUALS
%token LBRACKET RBRACKET LPAREN RPAREN LBRACE RBRACE 
%token FILETYPE STAGE PIPELINE CALL VOLATILE SWEEP SPLIT USING SELF RETURN
%token IN OUT SRC
%token <val> ID LITSTRING NUM_FLOAT NUM_INT DOT
%token <val> PY GO SH EXEC
%token <val> MAP INT STRING FLOAT PATH FILE BOOL TRUE FALSE NULL DEFAULT

%%
file
    : dec_list
        {{ 
            global := NewAst($1, nil)
            mmlex.(*mmLexInfo).global = global
        }}
    | dec_list call_stm
        {{ 
            global := NewAst($1, $2)
            mmlex.(*mmLexInfo).global = global
        }}
    | call_stm
        {{
            global := NewAst([]Dec{}, $1)
            mmlex.(*mmLexInfo).global = global
        }}
    ;

dec_list
    : dec_list dec
        {{ $$ = append($1, $2) }}
    | dec
        {{ $$ = []Dec{$1} }}
    ;

dec
    : FILETYPE file_id SEMICOLON
        {{ $$ = &Filetype{NewAstNode(&mmlval), $2} }}
    | STAGE ID LPAREN in_param_list out_param_list src_stm RPAREN 
        {{ $$ = &Stage{NewAstNode(&mmlval), $2, $4, $5, $6, &Params{[]Param{}, map[string]Param{}} } }}
    | STAGE ID LPAREN in_param_list out_param_list src_stm RPAREN split_param_list
        {{ $$ = &Stage{NewAstNode(&mmlval), $2, $4, $5, $6, $8} }}
    | PIPELINE ID LPAREN in_param_list out_param_list RPAREN LBRACE call_stm_list return_stm RBRACE
        {{ $$ = &Pipeline{NewAstNode(&mmlval), $2, $4, $5, $8, &Callables{[]Callable{}, map[string]Callable{}}, $9} }}
    ;

file_id
    : ID DOT ID
        {{ $$ = $1 + $2 + $3 }}
    | ID
    ;

in_param_list
    : in_param_list in_param
        {{ 
            $1.list = append($1.list, $2)
            $$ = $1
        }}
    | in_param
        {{ $$ = &Params{[]Param{$1}, map[string]Param{}} }}
    ;

in_param
    : IN type ID help
        {{ $$ = &InParam{NewAstNode(&mmlval), $2, false, $3, unquote($4), false } }}
    | IN type LBRACKET RBRACKET ID help
        {{ $$ = &InParam{NewAstNode(&mmlval), $2, true, $5, unquote($6), false } }}
    ;

out_param_list
    : out_param_list out_param
        {{ 
            $1.list = append($1.list, $2)
            $$ = $1
        }}
    | out_param
        {{ $$ = &Params{[]Param{$1}, map[string]Param{}} }}
    ;

out_param
    : OUT type help 
        {{ $$ = &OutParam{NewAstNode(&mmlval), $2, false, "default", unquote($3), false } }}
    | OUT type ID help 
        {{ $$ = &OutParam{NewAstNode(&mmlval), $2, false, $3, unquote($4), false } }}
    | OUT type LBRACKET RBRACKET help 
        {{ $$ = &OutParam{NewAstNode(&mmlval), $2, true, "default", unquote($5), false } }}
    | OUT type LBRACKET RBRACKET ID help 
        {{ $$ = &OutParam{NewAstNode(&mmlval), $2, true, $5, unquote($6), false } }}    
    ;

src_stm
    : SRC src_lang LITSTRING COMMA
        {{ $$ = &SrcParam{NewAstNode(&mmlval), $2, unquote($3) } }}
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
    | MAP
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

split_param_list
    : SPLIT USING LPAREN in_param_list RPAREN
        {{ $$ = $4 }}
    ;

return_stm
    : RETURN LPAREN bind_stm_list RPAREN
        {{ $$ = &ReturnStm{NewAstNode(&mmlval), $3} }}
    ;

call_stm_list
    : call_stm_list call_stm
        {{ $$ = append($1, $2) }}
    | call_stm
        {{ $$ = []*CallStm{$1} }}
    ;

call_stm
    : CALL ID LPAREN bind_stm_list RPAREN
        {{ $$ = &CallStm{NewAstNode(&mmlval), false, $2, $4} }}
    | CALL VOLATILE ID LPAREN bind_stm_list RPAREN
        {{ $$ = &CallStm{NewAstNode(&mmlval), true, $3, $5} }}
    ;

bind_stm_list
    : bind_stm_list bind_stm
        {{ 
            $1.list = append($1.list, $2)
            $$ = $1
        }}
    | bind_stm
        {{ $$ = &BindStms{[]*BindStm{$1}, map[string]*BindStm{} } }}
    ;

bind_stm
    : ID EQUALS exp COMMA
        {{ $$ = &BindStm{NewAstNode(&mmlval), $1, $3, false, ""} }}
    | ID EQUALS SWEEP LPAREN exp_list RPAREN COMMA
        {{ $$ = &BindStm{NewAstNode(&mmlval), $1, &ValExp{node:NewAstNode(&mmlval), kind: "array", value: $5}, true, ""} }}
    ;

exp_list
    : exp_list COMMA exp
        {{ $$ = append($1, $3) }}
    | exp
        {{ $$ = []Exp{$1} }}
    ; 

kvpair_list
    : kvpair_list COMMA LITSTRING COLON exp
        {{ 
            $1[unquote($3)] = $5
            $$ = $1
        }}
    | LITSTRING COLON exp
        {{ $$ = map[string]Exp{unquote($1): $3} }}
    ;

exp
    : LBRACKET exp_list RBRACKET        
        {{ $$ = &ValExp{node:NewAstNode(&mmlval), kind: "array", value: $2} }}
    | LBRACKET RBRACKET
        {{ $$ = &ValExp{node:NewAstNode(&mmlval), kind: "array", value: []Exp{}} }}
    | LBRACE RBRACE
        {{ $$ = &ValExp{node:NewAstNode(&mmlval), kind: "map", value: map[string]interface{}{}} }}
    | LBRACE kvpair_list RBRACE
        {{ $$ = &ValExp{node:NewAstNode(&mmlval), kind: "map", value: $2} }}
    | PATH LPAREN LITSTRING RPAREN
        {{ $$ = &ValExp{node:NewAstNode(&mmlval), kind: $1, value: unquote($3)} }}
    | FILE LPAREN LITSTRING RPAREN
        {{ $$ = &ValExp{node:NewAstNode(&mmlval), kind: $1, value: unquote($3)} }}
    | NUM_FLOAT
        {{  // Lexer guarantees parseable float strings.
            f, _ := strconv.ParseFloat($1, 64)
            $$ = &ValExp{node:NewAstNode(&mmlval), kind: "float", value: f } 
        }}
    | NUM_INT
        {{  // Lexer guarantees parseable int strings.
            i, _ := strconv.ParseInt($1, 0, 64)
            $$ = &ValExp{node:NewAstNode(&mmlval), kind: "int", value: i } 
        }}
    | LITSTRING
        {{ $$ = &ValExp{node:NewAstNode(&mmlval), kind: "string", value: unquote($1)} }}
    | TRUE
        {{ $$ = &ValExp{node:NewAstNode(&mmlval), kind: "bool", value: true} }}
    | FALSE
        {{ $$ = &ValExp{node:NewAstNode(&mmlval), kind: "bool", value: false} }}
    | NULL
        {{ $$ = &ValExp{node:NewAstNode(&mmlval), kind: "null", value: nil} }}
    | ref_exp
        {{ $$ = $1 }}
    ;

ref_exp
    : ID DOT ID
        {{ $$ = &RefExp{NewAstNode(&mmlval), "call", $1, $3} }}
    | ID
        {{ $$ = &RefExp{NewAstNode(&mmlval), "call", $1, "default"} }}
    | SELF DOT ID
        {{ $$ = &RefExp{NewAstNode(&mmlval), "self", $3, ""} }}
    ;
%%
%{
//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Margo
//
package main

import (
    "fmt"
    "strconv"
    "strings"
)
%}

%union{
    global    *Ast
    loc       int
    val       string
    dec       Dec
    decs      []Dec
    inparam   *InParam
    outparam  *OutParam
    params    *Params
    src       *SrcParam
    exp       Exp
    exps      []Exp
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
%type <call>      call_stm 
%type <calls>     call_stm_list 
%type <binding>   bind_stm
%type <bindings>  bind_stm_list
%type <retstm>    return_stm

%token SKIP INVALID 
%token SEMICOLON LBRACKET RBRACKET LPAREN RPAREN LBRACE RBRACE COMMA EQUALS
%token FILETYPE STAGE PIPELINE CALL VOLATILE SWEEP SPLIT USING SELF RETURN
%token IN OUT SRC
%token <val> ID LITSTRING NUM_FLOAT NUM_INT DOT
%token <val> PY GO SH EXEC
%token <val> INT TSTRING FLOAT PATH FILE BOOL TRUE FALSE NULL DEFAULT

%%
file
    : dec_list
        {{ 
            fmt.Print()
            global := Ast{[]FileLoc{}, map[string]bool{}, []*Filetype{}, map[string]bool{}, []*Stage{}, []*Pipeline{}, &Callables{[]Callable{}, map[string]Callable{}}, nil}
            for _, dec := range $1 {
                switch dec := dec.(type) {
                case *Filetype:
                    global.filetypes      = append(global.filetypes, dec)
                case *Stage:
                    global.stages         = append(global.stages, dec)
                    global.callables.list = append(global.callables.list, dec)
                case *Pipeline:
                    global.pipelines      = append(global.pipelines, dec)
                    global.callables.list = append(global.callables.list, dec)
                }
            }
            mmlex.(*mmLexInfo).global = &global
        }}
    | call_stm
        {{ 
            global := Ast{[]FileLoc{}, map[string]bool{}, []*Filetype{}, map[string]bool{}, []*Stage{}, []*Pipeline{}, &Callables{[]Callable{}, map[string]Callable{}}, $1} 
            mmlex.(*mmLexInfo).global = &global
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
        {{ $$ = &Filetype{AstNode{mmlval.loc}, $2} }}
    | STAGE ID LPAREN in_param_list out_param_list src_stm RPAREN 
        {{ $$ = &Stage{AstNode{mmlval.loc}, $2, $4, $5, $6, &Params{[]Param{}, map[string]Param{}} } }}
    | STAGE ID LPAREN in_param_list out_param_list src_stm RPAREN split_param_list
        {{ $$ = &Stage{AstNode{mmlval.loc}, $2, $4, $5, $6, $8} }}
    | PIPELINE ID LPAREN in_param_list out_param_list RPAREN LBRACE call_stm_list return_stm RBRACE
        {{ $$ = &Pipeline{AstNode{mmlval.loc}, $2, $4, $5, $8, &Callables{[]Callable{}, map[string]Callable{}}, $9} }}
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
        {{ $$ = &InParam{AstNode{mmlval.loc}, $2, $3, $4, false } }}
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
        {{ $$ = &OutParam{AstNode{mmlval.loc}, $2, "default", $3, false } }}
    | OUT type ID help 
        {{ $$ = &OutParam{AstNode{mmlval.loc}, $2, $3, $4, false } }}
    ;

src_stm
    : SRC src_lang LITSTRING COMMA
        {{ $$ = &SrcParam{AstNode{mmlval.loc}, $2, strings.Replace($3, "\"", "", -1) } }}
    ;

help
    : LITSTRING COMMA
        {{ $$ = $1 }}
    | COMMA
        {{ $$ = "" }}
    ;

type
    : INT
    | TSTRING
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

split_param_list
    : SPLIT USING LPAREN in_param_list RPAREN
        {{ $$ = $4 }}
    ;

return_stm
    : RETURN LPAREN bind_stm_list RPAREN
        {{ $$ = &ReturnStm{AstNode{mmlval.loc}, $3} }}
    ;

call_stm_list
    : call_stm_list call_stm
        {{ $$ = append($1, $2) }}
    | call_stm
        {{ $$ = []*CallStm{$1} }}
    ;

call_stm
    : CALL ID LPAREN bind_stm_list RPAREN
        {{ $$ = &CallStm{AstNode{mmlval.loc}, false, $2, $4} }}
    | CALL VOLATILE ID LPAREN bind_stm_list RPAREN
        {{ $$ = &CallStm{AstNode{mmlval.loc}, true, $3, $5} }}
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
        {{ $$ = &BindStm{AstNode{mmlval.loc}, $1, $3, false, ""} }}
    | ID EQUALS SWEEP LPAREN exp RPAREN COMMA
        {{ $$ = &BindStm{AstNode{mmlval.loc}, $1, $5, true, ""} }}
    ;

exp_list
    : exp_list COMMA exp
        {{ $$ = append($1, $3) }}
    | exp
        {{ $$ = []Exp{$1} }}
    ; 

exp
    : LBRACKET exp_list RBRACKET        
        {{ $$ = &ValExp{node:AstNode{mmlval.loc}, kind: "array", value: $2} }}
    | LBRACKET RBRACKET
        {{ $$ = &ValExp{node:AstNode{mmlval.loc}, kind: "array", value: []Exp{}} }}
    | PATH LPAREN LITSTRING RPAREN
        {{ $$ = &ValExp{node:AstNode{mmlval.loc}, kind: $1, value: strings.Replace($3, "\"", "", -1) } }}
    | FILE LPAREN LITSTRING RPAREN
        {{ $$ = &ValExp{node:AstNode{mmlval.loc}, kind: $1, value: strings.Replace($3, "\"", "", -1) } }}
    | NUM_FLOAT
        {{  // Lexer guarantees parseable float strings.
            f, _ := strconv.ParseFloat($1, 64)
            $$ = &ValExp{node:AstNode{mmlval.loc}, kind: "float", value: f } 
        }}
    | NUM_INT
        {{  // Lexer guarantees parseable int strings.
            i, _ := strconv.ParseInt($1, 0, 64)
            $$ = &ValExp{node:AstNode{mmlval.loc}, kind: "int", value: i } 
        }}
    | LITSTRING
        {{ $$ = &ValExp{node:AstNode{mmlval.loc}, kind: "string", value: strings.Replace($1, "\"", "", -1)} }}
    | TRUE
        {{ $$ = &ValExp{node:AstNode{mmlval.loc}, kind: "bool", value: true} }}
    | FALSE
        {{ $$ = &ValExp{node:AstNode{mmlval.loc}, kind: "bool", value: false} }}
    | NULL
        {{ $$ = &ValExp{node:AstNode{mmlval.loc}, kind: "null", value: nil} }}
    | ref_exp
        {{ $$ = $1 }}
    ;

ref_exp
    : ID DOT ID
        {{ $$ = &RefExp{AstNode{mmlval.loc}, "call", $1, $3} }}
    | ID
        {{ $$ = &RefExp{AstNode{mmlval.loc}, "call", $1, "default"} }}
    | SELF DOT ID
        {{ $$ = &RefExp{AstNode{mmlval.loc}, "self", $3, ""} }}
    ;
%%
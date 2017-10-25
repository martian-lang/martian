%{
//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO grammar.
//

package syntax

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
    locmap    []FileLoc
    arr       int
    loc       int
    val       string
    modifiers *Modifiers
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
    pre_dir   []*preprocessorDirective
}

%type <pre_dir>   preprocess_directives
%type <val>       id_list type help type src_lang type outname
%type <modifiers> modifiers
%type <arr>       arr_list
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

%token SKIP COMMENT INVALID
%token SEMICOLON COLON COMMA EQUALS
%token LBRACKET RBRACKET LPAREN RPAREN LBRACE RBRACE
%token FILETYPE STAGE PIPELINE CALL LOCAL PREFLIGHT VOLATILE SWEEP SPLIT USING SELF RETURN
%token IN OUT SRC
%token <val> ID LITSTRING NUM_FLOAT NUM_INT DOT
%token <val> PY GO SH EXEC COMPILED
%token <val> MAP INT STRING FLOAT PATH BOOL TRUE FALSE NULL DEFAULT
%token <val> PREPROCESS_DIRECTIVE

%%
file
    : preprocess_directives dec_list
        {{
            global := NewAst($2, nil)
            global.preprocess = $1
            mmlex.(*mmLexInfo).global = global
        }}
    | preprocess_directives dec_list call_stm
        {{
            global := NewAst($2, $3)
            global.preprocess = $1
            mmlex.(*mmLexInfo).global = global
        }}
    | preprocess_directives call_stm
        {{
            global := NewAst([]Dec{}, $2)
            global.preprocess = $1
            mmlex.(*mmLexInfo).global = global
        }}
    | dec_list
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

preprocess_directives
    : preprocess_directives PREPROCESS_DIRECTIVE
        {{ $$ = append($1, &preprocessorDirective{NewAstNode($<loc>2, $<locmap>2), $2}) }}
    | PREPROCESS_DIRECTIVE
        {{ $$ = []*preprocessorDirective{
              &preprocessorDirective{
                  Node: NewAstNode($<loc>1, $<locmap>1),
                  Value: $1,
              },
           }
        }}

dec_list
    : dec_list dec
        {{ $$ = append($1, $2) }}
    | dec
        {{ $$ = []Dec{$1} }}
    ;

dec
    : FILETYPE id_list SEMICOLON
        {{ $$ = &UserType{NewAstNode($<loc>2, $<locmap>2), $2} }}
    | STAGE ID LPAREN in_param_list out_param_list src_stm RPAREN
        {{ $$ = &Stage{NewAstNode($<loc>2, $<locmap>2), $2, $4, $5, $6, &Params{[]Param{}, map[string]Param{}}, false} }}
    | STAGE ID LPAREN in_param_list out_param_list src_stm RPAREN split_param_list
        {{ $$ = &Stage{NewAstNode($<loc>2, $<locmap>2), $2, $4, $5, $6, $8, true} }}
    | PIPELINE ID LPAREN in_param_list out_param_list RPAREN LBRACE call_stm_list return_stm RBRACE
        {{ $$ = &Pipeline{NewAstNode($<loc>2, $<locmap>2), $2, $4, $5, $8, &Callables{[]Callable{}, map[string]Callable{}}, $9} }}
    ;

id_list
    : id_list DOT ID
        {{ $$ = $1 + $2 + $3 }}
    | ID
    ;

arr_list
    :
        {{ $$ = 0 }}
    | arr_list LBRACKET RBRACKET
        {{ $$ += 1 }}
    ;

in_param_list
    :
        {{ $$ = &Params{[]Param{}, map[string]Param{}} }}
    | in_param_list in_param
        {{
            $1.List = append($1.List, $2)
            $$ = $1
        }}
    ;

in_param
    : IN type arr_list ID help COMMA
        {{ $$ = &InParam{NewAstNode($<loc>1, $<locmap>1), $2, $3, $4, unquote($5), false } }}
    | IN type arr_list ID COMMA
        {{ $$ = &InParam{NewAstNode($<loc>1, $<locmap>1), $2, $3, $4, "", false } }}
    ;

out_param_list
    :
        {{ $$ = &Params{[]Param{}, map[string]Param{}} }}
    | out_param_list out_param
        {{
            $1.List = append($1.List, $2)
            $$ = $1
        }}
    ;

out_param
    : OUT type arr_list COMMA
        {{ $$ = &OutParam{NewAstNode($<loc>1, $<locmap>1), $2, $3, "default", "", "", false } }}
    | OUT type arr_list help COMMA
        {{ $$ = &OutParam{NewAstNode($<loc>1, $<locmap>1), $2, $3, "default", unquote($4), "", false } }}
    | OUT type arr_list help outname COMMA
        {{ $$ = &OutParam{NewAstNode($<loc>1, $<locmap>1), $2, $3, "default", unquote($4), unquote($5), false } }}
    | OUT type arr_list ID COMMA
        {{ $$ = &OutParam{NewAstNode($<loc>1, $<locmap>1), $2, $3, $4, "", "", false } }}
    | OUT type arr_list ID help COMMA
        {{ $$ = &OutParam{NewAstNode($<loc>1, $<locmap>1), $2, $3, $4, unquote($5), "", false } }}
    | OUT type arr_list ID help outname COMMA
        {{ $$ = &OutParam{NewAstNode($<loc>1, $<locmap>1), $2, $3, $4, unquote($5), unquote($6), false } }}
    ;

src_stm
    : SRC src_lang LITSTRING COMMA
        {{ stagecodeParts := strings.Split(unquote($3), " ")
	   $$ = &SrcParam{NewAstNode($<loc>1, $<locmap>1), StageLanguage($2), stagecodeParts[0], stagecodeParts[1:]} }}
    ;

help
    : LITSTRING
        {{ $$ = $1 }}
    ;

outname
    : LITSTRING
        {{ $$ = $1 }}
    ;

type
    : INT
    | STRING
    | PATH
    | FLOAT
    | BOOL
    | MAP
    | id_list
    ;

src_lang
    : PY
    | EXEC
    | COMPILED
    //| GO
    //| SH
    ;

split_param_list
    : SPLIT USING LPAREN in_param_list RPAREN
        {{ $$ = $4 }}
    ;

return_stm
    : RETURN LPAREN bind_stm_list RPAREN
        {{ $$ = &ReturnStm{NewAstNode($<loc>1, $<locmap>1), $3} }}
    ;

call_stm_list
    : call_stm_list call_stm
        {{ $$ = append($1, $2) }}
    | call_stm
        {{ $$ = []*CallStm{$1} }}
    ;

call_stm
    : CALL modifiers ID LPAREN bind_stm_list RPAREN
        {{ $$ = &CallStm{NewAstNode($<loc>1, $<locmap>1), $2, $3, $5} }}
    ;

modifiers
    :
      {{ $$ = &Modifiers{false, false, false} }}
    | modifiers LOCAL
      {{ $$.Local = true }}
    | modifiers PREFLIGHT
      {{ $$.Preflight = true }}
    | modifiers VOLATILE
      {{ $$.Volatile = true }}
    ;

bind_stm_list
    :
        {{ $$ = &BindStms{NewAstNode($<loc>0, $<locmap>0), []*BindStm{}, map[string]*BindStm{}} }}
    | bind_stm_list bind_stm
        {{
            $1.List = append($1.List, $2)
            $$ = $1
        }}
    ;

bind_stm
    : ID EQUALS exp COMMA
        {{ $$ = &BindStm{NewAstNode($<loc>1, $<locmap>1), $1, $3, false, ""} }}
    | ID EQUALS SWEEP LPAREN exp_list RPAREN COMMA
        {{ $$ = &BindStm{NewAstNode($<loc>1, $<locmap>1), $1, &ValExp{Node:NewAstNode($<loc>1, $<locmap>1), Kind: "array", Value: $5}, true, ""} }}
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
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<locmap>1), Kind: "array", Value: $2} }}
    | LBRACKET RBRACKET
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<locmap>1), Kind: "array", Value: []Exp{}} }}
    | LBRACE RBRACE
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<locmap>1), Kind: "map", Value: map[string]interface{}{}} }}
    | LBRACE kvpair_list RBRACE
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<locmap>1), Kind: "map", Value: $2} }}
    | NUM_FLOAT
        {{  // Lexer guarantees parseable float strings.
            f, _ := strconv.ParseFloat($1, 64)
            $$ = &ValExp{Node:NewAstNode($<loc>1, $<locmap>1), Kind: "float", Value: f }
        }}
    | NUM_INT
        {{  // Lexer guarantees parseable int strings.
            i, _ := strconv.ParseInt($1, 0, 64)
            $$ = &ValExp{Node:NewAstNode($<loc>1, $<locmap>1), Kind: "int", Value: i }
        }}
    | LITSTRING
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<locmap>1), Kind: "string", Value: unquote($1)} }}
    | TRUE
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<locmap>1), Kind: "bool", Value: true} }}
    | FALSE
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<locmap>1), Kind: "bool", Value: false} }}
    | NULL
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<locmap>1), Kind: "null", Value: nil} }}
    | ref_exp
        {{ $$ = $1 }}
    ;

ref_exp
    : ID DOT ID
        {{ $$ = &RefExp{NewAstNode($<loc>1, $<locmap>1), "call", $1, $3} }}
    | ID
        {{ $$ = &RefExp{NewAstNode($<loc>1, $<locmap>1), "call", $1, "default"} }}
    | SELF DOT ID
        {{ $$ = &RefExp{NewAstNode($<loc>1, $<locmap>1), "self", $3, ""} }}
    ;
%%

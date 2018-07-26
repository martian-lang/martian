%{
//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO grammar.
//

package syntax

import (
    "strings"
)

%}

%union{
    global    *Ast
    srcfile   *SourceFile
    arr       int16
    loc       int
    val       []byte
    modifiers *Modifiers
    dec       Dec
    decs      []Dec
    inparam   *InParam
    outparam  *OutParam
    retains   []*RetainParam
    stretains *RetainParams
    i_params  *InParams
    o_params  *OutParams
    res       *Resources
    par_tuple paramsTuple
    src       *SrcParam
    exp       Exp
    exps      []Exp
    rexp      *RefExp
    vexp      *ValExp
    kvpairs   map[string]Exp
    call      *CallStm
    calls     []*CallStm
    binding   *BindStm
    bindings  *BindStms
    retstm    *ReturnStm
    plretains *PipelineRetains
    reflist   []*RefExp
    includes  []*Include
    intern    *stringIntern
}

%type <includes>  includes
%type <val>       id id_list type help type src_lang type outname
%type <modifiers> modifiers
%type <arr>       arr_list
%type <dec>       dec stage pipeline
%type <decs>      dec_list
%type <inparam>   in_param
%type <outparam>  out_param
%type <retains>   stage_retain_list
%type <stretains> stage_retain
%type <reflist>   pipeline_retain_list
%type <plretains> pipeline_retain
%type <i_params>  in_param_list
%type <o_params>  out_param_list
%type <par_tuple> split_param_list
%type <src>       src_stm
%type <exp>       exp
%type <rexp>      ref_exp
%type <vexp>      val_exp bool_exp
%type <exps>      exp_list
%type <kvpairs>   kvpair_list
%type <call>      call_stm
%type <calls>     call_stm_list
%type <binding>   bind_stm modifier_stm
%type <bindings>  bind_stm_list modifier_stm_list
%type <retstm>    return_stm
%type <res>       resources resource_list

%token SKIP COMMENT INVALID
%token SEMICOLON COLON COMMA EQUALS
%token LBRACKET RBRACKET LPAREN RPAREN LBRACE RBRACE
%token SWEEP RETURN SELF
%token <val> FILETYPE STAGE PIPELINE CALL SPLIT USING RETAIN
%token <val> LOCAL PREFLIGHT VOLATILE DISABLED STRICT
%token IN OUT SRC AS
%token <val> THREADS MEM_GB SPECIAL
%token <val> ID LITSTRING NUM_FLOAT NUM_INT DOT
%token <val> PY EXEC COMPILED
%token <val> MAP INT STRING FLOAT PATH BOOL TRUE FALSE NULL DEFAULT
%token INCLUDE_DIRECTIVE

%%
file
    : includes dec_list
        {{
            global := NewAst($2, nil, $<srcfile>2)
            global.Includes = $1
            mmlex.(*mmLexInfo).global = global
        }}
    | includes dec_list call_stm
        {{
            global := NewAst($2, $3, $<srcfile>2)
            global.Includes = $1
            mmlex.(*mmLexInfo).global = global
        }}
    | includes call_stm
        {{
            global := NewAst(nil, $2, $<srcfile>2)
            global.Includes = $1
            mmlex.(*mmLexInfo).global = global
        }}
    | dec_list
        {{
            global := NewAst($1, nil, $<srcfile>1)
            mmlex.(*mmLexInfo).global = global
        }}
    | dec_list call_stm
        {{
            global := NewAst($1, $2, $<srcfile>1)
            mmlex.(*mmLexInfo).global = global
        }}
    | call_stm
        {{
            global := NewAst(nil, $1, $<srcfile>1)
            mmlex.(*mmLexInfo).global = global
        }}
    ;

includes
    : includes INCLUDE_DIRECTIVE LITSTRING
        {{ $$ = append($1, &Include{
            Node: NewAstNode($<loc>2, $<srcfile>2),
            Value: $<intern>3.unquote($3),
           })
        }}
    | INCLUDE_DIRECTIVE LITSTRING
        {{ $$ = []*Include{
              &Include{
                  Node: NewAstNode($<loc>1, $<srcfile>1),
                  Value: $<intern>2.unquote($2),
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
        {{ $$ = &UserType{
            Node: NewAstNode($<loc>2, $<srcfile>2),
            Id: $<intern>2.Get($2),
        } }}
    | stage
    | pipeline
    ;

pipeline
    : PIPELINE id LPAREN in_param_list out_param_list RPAREN LBRACE call_stm_list return_stm pipeline_retain RBRACE
        {{ $$ = &Pipeline{
            Node: NewAstNode($<loc>2, $<srcfile>2),
            Id: $<intern>2.Get($2),
            InParams: $4,
            OutParams: $5,
            Calls: $8,
            Callables: &Callables{Table: make(map[string]Callable)},
            Ret: $9,
            Retain: $10,
        } }}
    ;

stage
    : STAGE id LPAREN in_param_list out_param_list src_stm RPAREN split_param_list resources stage_retain
        {{ $$ = &Stage{
                Node: NewAstNode($<loc>2, $<srcfile>2),
                Id: $<intern>2.Get($2),
                InParams: $4,
                OutParams: $5,
                Src: $6,
                ChunkIns: $8.Ins,
                ChunkOuts: $8.Outs,
                Split: $8.Present,
                Resources: $9,
                Retain: $10,
           }
        }}
   ;

resources
    :
        {{ $$ = nil }}
    | USING LPAREN resource_list RPAREN
        {{
             $3.Node = NewAstNode($<loc>1, $<srcfile>1)
             $$ = $3
         }}
    ;

resource_list
    :
        {{ $$ = new(Resources) }}
    | resource_list THREADS EQUALS NUM_INT COMMA
        {{
            n := NewAstNode($<loc>2, $<srcfile>2)
            $1.ThreadNode = &n
            i := parseInt($4)
            $1.Threads = int16(i)
            $$ = $1
        }}
    | resource_list MEM_GB EQUALS NUM_INT COMMA
        {{
            n := NewAstNode($<loc>2, $<srcfile>2)
            $1.MemNode = &n
            i := parseInt($4)
            $1.MemGB = int16(i)
            $$ = $1
        }}
    | resource_list SPECIAL EQUALS LITSTRING COMMA
        {{
            n := NewAstNode($<loc>2, $<srcfile>2)
            $1.SpecialNode = &n
            $1.Special = $<intern>4.unquote($4)
            $$ = $1
        }}
    | resource_list VOLATILE EQUALS STRICT COMMA
        {{
            n := NewAstNode($<loc>2, $<srcfile>2)
            $1.VolatileNode = &n
            $1.StrictVolatile = true
            $$ = $1
        }}
    ;

stage_retain
    :
        {{ $$ = nil }}
    | RETAIN LPAREN stage_retain_list RPAREN
        {{
             $$ = &RetainParams{
                Node: NewAstNode($<loc>1, $<srcfile>1),
                Params: $3,
             }
         }}
    ;

stage_retain_list
    :
        {{ $$ = nil }}
    | stage_retain_list id COMMA
        {{
            $$ = append($1, &RetainParam{
                Node: NewAstNode($<loc>2, $<srcfile>2),
                Id: $<intern>2.Get($2),
            })
        }}
    ;


id_list
    : id_list DOT id
        {{
            idd := append($1, '.')
            $$ = append(idd, $3...)
        }}
    | id
        {{
            // set capacity == length so append doesn't overwrite
            // other parts of the buffer later.
            $$ = $1[:len($1):len($1)]
        }}
    ;

arr_list
    :
        {{ $$ = 0 }}
    | arr_list LBRACKET RBRACKET
        {{ $$++ }}
    ;

in_param_list
    :
        {{ $$ = &InParams{Table: make(map[string]*InParam)} }}
    | in_param_list in_param
        {{
            $1.List = append($1.List, $2)
            $$ = $1
        }}
    ;

in_param
    : IN type arr_list id help COMMA
        {{ $$ = &InParam{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Tname: $<intern>2.Get($2),
            ArrayDim: $3,
            Id: $<intern>4.Get($4),
            Help: unquote($5),
        } }}
    | IN type arr_list id COMMA
        {{ $$ = &InParam{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Tname: $<intern>2.Get($2),
            ArrayDim: $3,
            Id: $<intern>4.Get($4),
        } }}
    ;

out_param_list
    :
        {{ $$ = &OutParams{Table: make(map[string]*OutParam)} }}
    | out_param_list out_param
        {{
            $1.List = append($1.List, $2)
            $$ = $1
        }}
    ;

out_param
    : OUT type arr_list COMMA
        {{ $$ = &OutParam{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Tname: $<intern>2.Get($2),
            ArrayDim: $3,
            Id: default_out_name,
        } }}
    | OUT type arr_list help COMMA
        {{ $$ = &OutParam{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Tname: $<intern>2.Get($2),
            ArrayDim: $3,
            Id: default_out_name,
            Help: unquote($4),
        } }}
    | OUT type arr_list help outname COMMA
        {{ $$ = &OutParam{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Tname: $<intern>2.Get($2),
            ArrayDim: $3,
            Id: default_out_name,
            Help: unquote($4),
            OutName: $<intern>5.unquote($5),
        } }}
    | OUT type arr_list id COMMA
        {{ $$ = &OutParam{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Tname: $<intern>2.Get($2),
            ArrayDim: $3,
            Id: $<intern>4.Get($4),
        } }}
    | OUT type arr_list id help COMMA
        {{ $$ = &OutParam{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Tname: $<intern>2.Get($2),
            ArrayDim: $3,
            Id: $<intern>4.Get($4),
            Help: unquote($5),
        } }}
    | OUT type arr_list id help outname COMMA
        {{ $$ = &OutParam{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Tname: $<intern>2.Get($2),
            ArrayDim: $3,
            Id: $<intern>4.Get($4),
            Help: unquote($5),
            OutName: $<intern>6.unquote($6),
        } }}
    ;

src_stm
    : SRC src_lang LITSTRING COMMA
        {{ stagecodeParts := strings.Split($<intern>3.unquote($3), " ")
           $$ = &SrcParam{
               Node: NewAstNode($<loc>1, $<srcfile>1),
               Lang: StageLanguage($<intern>2.Get($2)),
               Path: stagecodeParts[0],
               Args: stagecodeParts[1:],
           } }}
    ;

help
    : LITSTRING
    ;

outname
    : LITSTRING
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
    ;

split_param_list
    :
        {{
            $$ = paramsTuple{
                Present: false,
                Ins: &InParams{Table: make(map[string]*InParam)},
                Outs: &OutParams{Table: make(map[string]*OutParam)},
            }
        }}
    | SPLIT USING LPAREN in_param_list out_param_list RPAREN
        {{ $$ = paramsTuple{
            Present: true,
            Ins: $4,
            Outs: $5,
        } }}
    | SPLIT LPAREN in_param_list out_param_list RPAREN
        {{ $$ = paramsTuple{
            Present: true,
            Ins: $3,
            Outs: $4,
        } }}
    ;

return_stm
    : RETURN LPAREN bind_stm_list RPAREN
        {{ $$ = &ReturnStm{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Bindings: $3,
        } }}
    ;

pipeline_retain
    :
        {{ $$ = nil }}
    | RETAIN LPAREN pipeline_retain_list RPAREN
        {{ $$ = &PipelineRetains{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Refs: $3,
        } }}

pipeline_retain_list
    :
        {{ $$ = nil }}
    | pipeline_retain_list ref_exp COMMA
        {{ $$ = append($1, $2) }}

call_stm_list
    : call_stm_list call_stm
        {{ $$ = append($1, $2) }}
    | call_stm
        {{ $$ = []*CallStm{$1} }}
    ;

call_stm
    : CALL modifiers id LPAREN bind_stm_list RPAREN
        {{  id := $<intern>3.Get($3)
            $$ = &CallStm{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Modifiers: $2,
            Id: id,
            DecId: id,
            Bindings: $5,
        } }}
    | CALL modifiers id AS id LPAREN bind_stm_list RPAREN
        {{ $$ = &CallStm{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Modifiers: $2,
            Id: $<intern>5.Get($5),
            DecId: $<intern>3.Get($3),
            Bindings: $7,
        } }}
    | call_stm USING LPAREN modifier_stm_list RPAREN
        {{
            $1.Modifiers.Bindings = $4
            $$ = $1
        }}
    ;

modifiers
    :
      {{ $$ = new(Modifiers) }}
    | modifiers LOCAL
      {{ $$.Local = true }}
    | modifiers PREFLIGHT
      {{ $$.Preflight = true }}
    | modifiers VOLATILE
      {{ $$.Volatile = true }}
    ;

modifier_stm_list
    :
        {{ $$ = &BindStms{
            Node: NewAstNode($<loc>0, $<srcfile>0),
            Table: make(map[string]*BindStm),
        } }}
    | modifier_stm_list modifier_stm
        {{
            $1.List = append($1.List, $2)
            $$ = $1
        }}
    ;

modifier_stm
    : LOCAL EQUALS bool_exp COMMA
        {{ $$ = &BindStm{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Id: local,
            Exp: $3,
        } }}
    | PREFLIGHT EQUALS bool_exp COMMA
        {{ $$ = &BindStm{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Id: preflight,
            Exp: $3,
        } }}
    | VOLATILE EQUALS bool_exp COMMA
        {{ $$ = &BindStm{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Id: volatile,
            Exp: $3,
        } }}
    | DISABLED EQUALS ref_exp COMMA
        {{ $$ = &BindStm{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Id: disabled,
            Exp: $3,
        } }}

bind_stm_list
    :
        {{ $$ = &BindStms{
            Node: NewAstNode($<loc>0, $<srcfile>0),
            Table: make(map[string]*BindStm),
        } }}
    | bind_stm_list bind_stm
        {{
            $1.List = append($1.List, $2)
            $$ = $1
        }}
    ;

bind_stm
    : id EQUALS exp COMMA
        {{ $$ = &BindStm{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Id: $<intern>1.Get($1),
            Exp: $3,
        } }}
    | id EQUALS SWEEP LPAREN exp_list COMMA RPAREN COMMA
        {{ $$ = &BindStm{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Id: $<intern>1.Get($1),
            Exp: &ValExp{
                Node: NewAstNode($<loc>1, $<srcfile>1),
                Kind: KindArray,
                Value: $5,
            },
            Sweep: true,
        } }}
    | id EQUALS SWEEP LPAREN exp_list RPAREN COMMA
        {{ $$ = &BindStm{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Id: $<intern>1.Get($1),
            Exp: &ValExp{
                Node: NewAstNode($<loc>1, $<srcfile>1),
                Kind: KindArray,
                Value: $5,
            },
            Sweep: true,
        } }}
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
    : val_exp
        {{ $$ = $1 }}
    | ref_exp
        {{ $$ = $1 }}

val_exp
    : LBRACKET exp_list RBRACKET
        {{ $$ = &ValExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindArray,
            Value: $2,
        } }}
    | LBRACKET exp_list COMMA RBRACKET
        {{ $$ = &ValExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindArray,
            Value: $2,
        } }}
    | LBRACKET RBRACKET
        {{ $$ = &ValExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindArray,
            Value: make([]Exp, 0),
        } }}
    | LBRACE RBRACE
        {{ $$ = &ValExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindMap,
            Value: make(map[string]interface{}, 0),
        } }}
    | LBRACE kvpair_list RBRACE
        {{ $$ = &ValExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindMap,
            Value: $2,
        } }}
    | LBRACE kvpair_list COMMA RBRACE
        {{ $$ = &ValExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindMap,
            Value: $2,
        } }}
    | NUM_FLOAT
        {{  // Lexer guarantees parseable float strings.
            f := parseFloat($1)
            $$ = &ValExp{
                Node: NewAstNode($<loc>1, $<srcfile>1),
                Kind: KindFloat,
                Value: f,
            }
        }}
    | NUM_INT
        {{  // Lexer guarantees parseable int strings.
            i := parseInt($1)
            $$ = &ValExp{
                Node: NewAstNode($<loc>1, $<srcfile>1),
                Kind: KindInt,
                Value: i,
            }
        }}
    | LITSTRING
        {{ $$ = &ValExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindString,
            Value: unquote($1),
        } }}
    | bool_exp
    | NULL
        {{ $$ = &ValExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindNull,
        } }}
    ;

bool_exp
    : TRUE
        {{ $$ = &ValExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindBool,
            Value: true,
        } }}
    | FALSE
        {{ $$ = &ValExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindBool,
            Value: false,
        } }}

ref_exp
    : id DOT id
        {{ $$ = &RefExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindCall,
            Id: $<intern>1.Get($1),
            OutputId: $<intern>3.Get($3),
        } }}
    | id
        {{ $$ = &RefExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindCall,
            Id: $<intern>1.Get($1),
            OutputId: default_out_name,
        } }}
    | SELF DOT id
        {{ $$ = &RefExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindSelf,
            Id: $<intern>3.Get($3),
        } }}
    ;

id
    : ID
    | COMPILED
    | DISABLED
    | EXEC
    | FILETYPE
    | LOCAL
    | MEM_GB
    | PREFLIGHT
    | RETAIN
    | SPECIAL
    | SPLIT
    | STRICT
    | THREADS
    | USING
    | VOLATILE
    ;
%%

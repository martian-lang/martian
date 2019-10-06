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
    s_member  *StructMember
    retains   []*RetainParam
    stretains *RetainParams
    i_params  *InParams
    o_params  *OutParams
    s_members []*StructMember
    res       *Resources
    par_tuple paramsTuple
    src       *SrcParam
    type_id   TypeId
    exp       Exp
    exps      []Exp
    rexp      *RefExp
    vexp      ValExp
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
%type <val>       id id_list nonmap_type type help src_lang outname
%type <modifiers> modifiers
%type <arr>       arr_list
%type <dec>       dec stage pipeline struct
%type <decs>      dec_list
%type <inparam>   in_param
%type <outparam>  out_param
%type <s_member>  struct_field
%type <retains>   stage_retain_list
%type <stretains> stage_retain
%type <reflist>   pipeline_retain_list
%type <plretains> pipeline_retain
%type <i_params>  in_param_list
%type <o_params>  out_param_list
%type <s_members> struct_field_list
%type <par_tuple> split_param_list
%type <src>       src_stm
%type <type_id>   type_id
%type <exp>       exp
%type <rexp>      ref_exp
%type <vexp>      val_exp bool_exp array_exp map_exp
%type <exps>      exp_list
%type <kvpairs>   kvpair_list struct_vals_list
%type <call>      call_stm
%type <calls>     call_stm_list
%type <binding>   bind_stm modifier_stm
%type <bindings>  bind_stm_list modifier_stm_list
%type <retstm>    return_stm
%type <res>       resources resource_list

%token SKIP COMMENT INVALID
%token SEMICOLON COLON COMMA EQUALS
%token LBRACKET RBRACKET LPAREN RPAREN LBRACE RBRACE LANGLE RANGLE
%token SWEEP RETURN SELF
%token <val> FILETYPE STAGE PIPELINE CALL SPLIT USING RETAIN
%token <val> LOCAL PREFLIGHT VOLATILE DISABLED STRICT STRUCT
%token IN OUT SRC AS
%token <val> THREADS MEM_GB VMEM_GB SPECIAL
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
    | val_exp
        {{
            global := NewAst(nil, nil, $<srcfile>1)
            mmlex.(*mmLexInfo).global = global
            mmlex.(*mmLexInfo).exp = $1
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
    | struct
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

struct
   : STRUCT id LPAREN struct_field_list RPAREN
        {{ $$ = &StructType{
                Node: NewAstNode($<loc>2, $<srcfile>2),
                Id: $<intern>2.Get($2),
                Members: $4,
           }
        }}

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
    | resource_list VMEM_GB EQUALS NUM_INT COMMA
        {{
            n := NewAstNode($<loc>2, $<srcfile>2)
            $1.VMemNode = &n
            i := parseInt($4)
            $1.VMemGB = int16(i)
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
            $$ = append(append($1, '.'), $3...)
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
    : IN type_id id help COMMA
        {{ $$ = &InParam{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Tname: $2,
            Id: $<intern>3.Get($3),
            Help: unquote($4),
        } }}
    | IN type_id id COMMA
        {{ $$ = &InParam{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Tname: $2,
            Id: $<intern>3.Get($3),
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
    : OUT type_id COMMA
        {{ $$ = &OutParam{
            StructMember: StructMember{
                Node: NewAstNode($<loc>1, $<srcfile>1),
                Tname: $2,
                Id: default_out_name,
            },
        } }}
    | OUT type_id help COMMA
        {{ $$ = &OutParam{
            StructMember: StructMember{
                Node: NewAstNode($<loc>1, $<srcfile>1),
                Tname: $2,
                Id: default_out_name,
                Help: unquote($3),
            },
        } }}
    | OUT type_id help outname COMMA
        {{ $$ = &OutParam{
            StructMember: StructMember{
                Node: NewAstNode($<loc>1, $<srcfile>1),
                Tname: $2,
                Id: default_out_name,
                OutName: $<intern>5.unquote($4),
                Help: unquote($3),
            },
        } }}
    | OUT struct_field
        {{ $$ = &OutParam{
            StructMember: *$2,
        } }}
    ;

struct_field_list
    : struct_field
        {{ $$ = []*StructMember{$1} }}
    | struct_field_list struct_field
        {{
            $$ = append($1, $2)
        }}
    ;

struct_field
    : type_id id COMMA
        {{ $$ = &StructMember{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Tname: $1,
            Id: $<intern>2.Get($2),
        } }}
    | type_id id help COMMA
        {{ $$ = &StructMember{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Tname: $1,
            Id: $<intern>2.Get($2),
            Help: unquote($3),
        } }}
    | type_id id help outname COMMA
        {{ $$ = &StructMember{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Tname: $1,
            Id: $<intern>2.Get($2),
            OutName: $<intern>4.unquote($4),
            Help: unquote($3),
        } }}
     ;

src_stm
    : SRC src_lang LITSTRING COMMA
        {{ cmd := strings.TrimSpace($<intern>3.unquote($3))
           stagecodeParts := strings.Fields($<intern>3.unquote($3))
           $$ = &SrcParam{
               Node: NewAstNode($<loc>1, $<srcfile>1),
               Lang: StageLanguage($<intern>2.Get($2)),
               cmd:  cmd,
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
    : nonmap_type
    | MAP
    ;

nonmap_type
    : INT
    | STRING
    | PATH
    | FLOAT
    | BOOL
    | id_list
    ;
 
type_id
    : MAP LANGLE nonmap_type arr_list RANGLE arr_list
        {{ $$ = TypeId{
            Tname: $<intern>3.Get($3),
            ArrayDim: $6,
            MapDim: 1 + $4,
        } }}
    | type arr_list
        {{ $$ = TypeId{
            Tname: $<intern>1.Get($1),
            ArrayDim: $2,
        } }}
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
            Exp: &SweepExp{
                valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
                Value: $5,
            },
            Sweep: true,
        } }}
    | id EQUALS SWEEP LPAREN exp_list RPAREN COMMA
        {{ $$ = &BindStm{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Id: $<intern>1.Get($1),
            Exp: &SweepExp{
                valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
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

struct_vals_list
    : struct_vals_list COMMA id COLON exp
        {{
            $1[$<intern>3.Get($3)] = $5
            $$ = $1
        }}
    | id COLON exp
        {{ $$ = map[string]Exp{$<intern>1.Get($1): $3} }}
    ;

exp
    : val_exp
        {{ $$ = $1 }}
    | ref_exp
        {{ $$ = $1 }}

val_exp
    : NUM_FLOAT
        {{  // Lexer guarantees parseable float strings.
            f := parseFloat($1)
            $$ = &FloatExp{
                valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
                Value: f,
            }
        }}
    | NUM_INT
        {{  // Lexer guarantees parseable int strings.
            i := parseInt($1)
            $$ = &IntExp{
                valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
                Value: i,
            }
        }}
    | LITSTRING
        {{ $$ = &StringExp{
            valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
            Kind: KindString,
            Value: unquote($1),
        } }}
    | array_exp
    | map_exp
    | bool_exp
    | NULL
        {{ $$ = &NullExp{
            valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
        } }}
    ;

array_exp
    : LBRACKET exp_list RBRACKET
        {{ $$ = &ArrayExp{
            valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
            Value: $2,
        } }}
    | LBRACKET exp_list COMMA RBRACKET
        {{ $$ = &ArrayExp{
            valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
            Value: $2,
        } }}
    | LBRACKET RBRACKET
        {{ $$ = &ArrayExp{
            valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
            Value: make([]Exp, 0),
        } }}

map_exp
    : LBRACE RBRACE
        {{ $$ = &MapExp{
            valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
            Kind: KindMap,
            Value: make(map[string]Exp, 0),
        } }}
    | LBRACE kvpair_list RBRACE
        {{ $$ = &MapExp{
            valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
            Kind: KindMap,
            Value: $2,
        } }}
    | LBRACE kvpair_list COMMA RBRACE
        {{ $$ = &MapExp{
            valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
            Kind: KindMap,
            Value: $2,
        } }}
    | LBRACE struct_vals_list RBRACE
        {{ $$ = &MapExp{
            valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
            Kind: KindStruct,
            Value: $2,
        } }}
    | LBRACE struct_vals_list COMMA RBRACE
        {{ $$ = &MapExp{
            valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
            Kind: KindStruct,
            Value: $2,
        } }}

bool_exp
    : TRUE
        {{ $$ = &BoolExp{
            valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
            Value: true,
        } }}
    | FALSE
        {{ $$ = &BoolExp{
            valExp: valExp{Node: NewAstNode($<loc>1, $<srcfile>1)},
            Value: false,
        } }}

ref_exp
    : id DOT id_list
        {{ $$ = &RefExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindCall,
            Id: $<intern>1.Get($1),
            OutputId: $<intern>3.Get($3),
        } }}
    | id DOT DEFAULT
        {{
            $$ = &RefExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindCall,
            Id: $<intern>1.Get($1),
            OutputId: default_out_name,
        } }}
    | id
        {{ $$ = &RefExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindCall,
            Id: $<intern>1.Get($1),
        } }}
    | SELF DOT id
        {{ $$ = &RefExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindSelf,
            Id: $<intern>3.Get($3),
        } }}
    | SELF DOT id DOT id_list
        {{ $$ = &RefExp{
            Node: NewAstNode($<loc>1, $<srcfile>1),
            Kind: KindSelf,
            Id: $<intern>3.Get($3),
            OutputId: $<intern>5.Get($5),
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
    | VMEM_GB
    | PREFLIGHT
    | RETAIN
    | SPECIAL
    | SPLIT
    | STRICT
    | STRUCT
    | THREADS
    | USING
    | VOLATILE
    ;
%%

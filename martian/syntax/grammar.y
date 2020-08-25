%{
//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
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
    arr       int16
    loc       SourceLoc
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
    f32       float32
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
%type <vexp>      val_exp bool_exp
%type <vexp>      array_exp nonempty_array_exp
%type <vexp>      map_exp nonempty_map_exp
%type <vexp>      nonempty_collection_exp
%type <exps>      exp_list exp_list_partial
%type <kvpairs>   kvpair_list kvpair_list_partial
%type <kvpairs>   struct_vals_list struct_vals_list_partial
%type <call>      call_stm call_stm_begin
%type <calls>     call_stm_list
%type <binding>   bind_stm split_bind_stm wildcard_bind modifier_stm
%type <bindings>  bind_stm_list nonempty_bind_stm_list
%type <bindings>  split_bind_stm_list_partial split_bind_stm_list
%type <bindings>  modifier_stm_list
%type <retstm>    return_stm
%type <res>       resources resource_list
%type <f32>       float_32

%token SKIP COMMENT INVALID
%token ';' ':' ',' '=' '.' '*'
%token '[' ']' '(' ')' '{' '}' '<' '>'
%token INCLUDE_DIRECTIVE STAGE PIPELINE CALL RETURN
%token IN OUT SRC AS
%token <val> FILETYPE MAP INT STRING FLOAT PATH BOOL
%token <val> SPLIT USING RETAIN
%token <val> LOCAL PREFLIGHT VOLATILE DISABLED STRICT STRUCT
%token <val> THREADS MEM_GB VMEM_GB SPECIAL
%token <val> ID LITSTRING NUM_FLOAT NUM_INT
%token <val> PY EXEC COMPILED
%token SELF TRUE FALSE NULL DEFAULT

%%
file
    : includes dec_list
        {
            global := NewAst($2, nil, $<loc>2.File)
            global.Includes = $1
            mmlex.(*mmLexInfo).global = global
        }
    | includes dec_list call_stm
        {
            global := NewAst($2, $3, $<loc>2.File)
            global.Includes = $1
            mmlex.(*mmLexInfo).global = global
        }
    | includes call_stm
        {
            global := NewAst(nil, $2, $<loc>2.File)
            global.Includes = $1
            mmlex.(*mmLexInfo).global = global
        }
    | dec_list
        {
            global := NewAst($1, nil, $<loc>1.File)
            mmlex.(*mmLexInfo).global = global
        }
    | dec_list call_stm
        {
            global := NewAst($1, $2, $<loc>1.File)
            mmlex.(*mmLexInfo).global = global
        }
    | call_stm
        {
            global := NewAst(nil, $1, $<loc>1.File)
            mmlex.(*mmLexInfo).global = global
        }
    | val_exp
        {
            global := NewAst(nil, nil, $<loc>1.File)
            mmlex.(*mmLexInfo).global = global
            mmlex.(*mmLexInfo).exp = $1
        }
    ;

includes
    : includes INCLUDE_DIRECTIVE LITSTRING
        { $$ = append($1, &Include{
            Node: NewAstNode($<loc>2),
            Value: $<intern>3.unquote($3),
           })
        }
    | INCLUDE_DIRECTIVE LITSTRING
        { $$ = []*Include{
              &Include{
                  Node: NewAstNode($<loc>1),
                  Value: $<intern>2.unquote($2),
              },
           }
        }

dec_list
    : dec_list dec
        { $$ = append($1, $2) }
    | dec
        { $$ = []Dec{$1} }
    ;

dec
    : FILETYPE id_list ';'
        { $$ = &UserType{
            Node: NewAstNode($<loc>2),
            Id: $<intern>2.Get($2),
        } }
    | stage
    | pipeline
    | struct
    ;

pipeline
    : PIPELINE id '(' in_param_list out_param_list ')' '{' call_stm_list return_stm pipeline_retain '}'
        { $$ = &Pipeline{
            Node: NewAstNode($<loc>2),
            Id: $<intern>2.Get($2),
            InParams: $4,
            OutParams: $5,
            Calls: $8,
            Ret: $9,
            Retain: $10,
        } }
    | PIPELINE id '(' in_param_list out_param_list ')' '{' return_stm pipeline_retain '}'
        { $$ = &Pipeline{
            Node: NewAstNode($<loc>2),
            Id: $<intern>2.Get($2),
            InParams: $4,
            OutParams: $5,
            Callables: new(Callables),
            Ret: $8,
            Retain: $9,
        } }
    ;

stage
    : STAGE id '(' in_param_list out_param_list src_stm ')' split_param_list resources stage_retain
        { $$ = &Stage{
                Node: NewAstNode($<loc>2),
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
        }
   ;

struct
   : STRUCT id '(' struct_field_list ')'
        { $$ = &StructType{
                Node: NewAstNode($<loc>2),
                Id: $<intern>2.Get($2),
                Members: $4,
           }
        }

resources
    :
        { $$ = nil }
    | USING '(' resource_list ')'
        {
            $3.Node = NewAstNode($<loc>1)
            $$ = $3
        }
    ;

resource_list
    :
        { $$ = new(Resources) }
    | resource_list THREADS '=' float_32 ','
        {
            n := NewAstNode($<loc>2)
            $1.ThreadNode = &n
            $1.Threads = roundUpTo($4, 100)
            $$ = $1
        }
    | resource_list MEM_GB '=' float_32 ','
        {
            n := NewAstNode($<loc>2)
            $1.MemNode = &n
            $1.MemGB = roundUpTo($4, 1024)
            $$ = $1
        }
    | resource_list VMEM_GB '=' float_32 ','
        {
            n := NewAstNode($<loc>2)
            $1.VMemNode = &n
            $1.VMemGB = roundUpTo($4, 1024)
            $$ = $1
        }
    | resource_list SPECIAL '=' LITSTRING ','
        {
            n := NewAstNode($<loc>2)
            $1.SpecialNode = &n
            $1.Special = $<intern>4.unquote($4)
            $$ = $1
        }
    | resource_list VOLATILE '=' STRICT ','
        {
            n := NewAstNode($<loc>2)
            $1.VolatileNode = &n
            $1.StrictVolatile = true
            $$ = $1
        }
    | resource_list VOLATILE '=' FALSE ','
        {
            n := NewAstNode($<loc>2)
            $1.VolatileNode = &n
            $1.StrictVolatile = false
            $$ = $1
        }
    ;

float_32
    : NUM_INT
        { $$ = float32(parseInt($1)) }
    | NUM_FLOAT
        { $$ = parseFloat32($1) }
    ;

stage_retain
    :
        { $$ = nil }
    | RETAIN '(' stage_retain_list ')'
        {
            $$ = &RetainParams{
                Node: NewAstNode($<loc>1),
                Params: $3,
             }
         }
    ;

stage_retain_list
    :
        { $$ = nil }
    | stage_retain_list id ','
        {
            $$ = append($1, &RetainParam{
                Node: NewAstNode($<loc>2),
                Id: $<intern>2.Get($2),
            })
        }
    ;


id_list
    : id_list '.' id
        {
            $$ = append(append($1, '.'), $3...)
        }
    | id
        {
            // set capacity == length so append doesn't overwrite
            // other parts of the buffer later.
            $$ = $1[:len($1):len($1)]
        }
    ;

arr_list
    :
        { $$ = 0 }
    | arr_list '[' ']'
        { $$++ }
    ;

in_param_list
    :
        { $$ = new(InParams) }
    | in_param_list in_param
        {
            $1.List = append($1.List, $2)
            $$ = $1
        }
    ;

in_param
    : IN type_id id help ','
        { $$ = &InParam{
            Node: NewAstNode($<loc>1),
            Tname: $2,
            Id: $<intern>3.Get($3),
            Help: unquote($4),
        } }
    | IN type_id id ','
        { $$ = &InParam{
            Node: NewAstNode($<loc>1),
            Tname: $2,
            Id: $<intern>3.Get($3),
        } }
    ;

out_param_list
    :
        { $$ = new(OutParams) }
    | out_param_list out_param
        {
            $1.List = append($1.List, $2)
            $$ = $1
        }
    ;

out_param
    : OUT type_id ','
        { $$ = &OutParam{
            StructMember: StructMember{
                Node: NewAstNode($<loc>1),
                Tname: $2,
                Id: defaultOutName,
            },
        } }
    | OUT type_id help ','
        { $$ = &OutParam{
            StructMember: StructMember{
                Node: NewAstNode($<loc>1),
                Tname: $2,
                Id: defaultOutName,
                Help: unquote($3),
            },
        } }
    | OUT type_id help outname ','
        { $$ = &OutParam{
            StructMember: StructMember{
                Node: NewAstNode($<loc>1),
                Tname: $2,
                Id: defaultOutName,
                OutName: $<intern>5.unquote($4),
                Help: unquote($3),
            },
        } }
    | OUT struct_field
        { $$ = &OutParam{
            StructMember: *$2,
        } }
    ;

struct_field_list
    : struct_field
        { $$ = []*StructMember{$1} }
    | struct_field_list struct_field
        {
            $$ = append($1, $2)
        }
    ;

struct_field
    : type_id id ','
        { $$ = &StructMember{
            Node: NewAstNode($<loc>1),
            Tname: $1,
            Id: $<intern>2.Get($2),
        } }
    | type_id id help ','
        { $$ = &StructMember{
            Node: NewAstNode($<loc>1),
            Tname: $1,
            Id: $<intern>2.Get($2),
            Help: unquote($3),
        } }
    | type_id id help outname ','
        { $$ = &StructMember{
            Node: NewAstNode($<loc>1),
            Tname: $1,
            Id: $<intern>2.Get($2),
            OutName: $<intern>4.unquote($4),
            Help: unquote($3),
        } }
     ;

src_stm
    : SRC src_lang LITSTRING ','
        {
            cmd := strings.TrimSpace($<intern>3.unquote($3))
            stagecodeParts := strings.Fields(cmd)
            $$ = &SrcParam{
                Node: NewAstNode($<loc>1),
                Lang: StageLanguage($<intern>2.Get($2)),
                cmd:  cmd,
                Path: stagecodeParts[0],
                Args: stagecodeParts[1:],
            }
        }
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
    : MAP '<' nonmap_type arr_list '>' arr_list
        { $$ = TypeId{
            Tname: $<intern>3.Get($3),
            ArrayDim: $6,
            MapDim: 1 + $4,
        } }
    | type arr_list
        { $$ = TypeId{
            Tname: $<intern>1.Get($1),
            ArrayDim: $2,
        } }
    ;

src_lang
    : PY
    | EXEC
    | COMPILED
    ;

split_param_list
    :
        {
            $$ = paramsTuple{
                Present: false,
                Ins: new(InParams),
                Outs: new(OutParams),
            }
        }
    | SPLIT USING '(' in_param_list out_param_list ')'
        { $$ = paramsTuple{
            Present: true,
            Ins: $4,
            Outs: $5,
        } }
    | SPLIT '(' in_param_list out_param_list ')'
        { $$ = paramsTuple{
            Present: true,
            Ins: $3,
            Outs: $4,
        } }
    ;

return_stm
    : RETURN '(' bind_stm_list ')'
        { $$ = &ReturnStm{
            Node: NewAstNode($<loc>1),
            Bindings: $3,
        } }
    ;

pipeline_retain
    :
        { $$ = nil }
    | RETAIN '(' pipeline_retain_list ')'
        { $$ = &PipelineRetains{
            Node: NewAstNode($<loc>1),
            Refs: $3,
        } }

pipeline_retain_list
    :
        { $$ = nil }
    | pipeline_retain_list ref_exp ','
        { $$ = append($1, $2) }

call_stm_list
    : call_stm_list call_stm
        { $$ = append($1, $2) }
    | call_stm
        { $$ = []*CallStm{$1} }
    ;

call_stm_begin
    : CALL modifiers id
        { id := $<intern>3.Get($3)
          $$ = &CallStm{
            Node: NewAstNode($<loc>1),
            Modifiers: $2,
            Id: id,
            DecId: id,
        } }
    | CALL modifiers id AS id
        { $$ = &CallStm{
            Node: NewAstNode($<loc>1),
            Modifiers: $2,
            Id: $<intern>5.Get($5),
            DecId: $<intern>3.Get($3),
        } }


call_stm
    : call_stm_begin '(' bind_stm_list ')'
        { 
            $1.Bindings = $3
            $$ = $1
        }
    | MAP call_stm_begin '(' split_bind_stm_list ')'
        {
            $2.Bindings = $4
            $2.Mapping = &mapSourcePlaceholder
            $$ = $2
        }
    | call_stm USING '(' modifier_stm_list ')'
        {
            $1.Modifiers.Bindings = $4
            $$ = $1
        }
    ;

modifiers
    :
      { $$ = new(Modifiers) }
    | modifiers LOCAL
      { $$.Local = true }
    | modifiers PREFLIGHT
      { $$.Preflight = true }
    | modifiers VOLATILE
      { $$.Volatile = true }
    ;

modifier_stm_list
    :
        { $$ = &BindStms{
            Node: NewAstNode($<loc>0),
        } }
    | modifier_stm_list modifier_stm
        {
            $1.List = append($1.List, $2)
            $$ = $1
        }
    ;

modifier_stm
    : LOCAL '=' bool_exp ','
        { $$ = &BindStm{
            Node: NewAstNode($<loc>1),
            Id: local,
            Exp: $3,
        } }
    | PREFLIGHT '=' bool_exp ','
        { $$ = &BindStm{
            Node: NewAstNode($<loc>1),
            Id: preflight,
            Exp: $3,
        } }
    | VOLATILE '=' bool_exp ','
        { $$ = &BindStm{
            Node: NewAstNode($<loc>1),
            Id: volatile,
            Exp: $3,
        } }
    | DISABLED '=' ref_exp ','
        { $$ = &BindStm{
            Node: NewAstNode($<loc>1),
            Id: disabled,
            Exp: $3,
        } }
    ;

nonempty_bind_stm_list
    : nonempty_bind_stm_list bind_stm
        {
            $1.List = append($1.List, $2)
            $$ = $1
        }
    | bind_stm
        { $$ = &BindStms{
            Node: NewAstNode($<loc>0),
            List: []*BindStm{$1},
        } }
    ;

bind_stm_list
    : nonempty_bind_stm_list
    | nonempty_bind_stm_list wildcard_bind
        {
            $1.List = append($1.List, $2)
            $$ = $1
        }
    | wildcard_bind
        { $$ = &BindStms{
            Node: NewAstNode($<loc>0),
            List: []*BindStm{$1},
        } }
    |
        { $$ = &BindStms{
            Node: NewAstNode($<loc>0),
        } }
    ;

split_bind_stm_list_partial
    : split_bind_stm
        { $$ = &BindStms{
            Node: NewAstNode($<loc>0),
            List: []*BindStm{$1},
        } }
    | nonempty_bind_stm_list split_bind_stm
        {
            $1.List = append($1.List, $2)
            $$ = $1
        }
    | split_bind_stm_list_partial split_bind_stm
        {
            $1.List = append($1.List, $2)
            $$ = $1
        }
    | split_bind_stm_list_partial bind_stm
        {
            $1.List = append($1.List, $2)
            $$ = $1
        }
    ;

split_bind_stm_list
    : split_bind_stm_list_partial
    | split_bind_stm_list_partial wildcard_bind
        {
            $1.List = append($1.List, $2)
            $$ = $1
        }
    ;

bind_stm
    : id '=' exp ','
        { $$ = &BindStm{
            Node: NewAstNode($<loc>1),
            Id: $<intern>1.Get($1),
            Exp: $3,
        } }
    ;

wildcard_bind
    : '*' '=' ref_exp ','
        { $$ = &BindStm{
            Node: NewAstNode($<loc>1),
            Id: "*",
            Exp: $3,
        } }
    | '*' '=' SELF ','
      { $$ = &BindStm{
            Node: NewAstNode($<loc>1),
            Id: "*",
            Exp: &RefExp{
                Node: NewAstNode($<loc>3),
                Kind: KindSelf,
            },
       } }
    ;

split_bind_stm
    : id '=' SPLIT nonempty_collection_exp ','
        { $$ = &BindStm{
            Node: NewAstNode($<loc>1),
            Id: $<intern>1.Get($1),
            Exp: &SplitExp{
                valExp: valExp{Node: NewAstNode($<loc>3)},
                Value: $4,
                Source: $4.(MapCallSource),
            },
        } }
    | id '=' SPLIT ref_exp ','
        { $$ = &BindStm{
            Node: NewAstNode($<loc>1),
            Id: $<intern>1.Get($1),
            Exp: &SplitExp{
                valExp: valExp{Node: NewAstNode($<loc>3)},
                Value: $4,
            },
        } }

nonempty_collection_exp
    : nonempty_array_exp
    | nonempty_map_exp
    ;

exp_list_partial
    : exp_list_partial ',' exp
        { $$ = append($1, $3) }
    | exp
        { $$ = []Exp{$1} }
    ;

exp_list
    : exp_list_partial ','
    | exp_list_partial
    ;

kvpair_list_partial
    : kvpair_list_partial ',' LITSTRING ':' exp
        {
            $1[unquote($3)] = $5
            $$ = $1
        }
    | LITSTRING ':' exp
        { $$ = map[string]Exp{unquote($1): $3} }
    ;

kvpair_list
    : kvpair_list_partial ','
    | kvpair_list_partial
    ;

struct_vals_list_partial
    : struct_vals_list_partial ',' id ':' exp
        {
            $1[$<intern>3.Get($3)] = $5
            $$ = $1
        }
    | id ':' exp
        { $$ = map[string]Exp{$<intern>1.Get($1): $3} }
    ;

struct_vals_list
    : struct_vals_list_partial ','
    | struct_vals_list_partial
    ;

exp
    : val_exp
        { $$ = $1 }
    | ref_exp
        { $$ = $1 }
    ;

val_exp
    : NUM_FLOAT
        { // Lexer guarantees parseable float strings.
          f := parseFloat($1)
          $$ = &FloatExp{
              valExp: valExp{Node: NewAstNode($<loc>1)},
              Value: f,
          }
        }
    | NUM_INT
        {  // Lexer guarantees parseable int strings.
            i := parseInt($1)
            $$ = &IntExp{
                valExp: valExp{Node: NewAstNode($<loc>1)},
                Value: i,
            }
        }
    | LITSTRING
        { $$ = &StringExp{
            valExp: valExp{Node: NewAstNode($<loc>1)},
            Value: unquote($1),
        } }
    | array_exp
    | map_exp
    | bool_exp
    | NULL
        { $$ = &NullExp{
            valExp: valExp{Node: NewAstNode($<loc>1)},
        } }
    ;

nonempty_array_exp
    : '[' exp_list ']'
        { $$ = &ArrayExp{
            valExp: valExp{Node: NewAstNode($<loc>1)},
            Value: $2,
        } }
    ;

array_exp
    : nonempty_array_exp
    | '[' ']'
        { $$ = &ArrayExp{
            valExp: valExp{Node: NewAstNode($<loc>1)},
            Value: make([]Exp, 0),
        } }
    ;

nonempty_map_exp
    : '{' kvpair_list '}'
        { $$ = &MapExp{
            valExp: valExp{Node: NewAstNode($<loc>1)},
            Kind: KindMap,
            Value: $2,
        } }
    ;

map_exp
    : nonempty_map_exp
    | '{' struct_vals_list '}'
        { $$ = &MapExp{
            valExp: valExp{Node: NewAstNode($<loc>1)},
            Kind: KindStruct,
            Value: $2,
        } }
    | '{' '}'
        { $$ = &MapExp{
            valExp: valExp{Node: NewAstNode($<loc>1)},
            Kind: KindMap,
            Value: make(map[string]Exp, 0),
        } }
    ;

bool_exp
    : TRUE
        { $$ = &BoolExp{
            valExp: valExp{Node: NewAstNode($<loc>1)},
            Value: true,
        } }
    | FALSE
        { $$ = &BoolExp{
            valExp: valExp{Node: NewAstNode($<loc>1)},
            Value: false,
        } }
    ;

ref_exp
    : id '.' id_list
        { $$ = &RefExp{
            Node: NewAstNode($<loc>1),
            Kind: KindCall,
            Id: $<intern>1.Get($1),
            OutputId: $<intern>3.Get($3),
        } }
    | id '.' DEFAULT
        {
            $$ = &RefExp{
            Node: NewAstNode($<loc>1),
            Kind: KindCall,
            Id: $<intern>1.Get($1),
            OutputId: defaultOutName,
        } }
    | id
        { $$ = &RefExp{
            Node: NewAstNode($<loc>1),
            Kind: KindCall,
            Id: $<intern>1.Get($1),
        } }
    | SELF '.' id
        { $$ = &RefExp{
            Node: NewAstNode($<loc>1),
            Kind: KindSelf,
            Id: $<intern>3.Get($3),
        } }
    | SELF '.' id '.' id_list
        { $$ = &RefExp{
            Node: NewAstNode($<loc>1),
            Kind: KindSelf,
            Id: $<intern>3.Get($3),
            OutputId: $<intern>5.Get($5),
        } }
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

" Martian (MRO) Syntax
" Author: Matt Sooknah

if exists("b:current_syntax")
  finish
endif

syn match include '^\s*@include' nextgroup=mroString skipwhite

syn keyword filetype  filetype nextgroup=parType skipwhite
syn keyword parameter in out  nextgroup=parType skipwhite contained
syn keyword src       src nextgroup=srctype skipwhite contained
syn keyword srctype   py comp exe nextgroup=mroString contained skipwhite
syn keyword restype   mem_gb vmem_gb threads special volatile nextgroup=assign contained skipwhite
syn keyword modifier  local preflight volatile nextgroup=modifier,callTarg skipwhite contained
syn keyword boundMod  local preflight volatile disabled nextgroup=assign contained skipwhite
syn keyword sweep     sweep nextgroup=sweepArray contained

syn keyword declaration pipeline stage nextgroup=pipeName skipwhite
syn keyword call        call nextgroup=modifier,callTarg skipwhite
syn keyword return      return nextgroup=callParams skipwhite contained

syn keyword mroNull  null       contained
syn keyword mroBool  true false contained

syn keyword split      split  nextgroup=splitUsing,paramBlock skipwhite contained
syn keyword splitUsing using  nextgroup=paramBlock skipwhite contained
syn keyword using      using  nextgroup=resParams  skipwhite contained
syn keyword retain     retain nextgroup=idList     skipwhite contained
syn keyword callUsing  using  nextgroup=modBlock   skipwhite contained
syn keyword as         as     nextgroup=callTarg   skipwhite contained
syn keyword self       self   nextgroup=dot contained

syn match dot    '\.' nextgroup=parName contained
syn match mapSep ':'  nextgroup=mroString,mroNull,mroBool,arrayLit,mapLit skipwhite contained
syn match assign '='  nextgroup=self,sweep,mroString,mroNumber,mroNull,mroBool,parName,arrayLit,mapLit skipwhite contained

syn match assignment '_\?[A-Za-z][A-Za-z0-9_]*\s*=' contains=parName,assign contained skipwhite nextgroup=self,sweep,mroString,mroNumber,mroNull,mroBool,parName,arrayLit,mapLit
syn match parType '_\?[A-Za-z][.A-Za-z0-9_]*' nextgroup=arrayDim,parName skipwhite contained
syn match arrayDim '\[\]' nextgroup=arraydim,parName skipwhite contained
syn match parName '_\?[A-Za-z][A-Za-z0-9_]*' nextgroup=dot,mroString skipwhite contained
syn match pipeName '_\?[A-Za-z][A-Za-z0-9_]*' nextgroup=paramBlock contained
syn match callTarg '_\?[A-Za-z][A-Za-z0-9_]*' nextgroup=as,callParams skipwhite contained

syn match commentLine '#.*$'

syn match mroNumber '\v<\d+>' contained skipwhite nextgroup=mapSep
syn match mroNumber '\v<\d+\.\d+>' contained skipwhite nextgroup=mapSep

syn region paramBlock start="(" end=")" fold transparent nextgroup=split,using,retain,callBlock skipwhite skipnl contained contains=parameter,src
syn region callParams start="(" end=")" fold transparent nextgroup=callUsing,retain skipwhite contained contains=assignment
syn region resParams  start="(" end=")" fold transparent nextgroup=retain contained skipwhite skipnl contains=restype
syn region idList     start="(" end=")" fold transparent contained contains=parName
syn region modBlock   start="(" end=")" fold transparent contained contains=boundMod
syn region callBlock  start="{" end="}" fold transparent contains=call,return contained skipwhite
syn region mroString  start=/"/ skip=/\\"/ end=/"/ nextgroup=mapSep skipwhite contained
syn region arrayLit   start='\[' end='\]' transparent contains=mroString,mroNumber,mroNull,mroBool,arrayLit,mapLit,parName contained
syn region sweepArray start='(' end=')' transparent contains=mroString,mroNumber,mroNull,mroBool,arrayLit,mapLit contained
syn region mapLit     start="{" end="}" transparent contains=mroString,mroNumber contained

let b:current_syntax = "mro"

hi def link commentLine   Comment

hi def link include       PreProcessor
hi def link filetype      Statement
hi def link parameter     Statement
hi def link src           Statement
hi def link declaration   Statement
hi def link return        Statement
hi def link call          Statement
hi def link modifier      Statement
hi def link split         Statement
hi def link splitUsing    Statement
hi def link callUsing     Statement
hi def link using         Statement
hi def link as            Keyword
hi def link restype       Keyword
hi def link boundMod      Keyword
hi def link self          Keyword
hi def link sweep         Keyword
hi def link using         Statement
hi def link retain        Statement

hi def link parType       Type
hi def link srcType       Type

hi def link pipeName      Type
hi def link callTarg      Type
hi def link parName       Identifier

hi def link dot       Operator

hi def link mroNumber Constant
hi def link mroString Constant
hi def link mroNull   Constant
hi def link mroTrue   Constant
hi def link mroFalse  Constant

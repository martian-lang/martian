" Martian (MRO) Syntax
" Author: Matt Sooknah

if exists("b:current_syntax")
  finish
endif

syn keyword include @include

syn keyword filetype filetype nextgroup=parType skipwhite
syn keyword parameter in out src nextgroup=parType skipwhite

syn keyword declaration pipeline stage nextgroup=pipeName skipwhite
syn keyword action call return nextgroup=modifier,pipeName skipwhite

syn keyword modifier volatile local preflight nextgroup=modifier,pipeName skipwhite contained

syn keyword mroNull null

syn match splitUsing 'split using'

syn match parType '\w\+' contained
syn match parType '\w\+\.\w\+' contained
syn match pipeName '[A-Za-z_]\+' contained 

syn match commentLine '#.*$'

syn match mroNumber '\v<\d+>'
syn match mroNumber '\v<\d+\.\d+>'

syn region mroString start=/"/ skip=/\\"/ end=/"/

let b:current_syntax = "mro"

hi def link commentLine   Comment

hi def link include       Statement
hi def link filetype      Statement
hi def link parameter     Statement
hi def link declaration   Statement
hi def link action        Statement
hi def link modifier      Statement
hi def link splitUsing    Statement

hi def link parType       Type
hi def link pipeName      Type

hi def link mroNumber Constant
hi def link mroString Constant
hi def link mroNull   Constant

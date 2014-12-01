//line src/mario/core/grammar.y:2

//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO grammar.
//
package core

import __yyfmt__ "fmt"

//line src/mario/core/grammar.y:7
import (
	"strconv"
	"strings"
)

func unquote(qs string) string {
	return strings.Replace(qs, "\"", "", -1)
}

//line src/mario/core/grammar.y:19
type mmSymType struct {
	yys      int
	global   *Ast
	arr      int
	loc      int
	val      string
	comments string
	dec      Dec
	decs     []Dec
	inparam  *InParam
	outparam *OutParam
	params   *Params
	src      *SrcParam
	exp      Exp
	exps     []Exp
	kvpairs  map[string]Exp
	call     *CallStm
	calls    []*CallStm
	binding  *BindStm
	bindings *BindStms
	retstm   *ReturnStm
}

const SKIP = 57346
const INVALID = 57347
const SEMICOLON = 57348
const COLON = 57349
const COMMA = 57350
const EQUALS = 57351
const LBRACKET = 57352
const RBRACKET = 57353
const LPAREN = 57354
const RPAREN = 57355
const LBRACE = 57356
const RBRACE = 57357
const FILETYPE = 57358
const STAGE = 57359
const PIPELINE = 57360
const CALL = 57361
const VOLATILE = 57362
const SWEEP = 57363
const SPLIT = 57364
const USING = 57365
const SELF = 57366
const RETURN = 57367
const IN = 57368
const OUT = 57369
const SRC = 57370
const ID = 57371
const LITSTRING = 57372
const NUM_FLOAT = 57373
const NUM_INT = 57374
const DOT = 57375
const PY = 57376
const GO = 57377
const SH = 57378
const EXEC = 57379
const MAP = 57380
const INT = 57381
const STRING = 57382
const FLOAT = 57383
const PATH = 57384
const FILE = 57385
const BOOL = 57386
const TRUE = 57387
const FALSE = 57388
const NULL = 57389
const DEFAULT = 57390

var mmToknames = []string{
	"SKIP",
	"INVALID",
	"SEMICOLON",
	"COLON",
	"COMMA",
	"EQUALS",
	"LBRACKET",
	"RBRACKET",
	"LPAREN",
	"RPAREN",
	"LBRACE",
	"RBRACE",
	"FILETYPE",
	"STAGE",
	"PIPELINE",
	"CALL",
	"VOLATILE",
	"SWEEP",
	"SPLIT",
	"USING",
	"SELF",
	"RETURN",
	"IN",
	"OUT",
	"SRC",
	"ID",
	"LITSTRING",
	"NUM_FLOAT",
	"NUM_INT",
	"DOT",
	"PY",
	"GO",
	"SH",
	"EXEC",
	"MAP",
	"INT",
	"STRING",
	"FLOAT",
	"PATH",
	"FILE",
	"BOOL",
	"TRUE",
	"FALSE",
	"NULL",
	"DEFAULT",
}
var mmStatenames = []string{}

const mmEofCode = 1
const mmErrCode = 2
const mmMaxDepth = 200

//line src/mario/core/grammar.y:281

//line yacctab:1
var mmExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
}

const mmNprod = 64
const mmPrivate = 57344

var mmTokenNames []string
var mmStates []string

const mmLast = 167

var mmAct = []int{

	78, 23, 26, 107, 3, 76, 71, 9, 53, 77,
	42, 68, 54, 85, 69, 84, 72, 20, 79, 118,
	110, 102, 65, 101, 88, 27, 31, 64, 59, 57,
	58, 133, 90, 81, 104, 53, 103, 51, 92, 54,
	55, 56, 109, 60, 61, 62, 52, 30, 37, 65,
	49, 91, 70, 28, 64, 59, 57, 58, 25, 48,
	43, 44, 46, 45, 30, 47, 12, 55, 56, 30,
	60, 61, 62, 18, 16, 11, 15, 89, 94, 41,
	40, 95, 14, 32, 34, 53, 50, 5, 110, 54,
	90, 125, 105, 115, 87, 112, 5, 117, 114, 65,
	41, 119, 132, 96, 64, 59, 57, 58, 116, 108,
	109, 35, 123, 73, 121, 34, 98, 55, 56, 120,
	60, 61, 62, 99, 66, 129, 36, 126, 130, 131,
	6, 7, 8, 5, 122, 96, 83, 82, 97, 19,
	75, 24, 22, 21, 17, 111, 127, 124, 106, 74,
	128, 100, 4, 1, 113, 10, 29, 93, 80, 63,
	38, 86, 39, 33, 2, 67, 13,
}
var mmPact = []int{

	114, -1000, 114, -1000, -1000, 46, 53, 47, 45, -1000,
	-1000, 132, 44, 133, -16, 131, 130, -1000, 129, -1000,
	29, -1000, -1000, 40, -1000, -1000, 58, 58, -1000, -1000,
	117, 35, 52, -1000, 21, 73, 25, -1000, 111, -1000,
	-23, 21, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -17,
	99, 141, 128, -2, 3, 125, 124, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -18, -20, 72, -6, -1000, -1000,
	-1000, 22, 9, 77, -1000, 75, 127, -1000, -1000, -1000,
	108, 144, -7, -9, 7, 5, -1000, 69, 140, 80,
	134, 12, -1000, 68, -1000, 95, 75, -1000, -11, -1000,
	75, 106, 101, -1000, -1000, 122, -1000, -1000, 12, 139,
	-1000, -1000, -1000, 76, -1000, 115, 138, -1000, 143, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, 75, 89,
	18, -1000, -1000, -1000,
}
var mmPgo = []int{

	0, 166, 10, 3, 165, 6, 152, 164, 163, 162,
	2, 83, 161, 160, 0, 159, 5, 158, 4, 157,
	156, 1, 154, 153,
}
var mmR1 = []int{

	0, 23, 23, 23, 7, 7, 6, 6, 6, 6,
	1, 1, 5, 5, 10, 10, 8, 11, 11, 9,
	9, 13, 3, 3, 2, 2, 2, 2, 2, 2,
	2, 2, 4, 4, 12, 22, 19, 19, 18, 18,
	21, 21, 20, 20, 16, 16, 17, 17, 14, 14,
	14, 14, 14, 14, 14, 14, 14, 14, 14, 14,
	14, 15, 15, 15,
}
var mmR2 = []int{

	0, 1, 2, 1, 2, 1, 3, 7, 8, 10,
	3, 1, 0, 3, 0, 2, 5, 0, 2, 4,
	5, 4, 2, 1, 1, 1, 1, 1, 1, 1,
	1, 3, 1, 1, 5, 4, 2, 1, 5, 6,
	0, 2, 4, 7, 3, 1, 5, 3, 3, 2,
	2, 3, 4, 4, 1, 1, 1, 1, 1, 1,
	1, 3, 1, 3,
}
var mmChk = []int{

	-1000, -23, -7, -18, -6, 19, 16, 17, 18, -18,
	-6, 29, 20, -1, 29, 29, 29, 12, 29, 6,
	33, 12, 12, -21, 12, 29, -10, -10, 13, -20,
	29, -21, -11, -8, 26, -11, 9, 13, -13, -9,
	28, 27, -2, 39, 40, 42, 41, 44, 38, 29,
	13, -14, 21, 10, 14, 42, 43, 31, 32, 30,
	45, 46, 47, -15, 29, 24, 13, -4, 34, 37,
	-2, -5, 33, 14, 8, 12, -16, 11, -14, 15,
	-17, 30, 12, 12, 33, 33, -12, 22, 30, -5,
	10, 29, 29, -19, -18, -16, 8, 11, 8, 15,
	7, 30, 30, 29, 29, 23, 8, -3, 29, 30,
	8, 11, -3, -22, -18, 25, 13, -14, 30, -14,
	13, 13, 12, -3, 8, 15, 12, 8, 7, -10,
	-21, -14, 13, 13,
}
var mmDef = []int{

	0, -2, 1, 3, 5, 0, 0, 0, 0, 2,
	4, 0, 0, 0, 11, 0, 0, 40, 0, 6,
	0, 14, 14, 0, 40, 10, 17, 17, 38, 41,
	0, 0, 0, 15, 0, 0, 0, 39, 0, 18,
	0, 0, 12, 24, 25, 26, 27, 28, 29, 30,
	0, 0, 0, 0, 0, 0, 0, 54, 55, 56,
	57, 58, 59, 60, 62, 0, 7, 0, 32, 33,
	12, 0, 0, 0, 42, 0, 0, 49, 45, 50,
	0, 0, 0, 0, 0, 0, 8, 0, 0, 0,
	0, 0, 31, 0, 37, 0, 0, 48, 0, 51,
	0, 0, 0, 61, 63, 0, 21, 19, 0, 0,
	23, 13, 16, 0, 36, 0, 0, 44, 0, 47,
	52, 53, 14, 20, 22, 9, 40, 43, 0, 0,
	0, 46, 34, 35,
}
var mmTok1 = []int{

	1,
}
var mmTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
	32, 33, 34, 35, 36, 37, 38, 39, 40, 41,
	42, 43, 44, 45, 46, 47, 48,
}
var mmTok3 = []int{
	0,
}

//line yaccpar:1

/*	parser for yacc output	*/

var mmDebug = 0

type mmLexer interface {
	Lex(lval *mmSymType) int
	Error(s string)
}

const mmFlag = -1000

func mmTokname(c int) string {
	// 4 is TOKSTART above
	if c >= 4 && c-4 < len(mmToknames) {
		if mmToknames[c-4] != "" {
			return mmToknames[c-4]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func mmStatname(s int) string {
	if s >= 0 && s < len(mmStatenames) {
		if mmStatenames[s] != "" {
			return mmStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func mmlex1(lex mmLexer, lval *mmSymType) int {
	c := 0
	char := lex.Lex(lval)
	if char <= 0 {
		c = mmTok1[0]
		goto out
	}
	if char < len(mmTok1) {
		c = mmTok1[char]
		goto out
	}
	if char >= mmPrivate {
		if char < mmPrivate+len(mmTok2) {
			c = mmTok2[char-mmPrivate]
			goto out
		}
	}
	for i := 0; i < len(mmTok3); i += 2 {
		c = mmTok3[i+0]
		if c == char {
			c = mmTok3[i+1]
			goto out
		}
	}

out:
	if c == 0 {
		c = mmTok2[1] /* unknown char */
	}
	if mmDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", mmTokname(c), uint(char))
	}
	return c
}

func mmParse(mmlex mmLexer) int {
	var mmn int
	var mmlval mmSymType
	var mmVAL mmSymType
	mmS := make([]mmSymType, mmMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	mmstate := 0
	mmchar := -1
	mmp := -1
	goto mmstack

ret0:
	return 0

ret1:
	return 1

mmstack:
	/* put a state and value onto the stack */
	if mmDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", mmTokname(mmchar), mmStatname(mmstate))
	}

	mmp++
	if mmp >= len(mmS) {
		nyys := make([]mmSymType, len(mmS)*2)
		copy(nyys, mmS)
		mmS = nyys
	}
	mmS[mmp] = mmVAL
	mmS[mmp].yys = mmstate

mmnewstate:
	mmn = mmPact[mmstate]
	if mmn <= mmFlag {
		goto mmdefault /* simple state */
	}
	if mmchar < 0 {
		mmchar = mmlex1(mmlex, &mmlval)
	}
	mmn += mmchar
	if mmn < 0 || mmn >= mmLast {
		goto mmdefault
	}
	mmn = mmAct[mmn]
	if mmChk[mmn] == mmchar { /* valid shift */
		mmchar = -1
		mmVAL = mmlval
		mmstate = mmn
		if Errflag > 0 {
			Errflag--
		}
		goto mmstack
	}

mmdefault:
	/* default state action */
	mmn = mmDef[mmstate]
	if mmn == -2 {
		if mmchar < 0 {
			mmchar = mmlex1(mmlex, &mmlval)
		}

		/* look through exception table */
		xi := 0
		for {
			if mmExca[xi+0] == -1 && mmExca[xi+1] == mmstate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			mmn = mmExca[xi+0]
			if mmn < 0 || mmn == mmchar {
				break
			}
		}
		mmn = mmExca[xi+1]
		if mmn < 0 {
			goto ret0
		}
	}
	if mmn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			mmlex.Error("syntax error")
			Nerrs++
			if mmDebug >= 1 {
				__yyfmt__.Printf("%s", mmStatname(mmstate))
				__yyfmt__.Printf(" saw %s\n", mmTokname(mmchar))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for mmp >= 0 {
				mmn = mmPact[mmS[mmp].yys] + mmErrCode
				if mmn >= 0 && mmn < mmLast {
					mmstate = mmAct[mmn] /* simulate a shift of "error" */
					if mmChk[mmstate] == mmErrCode {
						goto mmstack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if mmDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", mmS[mmp].yys)
				}
				mmp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if mmDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", mmTokname(mmchar))
			}
			if mmchar == mmEofCode {
				goto ret1
			}
			mmchar = -1
			goto mmnewstate /* try again in the same state */
		}
	}

	/* reduction by production mmn */
	if mmDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", mmn, mmStatname(mmstate))
	}

	mmnt := mmn
	mmpt := mmp
	_ = mmpt // guard against "declared and not used"

	mmp -= mmR2[mmn]
	mmVAL = mmS[mmp+1]

	/* consult goto table to find next state */
	mmn = mmR1[mmn]
	mmg := mmPgo[mmn]
	mmj := mmg + mmS[mmp].yys + 1

	if mmj >= mmLast {
		mmstate = mmAct[mmg]
	} else {
		mmstate = mmAct[mmj]
		if mmChk[mmstate] != -mmn {
			mmstate = mmAct[mmg]
		}
	}
	// dummy call; replaced with literal code
	switch mmnt {

	case 1:
		//line src/mario/core/grammar.y:70
		{
			{
				global := NewAst(mmS[mmpt-0].decs, nil)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 2:
		//line src/mario/core/grammar.y:75
		{
			{
				global := NewAst(mmS[mmpt-1].decs, mmS[mmpt-0].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 3:
		//line src/mario/core/grammar.y:80
		{
			{
				global := NewAst([]Dec{}, mmS[mmpt-0].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 4:
		//line src/mario/core/grammar.y:88
		{
			{
				mmVAL.decs = append(mmS[mmpt-1].decs, mmS[mmpt-0].dec)
			}
		}
	case 5:
		//line src/mario/core/grammar.y:90
		{
			{
				mmVAL.decs = []Dec{mmS[mmpt-0].dec}
			}
		}
	case 6:
		//line src/mario/core/grammar.y:95
		{
			{
				mmVAL.dec = &Filetype{NewAstNode(&mmlval), mmS[mmpt-1].val}
			}
		}
	case 7:
		//line src/mario/core/grammar.y:97
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-5].val, mmS[mmpt-3].params, mmS[mmpt-2].params, mmS[mmpt-1].src, &Params{[]Param{}, map[string]Param{}}}
			}
		}
	case 8:
		//line src/mario/core/grammar.y:99
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-6].val, mmS[mmpt-4].params, mmS[mmpt-3].params, mmS[mmpt-2].src, mmS[mmpt-0].params}
			}
		}
	case 9:
		//line src/mario/core/grammar.y:101
		{
			{
				mmVAL.dec = &Pipeline{NewAstNode(&mmlval), mmS[mmpt-8].val, mmS[mmpt-6].params, mmS[mmpt-5].params, mmS[mmpt-2].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmS[mmpt-1].retstm}
			}
		}
	case 10:
		//line src/mario/core/grammar.y:106
		{
			{
				mmVAL.val = mmS[mmpt-2].val + mmS[mmpt-1].val + mmS[mmpt-0].val
			}
		}
	case 11:
		mmVAL.val = mmS[mmpt-0].val
	case 12:
		//line src/mario/core/grammar.y:112
		{
			{
				mmVAL.arr = 0
			}
		}
	case 13:
		//line src/mario/core/grammar.y:114
		{
			{
				mmVAL.arr += 1
			}
		}
	case 14:
		//line src/mario/core/grammar.y:119
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 15:
		//line src/mario/core/grammar.y:121
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].inparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 16:
		//line src/mario/core/grammar.y:129
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(&mmlval), mmS[mmpt-3].val, mmS[mmpt-2].arr, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 17:
		//line src/mario/core/grammar.y:134
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 18:
		//line src/mario/core/grammar.y:136
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].outparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 19:
		//line src/mario/core/grammar.y:144
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-2].val, mmS[mmpt-1].arr, "default", unquote(mmS[mmpt-0].val), false}
			}
		}
	case 20:
		//line src/mario/core/grammar.y:146
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-3].val, mmS[mmpt-2].arr, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 21:
		//line src/mario/core/grammar.y:151
		{
			{
				mmVAL.src = &SrcParam{NewAstNode(&mmlval), mmS[mmpt-2].val, unquote(mmS[mmpt-1].val)}
			}
		}
	case 22:
		//line src/mario/core/grammar.y:156
		{
			{
				mmVAL.val = mmS[mmpt-1].val
			}
		}
	case 23:
		//line src/mario/core/grammar.y:158
		{
			{
				mmVAL.val = ""
			}
		}
	case 24:
		mmVAL.val = mmS[mmpt-0].val
	case 25:
		mmVAL.val = mmS[mmpt-0].val
	case 26:
		mmVAL.val = mmS[mmpt-0].val
	case 27:
		mmVAL.val = mmS[mmpt-0].val
	case 28:
		mmVAL.val = mmS[mmpt-0].val
	case 29:
		mmVAL.val = mmS[mmpt-0].val
	case 30:
		mmVAL.val = mmS[mmpt-0].val
	case 31:
		//line src/mario/core/grammar.y:170
		{
			{
				mmVAL.val = mmS[mmpt-2].val + "." + mmS[mmpt-0].val
			}
		}
	case 32:
		mmVAL.val = mmS[mmpt-0].val
	case 33:
		mmVAL.val = mmS[mmpt-0].val
	case 34:
		//line src/mario/core/grammar.y:182
		{
			{
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 35:
		//line src/mario/core/grammar.y:187
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(&mmlval), mmS[mmpt-1].bindings}
			}
		}
	case 36:
		//line src/mario/core/grammar.y:192
		{
			{
				mmVAL.calls = append(mmS[mmpt-1].calls, mmS[mmpt-0].call)
			}
		}
	case 37:
		//line src/mario/core/grammar.y:194
		{
			{
				mmVAL.calls = []*CallStm{mmS[mmpt-0].call}
			}
		}
	case 38:
		//line src/mario/core/grammar.y:199
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), false, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 39:
		//line src/mario/core/grammar.y:201
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), true, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 40:
		//line src/mario/core/grammar.y:206
		{
			{
				mmVAL.bindings = &BindStms{[]*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 41:
		//line src/mario/core/grammar.y:208
		{
			{
				mmS[mmpt-1].bindings.list = append(mmS[mmpt-1].bindings.list, mmS[mmpt-0].binding)
				mmVAL.bindings = mmS[mmpt-1].bindings
			}
		}
	case 42:
		//line src/mario/core/grammar.y:216
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-3].val, mmS[mmpt-1].exp, false, ""}
			}
		}
	case 43:
		//line src/mario/core/grammar.y:218
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-6].val, &ValExp{node: NewAstNode(&mmlval), kind: "array", value: mmS[mmpt-2].exps}, true, ""}
			}
		}
	case 44:
		//line src/mario/core/grammar.y:223
		{
			{
				mmVAL.exps = append(mmS[mmpt-2].exps, mmS[mmpt-0].exp)
			}
		}
	case 45:
		//line src/mario/core/grammar.y:225
		{
			{
				mmVAL.exps = []Exp{mmS[mmpt-0].exp}
			}
		}
	case 46:
		//line src/mario/core/grammar.y:230
		{
			{
				mmS[mmpt-4].kvpairs[unquote(mmS[mmpt-2].val)] = mmS[mmpt-0].exp
				mmVAL.kvpairs = mmS[mmpt-4].kvpairs
			}
		}
	case 47:
		//line src/mario/core/grammar.y:235
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmS[mmpt-2].val): mmS[mmpt-0].exp}
			}
		}
	case 48:
		//line src/mario/core/grammar.y:240
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "array", value: mmS[mmpt-1].exps}
			}
		}
	case 49:
		//line src/mario/core/grammar.y:242
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "array", value: []Exp{}}
			}
		}
	case 50:
		//line src/mario/core/grammar.y:244
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "map", value: map[string]interface{}{}}
			}
		}
	case 51:
		//line src/mario/core/grammar.y:246
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "map", value: mmS[mmpt-1].kvpairs}
			}
		}
	case 52:
		//line src/mario/core/grammar.y:248
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: mmS[mmpt-3].val, value: unquote(mmS[mmpt-1].val)}
			}
		}
	case 53:
		//line src/mario/core/grammar.y:250
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: mmS[mmpt-3].val, value: unquote(mmS[mmpt-1].val)}
			}
		}
	case 54:
		//line src/mario/core/grammar.y:252
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmS[mmpt-0].val, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "float", value: f}
			}
		}
	case 55:
		//line src/mario/core/grammar.y:257
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmS[mmpt-0].val, 0, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "int", value: i}
			}
		}
	case 56:
		//line src/mario/core/grammar.y:262
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "string", value: unquote(mmS[mmpt-0].val)}
			}
		}
	case 57:
		//line src/mario/core/grammar.y:264
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "bool", value: true}
			}
		}
	case 58:
		//line src/mario/core/grammar.y:266
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "bool", value: false}
			}
		}
	case 59:
		//line src/mario/core/grammar.y:268
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "null", value: nil}
			}
		}
	case 60:
		//line src/mario/core/grammar.y:270
		{
			{
				mmVAL.exp = mmS[mmpt-0].exp
			}
		}
	case 61:
		//line src/mario/core/grammar.y:275
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-2].val, mmS[mmpt-0].val}
			}
		}
	case 62:
		//line src/mario/core/grammar.y:277
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-0].val, "default"}
			}
		}
	case 63:
		//line src/mario/core/grammar.y:279
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "self", mmS[mmpt-0].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

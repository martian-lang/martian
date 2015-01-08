//line src/martian/core/grammar.y:2

//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO grammar.
//
package core

import __yyfmt__ "fmt"

//line src/martian/core/grammar.y:7
import (
	"strconv"
	"strings"
)

func unquote(qs string) string {
	return strings.Replace(qs, "\"", "", -1)
}

//line src/martian/core/grammar.y:19
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
const LOCAL = 57362
const VOLATILE = 57363
const SWEEP = 57364
const SPLIT = 57365
const USING = 57366
const SELF = 57367
const RETURN = 57368
const IN = 57369
const OUT = 57370
const SRC = 57371
const ID = 57372
const LITSTRING = 57373
const NUM_FLOAT = 57374
const NUM_INT = 57375
const DOT = 57376
const PY = 57377
const GO = 57378
const SH = 57379
const EXEC = 57380
const MAP = 57381
const INT = 57382
const STRING = 57383
const FLOAT = 57384
const PATH = 57385
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
	"LOCAL",
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

//line src/martian/core/grammar.y:282

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

	86, 26, 31, 111, 3, 84, 79, 9, 58, 91,
	90, 62, 85, 51, 80, 63, 23, 57, 52, 53,
	55, 54, 56, 39, 76, 135, 72, 77, 32, 36,
	37, 71, 66, 64, 65, 87, 122, 62, 94, 114,
	46, 63, 35, 108, 60, 107, 67, 68, 69, 61,
	73, 89, 72, 13, 12, 96, 42, 71, 66, 64,
	65, 62, 113, 11, 78, 63, 98, 35, 114, 30,
	96, 45, 67, 68, 69, 97, 72, 44, 33, 29,
	19, 71, 66, 64, 65, 95, 100, 17, 35, 101,
	112, 113, 16, 21, 35, 35, 67, 68, 69, 50,
	49, 116, 20, 121, 118, 15, 59, 123, 134, 41,
	5, 109, 93, 5, 127, 81, 125, 119, 74, 43,
	128, 50, 41, 6, 7, 8, 5, 131, 104, 102,
	132, 133, 130, 115, 120, 105, 124, 102, 83, 38,
	103, 22, 28, 27, 25, 24, 18, 129, 126, 110,
	82, 106, 4, 1, 117, 10, 34, 99, 88, 70,
	47, 92, 48, 40, 2, 75, 14,
}
var mmPact = []int{

	107, -1000, 107, -1000, -1000, 33, 75, 62, 57, -1000,
	-1000, 134, 50, 72, 135, -18, 133, 132, -1000, 131,
	130, 49, -1000, 39, -1000, -1000, 65, -1000, -1000, 127,
	-1000, 82, 82, -1000, -1000, 110, 64, 58, -1000, 71,
	-1000, -22, 93, 27, -1000, -1000, 37, 105, -1000, -11,
	-22, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -20, 101,
	142, 126, 1, 20, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -24, -25, -1000, 89, 7, -1000, -1000, -1000, 45,
	36, 94, -1000, 51, 129, -1000, -1000, -1000, 120, 144,
	15, 13, -1000, 87, 141, 60, 122, 31, -1000, 91,
	-1000, 121, 51, -1000, 5, -1000, 51, -1000, -1000, 124,
	-1000, -1000, 31, 140, -1000, -1000, -1000, 99, -1000, 108,
	139, -1000, 125, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	51, 95, 12, -1000, -1000, -1000,
}
var mmPgo = []int{

	0, 166, 13, 3, 165, 6, 152, 164, 163, 162,
	2, 23, 161, 160, 0, 159, 5, 158, 4, 157,
	156, 1, 154, 153,
}
var mmR1 = []int{

	0, 23, 23, 23, 7, 7, 6, 6, 6, 6,
	1, 1, 5, 5, 10, 10, 8, 11, 11, 9,
	9, 13, 3, 3, 2, 2, 2, 2, 2, 2,
	2, 2, 4, 4, 12, 22, 19, 19, 18, 18,
	18, 18, 21, 21, 20, 20, 16, 16, 17, 17,
	14, 14, 14, 14, 14, 14, 14, 14, 14, 14,
	14, 15, 15, 15,
}
var mmR2 = []int{

	0, 1, 2, 1, 2, 1, 3, 7, 8, 10,
	3, 1, 0, 3, 0, 2, 5, 0, 2, 4,
	5, 4, 2, 1, 1, 1, 1, 1, 1, 1,
	1, 3, 1, 1, 5, 4, 2, 1, 5, 6,
	6, 7, 0, 2, 4, 7, 3, 1, 5, 3,
	3, 2, 2, 3, 1, 1, 1, 1, 1, 1,
	1, 3, 1, 3,
}
var mmChk = []int{

	-1000, -23, -7, -18, -6, 19, 16, 17, 18, -18,
	-6, 30, 21, 20, -1, 30, 30, 30, 12, 30,
	30, 21, 6, 34, 12, 12, -21, 12, 12, 30,
	30, -10, -10, 13, -20, 30, -21, -21, 12, -11,
	-8, 27, -11, 9, 13, 13, -21, -13, -9, 29,
	28, -2, 40, 41, 43, 42, 44, 39, 30, 13,
	-14, 22, 10, 14, 32, 33, 31, 45, 46, 47,
	-15, 30, 25, 13, 13, -4, 35, 38, -2, -5,
	34, 14, 8, 12, -16, 11, -14, 15, -17, 31,
	34, 34, -12, 23, 31, -5, 10, 30, 30, -19,
	-18, -16, 8, 11, 8, 15, 7, 30, 30, 24,
	8, -3, 30, 31, 8, 11, -3, -22, -18, 26,
	13, -14, 31, -14, 12, -3, 8, 15, 12, 8,
	7, -10, -21, -14, 13, 13,
}
var mmDef = []int{

	0, -2, 1, 3, 5, 0, 0, 0, 0, 2,
	4, 0, 0, 0, 0, 11, 0, 0, 42, 0,
	0, 0, 6, 0, 14, 14, 0, 42, 42, 0,
	10, 17, 17, 38, 43, 0, 0, 0, 42, 0,
	15, 0, 0, 0, 39, 40, 0, 0, 18, 0,
	0, 12, 24, 25, 26, 27, 28, 29, 30, 0,
	0, 0, 0, 0, 54, 55, 56, 57, 58, 59,
	60, 62, 0, 41, 7, 0, 32, 33, 12, 0,
	0, 0, 44, 0, 0, 51, 47, 52, 0, 0,
	0, 0, 8, 0, 0, 0, 0, 0, 31, 0,
	37, 0, 0, 50, 0, 53, 0, 61, 63, 0,
	21, 19, 0, 0, 23, 13, 16, 0, 36, 0,
	0, 46, 0, 49, 14, 20, 22, 9, 42, 45,
	0, 0, 0, 48, 34, 35,
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
		//line src/martian/core/grammar.y:70
		{
			{
				global := NewAst(mmS[mmpt-0].decs, nil)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 2:
		//line src/martian/core/grammar.y:75
		{
			{
				global := NewAst(mmS[mmpt-1].decs, mmS[mmpt-0].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 3:
		//line src/martian/core/grammar.y:80
		{
			{
				global := NewAst([]Dec{}, mmS[mmpt-0].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 4:
		//line src/martian/core/grammar.y:88
		{
			{
				mmVAL.decs = append(mmS[mmpt-1].decs, mmS[mmpt-0].dec)
			}
		}
	case 5:
		//line src/martian/core/grammar.y:90
		{
			{
				mmVAL.decs = []Dec{mmS[mmpt-0].dec}
			}
		}
	case 6:
		//line src/martian/core/grammar.y:95
		{
			{
				mmVAL.dec = &Filetype{NewAstNode(&mmlval), mmS[mmpt-1].val}
			}
		}
	case 7:
		//line src/martian/core/grammar.y:97
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-5].val, mmS[mmpt-3].params, mmS[mmpt-2].params, mmS[mmpt-1].src, &Params{[]Param{}, map[string]Param{}}}
			}
		}
	case 8:
		//line src/martian/core/grammar.y:99
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-6].val, mmS[mmpt-4].params, mmS[mmpt-3].params, mmS[mmpt-2].src, mmS[mmpt-0].params}
			}
		}
	case 9:
		//line src/martian/core/grammar.y:101
		{
			{
				mmVAL.dec = &Pipeline{NewAstNode(&mmlval), mmS[mmpt-8].val, mmS[mmpt-6].params, mmS[mmpt-5].params, mmS[mmpt-2].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmS[mmpt-1].retstm}
			}
		}
	case 10:
		//line src/martian/core/grammar.y:106
		{
			{
				mmVAL.val = mmS[mmpt-2].val + mmS[mmpt-1].val + mmS[mmpt-0].val
			}
		}
	case 11:
		mmVAL.val = mmS[mmpt-0].val
	case 12:
		//line src/martian/core/grammar.y:112
		{
			{
				mmVAL.arr = 0
			}
		}
	case 13:
		//line src/martian/core/grammar.y:114
		{
			{
				mmVAL.arr += 1
			}
		}
	case 14:
		//line src/martian/core/grammar.y:119
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 15:
		//line src/martian/core/grammar.y:121
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].inparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 16:
		//line src/martian/core/grammar.y:129
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(&mmlval), mmS[mmpt-3].val, mmS[mmpt-2].arr, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 17:
		//line src/martian/core/grammar.y:134
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 18:
		//line src/martian/core/grammar.y:136
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].outparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 19:
		//line src/martian/core/grammar.y:144
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-2].val, mmS[mmpt-1].arr, "default", unquote(mmS[mmpt-0].val), false}
			}
		}
	case 20:
		//line src/martian/core/grammar.y:146
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-3].val, mmS[mmpt-2].arr, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 21:
		//line src/martian/core/grammar.y:151
		{
			{
				stagecodeParts := strings.Split(unquote(mmS[mmpt-1].val), " ")
				mmVAL.src = &SrcParam{NewAstNode(&mmlval), mmS[mmpt-2].val, stagecodeParts[0], stagecodeParts[1:]}
			}
		}
	case 22:
		//line src/martian/core/grammar.y:157
		{
			{
				mmVAL.val = mmS[mmpt-1].val
			}
		}
	case 23:
		//line src/martian/core/grammar.y:159
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
		//line src/martian/core/grammar.y:171
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
		//line src/martian/core/grammar.y:183
		{
			{
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 35:
		//line src/martian/core/grammar.y:188
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(&mmlval), mmS[mmpt-1].bindings}
			}
		}
	case 36:
		//line src/martian/core/grammar.y:193
		{
			{
				mmVAL.calls = append(mmS[mmpt-1].calls, mmS[mmpt-0].call)
			}
		}
	case 37:
		//line src/martian/core/grammar.y:195
		{
			{
				mmVAL.calls = []*CallStm{mmS[mmpt-0].call}
			}
		}
	case 38:
		//line src/martian/core/grammar.y:200
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), false, false, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 39:
		//line src/martian/core/grammar.y:202
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), true, false, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 40:
		//line src/martian/core/grammar.y:204
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), false, true, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 41:
		//line src/martian/core/grammar.y:206
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), true, true, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 42:
		//line src/martian/core/grammar.y:211
		{
			{
				mmVAL.bindings = &BindStms{[]*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 43:
		//line src/martian/core/grammar.y:213
		{
			{
				mmS[mmpt-1].bindings.list = append(mmS[mmpt-1].bindings.list, mmS[mmpt-0].binding)
				mmVAL.bindings = mmS[mmpt-1].bindings
			}
		}
	case 44:
		//line src/martian/core/grammar.y:221
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-3].val, mmS[mmpt-1].exp, false, ""}
			}
		}
	case 45:
		//line src/martian/core/grammar.y:223
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-6].val, &ValExp{node: NewAstNode(&mmlval), kind: "array", value: mmS[mmpt-2].exps}, true, ""}
			}
		}
	case 46:
		//line src/martian/core/grammar.y:228
		{
			{
				mmVAL.exps = append(mmS[mmpt-2].exps, mmS[mmpt-0].exp)
			}
		}
	case 47:
		//line src/martian/core/grammar.y:230
		{
			{
				mmVAL.exps = []Exp{mmS[mmpt-0].exp}
			}
		}
	case 48:
		//line src/martian/core/grammar.y:235
		{
			{
				mmS[mmpt-4].kvpairs[unquote(mmS[mmpt-2].val)] = mmS[mmpt-0].exp
				mmVAL.kvpairs = mmS[mmpt-4].kvpairs
			}
		}
	case 49:
		//line src/martian/core/grammar.y:240
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmS[mmpt-2].val): mmS[mmpt-0].exp}
			}
		}
	case 50:
		//line src/martian/core/grammar.y:245
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "array", value: mmS[mmpt-1].exps}
			}
		}
	case 51:
		//line src/martian/core/grammar.y:247
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "array", value: []Exp{}}
			}
		}
	case 52:
		//line src/martian/core/grammar.y:249
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "map", value: map[string]interface{}{}}
			}
		}
	case 53:
		//line src/martian/core/grammar.y:251
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "map", value: mmS[mmpt-1].kvpairs}
			}
		}
	case 54:
		//line src/martian/core/grammar.y:253
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmS[mmpt-0].val, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "float", value: f}
			}
		}
	case 55:
		//line src/martian/core/grammar.y:258
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmS[mmpt-0].val, 0, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "int", value: i}
			}
		}
	case 56:
		//line src/martian/core/grammar.y:263
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "string", value: unquote(mmS[mmpt-0].val)}
			}
		}
	case 57:
		//line src/martian/core/grammar.y:265
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "bool", value: true}
			}
		}
	case 58:
		//line src/martian/core/grammar.y:267
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "bool", value: false}
			}
		}
	case 59:
		//line src/martian/core/grammar.y:269
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "null", value: nil}
			}
		}
	case 60:
		//line src/martian/core/grammar.y:271
		{
			{
				mmVAL.exp = mmS[mmpt-0].exp
			}
		}
	case 61:
		//line src/martian/core/grammar.y:276
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-2].val, mmS[mmpt-0].val}
			}
		}
	case 62:
		//line src/martian/core/grammar.y:278
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-0].val, "default"}
			}
		}
	case 63:
		//line src/martian/core/grammar.y:280
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "self", mmS[mmpt-0].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

//line core/grammar.y:2

//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// MRO grammar.
//
package core

import __yyfmt__ "fmt"

//line core/grammar.y:7
import (
	"strconv"
	"strings"
)

func unquote(qs string) string {
	return strings.Replace(qs, "\"", "", -1)
}

//line core/grammar.y:19
type mmSymType struct {
	yys      int
	global   *Ast
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

//line core/grammar.y:278

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

const mmLast = 183

var mmAct = []int{

	24, 23, 29, 77, 88, 75, 87, 3, 84, 28,
	9, 83, 71, 20, 118, 78, 109, 92, 104, 40,
	103, 51, 76, 92, 33, 52, 38, 25, 35, 36,
	80, 37, 31, 113, 37, 63, 33, 47, 49, 91,
	62, 57, 55, 56, 125, 91, 46, 41, 42, 44,
	43, 51, 45, 53, 54, 52, 58, 59, 60, 68,
	140, 48, 50, 66, 106, 63, 70, 105, 95, 12,
	62, 57, 55, 56, 93, 66, 25, 51, 11, 64,
	97, 52, 27, 53, 54, 69, 58, 59, 60, 39,
	67, 63, 32, 18, 110, 25, 62, 57, 55, 56,
	16, 15, 117, 92, 115, 90, 119, 14, 25, 53,
	54, 30, 58, 59, 60, 72, 124, 5, 126, 30,
	39, 139, 128, 129, 89, 91, 122, 108, 5, 39,
	133, 100, 134, 96, 30, 136, 98, 138, 101, 33,
	37, 116, 137, 6, 7, 8, 5, 121, 120, 85,
	135, 132, 98, 82, 81, 99, 19, 74, 26, 22,
	21, 17, 111, 94, 34, 130, 123, 112, 73, 131,
	102, 4, 1, 127, 10, 114, 79, 61, 65, 107,
	2, 86, 13,
}
var mmPact = []int{

	127, -1000, 127, -1000, -1000, 49, 78, 72, 71, -1000,
	-1000, 149, 64, 150, -20, 148, 147, -2, 146, -1000,
	53, 85, 85, 79, -1000, 155, -2, -1000, 93, -1000,
	8, 93, -1000, -1000, 41, 66, 62, -1000, -1000, 8,
	56, -1000, -1000, -1000, -1000, -1000, -1000, -21, 102, 160,
	145, 11, 0, 142, 141, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -22, -25, -1000, 136, -1000, -28, 95, 9,
	152, 39, 119, -1000, 67, 144, -1000, -1000, -1000, 123,
	163, -10, -12, 38, 35, 105, -14, -1000, -1000, 9,
	151, 159, -1000, -1000, 4, -1000, 109, 128, 67, -1000,
	-16, -1000, 67, 135, 134, -1000, -1000, -1000, 103, 158,
	-1000, 15, -1000, 9, 98, -1000, 157, -1000, 162, -1000,
	-1000, -1000, 139, -1000, -1000, 9, -1000, 117, -1000, 138,
	-1000, 67, 85, -1000, -1000, -2, -1000, 108, 47, -1000,
	-1000,
}
var mmPgo = []int{

	0, 182, 19, 4, 181, 171, 180, 2, 26, 9,
	29, 179, 178, 3, 177, 5, 176, 7, 175, 0,
	1, 173, 172,
}
var mmR1 = []int{

	0, 22, 22, 22, 6, 6, 5, 5, 5, 5,
	1, 1, 9, 9, 7, 7, 10, 10, 8, 8,
	8, 8, 12, 3, 3, 2, 2, 2, 2, 2,
	2, 2, 2, 4, 11, 21, 18, 18, 17, 17,
	20, 20, 19, 19, 15, 15, 16, 16, 13, 13,
	13, 13, 13, 13, 13, 13, 13, 13, 13, 13,
	13, 14, 14, 14,
}
var mmR2 = []int{

	0, 1, 2, 1, 2, 1, 3, 7, 8, 10,
	3, 1, 2, 1, 4, 6, 2, 1, 3, 4,
	5, 6, 4, 2, 1, 1, 1, 1, 1, 1,
	1, 1, 3, 1, 5, 4, 2, 1, 5, 6,
	2, 1, 4, 7, 3, 1, 5, 3, 3, 2,
	2, 3, 4, 4, 1, 1, 1, 1, 1, 1,
	1, 3, 1, 3,
}
var mmChk = []int{

	-1000, -22, -6, -17, -5, 19, 16, 17, 18, -17,
	-5, 29, 20, -1, 29, 29, 29, 12, 29, 6,
	33, 12, 12, -20, -19, 29, 12, 29, -9, -7,
	26, -9, 13, -19, 9, -20, -10, -7, -8, 27,
	-2, 39, 40, 42, 41, 44, 38, 29, -10, -13,
	21, 10, 14, 42, 43, 31, 32, 30, 45, 46,
	47, -14, 29, 24, 13, -12, -8, 28, -2, 29,
	10, 33, 13, 8, 12, -15, 11, -13, 15, -16,
	30, 12, 12, 33, 33, 13, -4, 34, -3, 29,
	10, 30, 8, -3, 11, 29, 14, -15, 8, 11,
	8, 15, 7, 30, 30, 29, 29, -11, 22, 30,
	-3, 11, 8, 29, -18, -17, 13, -13, 30, -13,
	13, 13, 23, 8, -3, 29, -3, -21, -17, 25,
	8, 7, 12, -3, 15, 12, -13, -9, -20, 13,
	13,
}
var mmDef = []int{

	0, -2, 1, 3, 5, 0, 0, 0, 0, 2,
	4, 0, 0, 0, 11, 0, 0, 0, 0, 6,
	0, 0, 0, 0, 41, 0, 0, 10, 0, 13,
	0, 0, 38, 40, 0, 0, 0, 12, 17, 0,
	0, 25, 26, 27, 28, 29, 30, 31, 0, 0,
	0, 0, 0, 0, 0, 54, 55, 56, 57, 58,
	59, 60, 62, 0, 39, 0, 16, 0, 0, 0,
	0, 0, 0, 42, 0, 0, 49, 45, 50, 0,
	0, 0, 0, 0, 0, 7, 0, 33, 18, 0,
	0, 0, 24, 14, 0, 32, 0, 0, 0, 48,
	0, 51, 0, 0, 0, 61, 63, 8, 0, 0,
	19, 0, 23, 0, 0, 37, 0, 44, 0, 47,
	52, 53, 0, 22, 20, 0, 15, 0, 36, 0,
	43, 0, 0, 21, 9, 0, 46, 0, 0, 34,
	35,
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
		//line core/grammar.y:68
		{
			{
				global := NewAst(mmS[mmpt-0].decs, nil)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 2:
		//line core/grammar.y:73
		{
			{
				global := NewAst(mmS[mmpt-1].decs, mmS[mmpt-0].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 3:
		//line core/grammar.y:78
		{
			{
				global := NewAst([]Dec{}, mmS[mmpt-0].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 4:
		//line core/grammar.y:86
		{
			{
				mmVAL.decs = append(mmS[mmpt-1].decs, mmS[mmpt-0].dec)
			}
		}
	case 5:
		//line core/grammar.y:88
		{
			{
				mmVAL.decs = []Dec{mmS[mmpt-0].dec}
			}
		}
	case 6:
		//line core/grammar.y:93
		{
			{
				mmVAL.dec = &Filetype{NewAstNode(&mmlval), mmS[mmpt-1].val}
			}
		}
	case 7:
		//line core/grammar.y:95
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-5].val, mmS[mmpt-3].params, mmS[mmpt-2].params, mmS[mmpt-1].src, &Params{[]Param{}, map[string]Param{}}}
			}
		}
	case 8:
		//line core/grammar.y:97
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-6].val, mmS[mmpt-4].params, mmS[mmpt-3].params, mmS[mmpt-2].src, mmS[mmpt-0].params}
			}
		}
	case 9:
		//line core/grammar.y:99
		{
			{
				mmVAL.dec = &Pipeline{NewAstNode(&mmlval), mmS[mmpt-8].val, mmS[mmpt-6].params, mmS[mmpt-5].params, mmS[mmpt-2].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmS[mmpt-1].retstm}
			}
		}
	case 10:
		//line core/grammar.y:104
		{
			{
				mmVAL.val = mmS[mmpt-2].val + mmS[mmpt-1].val + mmS[mmpt-0].val
			}
		}
	case 11:
		mmVAL.val = mmS[mmpt-0].val
	case 12:
		//line core/grammar.y:110
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].inparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 13:
		//line core/grammar.y:115
		{
			{
				mmVAL.params = &Params{[]Param{mmS[mmpt-0].inparam}, map[string]Param{}}
			}
		}
	case 14:
		//line core/grammar.y:120
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(&mmlval), mmS[mmpt-2].val, false, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 15:
		//line core/grammar.y:122
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(&mmlval), mmS[mmpt-4].val, true, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 16:
		//line core/grammar.y:127
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].outparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 17:
		//line core/grammar.y:132
		{
			{
				mmVAL.params = &Params{[]Param{mmS[mmpt-0].outparam}, map[string]Param{}}
			}
		}
	case 18:
		//line core/grammar.y:137
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-1].val, false, "default", unquote(mmS[mmpt-0].val), false}
			}
		}
	case 19:
		//line core/grammar.y:139
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-2].val, false, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 20:
		//line core/grammar.y:141
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-3].val, true, "default", unquote(mmS[mmpt-0].val), false}
			}
		}
	case 21:
		//line core/grammar.y:143
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-4].val, true, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 22:
		//line core/grammar.y:148
		{
			{
				mmVAL.src = &SrcParam{NewAstNode(&mmlval), mmS[mmpt-2].val, unquote(mmS[mmpt-1].val)}
			}
		}
	case 23:
		//line core/grammar.y:153
		{
			{
				mmVAL.val = mmS[mmpt-1].val
			}
		}
	case 24:
		//line core/grammar.y:155
		{
			{
				mmVAL.val = ""
			}
		}
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
		mmVAL.val = mmS[mmpt-0].val
	case 32:
		//line core/grammar.y:167
		{
			{
				mmVAL.val = mmS[mmpt-2].val + "." + mmS[mmpt-0].val
			}
		}
	case 33:
		mmVAL.val = mmS[mmpt-0].val
	case 34:
		//line core/grammar.y:179
		{
			{
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 35:
		//line core/grammar.y:184
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(&mmlval), mmS[mmpt-1].bindings}
			}
		}
	case 36:
		//line core/grammar.y:189
		{
			{
				mmVAL.calls = append(mmS[mmpt-1].calls, mmS[mmpt-0].call)
			}
		}
	case 37:
		//line core/grammar.y:191
		{
			{
				mmVAL.calls = []*CallStm{mmS[mmpt-0].call}
			}
		}
	case 38:
		//line core/grammar.y:196
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), false, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 39:
		//line core/grammar.y:198
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), true, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 40:
		//line core/grammar.y:203
		{
			{
				mmS[mmpt-1].bindings.list = append(mmS[mmpt-1].bindings.list, mmS[mmpt-0].binding)
				mmVAL.bindings = mmS[mmpt-1].bindings
			}
		}
	case 41:
		//line core/grammar.y:208
		{
			{
				mmVAL.bindings = &BindStms{[]*BindStm{mmS[mmpt-0].binding}, map[string]*BindStm{}}
			}
		}
	case 42:
		//line core/grammar.y:213
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-3].val, mmS[mmpt-1].exp, false, ""}
			}
		}
	case 43:
		//line core/grammar.y:215
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-6].val, &ValExp{node: NewAstNode(&mmlval), kind: "array", value: mmS[mmpt-2].exps}, true, ""}
			}
		}
	case 44:
		//line core/grammar.y:220
		{
			{
				mmVAL.exps = append(mmS[mmpt-2].exps, mmS[mmpt-0].exp)
			}
		}
	case 45:
		//line core/grammar.y:222
		{
			{
				mmVAL.exps = []Exp{mmS[mmpt-0].exp}
			}
		}
	case 46:
		//line core/grammar.y:227
		{
			{
				mmS[mmpt-4].kvpairs[unquote(mmS[mmpt-2].val)] = mmS[mmpt-0].exp
				mmVAL.kvpairs = mmS[mmpt-4].kvpairs
			}
		}
	case 47:
		//line core/grammar.y:232
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmS[mmpt-2].val): mmS[mmpt-0].exp}
			}
		}
	case 48:
		//line core/grammar.y:237
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "array", value: mmS[mmpt-1].exps}
			}
		}
	case 49:
		//line core/grammar.y:239
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "array", value: []Exp{}}
			}
		}
	case 50:
		//line core/grammar.y:241
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "map", value: map[string]interface{}{}}
			}
		}
	case 51:
		//line core/grammar.y:243
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "map", value: mmS[mmpt-1].kvpairs}
			}
		}
	case 52:
		//line core/grammar.y:245
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: mmS[mmpt-3].val, value: unquote(mmS[mmpt-1].val)}
			}
		}
	case 53:
		//line core/grammar.y:247
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: mmS[mmpt-3].val, value: unquote(mmS[mmpt-1].val)}
			}
		}
	case 54:
		//line core/grammar.y:249
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmS[mmpt-0].val, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "float", value: f}
			}
		}
	case 55:
		//line core/grammar.y:254
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmS[mmpt-0].val, 0, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "int", value: i}
			}
		}
	case 56:
		//line core/grammar.y:259
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "string", value: unquote(mmS[mmpt-0].val)}
			}
		}
	case 57:
		//line core/grammar.y:261
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "bool", value: true}
			}
		}
	case 58:
		//line core/grammar.y:263
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "bool", value: false}
			}
		}
	case 59:
		//line core/grammar.y:265
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "null", value: nil}
			}
		}
	case 60:
		//line core/grammar.y:267
		{
			{
				mmVAL.exp = mmS[mmpt-0].exp
			}
		}
	case 61:
		//line core/grammar.y:272
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-2].val, mmS[mmpt-0].val}
			}
		}
	case 62:
		//line core/grammar.y:274
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-0].val, "default"}
			}
		}
	case 63:
		//line core/grammar.y:276
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "self", mmS[mmpt-0].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

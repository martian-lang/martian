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

//line core/grammar.y:273

//line yacctab:1
var mmExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
}

const mmNprod = 63
const mmPrivate = 57344

var mmTokenNames []string
var mmStates []string

const mmLast = 184

var mmAct = []int{

	28, 56, 24, 23, 75, 7, 27, 42, 35, 33,
	74, 79, 95, 94, 54, 17, 41, 36, 37, 39,
	38, 125, 40, 26, 89, 79, 32, 77, 45, 32,
	58, 87, 118, 78, 59, 112, 79, 47, 139, 91,
	111, 49, 98, 51, 70, 71, 76, 78, 45, 69,
	64, 62, 63, 49, 29, 53, 44, 80, 78, 29,
	88, 29, 60, 61, 58, 65, 66, 67, 59, 114,
	15, 113, 29, 102, 52, 57, 82, 22, 70, 14,
	21, 99, 13, 69, 64, 62, 63, 105, 12, 104,
	34, 50, 55, 11, 25, 34, 60, 61, 138, 65,
	66, 67, 25, 115, 97, 117, 34, 119, 124, 121,
	58, 25, 126, 31, 59, 9, 9, 4, 5, 6,
	9, 122, 131, 130, 70, 4, 5, 6, 108, 69,
	64, 62, 63, 135, 128, 109, 137, 45, 32, 136,
	43, 83, 60, 61, 127, 65, 66, 67, 123, 72,
	132, 129, 106, 93, 92, 107, 16, 85, 30, 20,
	19, 18, 100, 81, 46, 133, 116, 101, 84, 134,
	110, 3, 1, 120, 8, 103, 90, 86, 68, 48,
	96, 2, 73, 10,
}
var mmPact = []int{

	109, -1000, 101, -1000, 64, 59, 53, -1000, -1000, 50,
	150, -18, 149, 148, 147, 51, -1000, 48, 76, 76,
	30, 146, -1000, 68, -1000, -22, 68, 43, -1000, 155,
	30, 63, -1000, -1000, -22, 45, -1000, -1000, -1000, -1000,
	-1000, -1000, -19, 79, -1000, -1000, 54, 32, 136, -1000,
	-24, 17, 28, 152, 47, 127, 160, 145, 20, 9,
	142, 141, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -20,
	-21, -1000, 82, 12, -1000, -1000, 28, 151, 159, -1000,
	-1000, 44, -1000, 97, -1000, 100, 144, -1000, -1000, -1000,
	120, 163, 10, 5, 42, 40, -1000, 80, 158, -1000,
	3, -1000, 28, 96, -1000, 135, 100, -1000, -9, -1000,
	100, 131, 121, -1000, -1000, 139, -1000, -1000, 28, -1000,
	107, -1000, 138, 157, -1000, 162, -1000, -1000, -1000, 76,
	-1000, -1000, 30, -1000, 100, 85, 25, -1000, -1000, -1000,
}
var mmPgo = []int{

	0, 183, 8, 4, 182, 171, 181, 2, 9, 3,
	113, 180, 179, 1, 178, 177, 176, 5, 175, 0,
	6, 173, 172,
}
var mmR1 = []int{

	0, 22, 22, 6, 6, 5, 5, 5, 5, 1,
	1, 9, 9, 7, 7, 10, 10, 8, 8, 8,
	8, 12, 3, 3, 2, 2, 2, 2, 2, 2,
	2, 2, 4, 11, 21, 18, 18, 17, 17, 20,
	20, 19, 19, 15, 15, 16, 16, 13, 13, 13,
	13, 13, 13, 13, 13, 13, 13, 13, 13, 13,
	14, 14, 14,
}
var mmR2 = []int{

	0, 1, 2, 2, 1, 3, 7, 8, 10, 3,
	1, 2, 1, 4, 6, 2, 1, 3, 4, 5,
	6, 4, 2, 1, 1, 1, 1, 1, 1, 1,
	1, 3, 1, 5, 4, 2, 1, 5, 6, 2,
	1, 4, 7, 3, 1, 5, 3, 3, 2, 2,
	3, 4, 4, 1, 1, 1, 1, 1, 1, 1,
	3, 1, 3,
}
var mmChk = []int{

	-1000, -22, -6, -5, 16, 17, 18, -17, -5, 19,
	-1, 29, 29, 29, 29, 20, 6, 33, 12, 12,
	12, 29, 29, -9, -7, 26, -9, -20, -19, 29,
	12, -10, -7, -8, 27, -2, 39, 40, 42, 41,
	44, 38, 29, -10, 13, -19, 9, -20, -12, -8,
	28, -2, 29, 10, 33, 13, -13, 21, 10, 14,
	42, 43, 31, 32, 30, 45, 46, 47, -14, 29,
	24, 13, 13, -4, 34, -3, 29, 10, 30, 8,
	-3, 11, 29, 14, 8, 12, -15, 11, -13, 15,
	-16, 30, 12, 12, 33, 33, -11, 22, 30, -3,
	11, 8, 29, -18, -17, -13, 8, 11, 8, 15,
	7, 30, 30, 29, 29, 23, 8, -3, 29, -3,
	-21, -17, 25, 13, -13, 30, -13, 13, 13, 12,
	-3, 15, 12, 8, 7, -9, -20, -13, 13, 13,
}
var mmDef = []int{

	0, -2, 1, 4, 0, 0, 0, 2, 3, 0,
	0, 10, 0, 0, 0, 0, 5, 0, 0, 0,
	0, 0, 9, 0, 12, 0, 0, 0, 40, 0,
	0, 0, 11, 16, 0, 0, 24, 25, 26, 27,
	28, 29, 30, 0, 37, 39, 0, 0, 0, 15,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 53, 54, 55, 56, 57, 58, 59, 61,
	0, 38, 6, 0, 32, 17, 0, 0, 0, 23,
	13, 0, 31, 0, 41, 0, 0, 48, 44, 49,
	0, 0, 0, 0, 0, 0, 7, 0, 0, 18,
	0, 22, 0, 0, 36, 0, 0, 47, 0, 50,
	0, 0, 0, 60, 62, 0, 21, 19, 0, 14,
	0, 35, 0, 0, 43, 0, 46, 51, 52, 0,
	20, 8, 0, 42, 0, 0, 0, 45, 33, 34,
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
		//line core/grammar.y:81
		{
			{
				mmVAL.decs = append(mmS[mmpt-1].decs, mmS[mmpt-0].dec)
			}
		}
	case 4:
		//line core/grammar.y:83
		{
			{
				mmVAL.decs = []Dec{mmS[mmpt-0].dec}
			}
		}
	case 5:
		//line core/grammar.y:88
		{
			{
				mmVAL.dec = &Filetype{NewAstNode(&mmlval), mmS[mmpt-1].val}
			}
		}
	case 6:
		//line core/grammar.y:90
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-5].val, mmS[mmpt-3].params, mmS[mmpt-2].params, mmS[mmpt-1].src, &Params{[]Param{}, map[string]Param{}}}
			}
		}
	case 7:
		//line core/grammar.y:92
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-6].val, mmS[mmpt-4].params, mmS[mmpt-3].params, mmS[mmpt-2].src, mmS[mmpt-0].params}
			}
		}
	case 8:
		//line core/grammar.y:94
		{
			{
				mmVAL.dec = &Pipeline{NewAstNode(&mmlval), mmS[mmpt-8].val, mmS[mmpt-6].params, mmS[mmpt-5].params, mmS[mmpt-2].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmS[mmpt-1].retstm}
			}
		}
	case 9:
		//line core/grammar.y:99
		{
			{
				mmVAL.val = mmS[mmpt-2].val + mmS[mmpt-1].val + mmS[mmpt-0].val
			}
		}
	case 10:
		mmVAL.val = mmS[mmpt-0].val
	case 11:
		//line core/grammar.y:105
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].inparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 12:
		//line core/grammar.y:110
		{
			{
				mmVAL.params = &Params{[]Param{mmS[mmpt-0].inparam}, map[string]Param{}}
			}
		}
	case 13:
		//line core/grammar.y:115
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(&mmlval), mmS[mmpt-2].val, false, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 14:
		//line core/grammar.y:117
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(&mmlval), mmS[mmpt-4].val, true, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 15:
		//line core/grammar.y:122
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].outparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 16:
		//line core/grammar.y:127
		{
			{
				mmVAL.params = &Params{[]Param{mmS[mmpt-0].outparam}, map[string]Param{}}
			}
		}
	case 17:
		//line core/grammar.y:132
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-1].val, false, "default", unquote(mmS[mmpt-0].val), false}
			}
		}
	case 18:
		//line core/grammar.y:134
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-2].val, false, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 19:
		//line core/grammar.y:136
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-3].val, true, "default", unquote(mmS[mmpt-0].val), false}
			}
		}
	case 20:
		//line core/grammar.y:138
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-4].val, true, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 21:
		//line core/grammar.y:143
		{
			{
				mmVAL.src = &SrcParam{NewAstNode(&mmlval), mmS[mmpt-2].val, unquote(mmS[mmpt-1].val)}
			}
		}
	case 22:
		//line core/grammar.y:148
		{
			{
				mmVAL.val = mmS[mmpt-1].val
			}
		}
	case 23:
		//line core/grammar.y:150
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
		//line core/grammar.y:162
		{
			{
				mmVAL.val = mmS[mmpt-2].val + "." + mmS[mmpt-0].val
			}
		}
	case 32:
		mmVAL.val = mmS[mmpt-0].val
	case 33:
		//line core/grammar.y:174
		{
			{
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 34:
		//line core/grammar.y:179
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(&mmlval), mmS[mmpt-1].bindings}
			}
		}
	case 35:
		//line core/grammar.y:184
		{
			{
				mmVAL.calls = append(mmS[mmpt-1].calls, mmS[mmpt-0].call)
			}
		}
	case 36:
		//line core/grammar.y:186
		{
			{
				mmVAL.calls = []*CallStm{mmS[mmpt-0].call}
			}
		}
	case 37:
		//line core/grammar.y:191
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), false, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 38:
		//line core/grammar.y:193
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), true, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 39:
		//line core/grammar.y:198
		{
			{
				mmS[mmpt-1].bindings.List = append(mmS[mmpt-1].bindings.List, mmS[mmpt-0].binding)
				mmVAL.bindings = mmS[mmpt-1].bindings
			}
		}
	case 40:
		//line core/grammar.y:203
		{
			{
				mmVAL.bindings = &BindStms{[]*BindStm{mmS[mmpt-0].binding}, map[string]*BindStm{}}
			}
		}
	case 41:
		//line core/grammar.y:208
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-3].val, mmS[mmpt-1].exp, false, ""}
			}
		}
	case 42:
		//line core/grammar.y:210
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-6].val, mmS[mmpt-2].exp, true, ""}
			}
		}
	case 43:
		//line core/grammar.y:215
		{
			{
				mmVAL.exps = append(mmS[mmpt-2].exps, mmS[mmpt-0].exp)
			}
		}
	case 44:
		//line core/grammar.y:217
		{
			{
				mmVAL.exps = []Exp{mmS[mmpt-0].exp}
			}
		}
	case 45:
		//line core/grammar.y:222
		{
			{
				mmS[mmpt-4].kvpairs[unquote(mmS[mmpt-2].val)] = mmS[mmpt-0].exp
				mmVAL.kvpairs = mmS[mmpt-4].kvpairs
			}
		}
	case 46:
		//line core/grammar.y:227
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmS[mmpt-2].val): mmS[mmpt-0].exp}
			}
		}
	case 47:
		//line core/grammar.y:232
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "array", Value: mmS[mmpt-1].exps}
			}
		}
	case 48:
		//line core/grammar.y:234
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "array", Value: []Exp{}}
			}
		}
	case 49:
		//line core/grammar.y:236
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "map", Value: map[string]interface{}{}}
			}
		}
	case 50:
		//line core/grammar.y:238
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "map", Value: mmS[mmpt-1].kvpairs}
			}
		}
	case 51:
		//line core/grammar.y:240
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: mmS[mmpt-3].val, Value: unquote(mmS[mmpt-1].val)}
			}
		}
	case 52:
		//line core/grammar.y:242
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: mmS[mmpt-3].val, Value: unquote(mmS[mmpt-1].val)}
			}
		}
	case 53:
		//line core/grammar.y:244
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmS[mmpt-0].val, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "float", Value: f}
			}
		}
	case 54:
		//line core/grammar.y:249
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmS[mmpt-0].val, 0, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "int", Value: i}
			}
		}
	case 55:
		//line core/grammar.y:254
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "string", Value: unquote(mmS[mmpt-0].val)}
			}
		}
	case 56:
		//line core/grammar.y:256
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "bool", Value: true}
			}
		}
	case 57:
		//line core/grammar.y:258
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "bool", Value: false}
			}
		}
	case 58:
		//line core/grammar.y:260
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "null", Value: nil}
			}
		}
	case 59:
		//line core/grammar.y:262
		{
			{
				mmVAL.exp = mmS[mmpt-0].exp
			}
		}
	case 60:
		//line core/grammar.y:267
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-2].val, mmS[mmpt-0].val}
			}
		}
	case 61:
		//line core/grammar.y:269
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-0].val, "default"}
			}
		}
	case 62:
		//line core/grammar.y:271
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "self", mmS[mmpt-0].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

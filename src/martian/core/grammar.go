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
	tags     *Tags
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
const PREFLIGHT = 57363
const VOLATILE = 57364
const SWEEP = 57365
const SPLIT = 57366
const USING = 57367
const SELF = 57368
const RETURN = 57369
const IN = 57370
const OUT = 57371
const SRC = 57372
const ID = 57373
const LITSTRING = 57374
const NUM_FLOAT = 57375
const NUM_INT = 57376
const DOT = 57377
const PY = 57378
const GO = 57379
const SH = 57380
const EXEC = 57381
const MAP = 57382
const INT = 57383
const STRING = 57384
const FLOAT = 57385
const PATH = 57386
const BOOL = 57387
const TRUE = 57388
const FALSE = 57389
const NULL = 57390
const DEFAULT = 57391

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
	"PREFLIGHT",
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

//line src/martian/core/grammar.y:289

//line yacctab:1
var mmExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
}

const mmNprod = 65
const mmPrivate = 57344

var mmTokenNames []string
var mmStates []string

const mmLast = 153

var mmAct = []int{

	84, 28, 26, 92, 82, 3, 60, 83, 9, 55,
	61, 52, 40, 89, 53, 95, 88, 75, 56, 21,
	85, 116, 70, 17, 18, 19, 27, 69, 64, 62,
	63, 123, 60, 95, 16, 108, 61, 87, 93, 94,
	75, 73, 65, 66, 67, 59, 107, 33, 70, 35,
	58, 77, 54, 69, 64, 62, 63, 94, 39, 38,
	122, 76, 25, 79, 74, 35, 15, 14, 65, 66,
	67, 13, 60, 48, 5, 31, 61, 31, 29, 90,
	97, 72, 100, 5, 99, 104, 101, 112, 70, 39,
	57, 50, 105, 69, 64, 62, 63, 110, 6, 7,
	8, 5, 113, 115, 109, 81, 32, 117, 65, 66,
	67, 102, 118, 24, 47, 119, 114, 102, 23, 22,
	103, 96, 124, 46, 41, 42, 44, 43, 45, 49,
	120, 111, 91, 80, 121, 106, 20, 4, 1, 98,
	10, 34, 78, 86, 68, 36, 71, 37, 30, 2,
	11, 51, 12,
}
var mmPact = []int{

	82, -1000, 82, -1000, -1000, -1000, 40, 36, 35, -1000,
	-1000, 3, 130, -16, 107, 106, 101, -1000, -1000, -1000,
	-1000, 31, -1000, -1000, -1000, -1000, 49, 49, 34, 29,
	-1000, 83, 60, -1000, -1000, 120, 78, -1000, -25, 83,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -17, 76, 22,
	57, 9, -1000, -1000, -1000, 30, 20, 64, 125, 93,
	-4, 5, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -19,
	-22, -1000, 54, 124, 7, 110, 25, -1000, 55, -1000,
	-1000, 62, 109, -1000, -1000, -1000, 77, 128, 15, 4,
	92, -1000, -1000, 25, 123, -1000, -1000, -1000, 72, -1000,
	90, 103, 62, -1000, -11, -1000, 62, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 122, -1000, 127, -1000, 47, 18,
	-1000, 62, -1000, -1000, -1000,
}
var mmPgo = []int{

	0, 152, 12, 3, 151, 150, 9, 137, 149, 148,
	147, 2, 78, 146, 145, 0, 144, 4, 143, 5,
	142, 141, 1, 139, 138,
}
var mmR1 = []int{

	0, 24, 24, 24, 8, 8, 7, 7, 7, 7,
	1, 1, 6, 6, 11, 11, 9, 12, 12, 10,
	10, 14, 3, 3, 2, 2, 2, 2, 2, 2,
	2, 2, 4, 4, 13, 23, 20, 20, 19, 5,
	5, 5, 5, 22, 22, 21, 21, 17, 17, 18,
	18, 15, 15, 15, 15, 15, 15, 15, 15, 15,
	15, 15, 16, 16, 16,
}
var mmR2 = []int{

	0, 1, 2, 1, 2, 1, 3, 7, 8, 10,
	3, 1, 0, 3, 0, 2, 5, 0, 2, 4,
	5, 4, 2, 1, 1, 1, 1, 1, 1, 1,
	1, 3, 1, 1, 5, 4, 2, 1, 6, 0,
	2, 2, 2, 0, 2, 4, 7, 3, 1, 5,
	3, 3, 2, 2, 3, 1, 1, 1, 1, 1,
	1, 1, 3, 1, 3,
}
var mmChk = []int{

	-1000, -24, -8, -19, -7, 19, 16, 17, 18, -19,
	-7, -5, -1, 31, 31, 31, 31, 20, 21, 22,
	6, 35, 12, 12, 12, 31, -11, -11, -22, -12,
	-9, 28, -12, 13, -21, 31, -14, -10, 30, 29,
	-2, 41, 42, 44, 43, 45, 40, 31, 13, 9,
	13, -4, 36, 39, -2, -6, 35, 14, -15, 23,
	10, 14, 33, 34, 32, 46, 47, 48, -16, 31,
	26, -13, 24, 32, -6, 10, 31, 31, -20, -19,
	8, 12, -17, 11, -15, 15, -18, 32, 35, 35,
	25, 8, -3, 31, 32, 8, 11, -3, -23, -19,
	27, -17, 8, 11, 8, 15, 7, 31, 31, 12,
	-3, 8, 15, 12, 13, -15, 32, -15, -11, -22,
	8, 7, 13, 13, -15,
}
var mmDef = []int{

	0, -2, 1, 3, 5, 39, 0, 0, 0, 2,
	4, 0, 0, 11, 0, 0, 0, 40, 41, 42,
	6, 0, 14, 14, 43, 10, 17, 17, 0, 0,
	15, 0, 0, 38, 44, 0, 0, 18, 0, 0,
	12, 24, 25, 26, 27, 28, 29, 30, 0, 0,
	7, 0, 32, 33, 12, 0, 0, 0, 0, 0,
	0, 0, 55, 56, 57, 58, 59, 60, 61, 63,
	0, 8, 0, 0, 0, 0, 0, 31, 0, 37,
	45, 0, 0, 52, 48, 53, 0, 0, 0, 0,
	0, 21, 19, 0, 0, 23, 13, 16, 0, 36,
	0, 0, 0, 51, 0, 54, 0, 62, 64, 14,
	20, 22, 9, 43, 0, 47, 0, 50, 0, 0,
	46, 0, 34, 35, 49,
}
var mmTok1 = []int{

	1,
}
var mmTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
	32, 33, 34, 35, 36, 37, 38, 39, 40, 41,
	42, 43, 44, 45, 46, 47, 48, 49,
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
		//line src/martian/core/grammar.y:72
		{
			{
				global := NewAst(mmS[mmpt-0].decs, nil)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 2:
		//line src/martian/core/grammar.y:77
		{
			{
				global := NewAst(mmS[mmpt-1].decs, mmS[mmpt-0].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 3:
		//line src/martian/core/grammar.y:82
		{
			{
				global := NewAst([]Dec{}, mmS[mmpt-0].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 4:
		//line src/martian/core/grammar.y:90
		{
			{
				mmVAL.decs = append(mmS[mmpt-1].decs, mmS[mmpt-0].dec)
			}
		}
	case 5:
		//line src/martian/core/grammar.y:92
		{
			{
				mmVAL.decs = []Dec{mmS[mmpt-0].dec}
			}
		}
	case 6:
		//line src/martian/core/grammar.y:97
		{
			{
				mmVAL.dec = &Filetype{NewAstNode(&mmlval), mmS[mmpt-1].val}
			}
		}
	case 7:
		//line src/martian/core/grammar.y:99
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-5].val, mmS[mmpt-3].params, mmS[mmpt-2].params, mmS[mmpt-1].src, &Params{[]Param{}, map[string]Param{}}}
			}
		}
	case 8:
		//line src/martian/core/grammar.y:101
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-6].val, mmS[mmpt-4].params, mmS[mmpt-3].params, mmS[mmpt-2].src, mmS[mmpt-0].params}
			}
		}
	case 9:
		//line src/martian/core/grammar.y:103
		{
			{
				mmVAL.dec = &Pipeline{NewAstNode(&mmlval), mmS[mmpt-8].val, mmS[mmpt-6].params, mmS[mmpt-5].params, mmS[mmpt-2].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmS[mmpt-1].retstm}
			}
		}
	case 10:
		//line src/martian/core/grammar.y:108
		{
			{
				mmVAL.val = mmS[mmpt-2].val + mmS[mmpt-1].val + mmS[mmpt-0].val
			}
		}
	case 11:
		mmVAL.val = mmS[mmpt-0].val
	case 12:
		//line src/martian/core/grammar.y:114
		{
			{
				mmVAL.arr = 0
			}
		}
	case 13:
		//line src/martian/core/grammar.y:116
		{
			{
				mmVAL.arr += 1
			}
		}
	case 14:
		//line src/martian/core/grammar.y:121
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 15:
		//line src/martian/core/grammar.y:123
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].inparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 16:
		//line src/martian/core/grammar.y:131
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(&mmlval), mmS[mmpt-3].val, mmS[mmpt-2].arr, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 17:
		//line src/martian/core/grammar.y:136
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 18:
		//line src/martian/core/grammar.y:138
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].outparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 19:
		//line src/martian/core/grammar.y:146
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-2].val, mmS[mmpt-1].arr, "default", unquote(mmS[mmpt-0].val), false}
			}
		}
	case 20:
		//line src/martian/core/grammar.y:148
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-3].val, mmS[mmpt-2].arr, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 21:
		//line src/martian/core/grammar.y:153
		{
			{
				stagecodeParts := strings.Split(unquote(mmS[mmpt-1].val), " ")
				mmVAL.src = &SrcParam{NewAstNode(&mmlval), mmS[mmpt-2].val, stagecodeParts[0], stagecodeParts[1:]}
			}
		}
	case 22:
		//line src/martian/core/grammar.y:159
		{
			{
				mmVAL.val = mmS[mmpt-1].val
			}
		}
	case 23:
		//line src/martian/core/grammar.y:161
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
		//line src/martian/core/grammar.y:173
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
		//line src/martian/core/grammar.y:185
		{
			{
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 35:
		//line src/martian/core/grammar.y:190
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(&mmlval), mmS[mmpt-1].bindings}
			}
		}
	case 36:
		//line src/martian/core/grammar.y:195
		{
			{
				mmVAL.calls = append(mmS[mmpt-1].calls, mmS[mmpt-0].call)
			}
		}
	case 37:
		//line src/martian/core/grammar.y:197
		{
			{
				mmVAL.calls = []*CallStm{mmS[mmpt-0].call}
			}
		}
	case 38:
		//line src/martian/core/grammar.y:202
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), mmS[mmpt-4].tags, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 39:
		//line src/martian/core/grammar.y:207
		{
			{
				mmVAL.tags = &Tags{false, false, false}
			}
		}
	case 40:
		//line src/martian/core/grammar.y:209
		{
			{
				mmVAL.tags.local = true
			}
		}
	case 41:
		//line src/martian/core/grammar.y:211
		{
			{
				mmVAL.tags.preflight = true
			}
		}
	case 42:
		//line src/martian/core/grammar.y:213
		{
			{
				mmVAL.tags.volatile = true
			}
		}
	case 43:
		//line src/martian/core/grammar.y:218
		{
			{
				mmVAL.bindings = &BindStms{[]*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 44:
		//line src/martian/core/grammar.y:220
		{
			{
				mmS[mmpt-1].bindings.list = append(mmS[mmpt-1].bindings.list, mmS[mmpt-0].binding)
				mmVAL.bindings = mmS[mmpt-1].bindings
			}
		}
	case 45:
		//line src/martian/core/grammar.y:228
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-3].val, mmS[mmpt-1].exp, false, ""}
			}
		}
	case 46:
		//line src/martian/core/grammar.y:230
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-6].val, &ValExp{node: NewAstNode(&mmlval), kind: "array", value: mmS[mmpt-2].exps}, true, ""}
			}
		}
	case 47:
		//line src/martian/core/grammar.y:235
		{
			{
				mmVAL.exps = append(mmS[mmpt-2].exps, mmS[mmpt-0].exp)
			}
		}
	case 48:
		//line src/martian/core/grammar.y:237
		{
			{
				mmVAL.exps = []Exp{mmS[mmpt-0].exp}
			}
		}
	case 49:
		//line src/martian/core/grammar.y:242
		{
			{
				mmS[mmpt-4].kvpairs[unquote(mmS[mmpt-2].val)] = mmS[mmpt-0].exp
				mmVAL.kvpairs = mmS[mmpt-4].kvpairs
			}
		}
	case 50:
		//line src/martian/core/grammar.y:247
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmS[mmpt-2].val): mmS[mmpt-0].exp}
			}
		}
	case 51:
		//line src/martian/core/grammar.y:252
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "array", value: mmS[mmpt-1].exps}
			}
		}
	case 52:
		//line src/martian/core/grammar.y:254
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "array", value: []Exp{}}
			}
		}
	case 53:
		//line src/martian/core/grammar.y:256
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "map", value: map[string]interface{}{}}
			}
		}
	case 54:
		//line src/martian/core/grammar.y:258
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "map", value: mmS[mmpt-1].kvpairs}
			}
		}
	case 55:
		//line src/martian/core/grammar.y:260
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmS[mmpt-0].val, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "float", value: f}
			}
		}
	case 56:
		//line src/martian/core/grammar.y:265
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmS[mmpt-0].val, 0, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "int", value: i}
			}
		}
	case 57:
		//line src/martian/core/grammar.y:270
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "string", value: unquote(mmS[mmpt-0].val)}
			}
		}
	case 58:
		//line src/martian/core/grammar.y:272
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "bool", value: true}
			}
		}
	case 59:
		//line src/martian/core/grammar.y:274
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "bool", value: false}
			}
		}
	case 60:
		//line src/martian/core/grammar.y:276
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "null", value: nil}
			}
		}
	case 61:
		//line src/martian/core/grammar.y:278
		{
			{
				mmVAL.exp = mmS[mmpt-0].exp
			}
		}
	case 62:
		//line src/martian/core/grammar.y:283
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-2].val, mmS[mmpt-0].val}
			}
		}
	case 63:
		//line src/martian/core/grammar.y:285
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-0].val, "default"}
			}
		}
	case 64:
		//line src/martian/core/grammar.y:287
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "self", mmS[mmpt-0].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

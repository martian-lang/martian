//line mario/core/grammar.y:2

//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO grammar.
//
package core

import __yyfmt__ "fmt"

//line mario/core/grammar.y:7
import (
	"strconv"
	"strings"
)

func unquote(qs string) string {
	return strings.Replace(qs, "\"", "", -1)
}

//line mario/core/grammar.y:19
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

//line mario/core/grammar.y:278

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

const mmLast = 172

var mmAct = []int{

	78, 23, 89, 76, 3, 26, 49, 9, 42, 68,
	53, 77, 85, 84, 54, 48, 43, 44, 46, 45,
	72, 47, 20, 79, 65, 137, 31, 93, 27, 64,
	59, 57, 58, 93, 120, 91, 93, 51, 81, 106,
	105, 30, 55, 56, 88, 60, 61, 62, 53, 92,
	69, 114, 54, 71, 90, 92, 12, 126, 92, 52,
	37, 28, 65, 108, 107, 11, 96, 64, 59, 57,
	58, 25, 70, 94, 18, 16, 30, 30, 98, 99,
	55, 56, 15, 60, 61, 62, 41, 40, 32, 14,
	53, 5, 50, 111, 54, 136, 34, 117, 109, 87,
	5, 119, 116, 128, 65, 121, 41, 102, 34, 64,
	59, 57, 58, 123, 103, 125, 35, 127, 6, 7,
	8, 5, 55, 56, 73, 60, 61, 62, 100, 133,
	132, 134, 135, 118, 122, 66, 129, 124, 100, 83,
	82, 101, 19, 75, 24, 22, 21, 17, 112, 95,
	36, 130, 113, 110, 74, 131, 104, 4, 1, 115,
	10, 29, 97, 80, 63, 38, 86, 39, 33, 2,
	67, 13,
}
var mmPact = []int{

	102, -1000, 102, -1000, -1000, 36, 60, 53, 46, -1000,
	-1000, 135, 45, 136, -11, 134, 133, -1000, 132, -1000,
	42, -1000, -1000, 48, -1000, -1000, 70, 70, -1000, -1000,
	141, 47, 59, -1000, -23, 79, 38, -1000, 122, -1000,
	-25, -23, 43, -1000, -1000, -1000, -1000, -1000, -1000, -13,
	110, 146, 131, 0, 8, 128, 127, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -20, -21, 77, 14, -1000, 25,
	19, 138, 37, 81, -1000, 80, 130, -1000, -1000, -1000,
	99, 149, 10, 9, 35, 34, -1000, 75, 145, -1000,
	19, 137, 144, -1000, -1000, 22, -1000, 72, -1000, 120,
	80, -1000, 4, -1000, 80, 121, 100, -1000, -1000, 125,
	-1000, -1000, 28, -1000, 19, 88, -1000, 124, 143, -1000,
	148, -1000, -1000, -1000, -1000, -1000, 19, -1000, -1000, -1000,
	-1000, 80, 82, -1000, 12, -1000, -1000, -1000,
}
var mmPgo = []int{

	0, 171, 8, 2, 170, 157, 169, 168, 167, 5,
	88, 166, 165, 0, 164, 3, 163, 4, 162, 161,
	1, 159, 158,
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
	3, 1, 0, 2, 4, 6, 0, 2, 3, 4,
	5, 6, 4, 2, 1, 1, 1, 1, 1, 1,
	1, 1, 3, 1, 5, 4, 2, 1, 5, 6,
	0, 2, 4, 7, 3, 1, 5, 3, 3, 2,
	2, 3, 4, 4, 1, 1, 1, 1, 1, 1,
	1, 3, 1, 3,
}
var mmChk = []int{

	-1000, -22, -6, -17, -5, 19, 16, 17, 18, -17,
	-5, 29, 20, -1, 29, 29, 29, 12, 29, 6,
	33, 12, 12, -20, 12, 29, -9, -9, 13, -19,
	29, -20, -10, -7, 26, -10, 9, 13, -12, -8,
	28, 27, -2, 39, 40, 42, 41, 44, 38, 29,
	13, -13, 21, 10, 14, 42, 43, 31, 32, 30,
	45, 46, 47, -14, 29, 24, 13, -4, 34, -2,
	29, 10, 33, 14, 8, 12, -15, 11, -13, 15,
	-16, 30, 12, 12, 33, 33, -11, 22, 30, -3,
	29, 10, 30, 8, -3, 11, 29, -18, -17, -15,
	8, 11, 8, 15, 7, 30, 30, 29, 29, 23,
	8, -3, 11, 8, 29, -21, -17, 25, 13, -13,
	30, -13, 13, 13, 12, -3, 29, -3, 15, 12,
	8, 7, -9, -3, -20, -13, 13, 13,
}
var mmDef = []int{

	0, -2, 1, 3, 5, 0, 0, 0, 0, 2,
	4, 0, 0, 0, 11, 0, 0, 40, 0, 6,
	0, 12, 12, 0, 40, 10, 16, 16, 38, 41,
	0, 0, 0, 13, 0, 0, 0, 39, 0, 17,
	0, 0, 0, 25, 26, 27, 28, 29, 30, 31,
	0, 0, 0, 0, 0, 0, 0, 54, 55, 56,
	57, 58, 59, 60, 62, 0, 7, 0, 33, 0,
	0, 0, 0, 0, 42, 0, 0, 49, 45, 50,
	0, 0, 0, 0, 0, 0, 8, 0, 0, 18,
	0, 0, 0, 24, 14, 0, 32, 0, 37, 0,
	0, 48, 0, 51, 0, 0, 0, 61, 63, 0,
	22, 19, 0, 23, 0, 0, 36, 0, 0, 44,
	0, 47, 52, 53, 12, 20, 0, 15, 9, 40,
	43, 0, 0, 21, 0, 46, 34, 35,
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
		//line mario/core/grammar.y:68
		{
			{
				global := NewAst(mmS[mmpt-0].decs, nil)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 2:
		//line mario/core/grammar.y:73
		{
			{
				global := NewAst(mmS[mmpt-1].decs, mmS[mmpt-0].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 3:
		//line mario/core/grammar.y:78
		{
			{
				global := NewAst([]Dec{}, mmS[mmpt-0].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 4:
		//line mario/core/grammar.y:86
		{
			{
				mmVAL.decs = append(mmS[mmpt-1].decs, mmS[mmpt-0].dec)
			}
		}
	case 5:
		//line mario/core/grammar.y:88
		{
			{
				mmVAL.decs = []Dec{mmS[mmpt-0].dec}
			}
		}
	case 6:
		//line mario/core/grammar.y:93
		{
			{
				mmVAL.dec = &Filetype{NewAstNode(&mmlval), mmS[mmpt-1].val}
			}
		}
	case 7:
		//line mario/core/grammar.y:95
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-5].val, mmS[mmpt-3].params, mmS[mmpt-2].params, mmS[mmpt-1].src, &Params{[]Param{}, map[string]Param{}}}
			}
		}
	case 8:
		//line mario/core/grammar.y:97
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-6].val, mmS[mmpt-4].params, mmS[mmpt-3].params, mmS[mmpt-2].src, mmS[mmpt-0].params}
			}
		}
	case 9:
		//line mario/core/grammar.y:99
		{
			{
				mmVAL.dec = &Pipeline{NewAstNode(&mmlval), mmS[mmpt-8].val, mmS[mmpt-6].params, mmS[mmpt-5].params, mmS[mmpt-2].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmS[mmpt-1].retstm}
			}
		}
	case 10:
		//line mario/core/grammar.y:104
		{
			{
				mmVAL.val = mmS[mmpt-2].val + mmS[mmpt-1].val + mmS[mmpt-0].val
			}
		}
	case 11:
		mmVAL.val = mmS[mmpt-0].val
	case 12:
		//line mario/core/grammar.y:110
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 13:
		//line mario/core/grammar.y:115
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].inparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 14:
		//line mario/core/grammar.y:120
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(&mmlval), mmS[mmpt-2].val, false, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 15:
		//line mario/core/grammar.y:122
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(&mmlval), mmS[mmpt-4].val, true, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 16:
		//line mario/core/grammar.y:127
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 17:
		//line mario/core/grammar.y:132
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].outparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 18:
		//line mario/core/grammar.y:137
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-1].val, false, "default", unquote(mmS[mmpt-0].val), false}
			}
		}
	case 19:
		//line mario/core/grammar.y:139
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-2].val, false, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 20:
		//line mario/core/grammar.y:141
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-3].val, true, "default", unquote(mmS[mmpt-0].val), false}
			}
		}
	case 21:
		//line mario/core/grammar.y:143
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-4].val, true, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 22:
		//line mario/core/grammar.y:148
		{
			{
				mmVAL.src = &SrcParam{NewAstNode(&mmlval), mmS[mmpt-2].val, unquote(mmS[mmpt-1].val)}
			}
		}
	case 23:
		//line mario/core/grammar.y:153
		{
			{
				mmVAL.val = mmS[mmpt-1].val
			}
		}
	case 24:
		//line mario/core/grammar.y:155
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
		//line mario/core/grammar.y:167
		{
			{
				mmVAL.val = mmS[mmpt-2].val + "." + mmS[mmpt-0].val
			}
		}
	case 33:
		mmVAL.val = mmS[mmpt-0].val
	case 34:
		//line mario/core/grammar.y:179
		{
			{
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 35:
		//line mario/core/grammar.y:184
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(&mmlval), mmS[mmpt-1].bindings}
			}
		}
	case 36:
		//line mario/core/grammar.y:189
		{
			{
				mmVAL.calls = append(mmS[mmpt-1].calls, mmS[mmpt-0].call)
			}
		}
	case 37:
		//line mario/core/grammar.y:191
		{
			{
				mmVAL.calls = []*CallStm{mmS[mmpt-0].call}
			}
		}
	case 38:
		//line mario/core/grammar.y:196
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), false, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 39:
		//line mario/core/grammar.y:198
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), true, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 40:
		//line mario/core/grammar.y:203
		{
			{
				mmVAL.bindings = &BindStms{[]*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 41:
		//line mario/core/grammar.y:208
		{
			{
				mmS[mmpt-1].bindings.list = append(mmS[mmpt-1].bindings.list, mmS[mmpt-0].binding)
				mmVAL.bindings = mmS[mmpt-1].bindings
			}
		}
	case 42:
		//line mario/core/grammar.y:213
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-3].val, mmS[mmpt-1].exp, false, ""}
			}
		}
	case 43:
		//line mario/core/grammar.y:215
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-6].val, &ValExp{node: NewAstNode(&mmlval), kind: "array", value: mmS[mmpt-2].exps}, true, ""}
			}
		}
	case 44:
		//line mario/core/grammar.y:220
		{
			{
				mmVAL.exps = append(mmS[mmpt-2].exps, mmS[mmpt-0].exp)
			}
		}
	case 45:
		//line mario/core/grammar.y:222
		{
			{
				mmVAL.exps = []Exp{mmS[mmpt-0].exp}
			}
		}
	case 46:
		//line mario/core/grammar.y:227
		{
			{
				mmS[mmpt-4].kvpairs[unquote(mmS[mmpt-2].val)] = mmS[mmpt-0].exp
				mmVAL.kvpairs = mmS[mmpt-4].kvpairs
			}
		}
	case 47:
		//line mario/core/grammar.y:232
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmS[mmpt-2].val): mmS[mmpt-0].exp}
			}
		}
	case 48:
		//line mario/core/grammar.y:237
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "array", value: mmS[mmpt-1].exps}
			}
		}
	case 49:
		//line mario/core/grammar.y:239
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "array", value: []Exp{}}
			}
		}
	case 50:
		//line mario/core/grammar.y:241
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "map", value: map[string]interface{}{}}
			}
		}
	case 51:
		//line mario/core/grammar.y:243
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "map", value: mmS[mmpt-1].kvpairs}
			}
		}
	case 52:
		//line mario/core/grammar.y:245
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: mmS[mmpt-3].val, value: unquote(mmS[mmpt-1].val)}
			}
		}
	case 53:
		//line mario/core/grammar.y:247
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: mmS[mmpt-3].val, value: unquote(mmS[mmpt-1].val)}
			}
		}
	case 54:
		//line mario/core/grammar.y:249
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmS[mmpt-0].val, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "float", value: f}
			}
		}
	case 55:
		//line mario/core/grammar.y:254
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmS[mmpt-0].val, 0, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "int", value: i}
			}
		}
	case 56:
		//line mario/core/grammar.y:259
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "string", value: unquote(mmS[mmpt-0].val)}
			}
		}
	case 57:
		//line mario/core/grammar.y:261
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "bool", value: true}
			}
		}
	case 58:
		//line mario/core/grammar.y:263
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "bool", value: false}
			}
		}
	case 59:
		//line mario/core/grammar.y:265
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), kind: "null", value: nil}
			}
		}
	case 60:
		//line mario/core/grammar.y:267
		{
			{
				mmVAL.exp = mmS[mmpt-0].exp
			}
		}
	case 61:
		//line mario/core/grammar.y:272
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-2].val, mmS[mmpt-0].val}
			}
		}
	case 62:
		//line mario/core/grammar.y:274
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-0].val, "default"}
			}
		}
	case 63:
		//line mario/core/grammar.y:276
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "self", mmS[mmpt-0].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

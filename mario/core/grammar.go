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

//line core/grammar.y:285

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

	23, 22, 28, 48, 87, 39, 3, 86, 83, 27,
	82, 70, 37, 19, 77, 91, 91, 89, 117, 108,
	50, 75, 103, 32, 51, 102, 69, 34, 91, 79,
	36, 30, 35, 36, 62, 32, 88, 90, 90, 61,
	56, 54, 55, 139, 67, 68, 63, 50, 65, 124,
	90, 51, 52, 53, 76, 57, 58, 59, 49, 24,
	65, 62, 24, 47, 24, 31, 61, 56, 54, 55,
	112, 105, 104, 92, 94, 26, 46, 96, 17, 52,
	53, 24, 57, 58, 59, 45, 40, 41, 43, 42,
	50, 44, 15, 109, 51, 38, 66, 11, 14, 13,
	71, 116, 114, 138, 62, 118, 10, 29, 38, 61,
	56, 54, 55, 5, 38, 123, 29, 125, 29, 128,
	127, 121, 52, 53, 107, 57, 58, 59, 5, 132,
	6, 7, 8, 5, 135, 133, 137, 95, 32, 36,
	99, 136, 6, 7, 8, 110, 120, 100, 119, 115,
	84, 134, 131, 97, 81, 80, 98, 18, 73, 25,
	21, 20, 16, 93, 33, 129, 122, 111, 72, 130,
	101, 4, 1, 126, 9, 113, 78, 74, 60, 64,
	106, 2, 85, 12,
}
var mmPact = []int{

	114, -1000, 126, -1000, -1000, 77, 70, 69, 63, -1000,
	150, 49, 151, -20, 149, 148, 35, 147, -1000, 46,
	92, 92, 52, -1000, 155, 35, -1000, 81, -1000, 47,
	81, -1000, -1000, 37, 33, 68, -1000, -1000, 47, 16,
	-1000, -1000, -1000, -1000, -1000, -1000, -22, 87, 160, 146,
	10, -1, 143, 142, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -23, -25, -1000, 137, -1000, -27, 7, 8, 152,
	45, 123, -1000, 80, 145, -1000, -1000, -1000, 132, 163,
	-5, -8, 43, 42, 102, -11, -1000, -1000, 8, 134,
	159, -1000, -1000, 41, -1000, 109, 136, 80, -1000, -12,
	-1000, 80, 135, 133, -1000, -1000, -1000, 98, 158, -1000,
	20, -1000, 8, 94, -1000, 157, -1000, 162, -1000, -1000,
	-1000, 140, -1000, -1000, 8, -1000, 120, -1000, 139, -1000,
	80, 92, -1000, -1000, 35, -1000, 90, 30, -1000, -1000,
}
var mmPgo = []int{

	0, 183, 5, 4, 182, 171, 181, 2, 12, 9,
	32, 180, 179, 3, 178, 177, 176, 6, 175, 0,
	1, 173, 172,
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

	0, 1, 1, 2, 1, 3, 7, 8, 10, 3,
	1, 2, 1, 4, 6, 2, 1, 3, 4, 5,
	6, 4, 2, 1, 1, 1, 1, 1, 1, 1,
	1, 3, 1, 5, 4, 2, 1, 5, 6, 2,
	1, 4, 7, 3, 1, 5, 3, 3, 2, 2,
	3, 4, 4, 1, 1, 1, 1, 1, 1, 1,
	3, 1, 3,
}
var mmChk = []int{

	-1000, -22, -6, -17, -5, 19, 16, 17, 18, -5,
	29, 20, -1, 29, 29, 29, 12, 29, 6, 33,
	12, 12, -20, -19, 29, 12, 29, -9, -7, 26,
	-9, 13, -19, 9, -20, -10, -7, -8, 27, -2,
	39, 40, 42, 41, 44, 38, 29, -10, -13, 21,
	10, 14, 42, 43, 31, 32, 30, 45, 46, 47,
	-14, 29, 24, 13, -12, -8, 28, -2, 29, 10,
	33, 13, 8, 12, -15, 11, -13, 15, -16, 30,
	12, 12, 33, 33, 13, -4, 34, -3, 29, 10,
	30, 8, -3, 11, 29, 14, -13, 8, 11, 8,
	15, 7, 30, 30, 29, 29, -11, 22, 30, -3,
	11, 8, 29, -18, -17, 13, -13, 30, -13, 13,
	13, 23, 8, -3, 29, -3, -21, -17, 25, 8,
	7, 12, -3, 15, 12, -13, -9, -20, 13, 13,
}
var mmDef = []int{

	0, -2, 1, 2, 4, 0, 0, 0, 0, 3,
	0, 0, 0, 10, 0, 0, 0, 0, 5, 0,
	0, 0, 0, 40, 0, 0, 9, 0, 12, 0,
	0, 37, 39, 0, 0, 0, 11, 16, 0, 0,
	24, 25, 26, 27, 28, 29, 30, 0, 0, 0,
	0, 0, 0, 0, 53, 54, 55, 56, 57, 58,
	59, 61, 0, 38, 0, 15, 0, 0, 0, 0,
	0, 0, 41, 0, 0, 48, 44, 49, 0, 0,
	0, 0, 0, 0, 6, 0, 32, 17, 0, 0,
	0, 23, 13, 0, 31, 0, 0, 0, 47, 0,
	50, 0, 0, 0, 60, 62, 7, 0, 0, 18,
	0, 22, 0, 0, 36, 0, 43, 0, 46, 51,
	52, 0, 21, 19, 0, 14, 0, 35, 0, 42,
	0, 0, 20, 8, 0, 45, 0, 0, 33, 34,
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
				global := Ast{[]FileLoc{}, map[string]bool{}, []*Filetype{}, map[string]bool{}, []*Stage{}, []*Pipeline{}, &Callables{[]Callable{}, map[string]Callable{}}, nil}
				for _, dec := range mmS[mmpt-0].decs {
					switch dec := dec.(type) {
					case *Filetype:
						global.filetypes = append(global.filetypes, dec)
					case *Stage:
						global.Stages = append(global.Stages, dec)
						global.callables.list = append(global.callables.list, dec)
					case *Pipeline:
						global.Pipelines = append(global.Pipelines, dec)
						global.callables.list = append(global.callables.list, dec)
					}
				}
				mmlex.(*mmLexInfo).global = &global
			}
		}
	case 2:
		//line core/grammar.y:85
		{
			{
				global := Ast{[]FileLoc{}, map[string]bool{}, []*Filetype{}, map[string]bool{}, []*Stage{}, []*Pipeline{}, &Callables{[]Callable{}, map[string]Callable{}}, mmS[mmpt-0].call}
				mmlex.(*mmLexInfo).global = &global
			}
		}
	case 3:
		//line core/grammar.y:93
		{
			{
				mmVAL.decs = append(mmS[mmpt-1].decs, mmS[mmpt-0].dec)
			}
		}
	case 4:
		//line core/grammar.y:95
		{
			{
				mmVAL.decs = []Dec{mmS[mmpt-0].dec}
			}
		}
	case 5:
		//line core/grammar.y:100
		{
			{
				mmVAL.dec = &Filetype{NewAstNode(&mmlval), mmS[mmpt-1].val}
			}
		}
	case 6:
		//line core/grammar.y:102
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-5].val, mmS[mmpt-3].params, mmS[mmpt-2].params, mmS[mmpt-1].src, &Params{[]Param{}, map[string]Param{}}}
			}
		}
	case 7:
		//line core/grammar.y:104
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmS[mmpt-6].val, mmS[mmpt-4].params, mmS[mmpt-3].params, mmS[mmpt-2].src, mmS[mmpt-0].params}
			}
		}
	case 8:
		//line core/grammar.y:106
		{
			{
				mmVAL.dec = &Pipeline{NewAstNode(&mmlval), mmS[mmpt-8].val, mmS[mmpt-6].params, mmS[mmpt-5].params, mmS[mmpt-2].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmS[mmpt-1].retstm}
			}
		}
	case 9:
		//line core/grammar.y:111
		{
			{
				mmVAL.val = mmS[mmpt-2].val + mmS[mmpt-1].val + mmS[mmpt-0].val
			}
		}
	case 10:
		mmVAL.val = mmS[mmpt-0].val
	case 11:
		//line core/grammar.y:117
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].inparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 12:
		//line core/grammar.y:122
		{
			{
				mmVAL.params = &Params{[]Param{mmS[mmpt-0].inparam}, map[string]Param{}}
			}
		}
	case 13:
		//line core/grammar.y:127
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(&mmlval), mmS[mmpt-2].val, false, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 14:
		//line core/grammar.y:129
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(&mmlval), mmS[mmpt-4].val, true, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 15:
		//line core/grammar.y:134
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].outparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 16:
		//line core/grammar.y:139
		{
			{
				mmVAL.params = &Params{[]Param{mmS[mmpt-0].outparam}, map[string]Param{}}
			}
		}
	case 17:
		//line core/grammar.y:144
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-1].val, false, "default", unquote(mmS[mmpt-0].val), false}
			}
		}
	case 18:
		//line core/grammar.y:146
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-2].val, false, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 19:
		//line core/grammar.y:148
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-3].val, true, "default", unquote(mmS[mmpt-0].val), false}
			}
		}
	case 20:
		//line core/grammar.y:150
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmS[mmpt-4].val, true, mmS[mmpt-1].val, unquote(mmS[mmpt-0].val), false}
			}
		}
	case 21:
		//line core/grammar.y:155
		{
			{
				mmVAL.src = &SrcParam{NewAstNode(&mmlval), mmS[mmpt-2].val, unquote(mmS[mmpt-1].val)}
			}
		}
	case 22:
		//line core/grammar.y:160
		{
			{
				mmVAL.val = mmS[mmpt-1].val
			}
		}
	case 23:
		//line core/grammar.y:162
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
		//line core/grammar.y:174
		{
			{
				mmVAL.val = mmS[mmpt-2].val + "." + mmS[mmpt-0].val
			}
		}
	case 32:
		mmVAL.val = mmS[mmpt-0].val
	case 33:
		//line core/grammar.y:186
		{
			{
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 34:
		//line core/grammar.y:191
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(&mmlval), mmS[mmpt-1].bindings}
			}
		}
	case 35:
		//line core/grammar.y:196
		{
			{
				mmVAL.calls = append(mmS[mmpt-1].calls, mmS[mmpt-0].call)
			}
		}
	case 36:
		//line core/grammar.y:198
		{
			{
				mmVAL.calls = []*CallStm{mmS[mmpt-0].call}
			}
		}
	case 37:
		//line core/grammar.y:203
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), false, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 38:
		//line core/grammar.y:205
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), true, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 39:
		//line core/grammar.y:210
		{
			{
				mmS[mmpt-1].bindings.List = append(mmS[mmpt-1].bindings.List, mmS[mmpt-0].binding)
				mmVAL.bindings = mmS[mmpt-1].bindings
			}
		}
	case 40:
		//line core/grammar.y:215
		{
			{
				mmVAL.bindings = &BindStms{[]*BindStm{mmS[mmpt-0].binding}, map[string]*BindStm{}}
			}
		}
	case 41:
		//line core/grammar.y:220
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-3].val, mmS[mmpt-1].exp, false, ""}
			}
		}
	case 42:
		//line core/grammar.y:222
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmS[mmpt-6].val, mmS[mmpt-2].exp, true, ""}
			}
		}
	case 43:
		//line core/grammar.y:227
		{
			{
				mmVAL.exps = append(mmS[mmpt-2].exps, mmS[mmpt-0].exp)
			}
		}
	case 44:
		//line core/grammar.y:229
		{
			{
				mmVAL.exps = []Exp{mmS[mmpt-0].exp}
			}
		}
	case 45:
		//line core/grammar.y:234
		{
			{
				mmS[mmpt-4].kvpairs[unquote(mmS[mmpt-2].val)] = mmS[mmpt-0].exp
				mmVAL.kvpairs = mmS[mmpt-4].kvpairs
			}
		}
	case 46:
		//line core/grammar.y:239
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmS[mmpt-2].val): mmS[mmpt-0].exp}
			}
		}
	case 47:
		//line core/grammar.y:244
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "array", Value: mmS[mmpt-1].exps}
			}
		}
	case 48:
		//line core/grammar.y:246
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "array", Value: []Exp{}}
			}
		}
	case 49:
		//line core/grammar.y:248
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "map", Value: map[string]interface{}{}}
			}
		}
	case 50:
		//line core/grammar.y:250
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "map", Value: mmS[mmpt-1].kvpairs}
			}
		}
	case 51:
		//line core/grammar.y:252
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: mmS[mmpt-3].val, Value: unquote(mmS[mmpt-1].val)}
			}
		}
	case 52:
		//line core/grammar.y:254
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: mmS[mmpt-3].val, Value: unquote(mmS[mmpt-1].val)}
			}
		}
	case 53:
		//line core/grammar.y:256
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmS[mmpt-0].val, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "float", Value: f}
			}
		}
	case 54:
		//line core/grammar.y:261
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmS[mmpt-0].val, 0, 64)
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "int", Value: i}
			}
		}
	case 55:
		//line core/grammar.y:266
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "string", Value: unquote(mmS[mmpt-0].val)}
			}
		}
	case 56:
		//line core/grammar.y:268
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "bool", Value: true}
			}
		}
	case 57:
		//line core/grammar.y:270
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "bool", Value: false}
			}
		}
	case 58:
		//line core/grammar.y:272
		{
			{
				mmVAL.exp = &ValExp{node: NewAstNode(&mmlval), Kind: "null", Value: nil}
			}
		}
	case 59:
		//line core/grammar.y:274
		{
			{
				mmVAL.exp = mmS[mmpt-0].exp
			}
		}
	case 60:
		//line core/grammar.y:279
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-2].val, mmS[mmpt-0].val}
			}
		}
	case 61:
		//line core/grammar.y:281
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmS[mmpt-0].val, "default"}
			}
		}
	case 62:
		//line core/grammar.y:283
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "self", mmS[mmpt-0].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

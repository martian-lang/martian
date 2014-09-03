//line grammar.y:2

//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Margo
//
package core

import __yyfmt__ "fmt"

//line grammar.y:7
import (
	"strconv"
	"strings"
)

//line grammar.y:15
type mmSymType struct {
	yys      int
	global   *Ast
	loc      int
	val      string
	dec      Dec
	decs     []Dec
	inparam  *InParam
	outparam *OutParam
	params   *Params
	src      *SrcParam
	exp      Exp
	exps     []Exp
	call     *CallStm
	calls    []*CallStm
	binding  *BindStm
	bindings *BindStms
	retstm   *ReturnStm
}

const SKIP = 57346
const INVALID = 57347
const SEMICOLON = 57348
const LBRACKET = 57349
const RBRACKET = 57350
const LPAREN = 57351
const RPAREN = 57352
const LBRACE = 57353
const RBRACE = 57354
const COMMA = 57355
const EQUALS = 57356
const FILETYPE = 57357
const STAGE = 57358
const PIPELINE = 57359
const CALL = 57360
const VOLATILE = 57361
const SWEEP = 57362
const SPLIT = 57363
const USING = 57364
const SELF = 57365
const RETURN = 57366
const IN = 57367
const OUT = 57368
const SRC = 57369
const ID = 57370
const LITSTRING = 57371
const NUM_FLOAT = 57372
const NUM_INT = 57373
const DOT = 57374
const PY = 57375
const GO = 57376
const SH = 57377
const EXEC = 57378
const INT = 57379
const TSTRING = 57380
const FLOAT = 57381
const PATH = 57382
const FILE = 57383
const BOOL = 57384
const TRUE = 57385
const FALSE = 57386
const NULL = 57387
const DEFAULT = 57388

var mmToknames = []string{
	"SKIP",
	"INVALID",
	"SEMICOLON",
	"LBRACKET",
	"RBRACKET",
	"LPAREN",
	"RPAREN",
	"LBRACE",
	"RBRACE",
	"COMMA",
	"EQUALS",
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
	"INT",
	"TSTRING",
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

//line grammar.y:256

//line yacctab:1
var mmExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
}

const mmNprod = 55
const mmPrivate = 57344

var mmTokenNames []string
var mmStates []string

const mmLast = 159

var mmAct = []int{

	23, 22, 28, 3, 47, 81, 27, 39, 37, 80,
	45, 77, 76, 67, 35, 49, 72, 19, 84, 40,
	41, 43, 42, 32, 44, 84, 97, 34, 30, 118,
	36, 60, 92, 36, 83, 32, 59, 54, 52, 53,
	82, 83, 91, 49, 63, 46, 65, 24, 50, 51,
	61, 55, 56, 57, 73, 63, 48, 24, 94, 60,
	93, 11, 49, 86, 59, 54, 52, 53, 24, 31,
	10, 66, 85, 38, 64, 88, 50, 51, 60, 55,
	56, 57, 26, 59, 54, 52, 53, 24, 98, 17,
	15, 101, 14, 13, 103, 50, 51, 68, 55, 56,
	57, 29, 38, 29, 109, 117, 5, 106, 5, 90,
	96, 33, 110, 38, 89, 111, 116, 32, 36, 115,
	29, 6, 7, 8, 5, 6, 7, 8, 107, 99,
	69, 113, 87, 105, 104, 102, 78, 114, 112, 75,
	74, 70, 25, 21, 20, 16, 18, 4, 1, 108,
	9, 100, 71, 58, 62, 95, 2, 79, 12,
}
var mmPact = []int{

	106, -1000, 110, -1000, -1000, 42, 65, 64, 62, -1000,
	136, 61, 140, -15, 135, 134, 29, 133, -1000, 54,
	78, 78, 59, -1000, 97, 29, -1000, 76, -1000, -18,
	76, -1000, -1000, 36, 40, 47, -1000, -1000, -18, 43,
	-1000, -1000, -1000, -1000, -1000, -19, 87, 117, 132, 8,
	131, 130, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -20,
	-21, -1000, 126, -1000, -24, 12, 5, 35, 121, -1000,
	55, 101, -1000, -1000, 13, 3, 32, 30, 89, -3,
	-1000, -1000, 5, 116, -1000, -1000, -1000, 90, 125, 55,
	-1000, 124, 123, -1000, -1000, -1000, 85, 115, -1000, -1000,
	88, -1000, 102, -1000, -1000, -1000, 129, -1000, 119, -1000,
	128, -1000, 78, -1000, 29, 95, 19, -1000, -1000,
}
var mmPgo = []int{

	0, 158, 7, 5, 157, 147, 156, 2, 8, 6,
	14, 155, 154, 4, 153, 152, 3, 151, 0, 1,
	149, 148,
}
var mmR1 = []int{

	0, 21, 21, 6, 6, 5, 5, 5, 5, 1,
	1, 9, 9, 7, 10, 10, 8, 8, 12, 3,
	3, 2, 2, 2, 2, 2, 2, 2, 4, 11,
	20, 17, 17, 16, 16, 19, 19, 18, 18, 15,
	15, 13, 13, 13, 13, 13, 13, 13, 13, 13,
	13, 13, 14, 14, 14,
}
var mmR2 = []int{

	0, 1, 1, 2, 1, 3, 7, 8, 10, 3,
	1, 2, 1, 4, 2, 1, 3, 4, 4, 2,
	1, 1, 1, 1, 1, 1, 1, 3, 1, 5,
	4, 2, 1, 5, 6, 2, 1, 4, 7, 3,
	1, 3, 2, 4, 4, 1, 1, 1, 1, 1,
	1, 1, 3, 1, 3,
}
var mmChk = []int{

	-1000, -21, -6, -16, -5, 18, 15, 16, 17, -5,
	28, 19, -1, 28, 28, 28, 9, 28, 6, 32,
	9, 9, -19, -18, 28, 9, 28, -9, -7, 25,
	-9, 10, -18, 14, -19, -10, -7, -8, 26, -2,
	37, 38, 40, 39, 42, 28, -10, -13, 20, 7,
	40, 41, 30, 31, 29, 43, 44, 45, -14, 28,
	23, 10, -12, -8, 27, -2, 28, 32, 10, 13,
	9, -15, 8, -13, 9, 9, 32, 32, 10, -4,
	33, -3, 28, 29, 13, -3, 28, 11, -13, 13,
	8, 29, 29, 28, 28, -11, 21, 29, -3, 13,
	-17, -16, 10, -13, 10, 10, 22, 13, -20, -16,
	24, 13, 9, 12, 9, -9, -19, 10, 10,
}
var mmDef = []int{

	0, -2, 1, 2, 4, 0, 0, 0, 0, 3,
	0, 0, 0, 10, 0, 0, 0, 0, 5, 0,
	0, 0, 0, 36, 0, 0, 9, 0, 12, 0,
	0, 33, 35, 0, 0, 0, 11, 15, 0, 0,
	21, 22, 23, 24, 25, 26, 0, 0, 0, 0,
	0, 0, 45, 46, 47, 48, 49, 50, 51, 53,
	0, 34, 0, 14, 0, 0, 0, 0, 0, 37,
	0, 0, 42, 40, 0, 0, 0, 0, 6, 0,
	28, 16, 0, 0, 20, 13, 27, 0, 0, 0,
	41, 0, 0, 52, 54, 7, 0, 0, 17, 19,
	0, 32, 0, 39, 43, 44, 0, 18, 0, 31,
	0, 38, 0, 8, 0, 0, 0, 29, 30,
}
var mmTok1 = []int{

	1,
}
var mmTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
	32, 33, 34, 35, 36, 37, 38, 39, 40, 41,
	42, 43, 44, 45, 46,
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
		//line grammar.y:60
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
		//line grammar.y:77
		{
			{
				global := Ast{[]FileLoc{}, map[string]bool{}, []*Filetype{}, map[string]bool{}, []*Stage{}, []*Pipeline{}, &Callables{[]Callable{}, map[string]Callable{}}, mmS[mmpt-0].call}
				mmlex.(*mmLexInfo).global = &global
			}
		}
	case 3:
		//line grammar.y:85
		{
			{
				mmVAL.decs = append(mmS[mmpt-1].decs, mmS[mmpt-0].dec)
			}
		}
	case 4:
		//line grammar.y:87
		{
			{
				mmVAL.decs = []Dec{mmS[mmpt-0].dec}
			}
		}
	case 5:
		//line grammar.y:92
		{
			{
				mmVAL.dec = &Filetype{AstNode{mmlval.loc}, mmS[mmpt-1].val}
			}
		}
	case 6:
		//line grammar.y:94
		{
			{
				mmVAL.dec = &Stage{AstNode{mmlval.loc}, mmS[mmpt-5].val, mmS[mmpt-3].params, mmS[mmpt-2].params, mmS[mmpt-1].src, &Params{[]Param{}, map[string]Param{}}}
			}
		}
	case 7:
		//line grammar.y:96
		{
			{
				mmVAL.dec = &Stage{AstNode{mmlval.loc}, mmS[mmpt-6].val, mmS[mmpt-4].params, mmS[mmpt-3].params, mmS[mmpt-2].src, mmS[mmpt-0].params}
			}
		}
	case 8:
		//line grammar.y:98
		{
			{
				mmVAL.dec = &Pipeline{AstNode{mmlval.loc}, mmS[mmpt-8].val, mmS[mmpt-6].params, mmS[mmpt-5].params, mmS[mmpt-2].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmS[mmpt-1].retstm}
			}
		}
	case 9:
		//line grammar.y:103
		{
			{
				mmVAL.val = mmS[mmpt-2].val + mmS[mmpt-1].val + mmS[mmpt-0].val
			}
		}
	case 10:
		mmVAL.val = mmS[mmpt-0].val
	case 11:
		//line grammar.y:109
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].inparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 12:
		//line grammar.y:114
		{
			{
				mmVAL.params = &Params{[]Param{mmS[mmpt-0].inparam}, map[string]Param{}}
			}
		}
	case 13:
		//line grammar.y:119
		{
			{
				mmVAL.inparam = &InParam{AstNode{mmlval.loc}, mmS[mmpt-2].val, mmS[mmpt-1].val, mmS[mmpt-0].val, false}
			}
		}
	case 14:
		//line grammar.y:124
		{
			{
				mmS[mmpt-1].params.list = append(mmS[mmpt-1].params.list, mmS[mmpt-0].outparam)
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 15:
		//line grammar.y:129
		{
			{
				mmVAL.params = &Params{[]Param{mmS[mmpt-0].outparam}, map[string]Param{}}
			}
		}
	case 16:
		//line grammar.y:134
		{
			{
				mmVAL.outparam = &OutParam{AstNode{mmlval.loc}, mmS[mmpt-1].val, "default", mmS[mmpt-0].val, false}
			}
		}
	case 17:
		//line grammar.y:136
		{
			{
				mmVAL.outparam = &OutParam{AstNode{mmlval.loc}, mmS[mmpt-2].val, mmS[mmpt-1].val, mmS[mmpt-0].val, false}
			}
		}
	case 18:
		//line grammar.y:141
		{
			{
				mmVAL.src = &SrcParam{AstNode{mmlval.loc}, mmS[mmpt-2].val, strings.Replace(mmS[mmpt-1].val, "\"", "", -1)}
			}
		}
	case 19:
		//line grammar.y:146
		{
			{
				mmVAL.val = mmS[mmpt-1].val
			}
		}
	case 20:
		//line grammar.y:148
		{
			{
				mmVAL.val = ""
			}
		}
	case 21:
		mmVAL.val = mmS[mmpt-0].val
	case 22:
		mmVAL.val = mmS[mmpt-0].val
	case 23:
		mmVAL.val = mmS[mmpt-0].val
	case 24:
		mmVAL.val = mmS[mmpt-0].val
	case 25:
		mmVAL.val = mmS[mmpt-0].val
	case 26:
		mmVAL.val = mmS[mmpt-0].val
	case 27:
		//line grammar.y:159
		{
			{
				mmVAL.val = mmS[mmpt-2].val + "." + mmS[mmpt-0].val
			}
		}
	case 28:
		mmVAL.val = mmS[mmpt-0].val
	case 29:
		//line grammar.y:171
		{
			{
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 30:
		//line grammar.y:176
		{
			{
				mmVAL.retstm = &ReturnStm{AstNode{mmlval.loc}, mmS[mmpt-1].bindings}
			}
		}
	case 31:
		//line grammar.y:181
		{
			{
				mmVAL.calls = append(mmS[mmpt-1].calls, mmS[mmpt-0].call)
			}
		}
	case 32:
		//line grammar.y:183
		{
			{
				mmVAL.calls = []*CallStm{mmS[mmpt-0].call}
			}
		}
	case 33:
		//line grammar.y:188
		{
			{
				mmVAL.call = &CallStm{AstNode{mmlval.loc}, false, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 34:
		//line grammar.y:190
		{
			{
				mmVAL.call = &CallStm{AstNode{mmlval.loc}, true, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 35:
		//line grammar.y:195
		{
			{
				mmS[mmpt-1].bindings.List = append(mmS[mmpt-1].bindings.List, mmS[mmpt-0].binding)
				mmVAL.bindings = mmS[mmpt-1].bindings
			}
		}
	case 36:
		//line grammar.y:200
		{
			{
				mmVAL.bindings = &BindStms{[]*BindStm{mmS[mmpt-0].binding}, map[string]*BindStm{}}
			}
		}
	case 37:
		//line grammar.y:205
		{
			{
				mmVAL.binding = &BindStm{AstNode{mmlval.loc}, mmS[mmpt-3].val, mmS[mmpt-1].exp, false, ""}
			}
		}
	case 38:
		//line grammar.y:207
		{
			{
				mmVAL.binding = &BindStm{AstNode{mmlval.loc}, mmS[mmpt-6].val, mmS[mmpt-2].exp, true, ""}
			}
		}
	case 39:
		//line grammar.y:212
		{
			{
				mmVAL.exps = append(mmS[mmpt-2].exps, mmS[mmpt-0].exp)
			}
		}
	case 40:
		//line grammar.y:214
		{
			{
				mmVAL.exps = []Exp{mmS[mmpt-0].exp}
			}
		}
	case 41:
		//line grammar.y:219
		{
			{
				mmVAL.exp = &ValExp{node: AstNode{mmlval.loc}, Kind: "array", Value: mmS[mmpt-1].exps}
			}
		}
	case 42:
		//line grammar.y:221
		{
			{
				mmVAL.exp = &ValExp{node: AstNode{mmlval.loc}, Kind: "array", Value: []Exp{}}
			}
		}
	case 43:
		//line grammar.y:223
		{
			{
				mmVAL.exp = &ValExp{node: AstNode{mmlval.loc}, Kind: mmS[mmpt-3].val, Value: strings.Replace(mmS[mmpt-1].val, "\"", "", -1)}
			}
		}
	case 44:
		//line grammar.y:225
		{
			{
				mmVAL.exp = &ValExp{node: AstNode{mmlval.loc}, Kind: mmS[mmpt-3].val, Value: strings.Replace(mmS[mmpt-1].val, "\"", "", -1)}
			}
		}
	case 45:
		//line grammar.y:227
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmS[mmpt-0].val, 64)
				mmVAL.exp = &ValExp{node: AstNode{mmlval.loc}, Kind: "float", Value: f}
			}
		}
	case 46:
		//line grammar.y:232
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmS[mmpt-0].val, 0, 64)
				mmVAL.exp = &ValExp{node: AstNode{mmlval.loc}, Kind: "int", Value: i}
			}
		}
	case 47:
		//line grammar.y:237
		{
			{
				mmVAL.exp = &ValExp{node: AstNode{mmlval.loc}, Kind: "string", Value: strings.Replace(mmS[mmpt-0].val, "\"", "", -1)}
			}
		}
	case 48:
		//line grammar.y:239
		{
			{
				mmVAL.exp = &ValExp{node: AstNode{mmlval.loc}, Kind: "bool", Value: true}
			}
		}
	case 49:
		//line grammar.y:241
		{
			{
				mmVAL.exp = &ValExp{node: AstNode{mmlval.loc}, Kind: "bool", Value: false}
			}
		}
	case 50:
		//line grammar.y:243
		{
			{
				mmVAL.exp = &ValExp{node: AstNode{mmlval.loc}, Kind: "null", Value: nil}
			}
		}
	case 51:
		//line grammar.y:245
		{
			{
				mmVAL.exp = mmS[mmpt-0].exp
			}
		}
	case 52:
		//line grammar.y:250
		{
			{
				mmVAL.exp = &RefExp{AstNode{mmlval.loc}, "call", mmS[mmpt-2].val, mmS[mmpt-0].val}
			}
		}
	case 53:
		//line grammar.y:252
		{
			{
				mmVAL.exp = &RefExp{AstNode{mmlval.loc}, "call", mmS[mmpt-0].val, "default"}
			}
		}
	case 54:
		//line grammar.y:254
		{
			{
				mmVAL.exp = &RefExp{AstNode{mmlval.loc}, "self", mmS[mmpt-0].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

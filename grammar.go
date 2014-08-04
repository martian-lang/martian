//line grammar.y:2
package main

import __yyfmt__ "fmt"

//line grammar.y:2
import (
	"strconv"
	"strings"
)

//line grammar.y:10
type mmSymType struct {
	yys      int
	loc      int
	val      string
	dec      Dec
	decs     []Dec
	param    Param
	params   []Param
	exp      Exp
	exps     []Exp
	stm      Stm
	stms     []Stm
	call     *CallStm
	calls    []*CallStm
	binding  *BindStm
	bindings []*BindStm
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
const STRING = 57380
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

//line grammar.y:207

//line yacctab:1
var mmExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
}

const mmNprod = 53
const mmPrivate = 57344

var mmTokenNames []string
var mmStates []string

const mmLast = 158

var mmAct = []int{

	23, 22, 28, 27, 50, 3, 69, 45, 48, 83,
	52, 78, 82, 68, 19, 96, 40, 41, 43, 42,
	72, 44, 95, 34, 72, 32, 63, 36, 39, 52,
	38, 62, 57, 55, 56, 38, 71, 34, 113, 70,
	71, 73, 51, 53, 54, 63, 58, 59, 60, 24,
	62, 57, 55, 56, 111, 64, 24, 79, 33, 46,
	98, 97, 53, 54, 86, 58, 59, 60, 52, 29,
	30, 31, 84, 24, 85, 67, 24, 87, 49, 26,
	91, 92, 17, 15, 63, 29, 30, 31, 11, 62,
	57, 55, 56, 29, 30, 31, 101, 10, 104, 37,
	14, 53, 54, 107, 58, 59, 60, 13, 66, 5,
	38, 112, 5, 34, 29, 30, 31, 35, 102, 6,
	7, 8, 5, 6, 7, 8, 94, 110, 89, 88,
	75, 93, 108, 74, 106, 105, 103, 109, 99, 81,
	80, 76, 25, 21, 20, 16, 18, 4, 1, 100,
	9, 90, 77, 61, 65, 2, 47, 12,
}
var mmPact = []int{

	104, -1000, 108, -1000, -1000, 69, 79, 72, 55, -1000,
	136, 54, 140, -18, 135, 134, 21, 133, -1000, 51,
	60, 60, 48, -1000, 103, 21, -1000, 89, -1000, -21,
	-21, -25, 68, -1000, -1000, 22, 45, 87, -1000, 47,
	-1000, -1000, -1000, -1000, -1000, -19, 11, 12, -1000, 122,
	117, 132, 3, 131, 130, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -20, -23, -1000, -1000, 50, 7, 36, -1000,
	7, 116, -1000, 115, 91, -1000, 61, 118, -1000, -1000,
	-7, -14, 33, 32, 129, -1000, -1000, -1000, -1000, -1000,
	94, -1000, 126, 61, -1000, 125, 124, -1000, -1000, 60,
	120, -1000, 128, 114, -1000, -1000, -1000, 44, -1000, 21,
	-1000, -1000, 28, -1000,
}
var mmPgo = []int{

	0, 157, 28, 6, 156, 147, 155, 2, 3, 154,
	4, 153, 152, 5, 151, 0, 1, 149, 148,
}
var mmR1 = []int{

	0, 18, 18, 6, 6, 5, 5, 5, 5, 1,
	1, 8, 8, 7, 7, 7, 7, 3, 3, 2,
	2, 2, 2, 2, 2, 2, 4, 9, 17, 14,
	14, 13, 13, 16, 16, 15, 15, 12, 12, 10,
	10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
	11, 11, 11,
}
var mmR2 = []int{

	0, 1, 1, 2, 1, 3, 5, 6, 9, 3,
	1, 2, 1, 4, 3, 4, 4, 2, 1, 1,
	1, 1, 1, 1, 1, 3, 1, 5, 4, 2,
	1, 5, 6, 2, 1, 4, 7, 3, 1, 3,
	2, 4, 4, 1, 1, 1, 1, 1, 1, 1,
	3, 1, 3,
}
var mmChk = []int{

	-1000, -18, -6, -13, -5, 18, 15, 16, 17, -5,
	28, 19, -1, 28, 28, 28, 9, 28, 6, 32,
	9, 9, -16, -15, 28, 9, 28, -8, -7, 25,
	26, 27, -8, 10, -15, 14, -16, 10, -7, -2,
	37, 38, 40, 39, 42, 28, -2, -4, 33, 10,
	-10, 20, 7, 40, 41, 30, 31, 29, 43, 44,
	45, -11, 28, 23, 10, -9, 21, 28, 32, -3,
	28, 29, 13, 29, 11, 13, 9, -12, 8, -10,
	9, 9, 32, 32, 22, -3, 28, -3, 13, 13,
	-14, -13, -10, 13, 8, 29, 29, 28, 28, 9,
	-17, -13, 24, 10, -10, 10, 10, -8, 12, 9,
	13, 10, -16, 10,
}
var mmDef = []int{

	0, -2, 1, 2, 4, 0, 0, 0, 0, 3,
	0, 0, 0, 10, 0, 0, 0, 0, 5, 0,
	0, 0, 0, 34, 0, 0, 9, 0, 12, 0,
	0, 0, 0, 31, 33, 0, 0, 6, 11, 0,
	19, 20, 21, 22, 23, 24, 0, 0, 26, 0,
	0, 0, 0, 0, 0, 43, 44, 45, 46, 47,
	48, 49, 51, 0, 32, 7, 0, 0, 0, 14,
	0, 0, 18, 0, 0, 35, 0, 0, 40, 38,
	0, 0, 0, 0, 0, 13, 25, 15, 17, 16,
	0, 30, 0, 0, 39, 0, 0, 50, 52, 0,
	0, 29, 0, 0, 37, 41, 42, 0, 8, 0,
	36, 27, 0, 28,
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
		//line grammar.y:52
		{
			{
				ast = Ast{mmS[mmpt-0].decs, nil}
			}
		}
	case 2:
		//line grammar.y:54
		{
			{
				ast = Ast{nil, mmS[mmpt-0].call}
			}
		}
	case 3:
		//line grammar.y:59
		{
			{
				mmVAL.decs = append(mmS[mmpt-1].decs, mmS[mmpt-0].dec)
			}
		}
	case 4:
		//line grammar.y:61
		{
			{
				mmVAL.decs = []Dec{mmS[mmpt-0].dec}
			}
		}
	case 5:
		//line grammar.y:66
		{
			{
				mmVAL.dec = &FileTypeDec{Node{mmlval.loc}, mmS[mmpt-1].val}
			}
		}
	case 6:
		//line grammar.y:68
		{
			{
				mmVAL.dec = &StageDec{Node{mmlval.loc}, mmS[mmpt-3].val, mmS[mmpt-1].params, nil}
			}
		}
	case 7:
		//line grammar.y:70
		{
			{
				mmVAL.dec = &StageDec{Node{mmlval.loc}, mmS[mmpt-4].val, mmS[mmpt-2].params, mmS[mmpt-0].params}
			}
		}
	case 8:
		//line grammar.y:72
		{
			{
				mmVAL.dec = &PipelineDec{Node{mmlval.loc}, mmS[mmpt-7].val, mmS[mmpt-5].params, mmS[mmpt-2].calls, mmS[mmpt-1].retstm}
			}
		}
	case 9:
		//line grammar.y:77
		{
			{
				mmVAL.val = mmS[mmpt-2].val + mmS[mmpt-1].val + mmS[mmpt-0].val
			}
		}
	case 10:
		mmVAL.val = mmS[mmpt-0].val
	case 11:
		//line grammar.y:83
		{
			{
				mmVAL.params = append(mmS[mmpt-1].params, mmS[mmpt-0].param)
			}
		}
	case 12:
		//line grammar.y:85
		{
			{
				mmVAL.params = []Param{mmS[mmpt-0].param}
			}
		}
	case 13:
		//line grammar.y:90
		{
			{
				mmVAL.param = &InParam{Node{mmlval.loc}, mmS[mmpt-2].val, mmS[mmpt-1].val, mmS[mmpt-0].val}
			}
		}
	case 14:
		//line grammar.y:92
		{
			{
				mmVAL.param = &OutParam{Node{mmlval.loc}, mmS[mmpt-1].val, "default", mmS[mmpt-0].val}
			}
		}
	case 15:
		//line grammar.y:94
		{
			{
				mmVAL.param = &OutParam{Node{mmlval.loc}, mmS[mmpt-2].val, mmS[mmpt-1].val, mmS[mmpt-0].val}
			}
		}
	case 16:
		//line grammar.y:96
		{
			{
				mmVAL.param = &SourceParam{Node{mmlval.loc}, mmS[mmpt-2].val, mmS[mmpt-1].val}
			}
		}
	case 17:
		//line grammar.y:100
		{
			{
				mmVAL.val = mmS[mmpt-1].val
			}
		}
	case 18:
		//line grammar.y:102
		{
			{
				mmVAL.val = ""
			}
		}
	case 19:
		mmVAL.val = mmS[mmpt-0].val
	case 20:
		mmVAL.val = mmS[mmpt-0].val
	case 21:
		mmVAL.val = mmS[mmpt-0].val
	case 22:
		mmVAL.val = mmS[mmpt-0].val
	case 23:
		mmVAL.val = mmS[mmpt-0].val
	case 24:
		mmVAL.val = mmS[mmpt-0].val
	case 25:
		//line grammar.y:113
		{
			{
				mmVAL.val = mmS[mmpt-2].val + "." + mmS[mmpt-0].val
			}
		}
	case 26:
		mmVAL.val = mmS[mmpt-0].val
	case 27:
		//line grammar.y:125
		{
			{
				mmVAL.params = mmS[mmpt-1].params
			}
		}
	case 28:
		//line grammar.y:130
		{
			{
				mmVAL.retstm = &ReturnStm{Node{mmlval.loc}, mmS[mmpt-1].bindings}
			}
		}
	case 29:
		//line grammar.y:135
		{
			{
				mmVAL.calls = append(mmS[mmpt-1].calls, mmS[mmpt-0].call)
			}
		}
	case 30:
		//line grammar.y:137
		{
			{
				mmVAL.calls = []*CallStm{mmS[mmpt-0].call}
			}
		}
	case 31:
		//line grammar.y:142
		{
			{
				mmVAL.call = &CallStm{Node{mmlval.loc}, false, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 32:
		//line grammar.y:144
		{
			{
				mmVAL.call = &CallStm{Node{mmlval.loc}, true, mmS[mmpt-3].val, mmS[mmpt-1].bindings}
			}
		}
	case 33:
		//line grammar.y:149
		{
			{
				mmVAL.bindings = append(mmS[mmpt-1].bindings, mmS[mmpt-0].binding)
			}
		}
	case 34:
		//line grammar.y:151
		{
			{
				mmVAL.bindings = []*BindStm{mmS[mmpt-0].binding}
			}
		}
	case 35:
		//line grammar.y:156
		{
			{
				mmVAL.binding = &BindStm{Node{mmlval.loc}, mmS[mmpt-3].val, mmS[mmpt-1].exp, false}
			}
		}
	case 36:
		//line grammar.y:158
		{
			{
				mmVAL.binding = &BindStm{Node{mmlval.loc}, mmS[mmpt-6].val, mmS[mmpt-2].exp, true}
			}
		}
	case 37:
		//line grammar.y:163
		{
			{
				mmVAL.exps = append(mmS[mmpt-2].exps, mmS[mmpt-0].exp)
			}
		}
	case 38:
		//line grammar.y:165
		{
			{
				mmVAL.exps = []Exp{mmS[mmpt-0].exp}
			}
		}
	case 39:
		//line grammar.y:170
		{
			{
				mmVAL.exp = nil
			}
		}
	case 40:
		//line grammar.y:172
		{
			{
				mmVAL.exp = nil
			}
		}
	case 41:
		//line grammar.y:174
		{
			{
				mmVAL.exp = &ValExp{Node: Node{mmlval.loc}, kind: mmS[mmpt-3].val, sval: strings.Replace(mmS[mmpt-1].val, "\"", "", -1)}
			}
		}
	case 42:
		//line grammar.y:176
		{
			{
				mmVAL.exp = &ValExp{Node: Node{mmlval.loc}, kind: mmS[mmpt-3].val, sval: strings.Replace(mmS[mmpt-1].val, "\"", "", -1)}
			}
		}
	case 43:
		//line grammar.y:178
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmS[mmpt-0].val, 64)
				mmVAL.exp = &ValExp{Node: Node{mmlval.loc}, kind: "float", fval: f}
			}
		}
	case 44:
		//line grammar.y:183
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmS[mmpt-0].val, 0, 64)
				mmVAL.exp = &ValExp{Node: Node{mmlval.loc}, kind: "int", ival: i}
			}
		}
	case 45:
		//line grammar.y:188
		{
			{
				mmVAL.exp = &ValExp{Node: Node{mmlval.loc}, kind: "string", sval: strings.Replace(mmS[mmpt-0].val, "\"", "", -1)}
			}
		}
	case 46:
		//line grammar.y:190
		{
			{
				mmVAL.exp = &ValExp{Node: Node{mmlval.loc}, kind: "bool", bval: true}
			}
		}
	case 47:
		//line grammar.y:192
		{
			{
				mmVAL.exp = &ValExp{Node: Node{mmlval.loc}, kind: "bool", bval: false}
			}
		}
	case 48:
		//line grammar.y:194
		{
			{
				mmVAL.exp = &ValExp{Node: Node{mmlval.loc}, kind: "null", null: true}
			}
		}
	case 49:
		//line grammar.y:196
		{
			{
				mmVAL.exp = mmS[mmpt-0].exp
			}
		}
	case 50:
		//line grammar.y:201
		{
			{
				mmVAL.exp = &RefExp{Node{mmlval.loc}, "call", mmS[mmpt-2].val, mmS[mmpt-0].val}
			}
		}
	case 51:
		//line grammar.y:203
		{
			{
				mmVAL.exp = &RefExp{Node{mmlval.loc}, "call", mmS[mmpt-0].val, "default"}
			}
		}
	case 52:
		//line grammar.y:205
		{
			{
				mmVAL.exp = &RefExp{Node{mmlval.loc}, "self", mmS[mmpt-0].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

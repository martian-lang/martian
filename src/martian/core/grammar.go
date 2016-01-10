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
	yys       int
	global    *Ast
	arr       int
	loc       int
	val       string
	comments  string
	modifiers *Modifiers
	dec       Dec
	decs      []Dec
	inparam   *InParam
	outparam  *OutParam
	params    *Params
	src       *SrcParam
	exp       Exp
	exps      []Exp
	kvpairs   map[string]Exp
	call      *CallStm
	calls     []*CallStm
	binding   *BindStm
	bindings  *BindStms
	retstm    *ReturnStm
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

var mmToknames = [...]string{
	"$end",
	"error",
	"$unk",
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
var mmStatenames = [...]string{}

const mmEofCode = 1
const mmErrCode = 2
const mmMaxDepth = 200

//line src/martian/core/grammar.y:287

//line yacctab:1
var mmExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const mmNprod = 64
const mmPrivate = 57344

var mmTokenNames []string
var mmStates []string

const mmLast = 153

var mmAct = [...]int{

	82, 28, 26, 90, 80, 3, 59, 81, 9, 55,
	60, 52, 40, 87, 53, 86, 93, 20, 74, 21,
	83, 114, 69, 17, 18, 19, 27, 68, 63, 61,
	62, 121, 59, 93, 16, 106, 60, 85, 74, 91,
	92, 72, 64, 65, 66, 58, 21, 33, 69, 35,
	57, 105, 54, 68, 63, 61, 62, 92, 25, 75,
	39, 38, 77, 13, 73, 35, 15, 14, 64, 65,
	66, 13, 46, 41, 42, 44, 43, 45, 48, 95,
	5, 120, 97, 31, 99, 88, 71, 110, 98, 5,
	29, 56, 102, 59, 39, 108, 31, 60, 100, 103,
	50, 113, 94, 112, 111, 115, 107, 79, 100, 69,
	116, 101, 49, 117, 68, 63, 61, 62, 32, 24,
	122, 6, 7, 8, 5, 23, 22, 118, 109, 64,
	65, 66, 89, 78, 119, 104, 47, 4, 1, 96,
	10, 34, 76, 12, 84, 67, 36, 70, 37, 30,
	2, 11, 51,
}
var mmPact = [...]int{

	105, -1000, 105, -1000, -1000, -1000, 40, 36, 35, -1000,
	-1000, 3, 11, -1000, 114, 113, 107, -1000, -1000, -1000,
	-1000, 27, -1000, -1000, -1000, -1000, 55, 55, 34, 31,
	-1000, 32, 65, -1000, -1000, 103, 87, -1000, -25, 32,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -16, 77, 22,
	62, 9, -1000, -1000, -1000, 28, 70, 125, 95, -4,
	5, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -20, -22,
	-1000, 60, 124, 8, 91, 25, 61, -1000, -1000, 83,
	100, -1000, -1000, -1000, 84, 128, 20, 4, 94, -1000,
	-1000, 25, 120, -1000, -1000, -1000, 72, -1000, 92, 90,
	83, -1000, -11, -1000, 83, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, 119, -1000, 127, -1000, 68, 18, -1000, 83,
	-1000, -1000, -1000,
}
var mmPgo = [...]int{

	0, 136, 12, 3, 152, 151, 9, 137, 150, 149,
	148, 2, 90, 147, 146, 0, 145, 4, 144, 5,
	142, 141, 1, 139, 138,
}
var mmR1 = [...]int{

	0, 24, 24, 24, 8, 8, 7, 7, 7, 7,
	1, 1, 6, 6, 11, 11, 9, 12, 12, 10,
	10, 14, 3, 3, 2, 2, 2, 2, 2, 2,
	2, 4, 4, 13, 23, 20, 20, 19, 5, 5,
	5, 5, 22, 22, 21, 21, 17, 17, 18, 18,
	15, 15, 15, 15, 15, 15, 15, 15, 15, 15,
	15, 16, 16, 16,
}
var mmR2 = [...]int{

	0, 1, 2, 1, 2, 1, 3, 7, 8, 10,
	3, 1, 0, 3, 0, 2, 5, 0, 2, 4,
	5, 4, 2, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 5, 4, 2, 1, 6, 0, 2,
	2, 2, 0, 2, 4, 7, 3, 1, 5, 3,
	3, 2, 2, 3, 1, 1, 1, 1, 1, 1,
	1, 3, 1, 3,
}
var mmChk = [...]int{

	-1000, -24, -8, -19, -7, 19, 16, 17, 18, -19,
	-7, -5, -1, 31, 31, 31, 31, 20, 21, 22,
	6, 35, 12, 12, 12, 31, -11, -11, -22, -12,
	-9, 28, -12, 13, -21, 31, -14, -10, 30, 29,
	-2, 41, 42, 44, 43, 45, 40, -1, 13, 9,
	13, -4, 36, 39, -2, -6, 14, -15, 23, 10,
	14, 33, 34, 32, 46, 47, 48, -16, 31, 26,
	-13, 24, 32, -6, 10, 31, -20, -19, 8, 12,
	-17, 11, -15, 15, -18, 32, 35, 35, 25, 8,
	-3, 31, 32, 8, 11, -3, -23, -19, 27, -17,
	8, 11, 8, 15, 7, 31, 31, 12, -3, 8,
	15, 12, 13, -15, 32, -15, -11, -22, 8, 7,
	13, 13, -15,
}
var mmDef = [...]int{

	0, -2, 1, 3, 5, 38, 0, 0, 0, 2,
	4, 0, 0, 11, 0, 0, 0, 39, 40, 41,
	6, 0, 14, 14, 42, 10, 17, 17, 0, 0,
	15, 0, 0, 37, 43, 0, 0, 18, 0, 0,
	12, 24, 25, 26, 27, 28, 29, 30, 0, 0,
	7, 0, 31, 32, 12, 0, 0, 0, 0, 0,
	0, 54, 55, 56, 57, 58, 59, 60, 62, 0,
	8, 0, 0, 0, 0, 0, 0, 36, 44, 0,
	0, 51, 47, 52, 0, 0, 0, 0, 0, 21,
	19, 0, 0, 23, 13, 16, 0, 35, 0, 0,
	0, 50, 0, 53, 0, 61, 63, 14, 20, 22,
	9, 42, 0, 46, 0, 49, 0, 0, 45, 0,
	33, 34, 48,
}
var mmTok1 = [...]int{

	1,
}
var mmTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
	32, 33, 34, 35, 36, 37, 38, 39, 40, 41,
	42, 43, 44, 45, 46, 47, 48, 49,
}
var mmTok3 = [...]int{
	0,
}

var mmErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	mmDebug        = 0
	mmErrorVerbose = false
)

type mmLexer interface {
	Lex(lval *mmSymType) int
	Error(s string)
}

type mmParser interface {
	Parse(mmLexer) int
	Lookahead() int
}

type mmParserImpl struct {
	lookahead func() int
}

func (p *mmParserImpl) Lookahead() int {
	return p.lookahead()
}

func mmNewParser() mmParser {
	p := &mmParserImpl{
		lookahead: func() int { return -1 },
	}
	return p
}

const mmFlag = -1000

func mmTokname(c int) string {
	if c >= 1 && c-1 < len(mmToknames) {
		if mmToknames[c-1] != "" {
			return mmToknames[c-1]
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

func mmErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !mmErrorVerbose {
		return "syntax error"
	}

	for _, e := range mmErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + mmTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := mmPact[state]
	for tok := TOKSTART; tok-1 < len(mmToknames); tok++ {
		if n := base + tok; n >= 0 && n < mmLast && mmChk[mmAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if mmDef[state] == -2 {
		i := 0
		for mmExca[i] != -1 || mmExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; mmExca[i] >= 0; i += 2 {
			tok := mmExca[i]
			if tok < TOKSTART || mmExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if mmExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += mmTokname(tok)
	}
	return res
}

func mmlex1(lex mmLexer, lval *mmSymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = mmTok1[0]
		goto out
	}
	if char < len(mmTok1) {
		token = mmTok1[char]
		goto out
	}
	if char >= mmPrivate {
		if char < mmPrivate+len(mmTok2) {
			token = mmTok2[char-mmPrivate]
			goto out
		}
	}
	for i := 0; i < len(mmTok3); i += 2 {
		token = mmTok3[i+0]
		if token == char {
			token = mmTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = mmTok2[1] /* unknown char */
	}
	if mmDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", mmTokname(token), uint(char))
	}
	return char, token
}

func mmParse(mmlex mmLexer) int {
	return mmNewParser().Parse(mmlex)
}

func (mmrcvr *mmParserImpl) Parse(mmlex mmLexer) int {
	var mmn int
	var mmlval mmSymType
	var mmVAL mmSymType
	var mmDollar []mmSymType
	_ = mmDollar // silence set and not used
	mmS := make([]mmSymType, mmMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	mmstate := 0
	mmchar := -1
	mmtoken := -1 // mmchar translated into internal numbering
	mmrcvr.lookahead = func() int { return mmchar }
	defer func() {
		// Make sure we report no lookahead when not parsing.
		mmstate = -1
		mmchar = -1
		mmtoken = -1
	}()
	mmp := -1
	goto mmstack

ret0:
	return 0

ret1:
	return 1

mmstack:
	/* put a state and value onto the stack */
	if mmDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", mmTokname(mmtoken), mmStatname(mmstate))
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
		mmchar, mmtoken = mmlex1(mmlex, &mmlval)
	}
	mmn += mmtoken
	if mmn < 0 || mmn >= mmLast {
		goto mmdefault
	}
	mmn = mmAct[mmn]
	if mmChk[mmn] == mmtoken { /* valid shift */
		mmchar = -1
		mmtoken = -1
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
			mmchar, mmtoken = mmlex1(mmlex, &mmlval)
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
			if mmn < 0 || mmn == mmtoken {
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
			mmlex.Error(mmErrorMessage(mmstate, mmtoken))
			Nerrs++
			if mmDebug >= 1 {
				__yyfmt__.Printf("%s", mmStatname(mmstate))
				__yyfmt__.Printf(" saw %s\n", mmTokname(mmtoken))
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
				__yyfmt__.Printf("error recovery discards %s\n", mmTokname(mmtoken))
			}
			if mmtoken == mmEofCode {
				goto ret1
			}
			mmchar = -1
			mmtoken = -1
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
	// mmp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if mmp+1 >= len(mmS) {
		nyys := make([]mmSymType, len(mmS)*2)
		copy(nyys, mmS)
		mmS = nyys
	}
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
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:72
		{
			{
				global := NewAst(mmDollar[1].decs, nil)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 2:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:77
		{
			{
				global := NewAst(mmDollar[1].decs, mmDollar[2].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 3:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:82
		{
			{
				global := NewAst([]Dec{}, mmDollar[1].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 4:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:90
		{
			{
				mmVAL.decs = append(mmDollar[1].decs, mmDollar[2].dec)
			}
		}
	case 5:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:92
		{
			{
				mmVAL.decs = []Dec{mmDollar[1].dec}
			}
		}
	case 6:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:97
		{
			{
				mmVAL.dec = &Filetype{NewAstNode(&mmlval), mmDollar[2].val}
			}
		}
	case 7:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/core/grammar.y:99
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmDollar[2].val, mmDollar[4].params, mmDollar[5].params, mmDollar[6].src, &Params{[]Param{}, map[string]Param{}}, false}
			}
		}
	case 8:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line src/martian/core/grammar.y:101
		{
			{
				mmVAL.dec = &Stage{NewAstNode(&mmlval), mmDollar[2].val, mmDollar[4].params, mmDollar[5].params, mmDollar[6].src, mmDollar[8].params, true}
			}
		}
	case 9:
		mmDollar = mmS[mmpt-10 : mmpt+1]
		//line src/martian/core/grammar.y:103
		{
			{
				mmVAL.dec = &Pipeline{NewAstNode(&mmlval), mmDollar[2].val, mmDollar[4].params, mmDollar[5].params, mmDollar[8].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmDollar[9].retstm}
			}
		}
	case 10:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:108
		{
			{
				mmVAL.val = mmDollar[1].val + mmDollar[2].val + mmDollar[3].val
			}
		}
	case 12:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/core/grammar.y:114
		{
			{
				mmVAL.arr = 0
			}
		}
	case 13:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:116
		{
			{
				mmVAL.arr += 1
			}
		}
	case 14:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/core/grammar.y:121
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 15:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:123
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].inparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 16:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/core/grammar.y:131
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(&mmlval), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), false}
			}
		}
	case 17:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/core/grammar.y:136
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 18:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:138
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].outparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 19:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/core/grammar.y:146
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmDollar[2].val, mmDollar[3].arr, "default", unquote(mmDollar[4].val), false}
			}
		}
	case 20:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/core/grammar.y:148
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(&mmlval), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), false}
			}
		}
	case 21:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/core/grammar.y:153
		{
			{
				stagecodeParts := strings.Split(unquote(mmDollar[3].val), " ")
				mmVAL.src = &SrcParam{NewAstNode(&mmlval), mmDollar[2].val, stagecodeParts[0], stagecodeParts[1:]}
			}
		}
	case 22:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:159
		{
			{
				mmVAL.val = mmDollar[1].val
			}
		}
	case 23:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:161
		{
			{
				mmVAL.val = ""
			}
		}
	case 33:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/core/grammar.y:183
		{
			{
				mmVAL.params = mmDollar[4].params
			}
		}
	case 34:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/core/grammar.y:188
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(&mmlval), mmDollar[3].bindings}
			}
		}
	case 35:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:193
		{
			{
				mmVAL.calls = append(mmDollar[1].calls, mmDollar[2].call)
			}
		}
	case 36:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:195
		{
			{
				mmVAL.calls = []*CallStm{mmDollar[1].call}
			}
		}
	case 37:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/core/grammar.y:200
		{
			{
				mmVAL.call = &CallStm{NewAstNode(&mmlval), mmDollar[2].modifiers, mmDollar[3].val, mmDollar[5].bindings}
			}
		}
	case 38:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/core/grammar.y:205
		{
			{
				mmVAL.modifiers = &Modifiers{false, false, false}
			}
		}
	case 39:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:207
		{
			{
				mmVAL.modifiers.Local = true
			}
		}
	case 40:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:209
		{
			{
				mmVAL.modifiers.Preflight = true
			}
		}
	case 41:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:211
		{
			{
				mmVAL.modifiers.Volatile = true
			}
		}
	case 42:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/core/grammar.y:216
		{
			{
				mmVAL.bindings = &BindStms{NewAstNode(&mmlval), []*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 43:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:218
		{
			{
				mmDollar[1].bindings.List = append(mmDollar[1].bindings.List, mmDollar[2].binding)
				mmVAL.bindings = mmDollar[1].bindings
			}
		}
	case 44:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/core/grammar.y:226
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 45:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/core/grammar.y:228
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(&mmlval), mmDollar[1].val, &ValExp{Node: NewAstNode(&mmlval), Kind: "array", Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 46:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:233
		{
			{
				mmVAL.exps = append(mmDollar[1].exps, mmDollar[3].exp)
			}
		}
	case 47:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:235
		{
			{
				mmVAL.exps = []Exp{mmDollar[1].exp}
			}
		}
	case 48:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/core/grammar.y:240
		{
			{
				mmDollar[1].kvpairs[unquote(mmDollar[3].val)] = mmDollar[5].exp
				mmVAL.kvpairs = mmDollar[1].kvpairs
			}
		}
	case 49:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:245
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmDollar[1].val): mmDollar[3].exp}
			}
		}
	case 50:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:250
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(&mmlval), Kind: "array", Value: mmDollar[2].exps}
			}
		}
	case 51:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:252
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(&mmlval), Kind: "array", Value: []Exp{}}
			}
		}
	case 52:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:254
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(&mmlval), Kind: "map", Value: map[string]interface{}{}}
			}
		}
	case 53:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:256
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(&mmlval), Kind: "map", Value: mmDollar[2].kvpairs}
			}
		}
	case 54:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:258
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmDollar[1].val, 64)
				mmVAL.exp = &ValExp{Node: NewAstNode(&mmlval), Kind: "float", Value: f}
			}
		}
	case 55:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:263
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmDollar[1].val, 0, 64)
				mmVAL.exp = &ValExp{Node: NewAstNode(&mmlval), Kind: "int", Value: i}
			}
		}
	case 56:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:268
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(&mmlval), Kind: "string", Value: unquote(mmDollar[1].val)}
			}
		}
	case 57:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:270
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(&mmlval), Kind: "bool", Value: true}
			}
		}
	case 58:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:272
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(&mmlval), Kind: "bool", Value: false}
			}
		}
	case 59:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:274
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(&mmlval), Kind: "null", Value: nil}
			}
		}
	case 60:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:276
		{
			{
				mmVAL.exp = mmDollar[1].exp
			}
		}
	case 61:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:281
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmDollar[1].val, mmDollar[3].val}
			}
		}
	case 62:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:283
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "call", mmDollar[1].val, "default"}
			}
		}
	case 63:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:285
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(&mmlval), "self", mmDollar[3].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

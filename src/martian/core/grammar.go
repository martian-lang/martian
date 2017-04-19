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
	locmap    []FileLoc
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
	nodeGen   func() AstNode
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
const mmInitialStackSize = 16

//line src/martian/core/grammar.y:302

//line yacctab:1
var mmExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const mmPrivate = 57344

const mmLast = 163

var mmAct = [...]int{

	82, 28, 110, 26, 91, 80, 3, 59, 81, 9,
	55, 60, 52, 40, 87, 53, 86, 90, 20, 74,
	21, 119, 83, 69, 17, 18, 19, 27, 68, 63,
	61, 62, 72, 59, 123, 16, 112, 60, 109, 85,
	92, 93, 96, 64, 65, 66, 58, 21, 107, 69,
	57, 106, 74, 54, 68, 63, 61, 62, 111, 130,
	93, 33, 111, 77, 25, 73, 93, 39, 38, 64,
	65, 66, 59, 75, 15, 14, 60, 35, 13, 35,
	95, 31, 48, 98, 128, 100, 29, 71, 69, 88,
	115, 5, 5, 68, 63, 61, 62, 113, 39, 31,
	99, 56, 118, 50, 116, 108, 120, 79, 64, 65,
	66, 13, 121, 24, 32, 23, 124, 22, 125, 103,
	46, 41, 42, 44, 43, 45, 104, 94, 131, 6,
	7, 8, 5, 101, 101, 127, 49, 102, 117, 129,
	126, 122, 114, 89, 78, 105, 47, 4, 1, 97,
	10, 34, 76, 12, 84, 67, 36, 70, 37, 30,
	2, 11, 51,
}
var mmPact = [...]int{

	113, -1000, 113, -1000, -1000, -1000, 47, 44, 43, -1000,
	-1000, 4, 12, -1000, 105, 103, 101, -1000, -1000, -1000,
	-1000, 33, -1000, -1000, -1000, -1000, 53, 53, 48, 38,
	-1000, 80, 69, -1000, -1000, 127, 90, -1000, -24, 80,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -15, 87, 23,
	63, 0, -1000, -1000, -1000, 42, 72, 136, 95, -3,
	7, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -19, -21,
	-1000, 64, 135, 9, 116, 34, 73, -1000, -1000, 62,
	126, -1000, -1000, -1000, 111, 138, 20, 17, 93, -1000,
	-1000, 30, 28, -1000, -1000, 134, -1000, 75, -1000, 92,
	125, 62, -1000, -11, -1000, 62, -1000, -1000, -1000, -1000,
	133, -1000, -1000, 26, -1000, -1000, -1000, 132, -1000, 128,
	-1000, 71, -1000, -1000, 131, 46, -1000, 62, -1000, -1000,
	-1000, -1000,
}
var mmPgo = [...]int{

	0, 146, 13, 4, 162, 2, 161, 10, 147, 160,
	159, 158, 3, 86, 157, 156, 0, 155, 5, 154,
	6, 152, 151, 1, 149, 148,
}
var mmR1 = [...]int{

	0, 25, 25, 25, 9, 9, 8, 8, 8, 8,
	1, 1, 7, 7, 12, 12, 10, 10, 13, 13,
	11, 11, 11, 11, 11, 11, 15, 3, 5, 2,
	2, 2, 2, 2, 2, 2, 4, 4, 14, 24,
	21, 21, 20, 6, 6, 6, 6, 23, 23, 22,
	22, 18, 18, 19, 19, 16, 16, 16, 16, 16,
	16, 16, 16, 16, 16, 16, 17, 17, 17,
}
var mmR2 = [...]int{

	0, 1, 2, 1, 2, 1, 3, 7, 8, 10,
	3, 1, 0, 3, 0, 2, 6, 5, 0, 2,
	4, 5, 6, 5, 6, 7, 4, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 5, 4,
	2, 1, 6, 0, 2, 2, 2, 0, 2, 4,
	7, 3, 1, 5, 3, 3, 2, 2, 3, 1,
	1, 1, 1, 1, 1, 1, 3, 1, 3,
}
var mmChk = [...]int{

	-1000, -25, -9, -20, -8, 19, 16, 17, 18, -20,
	-8, -6, -1, 31, 31, 31, 31, 20, 21, 22,
	6, 35, 12, 12, 12, 31, -12, -12, -23, -13,
	-10, 28, -13, 13, -22, 31, -15, -11, 30, 29,
	-2, 41, 42, 44, 43, 45, 40, -1, 13, 9,
	13, -4, 36, 39, -2, -7, 14, -16, 23, 10,
	14, 33, 34, 32, 46, 47, 48, -17, 31, 26,
	-14, 24, 32, -7, 10, 31, -21, -20, 8, 12,
	-18, 11, -16, 15, -19, 32, 35, 35, 25, 8,
	8, -3, 31, 32, 11, -3, 8, -24, -20, 27,
	-18, 8, 11, 8, 15, 7, 31, 31, 12, 8,
	-5, 32, 8, -3, 8, 15, 12, 13, -16, 32,
	-16, -12, 8, 8, -5, -23, 8, 7, 13, 8,
	13, -16,
}
var mmDef = [...]int{

	0, -2, 1, 3, 5, 43, 0, 0, 0, 2,
	4, 0, 0, 11, 0, 0, 0, 44, 45, 46,
	6, 0, 14, 14, 47, 10, 18, 18, 0, 0,
	15, 0, 0, 42, 48, 0, 0, 19, 0, 0,
	12, 29, 30, 31, 32, 33, 34, 35, 0, 0,
	7, 0, 36, 37, 12, 0, 0, 0, 0, 0,
	0, 59, 60, 61, 62, 63, 64, 65, 67, 0,
	8, 0, 0, 0, 0, 0, 0, 41, 49, 0,
	0, 56, 52, 57, 0, 0, 0, 0, 0, 26,
	20, 0, 0, 27, 13, 0, 17, 0, 40, 0,
	0, 0, 55, 0, 58, 0, 66, 68, 14, 21,
	0, 28, 23, 0, 16, 9, 47, 0, 51, 0,
	54, 0, 22, 24, 0, 0, 50, 0, 38, 25,
	39, 53,
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
	lval  mmSymType
	stack [mmInitialStackSize]mmSymType
	char  int
}

func (p *mmParserImpl) Lookahead() int {
	return p.char
}

func mmNewParser() mmParser {
	return &mmParserImpl{}
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
	var mmVAL mmSymType
	var mmDollar []mmSymType
	_ = mmDollar // silence set and not used
	mmS := mmrcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	mmstate := 0
	mmrcvr.char = -1
	mmtoken := -1 // mmrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		mmstate = -1
		mmrcvr.char = -1
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
	if mmrcvr.char < 0 {
		mmrcvr.char, mmtoken = mmlex1(mmlex, &mmrcvr.lval)
	}
	mmn += mmtoken
	if mmn < 0 || mmn >= mmLast {
		goto mmdefault
	}
	mmn = mmAct[mmn]
	if mmChk[mmn] == mmtoken { /* valid shift */
		mmrcvr.char = -1
		mmtoken = -1
		mmVAL = mmrcvr.lval
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
		if mmrcvr.char < 0 {
			mmrcvr.char, mmtoken = mmlex1(mmlex, &mmrcvr.lval)
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
			mmrcvr.char = -1
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
		//line src/martian/core/grammar.y:74
		{
			{
				global := NewAst(mmDollar[1].decs, nil)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 2:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:79
		{
			{
				global := NewAst(mmDollar[1].decs, mmDollar[2].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 3:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:84
		{
			{
				global := NewAst([]Dec{}, mmDollar[1].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 4:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:92
		{
			{
				mmVAL.decs = append(mmDollar[1].decs, mmDollar[2].dec)
			}
		}
	case 5:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:94
		{
			{
				mmVAL.decs = []Dec{mmDollar[1].dec}
			}
		}
	case 6:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:99
		{
			{
				mmVAL.dec = &UserType{mmDollar[1].nodeGen(), mmDollar[2].val}
			}
		}
	case 7:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/core/grammar.y:101
		{
			{
				mmVAL.dec = &Stage{mmDollar[1].nodeGen(), mmDollar[2].val, mmDollar[4].params, mmDollar[5].params, mmDollar[6].src, &Params{[]Param{}, map[string]Param{}}, false}
			}
		}
	case 8:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line src/martian/core/grammar.y:103
		{
			{
				mmVAL.dec = &Stage{mmDollar[1].nodeGen(), mmDollar[2].val, mmDollar[4].params, mmDollar[5].params, mmDollar[6].src, mmDollar[8].params, true}
			}
		}
	case 9:
		mmDollar = mmS[mmpt-10 : mmpt+1]
		//line src/martian/core/grammar.y:105
		{
			{
				mmVAL.dec = &Pipeline{mmDollar[1].nodeGen(), mmDollar[2].val, mmDollar[4].params, mmDollar[5].params, mmDollar[8].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmDollar[9].retstm}
			}
		}
	case 10:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:110
		{
			{
				mmVAL.val = mmDollar[1].val + mmDollar[2].val + mmDollar[3].val
			}
		}
	case 12:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/core/grammar.y:116
		{
			{
				mmVAL.arr = 0
			}
		}
	case 13:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:118
		{
			{
				mmVAL.arr += 1
			}
		}
	case 14:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/core/grammar.y:123
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 15:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:125
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].inparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 16:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/core/grammar.y:133
		{
			{
				mmVAL.inparam = &InParam{mmDollar[1].nodeGen(), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), false}
			}
		}
	case 17:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/core/grammar.y:135
		{
			{
				mmVAL.inparam = &InParam{mmDollar[1].nodeGen(), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, "", false}
			}
		}
	case 18:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/core/grammar.y:140
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 19:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:142
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].outparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 20:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/core/grammar.y:150
		{
			{
				mmVAL.outparam = &OutParam{mmDollar[1].nodeGen(), mmDollar[2].val, mmDollar[3].arr, "default", "", "", false}
			}
		}
	case 21:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/core/grammar.y:152
		{
			{
				mmVAL.outparam = &OutParam{mmDollar[1].nodeGen(), mmDollar[2].val, mmDollar[3].arr, "default", unquote(mmDollar[4].val), "", false}
			}
		}
	case 22:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/core/grammar.y:154
		{
			{
				mmVAL.outparam = &OutParam{mmDollar[1].nodeGen(), mmDollar[2].val, mmDollar[3].arr, "default", unquote(mmDollar[4].val), unquote(mmDollar[5].val), false}
			}
		}
	case 23:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/core/grammar.y:156
		{
			{
				mmVAL.outparam = &OutParam{mmDollar[1].nodeGen(), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, "", "", false}
			}
		}
	case 24:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/core/grammar.y:158
		{
			{
				mmVAL.outparam = &OutParam{mmDollar[1].nodeGen(), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), "", false}
			}
		}
	case 25:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/core/grammar.y:160
		{
			{
				mmVAL.outparam = &OutParam{mmDollar[1].nodeGen(), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), unquote(mmDollar[6].val), false}
			}
		}
	case 26:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/core/grammar.y:165
		{
			{
				stagecodeParts := strings.Split(unquote(mmDollar[3].val), " ")
				mmVAL.src = &SrcParam{mmDollar[1].nodeGen(), mmDollar[2].val, stagecodeParts[0], stagecodeParts[1:]}
			}
		}
	case 27:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:171
		{
			{
				mmVAL.val = mmDollar[1].val
			}
		}
	case 28:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:176
		{
			{
				mmVAL.val = mmDollar[1].val
			}
		}
	case 38:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/core/grammar.y:198
		{
			{
				mmVAL.params = mmDollar[4].params
			}
		}
	case 39:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/core/grammar.y:203
		{
			{
				mmVAL.retstm = &ReturnStm{mmDollar[1].nodeGen(), mmDollar[3].bindings}
			}
		}
	case 40:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:208
		{
			{
				mmVAL.calls = append(mmDollar[1].calls, mmDollar[2].call)
			}
		}
	case 41:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:210
		{
			{
				mmVAL.calls = []*CallStm{mmDollar[1].call}
			}
		}
	case 42:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/core/grammar.y:215
		{
			{
				mmVAL.call = &CallStm{mmDollar[1].nodeGen(), mmDollar[2].modifiers, mmDollar[3].val, mmDollar[5].bindings}
			}
		}
	case 43:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/core/grammar.y:220
		{
			{
				mmVAL.modifiers = &Modifiers{false, false, false}
			}
		}
	case 44:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:222
		{
			{
				mmVAL.modifiers.Local = true
			}
		}
	case 45:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:224
		{
			{
				mmVAL.modifiers.Preflight = true
			}
		}
	case 46:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:226
		{
			{
				mmVAL.modifiers.Volatile = true
			}
		}
	case 47:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/core/grammar.y:231
		{
			{
				mmVAL.bindings = &BindStms{mmDollar[0].nodeGen(), []*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 48:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:233
		{
			{
				mmDollar[1].bindings.List = append(mmDollar[1].bindings.List, mmDollar[2].binding)
				mmVAL.bindings = mmDollar[1].bindings
			}
		}
	case 49:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/core/grammar.y:241
		{
			{
				mmVAL.binding = &BindStm{mmDollar[1].nodeGen(), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 50:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/core/grammar.y:243
		{
			{
				mmVAL.binding = &BindStm{mmDollar[1].nodeGen(), mmDollar[1].val, &ValExp{Node: mmDollar[1].nodeGen(), Kind: "array", Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 51:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:248
		{
			{
				mmVAL.exps = append(mmDollar[1].exps, mmDollar[3].exp)
			}
		}
	case 52:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:250
		{
			{
				mmVAL.exps = []Exp{mmDollar[1].exp}
			}
		}
	case 53:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/core/grammar.y:255
		{
			{
				mmDollar[1].kvpairs[unquote(mmDollar[3].val)] = mmDollar[5].exp
				mmVAL.kvpairs = mmDollar[1].kvpairs
			}
		}
	case 54:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:260
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmDollar[1].val): mmDollar[3].exp}
			}
		}
	case 55:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:265
		{
			{
				mmVAL.exp = &ValExp{Node: mmDollar[1].nodeGen(), Kind: "array", Value: mmDollar[2].exps}
			}
		}
	case 56:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:267
		{
			{
				mmVAL.exp = &ValExp{Node: mmDollar[1].nodeGen(), Kind: "array", Value: []Exp{}}
			}
		}
	case 57:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/core/grammar.y:269
		{
			{
				mmVAL.exp = &ValExp{Node: mmDollar[1].nodeGen(), Kind: "map", Value: map[string]interface{}{}}
			}
		}
	case 58:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:271
		{
			{
				mmVAL.exp = &ValExp{Node: mmDollar[1].nodeGen(), Kind: "map", Value: mmDollar[2].kvpairs}
			}
		}
	case 59:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:273
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmDollar[1].val, 64)
				mmVAL.exp = &ValExp{Node: mmDollar[1].nodeGen(), Kind: "float", Value: f}
			}
		}
	case 60:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:278
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmDollar[1].val, 0, 64)
				mmVAL.exp = &ValExp{Node: mmDollar[1].nodeGen(), Kind: "int", Value: i}
			}
		}
	case 61:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:283
		{
			{
				mmVAL.exp = &ValExp{Node: mmDollar[1].nodeGen(), Kind: "string", Value: unquote(mmDollar[1].val)}
			}
		}
	case 62:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:285
		{
			{
				mmVAL.exp = &ValExp{Node: mmDollar[1].nodeGen(), Kind: "bool", Value: true}
			}
		}
	case 63:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:287
		{
			{
				mmVAL.exp = &ValExp{Node: mmDollar[1].nodeGen(), Kind: "bool", Value: false}
			}
		}
	case 64:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:289
		{
			{
				mmVAL.exp = &ValExp{Node: mmDollar[1].nodeGen(), Kind: "null", Value: nil}
			}
		}
	case 65:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:291
		{
			{
				mmVAL.exp = mmDollar[1].exp
			}
		}
	case 66:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:296
		{
			{
				mmVAL.exp = &RefExp{mmDollar[1].nodeGen(), "call", mmDollar[1].val, mmDollar[3].val}
			}
		}
	case 67:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/core/grammar.y:298
		{
			{
				mmVAL.exp = &RefExp{mmDollar[1].nodeGen(), "call", mmDollar[1].val, "default"}
			}
		}
	case 68:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/core/grammar.y:300
		{
			{
				mmVAL.exp = &RefExp{mmDollar[1].nodeGen(), "self", mmDollar[3].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

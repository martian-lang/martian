//line src/martian/syntax/grammar.y:2

//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO grammar.
//

package syntax

import __yyfmt__ "fmt"

//line src/martian/syntax/grammar.y:8
import (
	"strconv"
	"strings"
)

func unquote(qs string) string {
	return strings.Replace(qs, "\"", "", -1)
}

//line src/martian/syntax/grammar.y:20
type mmSymType struct {
	yys       int
	global    *Ast
	locmap    []FileLoc
	arr       int
	loc       int
	val       string
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
	pre_dir   []*preprocessorDirective
}

const SKIP = 57346
const COMMENT = 57347
const INVALID = 57348
const SEMICOLON = 57349
const COLON = 57350
const COMMA = 57351
const EQUALS = 57352
const LBRACKET = 57353
const RBRACKET = 57354
const LPAREN = 57355
const RPAREN = 57356
const LBRACE = 57357
const RBRACE = 57358
const FILETYPE = 57359
const STAGE = 57360
const PIPELINE = 57361
const CALL = 57362
const LOCAL = 57363
const PREFLIGHT = 57364
const VOLATILE = 57365
const SWEEP = 57366
const SPLIT = 57367
const USING = 57368
const SELF = 57369
const RETURN = 57370
const IN = 57371
const OUT = 57372
const SRC = 57373
const ID = 57374
const LITSTRING = 57375
const NUM_FLOAT = 57376
const NUM_INT = 57377
const DOT = 57378
const PY = 57379
const GO = 57380
const SH = 57381
const EXEC = 57382
const COMPILED = 57383
const MAP = 57384
const INT = 57385
const STRING = 57386
const FLOAT = 57387
const PATH = 57388
const BOOL = 57389
const TRUE = 57390
const FALSE = 57391
const NULL = 57392
const DEFAULT = 57393
const PREPROCESS_DIRECTIVE = 57394

var mmToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"SKIP",
	"COMMENT",
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
	"COMPILED",
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
	"PREPROCESS_DIRECTIVE",
}
var mmStatenames = [...]string{}

const mmEofCode = 1
const mmErrCode = 2
const mmInitialStackSize = 16

//line src/martian/syntax/grammar.y:335

//line yacctab:1
var mmExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const mmPrivate = 57344

const mmLast = 179

var mmAct = [...]int{

	89, 34, 117, 32, 98, 4, 87, 94, 12, 14,
	62, 26, 93, 46, 66, 88, 58, 21, 67, 59,
	60, 27, 126, 8, 9, 10, 7, 90, 79, 114,
	76, 130, 137, 33, 113, 75, 70, 68, 69, 81,
	27, 66, 119, 116, 92, 67, 8, 9, 10, 7,
	41, 71, 72, 73, 65, 118, 64, 76, 13, 61,
	82, 37, 75, 70, 68, 69, 100, 118, 97, 84,
	81, 103, 80, 23, 24, 25, 31, 35, 71, 72,
	73, 5, 39, 20, 22, 45, 44, 102, 19, 105,
	18, 99, 100, 107, 66, 100, 54, 7, 67, 135,
	41, 95, 63, 78, 120, 106, 7, 122, 123, 125,
	76, 38, 45, 127, 37, 75, 70, 68, 69, 128,
	56, 115, 86, 131, 30, 132, 18, 8, 9, 10,
	7, 71, 72, 73, 29, 138, 52, 47, 48, 50,
	49, 51, 110, 108, 108, 28, 6, 109, 124, 111,
	15, 101, 55, 136, 133, 129, 121, 96, 15, 85,
	53, 134, 112, 3, 1, 104, 11, 40, 83, 17,
	91, 74, 42, 77, 43, 36, 16, 57, 2,
}
var mmPact = [...]int{

	29, -1000, 6, 110, -1000, -1000, -1000, -1000, 58, 56,
	51, 110, -1000, -1000, -1000, -1000, 52, 4, -1000, 132,
	121, -1000, 111, -1000, -1000, -1000, -1000, 44, -1000, -1000,
	-1000, -1000, 32, 32, 68, 55, -1000, 94, 82, -1000,
	-1000, 142, 106, -1000, -21, 94, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -15, 87, 30, 78, -5, -1000, -1000,
	-1000, -1000, 28, 86, 150, 109, 3, 11, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -24, -29, -1000, 75, 148,
	59, 139, 62, 77, -1000, -1000, 83, 135, -1000, -1000,
	-1000, 133, 154, 2, -3, 108, -1000, -1000, 34, 33,
	-1000, -1000, 147, -1000, 91, -1000, 95, 134, 83, -1000,
	-11, -1000, 83, -1000, -1000, -1000, -1000, 146, -1000, -1000,
	22, -1000, -1000, -1000, 145, -1000, 153, -1000, 85, -1000,
	-1000, 144, 18, -1000, 83, -1000, -1000, -1000, -1000,
}
var mmPgo = [...]int{

	0, 178, 160, 13, 4, 177, 2, 176, 10, 146,
	163, 175, 174, 3, 77, 173, 172, 0, 171, 6,
	170, 5, 168, 167, 1, 165, 164,
}
var mmR1 = [...]int{

	0, 26, 26, 26, 26, 26, 26, 1, 1, 10,
	10, 9, 9, 9, 9, 2, 2, 8, 8, 13,
	13, 11, 11, 14, 14, 12, 12, 12, 12, 12,
	12, 16, 4, 6, 3, 3, 3, 3, 3, 3,
	3, 5, 5, 5, 15, 25, 22, 22, 21, 7,
	7, 7, 7, 24, 24, 23, 23, 19, 19, 20,
	20, 17, 17, 17, 17, 17, 17, 17, 17, 17,
	17, 17, 18, 18, 18,
}
var mmR2 = [...]int{

	0, 2, 3, 2, 1, 2, 1, 2, 1, 2,
	1, 3, 7, 8, 10, 3, 1, 0, 3, 0,
	2, 6, 5, 0, 2, 4, 5, 6, 5, 6,
	7, 4, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 5, 4, 2, 1, 6, 0,
	2, 2, 2, 0, 2, 4, 7, 3, 1, 5,
	3, 3, 2, 2, 3, 1, 1, 1, 1, 1,
	1, 1, 3, 1, 3,
}
var mmChk = [...]int{

	-1000, -26, -1, -10, -21, 52, -9, 20, 17, 18,
	19, -10, -21, 52, -21, -9, -7, -2, 32, 32,
	32, -21, 32, 21, 22, 23, 7, 36, 13, 13,
	13, 32, -13, -13, -24, -14, -11, 29, -14, 14,
	-23, 32, -16, -12, 31, 30, -3, 43, 44, 46,
	45, 47, 42, -2, 14, 10, 14, -5, 37, 40,
	41, -3, -8, 15, -17, 24, 11, 15, 34, 35,
	33, 48, 49, 50, -18, 32, 27, -15, 25, 33,
	-8, 11, 32, -22, -21, 9, 13, -19, 12, -17,
	16, -20, 33, 36, 36, 26, 9, 9, -4, 32,
	33, 12, -4, 9, -25, -21, 28, -19, 9, 12,
	9, 16, 8, 32, 32, 13, 9, -6, 33, 9,
	-4, 9, 16, 13, 14, -17, 33, -17, -13, 9,
	9, -6, -24, 9, 8, 14, 9, 14, -17,
}
var mmDef = [...]int{

	0, -2, 0, 4, 6, 8, 10, 49, 0, 0,
	0, 1, 3, 7, 5, 9, 0, 0, 16, 0,
	0, 2, 0, 50, 51, 52, 11, 0, 19, 19,
	53, 15, 23, 23, 0, 0, 20, 0, 0, 48,
	54, 0, 0, 24, 0, 0, 17, 34, 35, 36,
	37, 38, 39, 40, 0, 0, 12, 0, 41, 42,
	43, 17, 0, 0, 0, 0, 0, 0, 65, 66,
	67, 68, 69, 70, 71, 73, 0, 13, 0, 0,
	0, 0, 0, 0, 47, 55, 0, 0, 62, 58,
	63, 0, 0, 0, 0, 0, 31, 25, 0, 0,
	32, 18, 0, 22, 0, 46, 0, 0, 0, 61,
	0, 64, 0, 72, 74, 19, 26, 0, 33, 28,
	0, 21, 14, 53, 0, 57, 0, 60, 0, 27,
	29, 0, 0, 56, 0, 44, 30, 45, 59,
}
var mmTok1 = [...]int{

	1,
}
var mmTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
	32, 33, 34, 35, 36, 37, 38, 39, 40, 41,
	42, 43, 44, 45, 46, 47, 48, 49, 50, 51,
	52,
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
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:76
		{
			{
				global := NewAst(mmDollar[2].decs, nil)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 2:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:82
		{
			{
				global := NewAst(mmDollar[2].decs, mmDollar[3].call)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 3:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:88
		{
			{
				global := NewAst([]Dec{}, mmDollar[2].call)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 4:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:94
		{
			{
				global := NewAst(mmDollar[1].decs, nil)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 5:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:99
		{
			{
				global := NewAst(mmDollar[1].decs, mmDollar[2].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 6:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:104
		{
			{
				global := NewAst([]Dec{}, mmDollar[1].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 7:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:112
		{
			{
				mmVAL.pre_dir = append(mmDollar[1].pre_dir, &preprocessorDirective{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val})
			}
		}
	case 8:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:114
		{
			{
				mmVAL.pre_dir = []*preprocessorDirective{
					&preprocessorDirective{
						Node:  NewAstNode(mmDollar[1].loc, mmDollar[1].locmap),
						Value: mmDollar[1].val,
					},
				}
			}
		}
	case 9:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:124
		{
			{
				mmVAL.decs = append(mmDollar[1].decs, mmDollar[2].dec)
			}
		}
	case 10:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:126
		{
			{
				mmVAL.decs = []Dec{mmDollar[1].dec}
			}
		}
	case 11:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:131
		{
			{
				mmVAL.dec = &UserType{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val}
			}
		}
	case 12:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/syntax/grammar.y:133
		{
			{
				mmVAL.dec = &Stage{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val, mmDollar[4].params, mmDollar[5].params, mmDollar[6].src, &Params{[]Param{}, map[string]Param{}}, false}
			}
		}
	case 13:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line src/martian/syntax/grammar.y:135
		{
			{
				mmVAL.dec = &Stage{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val, mmDollar[4].params, mmDollar[5].params, mmDollar[6].src, mmDollar[8].params, true}
			}
		}
	case 14:
		mmDollar = mmS[mmpt-10 : mmpt+1]
		//line src/martian/syntax/grammar.y:137
		{
			{
				mmVAL.dec = &Pipeline{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val, mmDollar[4].params, mmDollar[5].params, mmDollar[8].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmDollar[9].retstm}
			}
		}
	case 15:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:142
		{
			{
				mmVAL.val = mmDollar[1].val + mmDollar[2].val + mmDollar[3].val
			}
		}
	case 17:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:148
		{
			{
				mmVAL.arr = 0
			}
		}
	case 18:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:150
		{
			{
				mmVAL.arr += 1
			}
		}
	case 19:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:155
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 20:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:157
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].inparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 21:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:165
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), false}
			}
		}
	case 22:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:167
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, "", false}
			}
		}
	case 23:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:172
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 24:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:174
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].outparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 25:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:182
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, "default", "", "", false}
			}
		}
	case 26:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:184
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, "default", unquote(mmDollar[4].val), "", false}
			}
		}
	case 27:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:186
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, "default", unquote(mmDollar[4].val), unquote(mmDollar[5].val), false}
			}
		}
	case 28:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:188
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, "", "", false}
			}
		}
	case 29:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:190
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), "", false}
			}
		}
	case 30:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/syntax/grammar.y:192
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), unquote(mmDollar[6].val), false}
			}
		}
	case 31:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:197
		{
			{
				stagecodeParts := strings.Split(unquote(mmDollar[3].val), " ")
				mmVAL.src = &SrcParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), StageLanguage(mmDollar[2].val), stagecodeParts[0], stagecodeParts[1:]}
			}
		}
	case 32:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:203
		{
			{
				mmVAL.val = mmDollar[1].val
			}
		}
	case 33:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:208
		{
			{
				mmVAL.val = mmDollar[1].val
			}
		}
	case 44:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:231
		{
			{
				mmVAL.params = mmDollar[4].params
			}
		}
	case 45:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:236
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[3].bindings}
			}
		}
	case 46:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:241
		{
			{
				mmVAL.calls = append(mmDollar[1].calls, mmDollar[2].call)
			}
		}
	case 47:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:243
		{
			{
				mmVAL.calls = []*CallStm{mmDollar[1].call}
			}
		}
	case 48:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:248
		{
			{
				mmVAL.call = &CallStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].modifiers, mmDollar[3].val, mmDollar[5].bindings}
			}
		}
	case 49:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:253
		{
			{
				mmVAL.modifiers = &Modifiers{false, false, false}
			}
		}
	case 50:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:255
		{
			{
				mmVAL.modifiers.Local = true
			}
		}
	case 51:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:257
		{
			{
				mmVAL.modifiers.Preflight = true
			}
		}
	case 52:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:259
		{
			{
				mmVAL.modifiers.Volatile = true
			}
		}
	case 53:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:264
		{
			{
				mmVAL.bindings = &BindStms{NewAstNode(mmDollar[0].loc, mmDollar[0].locmap), []*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 54:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:266
		{
			{
				mmDollar[1].bindings.List = append(mmDollar[1].bindings.List, mmDollar[2].binding)
				mmVAL.bindings = mmDollar[1].bindings
			}
		}
	case 55:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:274
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 56:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/syntax/grammar.y:276
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: "array", Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 57:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:281
		{
			{
				mmVAL.exps = append(mmDollar[1].exps, mmDollar[3].exp)
			}
		}
	case 58:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:283
		{
			{
				mmVAL.exps = []Exp{mmDollar[1].exp}
			}
		}
	case 59:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:288
		{
			{
				mmDollar[1].kvpairs[unquote(mmDollar[3].val)] = mmDollar[5].exp
				mmVAL.kvpairs = mmDollar[1].kvpairs
			}
		}
	case 60:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:293
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmDollar[1].val): mmDollar[3].exp}
			}
		}
	case 61:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:298
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: "array", Value: mmDollar[2].exps}
			}
		}
	case 62:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:300
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: "array", Value: []Exp{}}
			}
		}
	case 63:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:302
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: "map", Value: map[string]interface{}{}}
			}
		}
	case 64:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:304
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: "map", Value: mmDollar[2].kvpairs}
			}
		}
	case 65:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:306
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmDollar[1].val, 64)
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: "float", Value: f}
			}
		}
	case 66:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:311
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmDollar[1].val, 0, 64)
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: "int", Value: i}
			}
		}
	case 67:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:316
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: "string", Value: unquote(mmDollar[1].val)}
			}
		}
	case 68:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:318
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: "bool", Value: true}
			}
		}
	case 69:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:320
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: "bool", Value: false}
			}
		}
	case 70:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:322
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: "null", Value: nil}
			}
		}
	case 71:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:324
		{
			{
				mmVAL.exp = mmDollar[1].exp
			}
		}
	case 72:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:329
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), "call", mmDollar[1].val, mmDollar[3].val}
			}
		}
	case 73:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:331
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), "call", mmDollar[1].val, "default"}
			}
		}
	case 74:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:333
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), "self", mmDollar[3].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

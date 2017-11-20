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
	res       *Resources
	par_tuple paramsTuple
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
const AS = 57374
const THREADS = 57375
const MEM_GB = 57376
const SPECIAL = 57377
const ID = 57378
const LITSTRING = 57379
const NUM_FLOAT = 57380
const NUM_INT = 57381
const DOT = 57382
const PY = 57383
const GO = 57384
const SH = 57385
const EXEC = 57386
const COMPILED = 57387
const MAP = 57388
const INT = 57389
const STRING = 57390
const FLOAT = 57391
const PATH = 57392
const BOOL = 57393
const TRUE = 57394
const FALSE = 57395
const NULL = 57396
const DEFAULT = 57397
const PREPROCESS_DIRECTIVE = 57398

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
	"AS",
	"THREADS",
	"MEM_GB",
	"SPECIAL",
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

//line src/martian/syntax/grammar.y:413

//line yacctab:1
var mmExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const mmPrivate = 57344

const mmLast = 235

var mmAct = [...]int{

	96, 34, 38, 128, 36, 106, 94, 4, 63, 49,
	13, 15, 66, 101, 73, 67, 68, 150, 74, 27,
	22, 73, 139, 100, 28, 74, 8, 11, 10, 7,
	82, 168, 35, 8, 11, 10, 7, 82, 41, 81,
	77, 75, 76, 167, 169, 145, 81, 77, 75, 76,
	60, 91, 28, 153, 124, 78, 79, 80, 62, 123,
	69, 37, 78, 79, 80, 14, 24, 25, 26, 85,
	141, 86, 5, 129, 19, 44, 87, 155, 33, 73,
	95, 23, 130, 74, 55, 50, 51, 53, 52, 54,
	40, 140, 103, 97, 110, 82, 156, 157, 158, 73,
	117, 88, 127, 74, 81, 77, 75, 76, 21, 113,
	108, 111, 70, 131, 99, 82, 105, 135, 87, 138,
	78, 79, 80, 142, 81, 77, 75, 76, 83, 73,
	129, 143, 20, 74, 19, 146, 148, 138, 149, 108,
	78, 79, 80, 107, 108, 82, 42, 31, 90, 166,
	44, 159, 125, 162, 81, 77, 75, 76, 48, 58,
	160, 46, 115, 7, 136, 48, 32, 120, 44, 137,
	78, 79, 80, 7, 121, 114, 48, 48, 61, 109,
	64, 104, 8, 11, 10, 7, 6, 134, 133, 126,
	16, 118, 93, 45, 119, 165, 30, 29, 164, 16,
	163, 59, 172, 171, 170, 161, 154, 151, 144, 132,
	116, 92, 56, 152, 122, 3, 1, 147, 12, 112,
	102, 18, 43, 84, 98, 71, 72, 57, 89, 47,
	39, 9, 17, 65, 2,
}
var mmPact = [...]int{

	16, -1000, 9, 165, -1000, -1000, -1000, -1000, 98, -1000,
	96, 72, 165, -1000, -1000, -1000, -1000, 45, 12, -1000,
	184, 183, -1000, 134, -1000, -1000, -1000, -1000, 42, -1000,
	-1000, -1000, 25, -1000, 61, 61, 132, 180, 147, -1000,
	38, 128, -1000, -1000, 191, -1000, 163, -1000, 38, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -16, 166, -29, 88,
	114, 143, -1000, 65, 123, 14, -1000, -1000, -1000, 202,
	179, -1000, -1000, 68, 77, -1000, -1000, -1000, -1000, -1000,
	-1000, -17, -27, -1000, 153, -1000, 107, 167, 102, 83,
	149, 201, -1000, 118, 182, -1000, -1000, -1000, 158, 206,
	23, 18, 136, -1000, 176, -1000, 93, 73, -1000, -1000,
	200, -1000, -1000, 175, 174, -1000, -1000, 155, 10, -1000,
	54, -1000, 118, -1000, -1000, -1000, -1000, -1000, 199, -1000,
	-1000, 36, -1000, -1000, -1000, 61, 3, 198, -1000, -1000,
	205, -1000, -1000, 39, -1000, -1000, 197, 63, 61, 146,
	196, -1000, 118, -1000, -1000, -1000, 190, 188, 185, 135,
	-1000, -1000, -1000, 4, -8, 7, -1000, 195, 194, 193,
	-1000, -1000, -1000,
}
var mmPgo = [...]int{

	0, 234, 212, 9, 5, 233, 3, 232, 8, 186,
	231, 215, 230, 229, 1, 2, 228, 227, 0, 226,
	225, 6, 224, 7, 223, 222, 4, 220, 219, 217,
	216,
}
var mmR1 = [...]int{

	0, 30, 30, 30, 30, 30, 30, 1, 1, 11,
	11, 9, 9, 9, 10, 28, 28, 29, 29, 29,
	29, 2, 2, 8, 8, 14, 14, 12, 12, 15,
	15, 13, 13, 13, 13, 13, 13, 17, 4, 6,
	3, 3, 3, 3, 3, 3, 3, 5, 5, 5,
	16, 16, 16, 27, 24, 24, 23, 23, 7, 7,
	7, 7, 26, 26, 25, 25, 25, 21, 21, 22,
	22, 18, 18, 20, 20, 20, 20, 20, 20, 20,
	20, 20, 20, 20, 20, 19, 19, 19,
}
var mmR2 = [...]int{

	0, 2, 3, 2, 1, 2, 1, 2, 1, 2,
	1, 3, 1, 10, 9, 0, 4, 0, 5, 5,
	5, 3, 1, 0, 3, 0, 2, 6, 5, 0,
	2, 4, 5, 6, 5, 6, 7, 4, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	0, 6, 5, 4, 2, 1, 6, 8, 0, 2,
	2, 2, 0, 2, 4, 8, 7, 3, 1, 5,
	3, 1, 1, 3, 4, 2, 2, 3, 4, 1,
	1, 1, 1, 1, 1, 3, 1, 3,
}
var mmChk = [...]int{

	-1000, -30, -1, -11, -23, 56, -9, 20, 17, -10,
	19, 18, -11, -23, 56, -23, -9, -7, -2, 36,
	36, 36, -23, 36, 21, 22, 23, 7, 40, 13,
	13, 13, 32, 36, -14, -14, -26, 36, -15, -12,
	29, -15, 14, -25, 36, 13, 14, -13, 30, -3,
	47, 48, 50, 49, 51, 46, -2, -17, 31, 10,
	-26, 15, -3, -8, 14, -5, 41, 44, 45, -18,
	24, -20, -19, 11, 15, 38, 39, 37, 52, 53,
	54, 36, 27, 14, -24, -23, -8, 11, 36, -16,
	25, 37, 9, 13, -21, 12, -18, 16, -22, 37,
	40, 40, -27, -23, 28, 9, -4, 36, 37, 12,
	-4, 9, -28, 26, 26, 13, 9, -21, 9, 12,
	9, 16, 8, 36, 36, 16, 13, 9, -6, 37,
	9, -4, 9, 13, 13, -14, 9, 14, -18, 12,
	37, 16, -18, -26, 9, 9, -6, -29, -14, -15,
	14, 9, 8, 14, 9, 14, 33, 34, 35, -15,
	14, 9, -18, 10, 10, 10, 14, 39, 39, 37,
	9, 9, 9,
}
var mmDef = [...]int{

	0, -2, 0, 4, 6, 8, 10, 58, 0, 12,
	0, 0, 1, 3, 7, 5, 9, 0, 0, 22,
	0, 0, 2, 0, 59, 60, 61, 11, 0, 25,
	25, 62, 0, 21, 29, 29, 0, 0, 0, 26,
	0, 0, 56, 63, 0, 62, 0, 30, 0, 23,
	40, 41, 42, 43, 44, 45, 46, 0, 0, 0,
	0, 0, 23, 0, 50, 0, 47, 48, 49, 0,
	0, 71, 72, 0, 0, 79, 80, 81, 82, 83,
	84, 86, 0, 57, 0, 55, 0, 0, 0, 15,
	0, 0, 64, 0, 0, 75, 68, 76, 0, 0,
	0, 0, 0, 54, 0, 31, 0, 0, 38, 24,
	0, 28, 14, 0, 0, 25, 37, 0, 0, 73,
	0, 77, 0, 85, 87, 13, 62, 32, 0, 39,
	34, 0, 27, 17, 25, 29, 0, 0, 67, 74,
	0, 78, 70, 0, 33, 35, 0, 0, 29, 0,
	0, 66, 0, 53, 36, 16, 0, 0, 0, 0,
	52, 65, 69, 0, 0, 0, 51, 0, 0, 0,
	18, 19, 20,
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
	52, 53, 54, 55, 56,
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
		//line src/martian/syntax/grammar.y:81
		{
			{
				global := NewAst(mmDollar[2].decs, nil)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 2:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:87
		{
			{
				global := NewAst(mmDollar[2].decs, mmDollar[3].call)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 3:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:93
		{
			{
				global := NewAst([]Dec{}, mmDollar[2].call)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 4:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:99
		{
			{
				global := NewAst(mmDollar[1].decs, nil)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 5:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:104
		{
			{
				global := NewAst(mmDollar[1].decs, mmDollar[2].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 6:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:109
		{
			{
				global := NewAst([]Dec{}, mmDollar[1].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 7:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:117
		{
			{
				mmVAL.pre_dir = append(mmDollar[1].pre_dir, &preprocessorDirective{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val})
			}
		}
	case 8:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:119
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
		//line src/martian/syntax/grammar.y:129
		{
			{
				mmVAL.decs = append(mmDollar[1].decs, mmDollar[2].dec)
			}
		}
	case 10:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:131
		{
			{
				mmVAL.decs = []Dec{mmDollar[1].dec}
			}
		}
	case 11:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:136
		{
			{
				mmVAL.dec = &UserType{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val}
			}
		}
	case 12:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:138
		{
			{
				mmVAL.dec = mmDollar[1].dec
			}
		}
	case 13:
		mmDollar = mmS[mmpt-10 : mmpt+1]
		//line src/martian/syntax/grammar.y:140
		{
			{
				mmVAL.dec = &Pipeline{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val, mmDollar[4].params, mmDollar[5].params, mmDollar[8].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmDollar[9].retstm}
			}
		}
	case 14:
		mmDollar = mmS[mmpt-9 : mmpt+1]
		//line src/martian/syntax/grammar.y:145
		{
			{
				mmVAL.dec = &Stage{
					Node:      NewAstNode(mmDollar[2].loc, mmDollar[2].locmap),
					Id:        mmDollar[2].val,
					InParams:  mmDollar[4].params,
					OutParams: mmDollar[5].params,
					Src:       mmDollar[6].src,
					ChunkIns:  mmDollar[8].par_tuple.Ins,
					ChunkOuts: mmDollar[8].par_tuple.Outs,
					Split:     mmDollar[8].par_tuple.Present,
					Resources: mmDollar[9].res,
				}
			}
		}
	case 15:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:160
		{
			{
				mmVAL.res = nil
			}
		}
	case 16:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:162
		{
			{
				mmDollar[3].res.Node = NewAstNode(mmDollar[1].loc, mmDollar[1].locmap)
				mmVAL.res = mmDollar[3].res
			}
		}
	case 17:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:170
		{
			{
				mmVAL.res = &Resources{}
			}
		}
	case 18:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:172
		{
			{
				n := NewAstNode(mmDollar[2].loc, mmDollar[2].locmap)
				mmDollar[1].res.ThreadNode = &n
				i, _ := strconv.ParseInt(mmDollar[4].val, 0, 64)
				mmDollar[1].res.Threads = int(i)
				mmVAL.res = mmDollar[1].res
			}
		}
	case 19:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:180
		{
			{
				n := NewAstNode(mmDollar[2].loc, mmDollar[2].locmap)
				mmDollar[1].res.MemNode = &n
				i, _ := strconv.ParseInt(mmDollar[4].val, 0, 64)
				mmDollar[1].res.MemGB = int(i)
				mmVAL.res = mmDollar[1].res
			}
		}
	case 20:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:188
		{
			{
				n := NewAstNode(mmDollar[2].loc, mmDollar[2].locmap)
				mmDollar[1].res.SpecialNode = &n
				mmDollar[1].res.Special = mmDollar[4].val
				mmVAL.res = mmDollar[1].res
			}
		}
	case 21:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:198
		{
			{
				mmVAL.val = mmDollar[1].val + mmDollar[2].val + mmDollar[3].val
			}
		}
	case 23:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:204
		{
			{
				mmVAL.arr = 0
			}
		}
	case 24:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:206
		{
			{
				mmVAL.arr += 1
			}
		}
	case 25:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:211
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 26:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:213
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].inparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 27:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:221
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), false}
			}
		}
	case 28:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:223
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, "", false}
			}
		}
	case 29:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:228
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 30:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:230
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].outparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 31:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:238
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, "default", "", "", false}
			}
		}
	case 32:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:240
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, "default", unquote(mmDollar[4].val), "", false}
			}
		}
	case 33:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:242
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, "default", unquote(mmDollar[4].val), unquote(mmDollar[5].val), false}
			}
		}
	case 34:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:244
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, "", "", false}
			}
		}
	case 35:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:246
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), "", false}
			}
		}
	case 36:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/syntax/grammar.y:248
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), unquote(mmDollar[6].val), false}
			}
		}
	case 37:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:253
		{
			{
				stagecodeParts := strings.Split(unquote(mmDollar[3].val), " ")
				mmVAL.src = &SrcParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), StageLanguage(mmDollar[2].val), stagecodeParts[0], stagecodeParts[1:]}
			}
		}
	case 38:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:259
		{
			{
				mmVAL.val = mmDollar[1].val
			}
		}
	case 39:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:264
		{
			{
				mmVAL.val = mmDollar[1].val
			}
		}
	case 50:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:287
		{
			{
				mmVAL.par_tuple = paramsTuple{
					false,
					&Params{[]Param{}, map[string]Param{}},
					&Params{[]Param{}, map[string]Param{}},
				}
			}
		}
	case 51:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:295
		{
			{
				mmVAL.par_tuple = paramsTuple{true, mmDollar[4].params, mmDollar[5].params}
			}
		}
	case 52:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:297
		{
			{
				mmVAL.par_tuple = paramsTuple{true, mmDollar[3].params, mmDollar[4].params}
			}
		}
	case 53:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:302
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[3].bindings}
			}
		}
	case 54:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:307
		{
			{
				mmVAL.calls = append(mmDollar[1].calls, mmDollar[2].call)
			}
		}
	case 55:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:309
		{
			{
				mmVAL.calls = []*CallStm{mmDollar[1].call}
			}
		}
	case 56:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:314
		{
			{
				mmVAL.call = &CallStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].modifiers, mmDollar[3].val, mmDollar[3].val, mmDollar[5].bindings}
			}
		}
	case 57:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line src/martian/syntax/grammar.y:316
		{
			{
				mmVAL.call = &CallStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].modifiers, mmDollar[5].val, mmDollar[3].val, mmDollar[7].bindings}
			}
		}
	case 58:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:321
		{
			{
				mmVAL.modifiers = &Modifiers{false, false, false}
			}
		}
	case 59:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:323
		{
			{
				mmVAL.modifiers.Local = true
			}
		}
	case 60:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:325
		{
			{
				mmVAL.modifiers.Preflight = true
			}
		}
	case 61:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:327
		{
			{
				mmVAL.modifiers.Volatile = true
			}
		}
	case 62:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:332
		{
			{
				mmVAL.bindings = &BindStms{NewAstNode(mmDollar[0].loc, mmDollar[0].locmap), []*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 63:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:334
		{
			{
				mmDollar[1].bindings.List = append(mmDollar[1].bindings.List, mmDollar[2].binding)
				mmVAL.bindings = mmDollar[1].bindings
			}
		}
	case 64:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:342
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 65:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line src/martian/syntax/grammar.y:344
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 66:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/syntax/grammar.y:346
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 67:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:351
		{
			{
				mmVAL.exps = append(mmDollar[1].exps, mmDollar[3].exp)
			}
		}
	case 68:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:353
		{
			{
				mmVAL.exps = []Exp{mmDollar[1].exp}
			}
		}
	case 69:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:358
		{
			{
				mmDollar[1].kvpairs[unquote(mmDollar[3].val)] = mmDollar[5].exp
				mmVAL.kvpairs = mmDollar[1].kvpairs
			}
		}
	case 70:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:363
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmDollar[1].val): mmDollar[3].exp}
			}
		}
	case 71:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:368
		{
			{
				mmVAL.exp = mmDollar[1].exp
			}
		}
	case 72:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:370
		{
			{
				mmVAL.exp = mmDollar[1].exp
			}
		}
	case 73:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:374
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[2].exps}
			}
		}
	case 74:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:376
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[2].exps}
			}
		}
	case 75:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:378
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: []Exp{}}
			}
		}
	case 76:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:380
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: map[string]interface{}{}}
			}
		}
	case 77:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:382
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: mmDollar[2].kvpairs}
			}
		}
	case 78:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:384
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: mmDollar[2].kvpairs}
			}
		}
	case 79:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:386
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmDollar[1].val, 64)
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindFloat, Value: f}
			}
		}
	case 80:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:391
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmDollar[1].val, 0, 64)
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindInt, Value: i}
			}
		}
	case 81:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:396
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindString, Value: unquote(mmDollar[1].val)}
			}
		}
	case 82:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:398
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindBool, Value: true}
			}
		}
	case 83:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:400
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindBool, Value: false}
			}
		}
	case 84:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:402
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindNull, Value: nil}
			}
		}
	case 85:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:407
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindCall, mmDollar[1].val, mmDollar[3].val}
			}
		}
	case 86:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:409
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindCall, mmDollar[1].val, "default"}
			}
		}
	case 87:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:411
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindSelf, mmDollar[3].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

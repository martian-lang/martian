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
const AS = 57374
const ID = 57375
const LITSTRING = 57376
const NUM_FLOAT = 57377
const NUM_INT = 57378
const DOT = 57379
const PY = 57380
const GO = 57381
const SH = 57382
const EXEC = 57383
const COMPILED = 57384
const MAP = 57385
const INT = 57386
const STRING = 57387
const FLOAT = 57388
const PATH = 57389
const BOOL = 57390
const TRUE = 57391
const FALSE = 57392
const NULL = 57393
const DEFAULT = 57394
const PREPROCESS_DIRECTIVE = 57395

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

//line src/martian/syntax/grammar.y:343

//line yacctab:1
var mmExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const mmPrivate = 57344

const mmLast = 210

var mmAct = [...]int{

	94, 122, 35, 33, 103, 4, 92, 99, 12, 14,
	66, 98, 70, 27, 37, 141, 71, 21, 84, 119,
	8, 9, 10, 7, 26, 49, 118, 36, 80, 8,
	9, 10, 7, 34, 79, 74, 72, 73, 62, 70,
	132, 63, 64, 71, 70, 93, 138, 59, 71, 40,
	75, 76, 77, 146, 27, 80, 13, 32, 86, 68,
	80, 79, 74, 72, 73, 5, 79, 74, 72, 73,
	124, 123, 43, 89, 65, 18, 85, 75, 76, 77,
	87, 134, 75, 76, 77, 55, 50, 51, 53, 52,
	54, 102, 107, 86, 110, 105, 20, 70, 112, 133,
	19, 71, 18, 121, 81, 23, 24, 25, 108, 125,
	69, 48, 47, 80, 131, 104, 105, 22, 135, 79,
	74, 72, 73, 43, 136, 70, 41, 139, 123, 71,
	131, 140, 30, 105, 95, 75, 76, 77, 57, 39,
	144, 80, 100, 7, 148, 43, 83, 79, 74, 72,
	73, 31, 97, 7, 48, 39, 8, 9, 10, 7,
	115, 111, 127, 75, 76, 77, 129, 116, 67, 60,
	106, 130, 128, 120, 113, 91, 44, 114, 6, 29,
	28, 58, 15, 147, 145, 142, 137, 126, 101, 90,
	15, 56, 143, 117, 3, 1, 109, 11, 42, 88,
	17, 96, 78, 45, 82, 46, 38, 16, 61, 2,
}
var mmPact = [...]int{

	12, -1000, 3, 139, -1000, -1000, -1000, -1000, 69, 67,
	63, 139, -1000, -1000, -1000, -1000, 84, 17, -1000, 167,
	166, -1000, 119, -1000, -1000, -1000, -1000, 24, -1000, -1000,
	-1000, -6, -1000, 110, 110, 112, 163, 81, -1000, 42,
	124, -1000, -1000, 171, -1000, 155, -1000, 0, 42, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -24, 153, 86, 90,
	121, -16, -1000, -1000, -1000, -1000, 47, 123, 180, 162,
	33, 118, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -26,
	-30, -1000, -1000, 116, 179, 82, 158, 99, 133, -1000,
	-1000, 114, 165, -1000, -1000, -1000, 151, 185, -7, -14,
	160, -1000, -1000, 94, 61, -1000, -1000, 178, -1000, 146,
	-1000, 159, 157, 28, -1000, 65, -1000, 114, -1000, -1000,
	-1000, -1000, 177, -1000, -1000, 37, -1000, -1000, -1000, 1,
	176, -1000, -1000, 184, -1000, -1000, 126, -1000, -1000, 175,
	39, 174, -1000, 114, -1000, -1000, -1000, -1000, -1000,
}
var mmPgo = [...]int{

	0, 209, 191, 25, 4, 208, 1, 207, 10, 178,
	194, 206, 205, 3, 14, 204, 203, 0, 202, 6,
	201, 5, 199, 198, 2, 196, 195,
}
var mmR1 = [...]int{

	0, 26, 26, 26, 26, 26, 26, 1, 1, 10,
	10, 9, 9, 9, 9, 2, 2, 8, 8, 13,
	13, 11, 11, 14, 14, 12, 12, 12, 12, 12,
	12, 16, 4, 6, 3, 3, 3, 3, 3, 3,
	3, 5, 5, 5, 15, 25, 22, 22, 21, 21,
	7, 7, 7, 7, 24, 24, 23, 23, 23, 19,
	19, 20, 20, 17, 17, 17, 17, 17, 17, 17,
	17, 17, 17, 17, 17, 17, 18, 18, 18,
}
var mmR2 = [...]int{

	0, 2, 3, 2, 1, 2, 1, 2, 1, 2,
	1, 3, 7, 8, 10, 3, 1, 0, 3, 0,
	2, 6, 5, 0, 2, 4, 5, 6, 5, 6,
	7, 4, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 5, 4, 2, 1, 6, 8,
	0, 2, 2, 2, 0, 2, 4, 8, 7, 3,
	1, 5, 3, 3, 4, 2, 2, 3, 4, 1,
	1, 1, 1, 1, 1, 1, 3, 1, 3,
}
var mmChk = [...]int{

	-1000, -26, -1, -10, -21, 53, -9, 20, 17, 18,
	19, -10, -21, 53, -21, -9, -7, -2, 33, 33,
	33, -21, 33, 21, 22, 23, 7, 37, 13, 13,
	13, 32, 33, -13, -13, -24, 33, -14, -11, 29,
	-14, 14, -23, 33, 13, -16, -12, 31, 30, -3,
	44, 45, 47, 46, 48, 43, -2, 14, 10, -24,
	14, -5, 38, 41, 42, -3, -8, 15, -17, 24,
	11, 15, 35, 36, 34, 49, 50, 51, -18, 33,
	27, 14, -15, 25, 34, -8, 11, 33, -22, -21,
	9, 13, -19, 12, -17, 16, -20, 34, 37, 37,
	26, 9, 9, -4, 33, 34, 12, -4, 9, -25,
	-21, 28, -19, 9, 12, 9, 16, 8, 33, 33,
	13, 9, -6, 34, 9, -4, 9, 16, 13, 9,
	14, -17, 12, 34, 16, -17, -13, 9, 9, -6,
	-24, 14, 9, 8, 14, 9, 14, 9, -17,
}
var mmDef = [...]int{

	0, -2, 0, 4, 6, 8, 10, 50, 0, 0,
	0, 1, 3, 7, 5, 9, 0, 0, 16, 0,
	0, 2, 0, 51, 52, 53, 11, 0, 19, 19,
	54, 0, 15, 23, 23, 0, 0, 0, 20, 0,
	0, 48, 55, 0, 54, 0, 24, 0, 0, 17,
	34, 35, 36, 37, 38, 39, 40, 0, 0, 0,
	12, 0, 41, 42, 43, 17, 0, 0, 0, 0,
	0, 0, 69, 70, 71, 72, 73, 74, 75, 77,
	0, 49, 13, 0, 0, 0, 0, 0, 0, 47,
	56, 0, 0, 65, 60, 66, 0, 0, 0, 0,
	0, 31, 25, 0, 0, 32, 18, 0, 22, 0,
	46, 0, 0, 0, 63, 0, 67, 0, 76, 78,
	19, 26, 0, 33, 28, 0, 21, 14, 54, 0,
	0, 59, 64, 0, 68, 62, 0, 27, 29, 0,
	0, 0, 58, 0, 44, 30, 45, 57, 61,
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
	52, 53,
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
				mmVAL.call = &CallStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].modifiers, mmDollar[3].val, mmDollar[3].val, mmDollar[5].bindings}
			}
		}
	case 49:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line src/martian/syntax/grammar.y:250
		{
			{
				mmVAL.call = &CallStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].modifiers, mmDollar[5].val, mmDollar[3].val, mmDollar[7].bindings}
			}
		}
	case 50:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:255
		{
			{
				mmVAL.modifiers = &Modifiers{false, false, false}
			}
		}
	case 51:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:257
		{
			{
				mmVAL.modifiers.Local = true
			}
		}
	case 52:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:259
		{
			{
				mmVAL.modifiers.Preflight = true
			}
		}
	case 53:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:261
		{
			{
				mmVAL.modifiers.Volatile = true
			}
		}
	case 54:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:266
		{
			{
				mmVAL.bindings = &BindStms{NewAstNode(mmDollar[0].loc, mmDollar[0].locmap), []*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 55:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:268
		{
			{
				mmDollar[1].bindings.List = append(mmDollar[1].bindings.List, mmDollar[2].binding)
				mmVAL.bindings = mmDollar[1].bindings
			}
		}
	case 56:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:276
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 57:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line src/martian/syntax/grammar.y:278
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 58:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/syntax/grammar.y:280
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 59:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:285
		{
			{
				mmVAL.exps = append(mmDollar[1].exps, mmDollar[3].exp)
			}
		}
	case 60:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:287
		{
			{
				mmVAL.exps = []Exp{mmDollar[1].exp}
			}
		}
	case 61:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:292
		{
			{
				mmDollar[1].kvpairs[unquote(mmDollar[3].val)] = mmDollar[5].exp
				mmVAL.kvpairs = mmDollar[1].kvpairs
			}
		}
	case 62:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:297
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmDollar[1].val): mmDollar[3].exp}
			}
		}
	case 63:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:302
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[2].exps}
			}
		}
	case 64:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:304
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[2].exps}
			}
		}
	case 65:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:306
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: []Exp{}}
			}
		}
	case 66:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:308
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: map[string]interface{}{}}
			}
		}
	case 67:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:310
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: mmDollar[2].kvpairs}
			}
		}
	case 68:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:312
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: mmDollar[2].kvpairs}
			}
		}
	case 69:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:314
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmDollar[1].val, 64)
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindFloat, Value: f}
			}
		}
	case 70:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:319
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmDollar[1].val, 0, 64)
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindInt, Value: i}
			}
		}
	case 71:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:324
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindString, Value: unquote(mmDollar[1].val)}
			}
		}
	case 72:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:326
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindBool, Value: true}
			}
		}
	case 73:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:328
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindBool, Value: false}
			}
		}
	case 74:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:330
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindNull, Value: nil}
			}
		}
	case 75:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:332
		{
			{
				mmVAL.exp = mmDollar[1].exp
			}
		}
	case 76:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:337
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindCall, mmDollar[1].val, mmDollar[3].val}
			}
		}
	case 77:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:339
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindCall, mmDollar[1].val, "default"}
			}
		}
	case 78:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:341
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindSelf, mmDollar[3].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

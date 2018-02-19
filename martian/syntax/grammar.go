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
const SWEEP = 57359
const RETURN = 57360
const SELF = 57361
const FILETYPE = 57362
const STAGE = 57363
const PIPELINE = 57364
const CALL = 57365
const SPLIT = 57366
const USING = 57367
const LOCAL = 57368
const PREFLIGHT = 57369
const VOLATILE = 57370
const DISABLED = 57371
const IN = 57372
const OUT = 57373
const SRC = 57374
const AS = 57375
const THREADS = 57376
const MEM_GB = 57377
const SPECIAL = 57378
const ID = 57379
const LITSTRING = 57380
const NUM_FLOAT = 57381
const NUM_INT = 57382
const DOT = 57383
const PY = 57384
const GO = 57385
const SH = 57386
const EXEC = 57387
const COMPILED = 57388
const MAP = 57389
const INT = 57390
const STRING = 57391
const FLOAT = 57392
const PATH = 57393
const BOOL = 57394
const TRUE = 57395
const FALSE = 57396
const NULL = 57397
const DEFAULT = 57398
const PREPROCESS_DIRECTIVE = 57399

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
	"SWEEP",
	"RETURN",
	"SELF",
	"FILETYPE",
	"STAGE",
	"PIPELINE",
	"CALL",
	"SPLIT",
	"USING",
	"LOCAL",
	"PREFLIGHT",
	"VOLATILE",
	"DISABLED",
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

//line src/martian/syntax/grammar.y:454

//line yacctab:1
var mmExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
	-1, 39,
	13, 101,
	33, 101,
	-2, 60,
	-1, 40,
	13, 102,
	33, 102,
	-2, 61,
	-1, 41,
	13, 103,
	33, 103,
	-2, 62,
}

const mmPrivate = 57344

const mmLast = 515

var mmAct = [...]int{

	91, 60, 135, 163, 58, 143, 50, 133, 97, 20,
	4, 34, 35, 13, 15, 75, 118, 112, 108, 38,
	86, 87, 42, 36, 8, 11, 10, 7, 8, 11,
	10, 7, 100, 107, 43, 101, 102, 203, 202, 204,
	180, 128, 176, 165, 49, 162, 190, 62, 148, 59,
	47, 136, 51, 63, 74, 84, 43, 160, 17, 70,
	95, 14, 127, 20, 175, 5, 191, 192, 193, 164,
	48, 201, 145, 138, 164, 20, 94, 145, 150, 52,
	195, 85, 88, 89, 72, 90, 7, 152, 74, 98,
	96, 54, 55, 56, 57, 70, 109, 74, 125, 151,
	142, 74, 124, 157, 146, 123, 122, 141, 129, 130,
	158, 31, 7, 169, 168, 32, 33, 26, 27, 28,
	25, 161, 132, 71, 144, 22, 23, 24, 21, 145,
	171, 147, 140, 188, 200, 172, 29, 30, 6, 31,
	154, 45, 16, 32, 33, 26, 27, 28, 25, 155,
	166, 16, 156, 22, 23, 24, 21, 44, 173, 170,
	37, 199, 177, 198, 29, 30, 178, 93, 67, 66,
	181, 65, 184, 64, 173, 207, 183, 206, 113, 70,
	205, 185, 114, 196, 189, 194, 92, 31, 186, 179,
	197, 32, 33, 26, 27, 28, 25, 8, 11, 10,
	7, 22, 23, 24, 21, 117, 115, 116, 1, 167,
	113, 174, 29, 30, 114, 153, 131, 106, 92, 31,
	86, 87, 119, 32, 33, 26, 27, 28, 25, 105,
	104, 103, 82, 22, 23, 24, 21, 117, 115, 116,
	187, 19, 113, 134, 29, 30, 114, 159, 182, 149,
	92, 31, 86, 87, 119, 32, 33, 26, 27, 28,
	25, 3, 139, 46, 12, 22, 23, 24, 21, 117,
	115, 116, 53, 69, 113, 121, 29, 30, 114, 137,
	110, 111, 92, 31, 86, 87, 119, 32, 33, 26,
	27, 28, 25, 83, 126, 73, 61, 22, 23, 24,
	21, 117, 115, 116, 9, 18, 113, 99, 29, 30,
	114, 2, 0, 0, 92, 31, 86, 87, 119, 32,
	33, 26, 27, 28, 25, 0, 0, 0, 0, 22,
	23, 24, 21, 117, 115, 116, 0, 0, 31, 0,
	29, 30, 32, 33, 26, 27, 28, 25, 86, 87,
	119, 0, 22, 23, 24, 21, 0, 0, 0, 0,
	0, 0, 124, 29, 30, 81, 76, 77, 79, 78,
	80, 31, 0, 0, 0, 32, 33, 26, 27, 28,
	25, 0, 0, 0, 0, 22, 23, 24, 21, 120,
	0, 0, 0, 0, 0, 31, 29, 30, 0, 32,
	33, 26, 27, 28, 25, 0, 0, 0, 0, 22,
	23, 24, 21, 0, 0, 0, 0, 92, 31, 0,
	29, 30, 32, 33, 26, 27, 28, 25, 0, 0,
	0, 0, 22, 23, 24, 21, 68, 0, 0, 0,
	0, 0, 31, 29, 30, 0, 32, 33, 26, 27,
	28, 25, 0, 0, 0, 0, 22, 23, 24, 21,
	0, 0, 0, 0, 0, 31, 0, 29, 30, 32,
	33, 26, 27, 28, 25, 0, 0, 0, 0, 22,
	23, 24, 21, 0, 0, 0, 0, 0, 31, 0,
	29, 30, 32, 33, 39, 40, 41, 25, 0, 0,
	0, 0, 22, 23, 24, 21, 0, 0, 0, 0,
	0, 0, 0, 29, 30,
}
var mmPact = [...]int{

	8, -1000, 4, 177, 33, -1000, -1000, -1000, 445, -1000,
	445, 445, 177, 33, -1000, 33, -1000, 147, 468, 15,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 144, 128, 33, -1000, 37, -1000,
	-1000, -1000, -1000, 445, -1000, -1000, 65, -1000, 445, -1000,
	17, 17, -1000, -1000, 163, 161, 159, 158, 422, 110,
	70, -1000, 318, 23, -33, -33, -33, 398, -1000, -1000,
	157, -1000, 45, -1000, 318, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -7, 75, -10, 222, -1000, -1000, 221, 220,
	208, -8, -23, 263, 375, 63, -1000, 351, 38, 3,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, 445, 445, 207,
	109, -1000, -1000, 231, 35, -1000, -1000, -1000, -1000, -1000,
	-1000, 89, 33, 91, 92, 39, 53, 74, 206, -1000,
	-1000, -1000, 295, 140, -1000, -1000, -1000, 94, 239, 41,
	33, 108, -1000, 36, 34, -1000, -1000, 200, -1000, -1000,
	101, 100, -1000, -1000, 121, 199, -1000, 26, -1000, 295,
	-1000, -1000, -1000, 180, -1000, -1000, 31, -1000, -1000, -1000,
	17, 167, 179, -1000, -1000, 232, -1000, -1000, 119, -1000,
	-1000, 175, 32, 17, 66, 174, -1000, 295, -1000, -1000,
	-1000, 153, 151, 124, 57, -1000, -1000, -1000, -2, -3,
	1, -1000, 171, 168, 166, -1000, -1000, -1000,
}
var mmPgo = [...]int{

	0, 311, 0, 232, 15, 5, 307, 3, 305, 8,
	138, 304, 261, 296, 295, 6, 1, 294, 293, 2,
	17, 281, 16, 7, 279, 10, 275, 273, 272, 4,
	263, 262, 249, 248, 208,
}
var mmR1 = [...]int{

	0, 34, 34, 34, 34, 34, 34, 1, 1, 12,
	12, 10, 10, 10, 11, 32, 32, 33, 33, 33,
	33, 3, 3, 9, 9, 15, 15, 13, 13, 16,
	16, 14, 14, 14, 14, 14, 14, 18, 5, 7,
	4, 4, 4, 4, 4, 4, 4, 6, 6, 6,
	17, 17, 17, 31, 26, 26, 25, 25, 25, 8,
	8, 8, 8, 30, 30, 28, 28, 28, 28, 29,
	29, 27, 27, 27, 23, 23, 24, 24, 19, 19,
	21, 21, 21, 21, 21, 21, 21, 21, 21, 21,
	21, 22, 22, 20, 20, 20, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2, 2,
}
var mmR2 = [...]int{

	0, 2, 3, 2, 1, 2, 1, 2, 1, 2,
	1, 3, 1, 10, 9, 0, 4, 0, 5, 5,
	5, 3, 1, 0, 3, 0, 2, 6, 5, 0,
	2, 4, 5, 6, 5, 6, 7, 4, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	0, 6, 5, 4, 2, 1, 6, 8, 5, 0,
	2, 2, 2, 0, 2, 4, 4, 4, 4, 0,
	2, 4, 8, 7, 3, 1, 5, 3, 1, 1,
	3, 4, 2, 2, 3, 4, 1, 1, 1, 1,
	1, 1, 1, 3, 1, 3, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1,
}
var mmChk = [...]int{

	-1000, -34, -1, -12, -25, 57, -10, 23, 20, -11,
	22, 21, -12, -25, 57, -25, -10, 25, -8, -3,
	-2, 37, 34, 35, 36, 29, 26, 27, 28, 45,
	46, 20, 24, 25, -2, -2, -25, 13, -2, 26,
	27, 28, 7, 41, 13, 13, -30, 13, 33, -2,
	-15, -15, 14, -28, 26, 27, 28, 29, -29, -2,
	-16, -13, 30, -16, 10, 10, 10, 10, 14, -27,
	-2, 13, 14, -14, 31, -4, 48, 49, 51, 50,
	52, 47, -3, -18, 32, -22, 53, 54, -22, -22,
	-20, -2, 19, 10, -29, 15, -4, -9, 14, -6,
	42, 45, 46, 9, 9, 9, 9, 41, 41, -19,
	17, -21, -20, 11, 15, 39, 40, 38, -22, 55,
	14, -26, -25, -9, 11, -2, -17, 24, 38, -2,
	-2, 9, 13, -23, 12, -19, 16, -24, 38, -31,
	-25, 18, 9, -5, -2, 38, 12, -5, 9, -32,
	25, 25, 13, 9, -23, 9, 12, 9, 16, 8,
	16, 13, 9, -7, 38, 9, -5, 9, 13, 13,
	-15, 9, 14, -19, 12, 38, 16, -19, -29, 9,
	9, -7, -33, -15, -16, 14, 9, 8, 14, 9,
	14, 34, 35, 36, -16, 14, 9, -19, 10, 10,
	10, 14, 40, 40, 38, 9, 9, 9,
}
var mmDef = [...]int{

	0, -2, 0, 4, 6, 8, 10, 59, 0, 12,
	0, 0, 1, 3, 7, 5, 9, 0, 0, 0,
	22, 96, 97, 98, 99, 100, 101, 102, 103, 104,
	105, 106, 107, 108, 0, 0, 2, 63, 0, -2,
	-2, -2, 11, 0, 25, 25, 0, 69, 0, 21,
	29, 29, 58, 64, 0, 0, 0, 0, 0, 0,
	0, 26, 0, 0, 0, 0, 0, 0, 56, 70,
	0, 69, 0, 30, 0, 23, 40, 41, 42, 43,
	44, 45, 46, 0, 0, 0, 91, 92, 0, 0,
	0, 94, 0, 0, 0, 0, 23, 0, 50, 0,
	47, 48, 49, 65, 66, 67, 68, 0, 0, 0,
	0, 78, 79, 0, 0, 86, 87, 88, 89, 90,
	57, 0, 55, 0, 0, 0, 15, 0, 0, 93,
	95, 71, 0, 0, 82, 75, 83, 0, 0, 0,
	54, 0, 31, 0, 0, 38, 24, 0, 28, 14,
	0, 0, 25, 37, 0, 0, 80, 0, 84, 0,
	13, 69, 32, 0, 39, 34, 0, 27, 17, 25,
	29, 0, 0, 74, 81, 0, 85, 77, 0, 33,
	35, 0, 0, 29, 0, 0, 73, 0, 53, 36,
	16, 0, 0, 0, 0, 52, 72, 76, 0, 0,
	0, 51, 0, 0, 0, 18, 19, 20,
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
	52, 53, 54, 55, 56, 57,
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
		//line src/martian/syntax/grammar.y:83
		{
			{
				global := NewAst(mmDollar[2].decs, nil)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 2:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:89
		{
			{
				global := NewAst(mmDollar[2].decs, mmDollar[3].call)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 3:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:95
		{
			{
				global := NewAst([]Dec{}, mmDollar[2].call)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 4:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:101
		{
			{
				global := NewAst(mmDollar[1].decs, nil)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 5:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:106
		{
			{
				global := NewAst(mmDollar[1].decs, mmDollar[2].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 6:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:111
		{
			{
				global := NewAst([]Dec{}, mmDollar[1].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 7:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:119
		{
			{
				mmVAL.pre_dir = append(mmDollar[1].pre_dir, &preprocessorDirective{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val})
			}
		}
	case 8:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:121
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
		//line src/martian/syntax/grammar.y:131
		{
			{
				mmVAL.decs = append(mmDollar[1].decs, mmDollar[2].dec)
			}
		}
	case 10:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:133
		{
			{
				mmVAL.decs = []Dec{mmDollar[1].dec}
			}
		}
	case 11:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:138
		{
			{
				mmVAL.dec = &UserType{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val}
			}
		}
	case 13:
		mmDollar = mmS[mmpt-10 : mmpt+1]
		//line src/martian/syntax/grammar.y:141
		{
			{
				mmVAL.dec = &Pipeline{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val, mmDollar[4].params, mmDollar[5].params, mmDollar[8].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmDollar[9].retstm}
			}
		}
	case 14:
		mmDollar = mmS[mmpt-9 : mmpt+1]
		//line src/martian/syntax/grammar.y:146
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
		//line src/martian/syntax/grammar.y:161
		{
			{
				mmVAL.res = nil
			}
		}
	case 16:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:163
		{
			{
				mmDollar[3].res.Node = NewAstNode(mmDollar[1].loc, mmDollar[1].locmap)
				mmVAL.res = mmDollar[3].res
			}
		}
	case 17:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:171
		{
			{
				mmVAL.res = &Resources{}
			}
		}
	case 18:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:173
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
		//line src/martian/syntax/grammar.y:181
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
		//line src/martian/syntax/grammar.y:189
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
		//line src/martian/syntax/grammar.y:199
		{
			{
				mmVAL.val = mmDollar[1].val + mmDollar[2].val + mmDollar[3].val
			}
		}
	case 23:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:205
		{
			{
				mmVAL.arr = 0
			}
		}
	case 24:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:207
		{
			{
				mmVAL.arr += 1
			}
		}
	case 25:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:212
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 26:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:214
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].inparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 27:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:222
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), false}
			}
		}
	case 28:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:224
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, "", false}
			}
		}
	case 29:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:229
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 30:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:231
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].outparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 31:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:239
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, "default", "", "", false}
			}
		}
	case 32:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:241
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, "default", unquote(mmDollar[4].val), "", false}
			}
		}
	case 33:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:243
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, "default", unquote(mmDollar[4].val), unquote(mmDollar[5].val), false}
			}
		}
	case 34:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:245
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, "", "", false}
			}
		}
	case 35:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:247
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), "", false}
			}
		}
	case 36:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/syntax/grammar.y:249
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), unquote(mmDollar[6].val), false}
			}
		}
	case 37:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:254
		{
			{
				stagecodeParts := strings.Split(unquote(mmDollar[3].val), " ")
				mmVAL.src = &SrcParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), StageLanguage(mmDollar[2].val), stagecodeParts[0], stagecodeParts[1:]}
			}
		}
	case 50:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:286
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
		//line src/martian/syntax/grammar.y:294
		{
			{
				mmVAL.par_tuple = paramsTuple{true, mmDollar[4].params, mmDollar[5].params}
			}
		}
	case 52:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:296
		{
			{
				mmVAL.par_tuple = paramsTuple{true, mmDollar[3].params, mmDollar[4].params}
			}
		}
	case 53:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:301
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[3].bindings}
			}
		}
	case 54:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:306
		{
			{
				mmVAL.calls = append(mmDollar[1].calls, mmDollar[2].call)
			}
		}
	case 55:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:308
		{
			{
				mmVAL.calls = []*CallStm{mmDollar[1].call}
			}
		}
	case 56:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:313
		{
			{
				mmVAL.call = &CallStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].modifiers, mmDollar[3].val, mmDollar[3].val, mmDollar[5].bindings}
			}
		}
	case 57:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line src/martian/syntax/grammar.y:315
		{
			{
				mmVAL.call = &CallStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].modifiers, mmDollar[5].val, mmDollar[3].val, mmDollar[7].bindings}
			}
		}
	case 58:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:317
		{
			{
				mmDollar[1].call.Modifiers.Bindings = mmDollar[4].bindings
				mmVAL.call = mmDollar[1].call
			}
		}
	case 59:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:325
		{
			{
				mmVAL.modifiers = &Modifiers{}
			}
		}
	case 60:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:327
		{
			{
				mmVAL.modifiers.Local = true
			}
		}
	case 61:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:329
		{
			{
				mmVAL.modifiers.Preflight = true
			}
		}
	case 62:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:331
		{
			{
				mmVAL.modifiers.Volatile = true
			}
		}
	case 63:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:336
		{
			{
				mmVAL.bindings = &BindStms{NewAstNode(mmDollar[0].loc, mmDollar[0].locmap), []*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 64:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:338
		{
			{
				mmDollar[1].bindings.List = append(mmDollar[1].bindings.List, mmDollar[2].binding)
				mmVAL.bindings = mmDollar[1].bindings
			}
		}
	case 65:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:346
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 66:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:348
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 67:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:350
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 68:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:352
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 69:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:356
		{
			{
				mmVAL.bindings = &BindStms{NewAstNode(mmDollar[0].loc, mmDollar[0].locmap), []*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 70:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:358
		{
			{
				mmDollar[1].bindings.List = append(mmDollar[1].bindings.List, mmDollar[2].binding)
				mmVAL.bindings = mmDollar[1].bindings
			}
		}
	case 71:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:366
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 72:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line src/martian/syntax/grammar.y:368
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 73:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/syntax/grammar.y:370
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 74:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:375
		{
			{
				mmVAL.exps = append(mmDollar[1].exps, mmDollar[3].exp)
			}
		}
	case 75:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:377
		{
			{
				mmVAL.exps = []Exp{mmDollar[1].exp}
			}
		}
	case 76:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:382
		{
			{
				mmDollar[1].kvpairs[unquote(mmDollar[3].val)] = mmDollar[5].exp
				mmVAL.kvpairs = mmDollar[1].kvpairs
			}
		}
	case 77:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:387
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmDollar[1].val): mmDollar[3].exp}
			}
		}
	case 80:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:396
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[2].exps}
			}
		}
	case 81:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:398
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[2].exps}
			}
		}
	case 82:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:400
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: []Exp{}}
			}
		}
	case 83:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:402
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: map[string]interface{}{}}
			}
		}
	case 84:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:404
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: mmDollar[2].kvpairs}
			}
		}
	case 85:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:406
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: mmDollar[2].kvpairs}
			}
		}
	case 86:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:408
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmDollar[1].val, 64)
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindFloat, Value: f}
			}
		}
	case 87:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:413
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmDollar[1].val, 0, 64)
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindInt, Value: i}
			}
		}
	case 88:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:418
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindString, Value: unquote(mmDollar[1].val)}
			}
		}
	case 90:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:421
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindNull, Value: nil}
			}
		}
	case 91:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:426
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindBool, Value: true}
			}
		}
	case 92:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:428
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindBool, Value: false}
			}
		}
	case 93:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:432
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindCall, mmDollar[1].val, mmDollar[3].val}
			}
		}
	case 94:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:434
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindCall, mmDollar[1].val, "default"}
			}
		}
	case 95:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:436
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindSelf, mmDollar[3].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

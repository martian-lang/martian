//line martian/syntax/grammar.y:2

//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO grammar.
//

package syntax

import __yyfmt__ "fmt"

//line martian/syntax/grammar.y:8
import (
	"strconv"
	"strings"
)

//line martian/syntax/grammar.y:17
type mmSymType struct {
	yys       int
	global    *Ast
	srcfile   *SourceFile
	arr       int
	loc       int
	val       string
	modifiers *Modifiers
	dec       Dec
	decs      []Dec
	inparam   *InParam
	outparam  *OutParam
	retains   []*RetainParam
	stretains *RetainParams
	params    *Params
	res       *Resources
	par_tuple paramsTuple
	src       *SrcParam
	exp       Exp
	exps      []Exp
	rexp      *RefExp
	vexp      *ValExp
	kvpairs   map[string]Exp
	call      *CallStm
	calls     []*CallStm
	binding   *BindStm
	bindings  *BindStms
	retstm    *ReturnStm
	plretains *PipelineRetains
	reflist   []*RefExp
	includes  []*Include
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
const RETAIN = 57368
const LOCAL = 57369
const PREFLIGHT = 57370
const VOLATILE = 57371
const DISABLED = 57372
const STRICT = 57373
const IN = 57374
const OUT = 57375
const SRC = 57376
const AS = 57377
const THREADS = 57378
const MEM_GB = 57379
const SPECIAL = 57380
const ID = 57381
const LITSTRING = 57382
const NUM_FLOAT = 57383
const NUM_INT = 57384
const DOT = 57385
const PY = 57386
const GO = 57387
const SH = 57388
const EXEC = 57389
const COMPILED = 57390
const MAP = 57391
const INT = 57392
const STRING = 57393
const FLOAT = 57394
const PATH = 57395
const BOOL = 57396
const TRUE = 57397
const FALSE = 57398
const NULL = 57399
const DEFAULT = 57400
const INCLUDE_DIRECTIVE = 57401

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
	"RETAIN",
	"LOCAL",
	"PREFLIGHT",
	"VOLATILE",
	"DISABLED",
	"STRICT",
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
	"INCLUDE_DIRECTIVE",
}
var mmStatenames = [...]string{}

const mmEofCode = 1
const mmErrCode = 2
const mmInitialStackSize = 16

//line martian/syntax/grammar.y:517

//line yacctab:1
var mmExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
	-1, 43,
	13, 110,
	35, 110,
	-2, 69,
	-1, 44,
	13, 112,
	35, 112,
	-2, 70,
	-1, 45,
	13, 119,
	35, 119,
	-2, 71,
}

const mmPrivate = 57344

const mmLast = 595

var mmAct = [...]int{

	95, 64, 139, 168, 116, 62, 54, 147, 137, 21,
	4, 37, 38, 13, 15, 101, 146, 112, 128, 122,
	42, 111, 79, 39, 90, 91, 104, 26, 46, 105,
	106, 32, 35, 30, 27, 29, 36, 24, 33, 47,
	223, 222, 183, 34, 28, 31, 22, 149, 53, 189,
	224, 132, 170, 63, 25, 23, 55, 67, 40, 18,
	66, 140, 225, 74, 47, 174, 182, 21, 78, 88,
	165, 17, 8, 11, 10, 7, 94, 131, 117, 21,
	169, 98, 118, 149, 114, 142, 96, 26, 89, 92,
	93, 32, 35, 30, 27, 29, 36, 24, 33, 74,
	113, 100, 129, 34, 28, 31, 22, 121, 119, 120,
	126, 14, 133, 134, 25, 23, 127, 202, 8, 11,
	10, 7, 90, 91, 123, 219, 156, 154, 148, 7,
	145, 185, 206, 128, 167, 7, 144, 151, 155, 203,
	204, 205, 26, 152, 78, 158, 32, 35, 30, 27,
	29, 36, 24, 33, 51, 99, 171, 5, 34, 28,
	31, 22, 180, 177, 161, 169, 184, 208, 76, 25,
	23, 162, 187, 178, 149, 190, 52, 56, 179, 194,
	102, 180, 191, 193, 186, 176, 78, 78, 74, 150,
	58, 59, 60, 61, 175, 207, 8, 11, 10, 7,
	210, 117, 214, 212, 195, 118, 166, 136, 159, 96,
	26, 160, 218, 75, 32, 35, 30, 27, 29, 36,
	24, 33, 49, 48, 41, 217, 34, 28, 31, 22,
	121, 119, 120, 216, 229, 117, 181, 25, 23, 118,
	215, 97, 71, 96, 26, 90, 91, 123, 32, 35,
	30, 27, 29, 36, 24, 33, 70, 69, 68, 228,
	34, 28, 31, 22, 121, 119, 120, 1, 227, 117,
	138, 25, 23, 118, 226, 221, 220, 96, 26, 90,
	91, 123, 32, 35, 30, 27, 29, 36, 24, 33,
	209, 200, 196, 188, 34, 28, 31, 22, 121, 119,
	120, 192, 172, 117, 157, 25, 23, 118, 135, 110,
	109, 96, 26, 90, 91, 123, 32, 35, 30, 27,
	29, 36, 24, 33, 108, 107, 86, 197, 34, 28,
	31, 22, 121, 119, 120, 20, 163, 153, 3, 25,
	23, 12, 143, 50, 57, 73, 26, 90, 91, 123,
	32, 35, 30, 27, 29, 36, 24, 33, 125, 141,
	115, 87, 34, 28, 31, 22, 130, 164, 198, 173,
	201, 77, 65, 25, 23, 85, 80, 81, 83, 82,
	84, 211, 9, 19, 103, 2, 96, 26, 0, 0,
	0, 32, 35, 30, 27, 29, 36, 24, 33, 0,
	0, 0, 0, 34, 28, 31, 22, 213, 0, 0,
	0, 0, 6, 26, 25, 23, 16, 32, 35, 30,
	27, 29, 36, 24, 33, 16, 0, 0, 0, 34,
	28, 31, 22, 199, 0, 0, 0, 0, 0, 26,
	25, 23, 0, 32, 35, 30, 27, 29, 36, 24,
	33, 0, 0, 0, 0, 34, 28, 31, 22, 124,
	0, 0, 0, 0, 0, 26, 25, 23, 0, 32,
	35, 30, 27, 29, 36, 24, 33, 0, 0, 0,
	0, 34, 28, 31, 22, 0, 0, 0, 0, 96,
	26, 0, 25, 23, 32, 35, 30, 27, 29, 36,
	24, 33, 0, 0, 0, 0, 34, 28, 31, 22,
	72, 0, 0, 0, 0, 0, 26, 25, 23, 0,
	32, 35, 30, 27, 29, 36, 24, 33, 0, 0,
	0, 0, 34, 28, 31, 22, 0, 0, 0, 0,
	0, 26, 0, 25, 23, 32, 35, 30, 27, 29,
	36, 24, 33, 0, 0, 0, 0, 34, 28, 31,
	22, 0, 0, 0, 0, 0, 26, 0, 25, 23,
	32, 35, 30, 43, 44, 45, 24, 33, 0, 0,
	0, 0, 34, 28, 31, 22, 0, 0, 0, 0,
	0, 0, 0, 25, 23,
}
var mmPact = [...]int{

	98, -1000, 52, 176, 46, 19, -1000, -1000, 521, -1000,
	521, 521, 176, 46, 18, 46, -1000, 211, -1000, 546,
	21, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, 210, 209, 46,
	-1000, -1000, 141, -1000, -1000, -1000, -1000, 521, -1000, -1000,
	163, -1000, 521, -1000, 28, 28, -1000, -1000, 248, 247,
	246, 232, 496, 200, 154, -1000, 326, 35, -31, -31,
	-31, 470, -1000, -1000, 231, -1000, 140, -1000, 326, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -4, 166, -18, 316,
	-1000, -1000, 315, 301, 300, -22, -26, 67, 445, 106,
	-1000, 122, 53, 11, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, 521, 521, 299, 194, -1000, -1000, 258, 45, -1000,
	-1000, -1000, -1000, -1000, -1000, 112, 46, 7, 177, 134,
	102, 113, 295, -1000, -1000, -1000, 292, 199, -1000, -1000,
	-1000, 155, 328, 44, 46, 193, -1000, 125, 43, -1000,
	-1000, 293, -1000, 39, 181, 172, -1000, -1000, 164, 224,
	-1000, 26, -1000, 292, 115, 171, -1000, -1000, 284, -1000,
	-1000, 40, -1000, -1000, 169, -1000, -1000, 28, 190, 283,
	-1000, -1000, 319, -1000, -1000, -1000, -1000, 419, -1000, -1000,
	282, -1000, 103, 28, 153, 281, -1000, 292, 367, -1000,
	-1000, 393, -1000, 230, 223, 215, 202, 111, -1000, -1000,
	-1000, -1000, 267, -1000, 266, -1, -2, 10, 31, -1000,
	-1000, -1000, 265, 259, 250, 225, -1000, -1000, -1000, -1000,
}
var mmPgo = [...]int{

	0, 385, 0, 326, 22, 7, 384, 3, 383, 15,
	412, 382, 338, 372, 371, 370, 369, 368, 367, 6,
	1, 366, 361, 2, 4, 360, 19, 8, 359, 10,
	358, 345, 344, 5, 343, 342, 337, 301, 267,
}
var mmR1 = [...]int{

	0, 38, 38, 38, 38, 38, 38, 1, 1, 12,
	12, 10, 10, 10, 11, 36, 36, 37, 37, 37,
	37, 37, 16, 16, 15, 15, 3, 3, 9, 9,
	19, 19, 13, 13, 20, 20, 14, 14, 14, 14,
	14, 14, 22, 5, 7, 4, 4, 4, 4, 4,
	4, 4, 6, 6, 6, 21, 21, 21, 35, 18,
	18, 17, 17, 30, 30, 29, 29, 29, 8, 8,
	8, 8, 34, 34, 32, 32, 32, 32, 33, 33,
	31, 31, 31, 27, 27, 28, 28, 23, 23, 25,
	25, 25, 25, 25, 25, 25, 25, 25, 25, 25,
	26, 26, 24, 24, 24, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
}
var mmR2 = [...]int{

	0, 2, 3, 2, 1, 2, 1, 3, 2, 2,
	1, 3, 1, 11, 10, 0, 4, 0, 5, 5,
	5, 5, 0, 4, 0, 3, 3, 1, 0, 3,
	0, 2, 6, 5, 0, 2, 4, 5, 6, 5,
	6, 7, 4, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 0, 6, 5, 4, 0,
	4, 0, 3, 2, 1, 6, 8, 5, 0, 2,
	2, 2, 0, 2, 4, 4, 4, 4, 0, 2,
	4, 8, 7, 3, 1, 5, 3, 1, 1, 3,
	4, 2, 2, 3, 4, 1, 1, 1, 1, 1,
	1, 1, 3, 1, 3, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
}
var mmChk = [...]int{

	-1000, -38, -1, -12, -29, 59, -10, 23, 20, -11,
	22, 21, -12, -29, 59, -29, -10, 25, 40, -8,
	-3, -2, 39, 48, 30, 47, 20, 27, 37, 28,
	26, 38, 24, 31, 36, 25, 29, -2, -2, -29,
	40, 13, -2, 27, 28, 29, 7, 43, 13, 13,
	-34, 13, 35, -2, -19, -19, 14, -32, 27, 28,
	29, 30, -33, -2, -20, -13, 32, -20, 10, 10,
	10, 10, 14, -31, -2, 13, 14, -14, 33, -4,
	50, 51, 53, 52, 54, 49, -3, -22, 34, -26,
	55, 56, -26, -26, -24, -2, 19, 10, -33, 15,
	-4, -9, 14, -6, 44, 47, 48, 9, 9, 9,
	9, 43, 43, -23, 17, -25, -24, 11, 15, 41,
	42, 40, -26, 57, 14, -30, -29, -9, 11, -2,
	-21, 24, 40, -2, -2, 9, 13, -27, 12, -23,
	16, -28, 40, -35, -29, 18, 9, -5, -2, 40,
	12, -5, 9, -36, 25, 25, 13, 9, -27, 9,
	12, 9, 16, 8, -18, 26, 13, 9, -7, 40,
	9, -5, 9, -16, 26, 13, 13, -19, 9, 14,
	-23, 12, 40, 16, -23, 16, 13, -33, 9, 9,
	-7, 13, -37, -19, -20, 14, 9, 8, -17, 14,
	9, -15, 14, 36, 37, 38, 29, -20, 14, 9,
	-23, 14, -24, 14, -2, 10, 10, 10, 10, 14,
	9, 9, 42, 42, 40, 31, 9, 9, 9, 9,
}
var mmDef = [...]int{

	0, -2, 0, 4, 6, 0, 10, 68, 0, 12,
	0, 0, 1, 3, 0, 5, 9, 0, 8, 0,
	0, 27, 105, 106, 107, 108, 109, 110, 111, 112,
	113, 114, 115, 116, 117, 118, 119, 0, 0, 2,
	7, 72, 0, -2, -2, -2, 11, 0, 30, 30,
	0, 78, 0, 26, 34, 34, 67, 73, 0, 0,
	0, 0, 0, 0, 0, 31, 0, 0, 0, 0,
	0, 0, 65, 79, 0, 78, 0, 35, 0, 28,
	45, 46, 47, 48, 49, 50, 51, 0, 0, 0,
	100, 101, 0, 0, 0, 103, 0, 0, 0, 0,
	28, 0, 55, 0, 52, 53, 54, 74, 75, 76,
	77, 0, 0, 0, 0, 87, 88, 0, 0, 95,
	96, 97, 98, 99, 66, 0, 64, 0, 0, 0,
	15, 0, 0, 102, 104, 80, 0, 0, 91, 84,
	92, 0, 0, 59, 63, 0, 36, 0, 0, 43,
	29, 0, 33, 22, 0, 0, 30, 42, 0, 0,
	89, 0, 93, 0, 0, 0, 78, 37, 0, 44,
	39, 0, 32, 14, 0, 17, 30, 34, 0, 0,
	83, 90, 0, 94, 86, 13, 61, 0, 38, 40,
	0, 24, 0, 34, 0, 0, 82, 0, 0, 58,
	41, 0, 16, 0, 0, 0, 0, 0, 57, 81,
	85, 60, 0, 23, 0, 0, 0, 0, 0, 56,
	62, 25, 0, 0, 0, 0, 18, 19, 20, 21,
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
	52, 53, 54, 55, 56, 57, 58, 59,
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
		//line martian/syntax/grammar.y:92
		{
			{
				global := NewAst(mmDollar[2].decs, nil, mmDollar[2].srcfile)
				global.Includes = mmDollar[1].includes
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 2:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:98
		{
			{
				global := NewAst(mmDollar[2].decs, mmDollar[3].call, mmDollar[2].srcfile)
				global.Includes = mmDollar[1].includes
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 3:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:104
		{
			{
				global := NewAst([]Dec{}, mmDollar[2].call, mmDollar[2].srcfile)
				global.Includes = mmDollar[1].includes
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 4:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:110
		{
			{
				global := NewAst(mmDollar[1].decs, nil, mmDollar[1].srcfile)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 5:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:115
		{
			{
				global := NewAst(mmDollar[1].decs, mmDollar[2].call, mmDollar[1].srcfile)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 6:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:120
		{
			{
				global := NewAst([]Dec{}, mmDollar[1].call, mmDollar[1].srcfile)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 7:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:128
		{
			{
				mmVAL.includes = append(mmDollar[1].includes, &Include{NewAstNode(mmDollar[2].loc, mmDollar[2].srcfile), unquote(mmDollar[3].val)})
			}
		}
	case 8:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:130
		{
			{
				mmVAL.includes = []*Include{
					&Include{
						Node:  NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile),
						Value: unquote(mmDollar[2].val),
					},
				}
			}
		}
	case 9:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:140
		{
			{
				mmVAL.decs = append(mmDollar[1].decs, mmDollar[2].dec)
			}
		}
	case 10:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:142
		{
			{
				mmVAL.decs = []Dec{mmDollar[1].dec}
			}
		}
	case 11:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:147
		{
			{
				mmVAL.dec = &UserType{NewAstNode(mmDollar[2].loc, mmDollar[2].srcfile), mmDollar[2].val}
			}
		}
	case 13:
		mmDollar = mmS[mmpt-11 : mmpt+1]
		//line martian/syntax/grammar.y:150
		{
			{
				mmVAL.dec = &Pipeline{NewAstNode(mmDollar[2].loc, mmDollar[2].srcfile), mmDollar[2].val, mmDollar[4].params, mmDollar[5].params, mmDollar[8].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmDollar[9].retstm, mmDollar[10].plretains}
			}
		}
	case 14:
		mmDollar = mmS[mmpt-10 : mmpt+1]
		//line martian/syntax/grammar.y:155
		{
			{
				mmVAL.dec = &Stage{
					Node:      NewAstNode(mmDollar[2].loc, mmDollar[2].srcfile),
					Id:        mmDollar[2].val,
					InParams:  mmDollar[4].params,
					OutParams: mmDollar[5].params,
					Src:       mmDollar[6].src,
					ChunkIns:  mmDollar[8].par_tuple.Ins,
					ChunkOuts: mmDollar[8].par_tuple.Outs,
					Split:     mmDollar[8].par_tuple.Present,
					Resources: mmDollar[9].res,
					Retain:    mmDollar[10].stretains,
				}
			}
		}
	case 15:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:171
		{
			{
				mmVAL.res = nil
			}
		}
	case 16:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:173
		{
			{
				mmDollar[3].res.Node = NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile)
				mmVAL.res = mmDollar[3].res
			}
		}
	case 17:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:181
		{
			{
				mmVAL.res = &Resources{}
			}
		}
	case 18:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:183
		{
			{
				n := NewAstNode(mmDollar[2].loc, mmDollar[2].srcfile)
				mmDollar[1].res.ThreadNode = &n
				i, _ := strconv.ParseInt(mmDollar[4].val, 0, 64)
				mmDollar[1].res.Threads = int(i)
				mmVAL.res = mmDollar[1].res
			}
		}
	case 19:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:191
		{
			{
				n := NewAstNode(mmDollar[2].loc, mmDollar[2].srcfile)
				mmDollar[1].res.MemNode = &n
				i, _ := strconv.ParseInt(mmDollar[4].val, 0, 64)
				mmDollar[1].res.MemGB = int(i)
				mmVAL.res = mmDollar[1].res
			}
		}
	case 20:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:199
		{
			{
				n := NewAstNode(mmDollar[2].loc, mmDollar[2].srcfile)
				mmDollar[1].res.SpecialNode = &n
				mmDollar[1].res.Special = mmDollar[4].val
				mmVAL.res = mmDollar[1].res
			}
		}
	case 21:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:206
		{
			{
				n := NewAstNode(mmDollar[2].loc, mmDollar[2].srcfile)
				mmDollar[1].res.VolatileNode = &n
				mmDollar[1].res.StrictVolatile = true
				mmVAL.res = mmDollar[1].res
			}
		}
	case 22:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:216
		{
			{
				mmVAL.stretains = nil
			}
		}
	case 23:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:218
		{
			{
				mmVAL.stretains = &RetainParams{
					Node:   NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile),
					Params: mmDollar[3].retains,
				}
			}
		}
	case 24:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:228
		{
			{
				mmVAL.retains = nil
			}
		}
	case 25:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:230
		{
			{
				mmVAL.retains = append(mmDollar[1].retains, &RetainParam{
					Node: NewAstNode(mmDollar[2].loc, mmDollar[2].srcfile),
					Id:   mmDollar[2].val,
				})
			}
		}
	case 26:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:241
		{
			{
				mmVAL.val = mmDollar[1].val + mmDollar[2].val + mmDollar[3].val
			}
		}
	case 28:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:247
		{
			{
				mmVAL.arr = 0
			}
		}
	case 29:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:249
		{
			{
				mmVAL.arr += 1
			}
		}
	case 30:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:254
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 31:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:256
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].inparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 32:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line martian/syntax/grammar.y:264
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), false}
			}
		}
	case 33:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:266
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, "", false}
			}
		}
	case 34:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:271
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 35:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:273
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].outparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 36:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:281
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[2].val, mmDollar[3].arr, "default", "", "", false}
			}
		}
	case 37:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:283
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[2].val, mmDollar[3].arr, "default", unquote(mmDollar[4].val), "", false}
			}
		}
	case 38:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line martian/syntax/grammar.y:285
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[2].val, mmDollar[3].arr, "default", unquote(mmDollar[4].val), unquote(mmDollar[5].val), false}
			}
		}
	case 39:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:287
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, "", "", false}
			}
		}
	case 40:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line martian/syntax/grammar.y:289
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), "", false}
			}
		}
	case 41:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line martian/syntax/grammar.y:291
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), unquote(mmDollar[6].val), false}
			}
		}
	case 42:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:296
		{
			{
				stagecodeParts := strings.Split(unquote(mmDollar[3].val), " ")
				mmVAL.src = &SrcParam{
					NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile),
					StageLanguage(mmDollar[2].val),
					stagecodeParts[0],
					stagecodeParts[1:],
				}
			}
		}
	case 55:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:333
		{
			{
				mmVAL.par_tuple = paramsTuple{
					false,
					&Params{[]Param{}, map[string]Param{}},
					&Params{[]Param{}, map[string]Param{}},
				}
			}
		}
	case 56:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line martian/syntax/grammar.y:341
		{
			{
				mmVAL.par_tuple = paramsTuple{true, mmDollar[4].params, mmDollar[5].params}
			}
		}
	case 57:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:343
		{
			{
				mmVAL.par_tuple = paramsTuple{true, mmDollar[3].params, mmDollar[4].params}
			}
		}
	case 58:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:348
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[3].bindings}
			}
		}
	case 59:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:353
		{
			{
				mmVAL.plretains = nil
			}
		}
	case 60:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:355
		{
			{
				mmVAL.plretains = &PipelineRetains{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[3].reflist}
			}
		}
	case 61:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:359
		{
			{
				mmVAL.reflist = nil
			}
		}
	case 62:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:361
		{
			{
				mmVAL.reflist = append(mmDollar[1].reflist, mmDollar[2].rexp)
			}
		}
	case 63:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:365
		{
			{
				mmVAL.calls = append(mmDollar[1].calls, mmDollar[2].call)
			}
		}
	case 64:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:367
		{
			{
				mmVAL.calls = []*CallStm{mmDollar[1].call}
			}
		}
	case 65:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line martian/syntax/grammar.y:372
		{
			{
				mmVAL.call = &CallStm{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[2].modifiers, mmDollar[3].val, mmDollar[3].val, mmDollar[5].bindings}
			}
		}
	case 66:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line martian/syntax/grammar.y:374
		{
			{
				mmVAL.call = &CallStm{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[2].modifiers, mmDollar[5].val, mmDollar[3].val, mmDollar[7].bindings}
			}
		}
	case 67:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:376
		{
			{
				mmDollar[1].call.Modifiers.Bindings = mmDollar[4].bindings
				mmVAL.call = mmDollar[1].call
			}
		}
	case 68:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:384
		{
			{
				mmVAL.modifiers = &Modifiers{}
			}
		}
	case 69:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:386
		{
			{
				mmVAL.modifiers.Local = true
			}
		}
	case 70:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:388
		{
			{
				mmVAL.modifiers.Preflight = true
			}
		}
	case 71:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:390
		{
			{
				mmVAL.modifiers.Volatile = true
			}
		}
	case 72:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:395
		{
			{
				mmVAL.bindings = &BindStms{NewAstNode(mmDollar[0].loc, mmDollar[0].srcfile), []*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 73:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:397
		{
			{
				mmDollar[1].bindings.List = append(mmDollar[1].bindings.List, mmDollar[2].binding)
				mmVAL.bindings = mmDollar[1].bindings
			}
		}
	case 74:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:405
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[1].val, mmDollar[3].vexp, false, ""}
			}
		}
	case 75:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:407
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[1].val, mmDollar[3].vexp, false, ""}
			}
		}
	case 76:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:409
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[1].val, mmDollar[3].vexp, false, ""}
			}
		}
	case 77:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:411
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[1].val, mmDollar[3].rexp, false, ""}
			}
		}
	case 78:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:415
		{
			{
				mmVAL.bindings = &BindStms{NewAstNode(mmDollar[0].loc, mmDollar[0].srcfile), []*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 79:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:417
		{
			{
				mmDollar[1].bindings.List = append(mmDollar[1].bindings.List, mmDollar[2].binding)
				mmVAL.bindings = mmDollar[1].bindings
			}
		}
	case 80:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:425
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 81:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line martian/syntax/grammar.y:427
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[1].val, &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindArray, Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 82:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line martian/syntax/grammar.y:429
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), mmDollar[1].val, &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindArray, Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 83:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:434
		{
			{
				mmVAL.exps = append(mmDollar[1].exps, mmDollar[3].exp)
			}
		}
	case 84:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:436
		{
			{
				mmVAL.exps = []Exp{mmDollar[1].exp}
			}
		}
	case 85:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:441
		{
			{
				mmDollar[1].kvpairs[unquote(mmDollar[3].val)] = mmDollar[5].exp
				mmVAL.kvpairs = mmDollar[1].kvpairs
			}
		}
	case 86:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:446
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmDollar[1].val): mmDollar[3].exp}
			}
		}
	case 87:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:451
		{
			{
				mmVAL.exp = mmDollar[1].vexp
			}
		}
	case 88:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:453
		{
			{
				mmVAL.exp = mmDollar[1].rexp
			}
		}
	case 89:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:457
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindArray, Value: mmDollar[2].exps}
			}
		}
	case 90:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:459
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindArray, Value: mmDollar[2].exps}
			}
		}
	case 91:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:461
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindArray, Value: []Exp{}}
			}
		}
	case 92:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:463
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindMap, Value: map[string]interface{}{}}
			}
		}
	case 93:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:465
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindMap, Value: mmDollar[2].kvpairs}
			}
		}
	case 94:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:467
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindMap, Value: mmDollar[2].kvpairs}
			}
		}
	case 95:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:469
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmDollar[1].val, 64)
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindFloat, Value: f}
			}
		}
	case 96:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:474
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmDollar[1].val, 0, 64)
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindInt, Value: i}
			}
		}
	case 97:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:479
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindString, Value: unquote(mmDollar[1].val)}
			}
		}
	case 99:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:482
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindNull, Value: nil}
			}
		}
	case 100:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:487
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindBool, Value: true}
			}
		}
	case 101:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:489
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), Kind: KindBool, Value: false}
			}
		}
	case 102:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:493
		{
			{
				mmVAL.rexp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), KindCall, mmDollar[1].val, mmDollar[3].val}
			}
		}
	case 103:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:495
		{
			{
				mmVAL.rexp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), KindCall, mmDollar[1].val, "default"}
			}
		}
	case 104:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:497
		{
			{
				mmVAL.rexp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].srcfile), KindSelf, mmDollar[3].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

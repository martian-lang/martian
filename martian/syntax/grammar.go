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

func unquote(qs string) string {
	return strings.Replace(qs, "\"", "", -1)
}

//line martian/syntax/grammar.y:20
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
const PREPROCESS_DIRECTIVE = 57401

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
	"PREPROCESS_DIRECTIVE",
}
var mmStatenames = [...]string{}

const mmEofCode = 1
const mmErrCode = 2
const mmInitialStackSize = 16

//line martian/syntax/grammar.y:520

//line yacctab:1
var mmExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
	-1, 41,
	13, 110,
	35, 110,
	-2, 69,
	-1, 42,
	13, 112,
	35, 112,
	-2, 70,
	-1, 43,
	13, 119,
	35, 119,
	-2, 71,
}

const mmPrivate = 57344

const mmLast = 592

var mmAct = [...]int{

	93, 62, 137, 166, 114, 60, 52, 145, 135, 20,
	4, 36, 37, 13, 15, 99, 126, 88, 89, 40,
	44, 110, 77, 38, 102, 25, 109, 103, 104, 31,
	34, 29, 26, 28, 35, 23, 32, 45, 221, 220,
	181, 33, 27, 30, 21, 200, 51, 187, 120, 222,
	168, 61, 24, 22, 53, 65, 45, 165, 130, 138,
	204, 72, 76, 86, 180, 20, 64, 201, 202, 203,
	8, 11, 10, 7, 92, 223, 115, 20, 167, 96,
	116, 147, 112, 140, 94, 25, 172, 150, 167, 31,
	34, 29, 26, 28, 35, 23, 32, 72, 111, 98,
	127, 33, 27, 30, 21, 119, 117, 118, 124, 14,
	131, 132, 24, 22, 125, 87, 90, 91, 147, 49,
	88, 89, 121, 217, 163, 154, 146, 8, 11, 10,
	7, 183, 129, 206, 142, 149, 74, 153, 17, 94,
	25, 50, 76, 156, 31, 34, 29, 26, 28, 35,
	23, 32, 76, 152, 169, 76, 33, 27, 30, 21,
	178, 175, 7, 97, 182, 159, 5, 24, 22, 54,
	185, 143, 160, 188, 100, 176, 7, 192, 189, 178,
	177, 191, 56, 57, 58, 59, 72, 8, 11, 10,
	7, 6, 184, 205, 174, 16, 173, 164, 208, 115,
	212, 210, 193, 116, 16, 134, 157, 94, 25, 158,
	216, 73, 31, 34, 29, 26, 28, 35, 23, 32,
	47, 46, 39, 148, 33, 27, 30, 21, 119, 117,
	118, 215, 227, 115, 179, 24, 22, 116, 214, 213,
	95, 94, 25, 88, 89, 121, 31, 34, 29, 26,
	28, 35, 23, 32, 69, 68, 67, 66, 33, 27,
	30, 21, 119, 117, 118, 1, 226, 115, 136, 24,
	22, 116, 225, 224, 219, 94, 25, 88, 89, 121,
	31, 34, 29, 26, 28, 35, 23, 32, 218, 207,
	198, 194, 33, 27, 30, 21, 119, 117, 118, 190,
	186, 115, 170, 24, 22, 116, 155, 133, 108, 94,
	25, 88, 89, 121, 31, 34, 29, 26, 28, 35,
	23, 32, 107, 106, 105, 84, 33, 27, 30, 21,
	119, 117, 118, 195, 19, 161, 3, 24, 22, 12,
	151, 141, 48, 55, 25, 88, 89, 121, 31, 34,
	29, 26, 28, 35, 23, 32, 71, 123, 139, 113,
	33, 27, 30, 21, 85, 128, 162, 196, 144, 171,
	126, 24, 22, 83, 78, 79, 81, 80, 82, 25,
	199, 75, 63, 31, 34, 29, 26, 28, 35, 23,
	32, 9, 18, 101, 2, 33, 27, 30, 21, 147,
	0, 0, 0, 209, 0, 0, 24, 22, 94, 25,
	0, 0, 0, 31, 34, 29, 26, 28, 35, 23,
	32, 0, 0, 0, 0, 33, 27, 30, 21, 211,
	0, 0, 0, 0, 0, 25, 24, 22, 0, 31,
	34, 29, 26, 28, 35, 23, 32, 0, 0, 0,
	0, 33, 27, 30, 21, 197, 0, 0, 0, 0,
	0, 25, 24, 22, 0, 31, 34, 29, 26, 28,
	35, 23, 32, 0, 0, 0, 0, 33, 27, 30,
	21, 122, 0, 0, 0, 0, 0, 25, 24, 22,
	0, 31, 34, 29, 26, 28, 35, 23, 32, 0,
	0, 0, 0, 33, 27, 30, 21, 70, 0, 0,
	0, 0, 0, 25, 24, 22, 0, 31, 34, 29,
	26, 28, 35, 23, 32, 0, 0, 0, 0, 33,
	27, 30, 21, 0, 0, 0, 0, 0, 25, 0,
	24, 22, 31, 34, 29, 26, 28, 35, 23, 32,
	0, 0, 0, 0, 33, 27, 30, 21, 0, 0,
	0, 0, 0, 25, 0, 24, 22, 31, 34, 29,
	41, 42, 43, 23, 32, 0, 0, 0, 0, 33,
	27, 30, 21, 0, 0, 0, 0, 0, 0, 0,
	24, 22,
}
var mmPact = [...]int{

	107, -1000, 50, 167, 113, -1000, -1000, -1000, 518, -1000,
	518, 518, 167, 113, -1000, 113, -1000, 209, 543, 13,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, 208, 207, 113, -1000,
	106, -1000, -1000, -1000, -1000, 518, -1000, -1000, 155, -1000,
	518, -1000, 34, 34, -1000, -1000, 247, 246, 245, 244,
	493, 198, 122, -1000, 324, 29, -38, -38, -38, 120,
	-1000, -1000, 230, -1000, 148, -1000, 324, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -6, 160, -20, 315, -1000, -1000,
	314, 313, 299, -17, -22, 65, 467, 139, -1000, 5,
	108, 18, -1000, -1000, -1000, -1000, -1000, -1000, -1000, 518,
	518, 298, 192, -1000, -1000, 256, 43, -1000, -1000, -1000,
	-1000, -1000, -1000, 153, 113, 359, 211, 78, 128, 112,
	297, -1000, -1000, -1000, 290, 197, -1000, -1000, -1000, 156,
	327, 98, 113, 184, -1000, 48, 41, -1000, -1000, 293,
	-1000, 60, 183, 181, -1000, -1000, 166, 222, -1000, 24,
	-1000, 290, 115, 179, -1000, -1000, 291, -1000, -1000, 38,
	-1000, -1000, 165, -1000, -1000, 34, 188, 282, -1000, -1000,
	325, -1000, -1000, -1000, -1000, 441, -1000, -1000, 281, -1000,
	31, 34, 119, 280, -1000, 290, 389, -1000, -1000, 415,
	-1000, 229, 228, 221, 200, 109, -1000, -1000, -1000, -1000,
	279, -1000, 265, -3, -4, 9, 44, -1000, -1000, -1000,
	264, 263, 257, 223, -1000, -1000, -1000, -1000,
}
var mmPgo = [...]int{

	0, 394, 0, 325, 22, 7, 393, 3, 392, 15,
	191, 391, 336, 382, 381, 380, 369, 367, 366, 6,
	1, 365, 364, 2, 4, 359, 48, 8, 358, 10,
	357, 356, 343, 5, 342, 341, 340, 299, 265,
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

	0, 2, 3, 2, 1, 2, 1, 2, 1, 2,
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
	22, 21, -12, -29, 59, -29, -10, 25, -8, -3,
	-2, 39, 48, 30, 47, 20, 27, 37, 28, 26,
	38, 24, 31, 36, 25, 29, -2, -2, -29, 13,
	-2, 27, 28, 29, 7, 43, 13, 13, -34, 13,
	35, -2, -19, -19, 14, -32, 27, 28, 29, 30,
	-33, -2, -20, -13, 32, -20, 10, 10, 10, 10,
	14, -31, -2, 13, 14, -14, 33, -4, 50, 51,
	53, 52, 54, 49, -3, -22, 34, -26, 55, 56,
	-26, -26, -24, -2, 19, 10, -33, 15, -4, -9,
	14, -6, 44, 47, 48, 9, 9, 9, 9, 43,
	43, -23, 17, -25, -24, 11, 15, 41, 42, 40,
	-26, 57, 14, -30, -29, -9, 11, -2, -21, 24,
	40, -2, -2, 9, 13, -27, 12, -23, 16, -28,
	40, -35, -29, 18, 9, -5, -2, 40, 12, -5,
	9, -36, 25, 25, 13, 9, -27, 9, 12, 9,
	16, 8, -18, 26, 13, 9, -7, 40, 9, -5,
	9, -16, 26, 13, 13, -19, 9, 14, -23, 12,
	40, 16, -23, 16, 13, -33, 9, 9, -7, 13,
	-37, -19, -20, 14, 9, 8, -17, 14, 9, -15,
	14, 36, 37, 38, 29, -20, 14, 9, -23, 14,
	-24, 14, -2, 10, 10, 10, 10, 14, 9, 9,
	42, 42, 40, 31, 9, 9, 9, 9,
}
var mmDef = [...]int{

	0, -2, 0, 4, 6, 8, 10, 68, 0, 12,
	0, 0, 1, 3, 7, 5, 9, 0, 0, 0,
	27, 105, 106, 107, 108, 109, 110, 111, 112, 113,
	114, 115, 116, 117, 118, 119, 0, 0, 2, 72,
	0, -2, -2, -2, 11, 0, 30, 30, 0, 78,
	0, 26, 34, 34, 67, 73, 0, 0, 0, 0,
	0, 0, 0, 31, 0, 0, 0, 0, 0, 0,
	65, 79, 0, 78, 0, 35, 0, 28, 45, 46,
	47, 48, 49, 50, 51, 0, 0, 0, 100, 101,
	0, 0, 0, 103, 0, 0, 0, 0, 28, 0,
	55, 0, 52, 53, 54, 74, 75, 76, 77, 0,
	0, 0, 0, 87, 88, 0, 0, 95, 96, 97,
	98, 99, 66, 0, 64, 0, 0, 0, 15, 0,
	0, 102, 104, 80, 0, 0, 91, 84, 92, 0,
	0, 59, 63, 0, 36, 0, 0, 43, 29, 0,
	33, 22, 0, 0, 30, 42, 0, 0, 89, 0,
	93, 0, 0, 0, 78, 37, 0, 44, 39, 0,
	32, 14, 0, 17, 30, 34, 0, 0, 83, 90,
	0, 94, 86, 13, 61, 0, 38, 40, 0, 24,
	0, 34, 0, 0, 82, 0, 0, 58, 41, 0,
	16, 0, 0, 0, 0, 0, 57, 81, 85, 60,
	0, 23, 0, 0, 0, 0, 0, 56, 62, 25,
	0, 0, 0, 0, 18, 19, 20, 21,
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
		//line martian/syntax/grammar.y:95
		{
			{
				global := NewAst(mmDollar[2].decs, nil)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 2:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:101
		{
			{
				global := NewAst(mmDollar[2].decs, mmDollar[3].call)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 3:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:107
		{
			{
				global := NewAst([]Dec{}, mmDollar[2].call)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 4:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:113
		{
			{
				global := NewAst(mmDollar[1].decs, nil)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 5:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:118
		{
			{
				global := NewAst(mmDollar[1].decs, mmDollar[2].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 6:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:123
		{
			{
				global := NewAst([]Dec{}, mmDollar[1].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 7:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:131
		{
			{
				mmVAL.pre_dir = append(mmDollar[1].pre_dir, &preprocessorDirective{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val})
			}
		}
	case 8:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:133
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
		//line martian/syntax/grammar.y:143
		{
			{
				mmVAL.decs = append(mmDollar[1].decs, mmDollar[2].dec)
			}
		}
	case 10:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:145
		{
			{
				mmVAL.decs = []Dec{mmDollar[1].dec}
			}
		}
	case 11:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:150
		{
			{
				mmVAL.dec = &UserType{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val}
			}
		}
	case 13:
		mmDollar = mmS[mmpt-11 : mmpt+1]
		//line martian/syntax/grammar.y:153
		{
			{
				mmVAL.dec = &Pipeline{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val, mmDollar[4].params, mmDollar[5].params, mmDollar[8].calls, &Callables{[]Callable{}, map[string]Callable{}}, mmDollar[9].retstm, mmDollar[10].plretains}
			}
		}
	case 14:
		mmDollar = mmS[mmpt-10 : mmpt+1]
		//line martian/syntax/grammar.y:158
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
					Retain:    mmDollar[10].stretains,
				}
			}
		}
	case 15:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:174
		{
			{
				mmVAL.res = nil
			}
		}
	case 16:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:176
		{
			{
				mmDollar[3].res.Node = NewAstNode(mmDollar[1].loc, mmDollar[1].locmap)
				mmVAL.res = mmDollar[3].res
			}
		}
	case 17:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:184
		{
			{
				mmVAL.res = &Resources{}
			}
		}
	case 18:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:186
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
		//line martian/syntax/grammar.y:194
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
		//line martian/syntax/grammar.y:202
		{
			{
				n := NewAstNode(mmDollar[2].loc, mmDollar[2].locmap)
				mmDollar[1].res.SpecialNode = &n
				mmDollar[1].res.Special = mmDollar[4].val
				mmVAL.res = mmDollar[1].res
			}
		}
	case 21:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:209
		{
			{
				n := NewAstNode(mmDollar[2].loc, mmDollar[2].locmap)
				mmDollar[1].res.VolatileNode = &n
				mmDollar[1].res.StrictVolatile = true
				mmVAL.res = mmDollar[1].res
			}
		}
	case 22:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:219
		{
			{
				mmVAL.stretains = nil
			}
		}
	case 23:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:221
		{
			{
				mmVAL.stretains = &RetainParams{
					Node:   NewAstNode(mmDollar[1].loc, mmDollar[1].locmap),
					Params: mmDollar[3].retains,
				}
			}
		}
	case 24:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:231
		{
			{
				mmVAL.retains = nil
			}
		}
	case 25:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:233
		{
			{
				mmVAL.retains = append(mmDollar[1].retains, &RetainParam{
					Node: NewAstNode(mmDollar[2].loc, mmDollar[2].locmap),
					Id:   mmDollar[2].val,
				})
			}
		}
	case 26:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:244
		{
			{
				mmVAL.val = mmDollar[1].val + mmDollar[2].val + mmDollar[3].val
			}
		}
	case 28:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:250
		{
			{
				mmVAL.arr = 0
			}
		}
	case 29:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:252
		{
			{
				mmVAL.arr += 1
			}
		}
	case 30:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:257
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 31:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:259
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].inparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 32:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line martian/syntax/grammar.y:267
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), false}
			}
		}
	case 33:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:269
		{
			{
				mmVAL.inparam = &InParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, "", false}
			}
		}
	case 34:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:274
		{
			{
				mmVAL.params = &Params{[]Param{}, map[string]Param{}}
			}
		}
	case 35:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:276
		{
			{
				mmDollar[1].params.List = append(mmDollar[1].params.List, mmDollar[2].outparam)
				mmVAL.params = mmDollar[1].params
			}
		}
	case 36:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:284
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, "default", "", "", false}
			}
		}
	case 37:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:286
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, "default", unquote(mmDollar[4].val), "", false}
			}
		}
	case 38:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line martian/syntax/grammar.y:288
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, "default", unquote(mmDollar[4].val), unquote(mmDollar[5].val), false}
			}
		}
	case 39:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:290
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, "", "", false}
			}
		}
	case 40:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line martian/syntax/grammar.y:292
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), "", false}
			}
		}
	case 41:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line martian/syntax/grammar.y:294
		{
			{
				mmVAL.outparam = &OutParam{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].val, mmDollar[3].arr, mmDollar[4].val, unquote(mmDollar[5].val), unquote(mmDollar[6].val), false}
			}
		}
	case 42:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:299
		{
			{
				stagecodeParts := strings.Split(unquote(mmDollar[3].val), " ")
				mmVAL.src = &SrcParam{
					NewAstNode(mmDollar[1].loc, mmDollar[1].locmap),
					StageLanguage(mmDollar[2].val),
					stagecodeParts[0],
					stagecodeParts[1:],
				}
			}
		}
	case 55:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:336
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
		//line martian/syntax/grammar.y:344
		{
			{
				mmVAL.par_tuple = paramsTuple{true, mmDollar[4].params, mmDollar[5].params}
			}
		}
	case 57:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:346
		{
			{
				mmVAL.par_tuple = paramsTuple{true, mmDollar[3].params, mmDollar[4].params}
			}
		}
	case 58:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:351
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[3].bindings}
			}
		}
	case 59:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:356
		{
			{
				mmVAL.plretains = nil
			}
		}
	case 60:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:358
		{
			{
				mmVAL.plretains = &PipelineRetains{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[3].reflist}
			}
		}
	case 61:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:362
		{
			{
				mmVAL.reflist = nil
			}
		}
	case 62:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:364
		{
			{
				mmVAL.reflist = append(mmDollar[1].reflist, mmDollar[2].rexp)
			}
		}
	case 63:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:368
		{
			{
				mmVAL.calls = append(mmDollar[1].calls, mmDollar[2].call)
			}
		}
	case 64:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:370
		{
			{
				mmVAL.calls = []*CallStm{mmDollar[1].call}
			}
		}
	case 65:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line martian/syntax/grammar.y:375
		{
			{
				mmVAL.call = &CallStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].modifiers, mmDollar[3].val, mmDollar[3].val, mmDollar[5].bindings}
			}
		}
	case 66:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line martian/syntax/grammar.y:377
		{
			{
				mmVAL.call = &CallStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].modifiers, mmDollar[5].val, mmDollar[3].val, mmDollar[7].bindings}
			}
		}
	case 67:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:379
		{
			{
				mmDollar[1].call.Modifiers.Bindings = mmDollar[4].bindings
				mmVAL.call = mmDollar[1].call
			}
		}
	case 68:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:387
		{
			{
				mmVAL.modifiers = &Modifiers{}
			}
		}
	case 69:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:389
		{
			{
				mmVAL.modifiers.Local = true
			}
		}
	case 70:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:391
		{
			{
				mmVAL.modifiers.Preflight = true
			}
		}
	case 71:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:393
		{
			{
				mmVAL.modifiers.Volatile = true
			}
		}
	case 72:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:398
		{
			{
				mmVAL.bindings = &BindStms{NewAstNode(mmDollar[0].loc, mmDollar[0].locmap), []*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 73:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:400
		{
			{
				mmDollar[1].bindings.List = append(mmDollar[1].bindings.List, mmDollar[2].binding)
				mmVAL.bindings = mmDollar[1].bindings
			}
		}
	case 74:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:408
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].vexp, false, ""}
			}
		}
	case 75:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:410
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].vexp, false, ""}
			}
		}
	case 76:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:412
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].vexp, false, ""}
			}
		}
	case 77:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:414
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].rexp, false, ""}
			}
		}
	case 78:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line martian/syntax/grammar.y:418
		{
			{
				mmVAL.bindings = &BindStms{NewAstNode(mmDollar[0].loc, mmDollar[0].locmap), []*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 79:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:420
		{
			{
				mmDollar[1].bindings.List = append(mmDollar[1].bindings.List, mmDollar[2].binding)
				mmVAL.bindings = mmDollar[1].bindings
			}
		}
	case 80:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:428
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 81:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line martian/syntax/grammar.y:430
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 82:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line martian/syntax/grammar.y:432
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 83:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:437
		{
			{
				mmVAL.exps = append(mmDollar[1].exps, mmDollar[3].exp)
			}
		}
	case 84:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:439
		{
			{
				mmVAL.exps = []Exp{mmDollar[1].exp}
			}
		}
	case 85:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line martian/syntax/grammar.y:444
		{
			{
				mmDollar[1].kvpairs[unquote(mmDollar[3].val)] = mmDollar[5].exp
				mmVAL.kvpairs = mmDollar[1].kvpairs
			}
		}
	case 86:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:449
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmDollar[1].val): mmDollar[3].exp}
			}
		}
	case 87:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:454
		{
			{
				mmVAL.exp = mmDollar[1].vexp
			}
		}
	case 88:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:456
		{
			{
				mmVAL.exp = mmDollar[1].rexp
			}
		}
	case 89:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:460
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[2].exps}
			}
		}
	case 90:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:462
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[2].exps}
			}
		}
	case 91:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:464
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: []Exp{}}
			}
		}
	case 92:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line martian/syntax/grammar.y:466
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: map[string]interface{}{}}
			}
		}
	case 93:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:468
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: mmDollar[2].kvpairs}
			}
		}
	case 94:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line martian/syntax/grammar.y:470
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: mmDollar[2].kvpairs}
			}
		}
	case 95:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:472
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmDollar[1].val, 64)
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindFloat, Value: f}
			}
		}
	case 96:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:477
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmDollar[1].val, 0, 64)
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindInt, Value: i}
			}
		}
	case 97:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:482
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindString, Value: unquote(mmDollar[1].val)}
			}
		}
	case 99:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:485
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindNull, Value: nil}
			}
		}
	case 100:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:490
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindBool, Value: true}
			}
		}
	case 101:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:492
		{
			{
				mmVAL.vexp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindBool, Value: false}
			}
		}
	case 102:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:496
		{
			{
				mmVAL.rexp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindCall, mmDollar[1].val, mmDollar[3].val}
			}
		}
	case 103:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line martian/syntax/grammar.y:498
		{
			{
				mmVAL.rexp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindCall, mmDollar[1].val, "default"}
			}
		}
	case 104:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line martian/syntax/grammar.y:500
		{
			{
				mmVAL.rexp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindSelf, mmDollar[3].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

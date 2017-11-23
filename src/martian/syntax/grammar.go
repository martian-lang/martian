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
const SWEEP = 57363
const SPLIT = 57364
const USING = 57365
const SELF = 57366
const RETURN = 57367
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
	"FILETYPE",
	"STAGE",
	"PIPELINE",
	"CALL",
	"SWEEP",
	"SPLIT",
	"USING",
	"SELF",
	"RETURN",
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

//line src/martian/syntax/grammar.y:443

//line yacctab:1
var mmExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const mmPrivate = 57344

const mmLast = 261

var mmAct = [...]int{

	122, 37, 47, 150, 45, 130, 120, 4, 84, 100,
	13, 15, 172, 101, 62, 73, 74, 100, 161, 99,
	23, 101, 79, 29, 8, 11, 10, 7, 190, 87,
	79, 105, 88, 89, 38, 78, 104, 102, 103, 175,
	95, 50, 94, 78, 104, 102, 103, 8, 11, 10,
	7, 73, 74, 106, 30, 189, 191, 30, 115, 73,
	74, 106, 57, 81, 14, 117, 107, 100, 121, 116,
	129, 101, 111, 46, 77, 163, 83, 55, 100, 167,
	79, 96, 101, 72, 75, 76, 152, 5, 97, 57,
	109, 79, 110, 78, 104, 102, 103, 162, 131, 132,
	57, 26, 27, 28, 78, 104, 102, 103, 151, 73,
	74, 106, 25, 149, 111, 132, 127, 123, 134, 20,
	73, 74, 106, 100, 177, 135, 141, 101, 36, 68,
	63, 64, 66, 65, 67, 79, 79, 153, 49, 125,
	112, 157, 151, 160, 178, 179, 180, 164, 78, 78,
	104, 102, 103, 165, 132, 22, 21, 168, 170, 160,
	171, 20, 34, 61, 71, 73, 74, 106, 139, 17,
	188, 39, 182, 181, 59, 184, 137, 147, 138, 82,
	85, 114, 35, 41, 42, 43, 44, 61, 7, 61,
	7, 61, 144, 128, 8, 11, 10, 7, 158, 145,
	6, 156, 133, 159, 16, 155, 148, 142, 119, 58,
	143, 174, 32, 16, 31, 24, 187, 186, 185, 80,
	54, 53, 52, 51, 194, 193, 192, 183, 176, 173,
	166, 154, 140, 118, 93, 92, 91, 90, 69, 146,
	3, 1, 169, 12, 136, 126, 33, 19, 40, 56,
	108, 124, 98, 70, 113, 60, 48, 9, 18, 86,
	2,
}
var mmPact = [...]int{

	30, -1000, 7, 177, 146, -1000, -1000, -1000, 124, -1000,
	119, 118, 177, 146, -1000, 146, -1000, 202, 75, 16,
	-1000, 201, 199, 146, -1000, 149, -1000, -1000, -1000, -1000,
	91, -1000, -1000, 157, -1000, 36, -1000, 108, 108, -1000,
	-1000, 213, 212, 211, 210, 63, 196, 160, -1000, 82,
	132, -38, -38, -38, 111, -1000, -1000, 209, -1000, 164,
	-1000, 82, -1000, -1000, -1000, -1000, -1000, -1000, -1000, 13,
	166, -13, 228, -1000, -1000, 227, 226, 225, 1, -1,
	67, 52, 170, -1000, 103, 159, 20, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 32, 28, 224, 195, -1000, -1000,
	56, 101, -1000, -1000, -1000, -1000, -1000, -1000, 168, 146,
	61, 190, 116, 153, 155, 223, -1000, -1000, -1000, 112,
	198, -1000, -1000, -1000, 183, 231, 161, 146, 193, -1000,
	104, 77, -1000, -1000, 222, -1000, -1000, 192, 188, -1000,
	-1000, 189, 6, -1000, 59, -1000, 112, -1000, -1000, -1000,
	221, -1000, -1000, 70, -1000, -1000, -1000, 108, -2, 220,
	-1000, -1000, 203, -1000, -1000, 25, -1000, -1000, 219, 110,
	108, 158, 218, -1000, 112, -1000, -1000, -1000, 208, 207,
	206, 156, -1000, -1000, -1000, 15, -12, 18, -1000, 217,
	216, 215, -1000, -1000, -1000,
}
var mmPgo = [...]int{

	0, 260, 238, 14, 5, 259, 3, 258, 8, 200,
	257, 240, 256, 255, 1, 2, 254, 253, 0, 19,
	252, 31, 6, 251, 7, 250, 249, 248, 4, 246,
	245, 244, 242, 241,
}
var mmR1 = [...]int{

	0, 33, 33, 33, 33, 33, 33, 1, 1, 11,
	11, 9, 9, 9, 10, 31, 31, 32, 32, 32,
	32, 2, 2, 8, 8, 14, 14, 12, 12, 15,
	15, 13, 13, 13, 13, 13, 13, 17, 4, 6,
	3, 3, 3, 3, 3, 3, 3, 5, 5, 5,
	16, 16, 16, 30, 25, 25, 24, 24, 24, 7,
	7, 7, 7, 29, 29, 27, 27, 27, 27, 28,
	28, 26, 26, 26, 22, 22, 23, 23, 18, 18,
	20, 20, 20, 20, 20, 20, 20, 20, 20, 20,
	20, 21, 21, 19, 19, 19,
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
	1, 1, 1, 3, 1, 3,
}
var mmChk = [...]int{

	-1000, -33, -1, -11, -24, 57, -9, 20, 17, -10,
	19, 18, -11, -24, 57, -24, -9, 23, -7, -2,
	37, 37, 37, -24, 13, 37, 26, 27, 28, 7,
	41, 13, 13, -29, 13, 33, 37, -14, -14, 14,
	-27, 26, 27, 28, 29, -28, 37, -15, -12, 30,
	-15, 10, 10, 10, 10, 14, -26, 37, 13, 14,
	-13, 31, -3, 48, 49, 51, 50, 52, 47, -2,
	-17, 32, -21, 53, 54, -21, -21, -19, 37, 24,
	10, -28, 15, -3, -8, 14, -5, 42, 45, 46,
	9, 9, 9, 9, 41, 41, -18, 21, -20, -19,
	11, 15, 39, 40, 38, -21, 55, 14, -25, -24,
	-8, 11, 37, -16, 22, 38, 37, 37, 9, 13,
	-22, 12, -18, 16, -23, 38, -30, -24, 25, 9,
	-4, 37, 38, 12, -4, 9, -31, 23, 23, 13,
	9, -22, 9, 12, 9, 16, 8, 16, 13, 9,
	-6, 38, 9, -4, 9, 13, 13, -14, 9, 14,
	-18, 12, 38, 16, -18, -28, 9, 9, -6, -32,
	-14, -15, 14, 9, 8, 14, 9, 14, 34, 35,
	36, -15, 14, 9, -18, 10, 10, 10, 14, 40,
	40, 38, 9, 9, 9,
}
var mmDef = [...]int{

	0, -2, 0, 4, 6, 8, 10, 59, 0, 12,
	0, 0, 1, 3, 7, 5, 9, 0, 0, 0,
	22, 0, 0, 2, 63, 0, 60, 61, 62, 11,
	0, 25, 25, 0, 69, 0, 21, 29, 29, 58,
	64, 0, 0, 0, 0, 0, 0, 0, 26, 0,
	0, 0, 0, 0, 0, 56, 70, 0, 69, 0,
	30, 0, 23, 40, 41, 42, 43, 44, 45, 46,
	0, 0, 0, 91, 92, 0, 0, 0, 94, 0,
	0, 0, 0, 23, 0, 50, 0, 47, 48, 49,
	65, 66, 67, 68, 0, 0, 0, 0, 78, 79,
	0, 0, 86, 87, 88, 89, 90, 57, 0, 55,
	0, 0, 0, 15, 0, 0, 93, 95, 71, 0,
	0, 82, 75, 83, 0, 0, 0, 54, 0, 31,
	0, 0, 38, 24, 0, 28, 14, 0, 0, 25,
	37, 0, 0, 80, 0, 84, 0, 13, 69, 32,
	0, 39, 34, 0, 27, 17, 25, 29, 0, 0,
	74, 81, 0, 85, 77, 0, 33, 35, 0, 0,
	29, 0, 0, 73, 0, 53, 36, 16, 0, 0,
	0, 0, 52, 72, 76, 0, 0, 0, 51, 0,
	0, 0, 18, 19, 20,
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
		//line src/martian/syntax/grammar.y:82
		{
			{
				global := NewAst(mmDollar[2].decs, nil)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 2:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:88
		{
			{
				global := NewAst(mmDollar[2].decs, mmDollar[3].call)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 3:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:94
		{
			{
				global := NewAst([]Dec{}, mmDollar[2].call)
				global.preprocess = mmDollar[1].pre_dir
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 4:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:100
		{
			{
				global := NewAst(mmDollar[1].decs, nil)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 5:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:105
		{
			{
				global := NewAst(mmDollar[1].decs, mmDollar[2].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 6:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:110
		{
			{
				global := NewAst([]Dec{}, mmDollar[1].call)
				mmlex.(*mmLexInfo).global = global
			}
		}
	case 7:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:118
		{
			{
				mmVAL.pre_dir = append(mmDollar[1].pre_dir, &preprocessorDirective{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val})
			}
		}
	case 8:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:120
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
		//line src/martian/syntax/grammar.y:130
		{
			{
				mmVAL.decs = append(mmDollar[1].decs, mmDollar[2].dec)
			}
		}
	case 10:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:132
		{
			{
				mmVAL.decs = []Dec{mmDollar[1].dec}
			}
		}
	case 11:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:137
		{
			{
				mmVAL.dec = &UserType{NewAstNode(mmDollar[2].loc, mmDollar[2].locmap), mmDollar[2].val}
			}
		}
	case 12:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:139
		{
			{
				mmVAL.dec = mmDollar[1].dec
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
	case 38:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:260
		{
			{
				mmVAL.val = mmDollar[1].val
			}
		}
	case 39:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:265
		{
			{
				mmVAL.val = mmDollar[1].val
			}
		}
	case 50:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:288
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
		//line src/martian/syntax/grammar.y:296
		{
			{
				mmVAL.par_tuple = paramsTuple{true, mmDollar[4].params, mmDollar[5].params}
			}
		}
	case 52:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:298
		{
			{
				mmVAL.par_tuple = paramsTuple{true, mmDollar[3].params, mmDollar[4].params}
			}
		}
	case 53:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:303
		{
			{
				mmVAL.retstm = &ReturnStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[3].bindings}
			}
		}
	case 54:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:308
		{
			{
				mmVAL.calls = append(mmDollar[1].calls, mmDollar[2].call)
			}
		}
	case 55:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:310
		{
			{
				mmVAL.calls = []*CallStm{mmDollar[1].call}
			}
		}
	case 56:
		mmDollar = mmS[mmpt-6 : mmpt+1]
		//line src/martian/syntax/grammar.y:315
		{
			{
				mmVAL.call = &CallStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].modifiers, mmDollar[3].val, mmDollar[3].val, mmDollar[5].bindings}
			}
		}
	case 57:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line src/martian/syntax/grammar.y:317
		{
			{
				mmVAL.call = &CallStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[2].modifiers, mmDollar[5].val, mmDollar[3].val, mmDollar[7].bindings}
			}
		}
	case 58:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:319
		{
			{
				mmDollar[1].call.Modifiers.Bindings = mmDollar[4].bindings
				mmVAL.call = mmDollar[1].call
			}
		}
	case 59:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:327
		{
			{
				mmVAL.modifiers = &Modifiers{}
			}
		}
	case 60:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:329
		{
			{
				mmVAL.modifiers.Local = true
			}
		}
	case 61:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:331
		{
			{
				mmVAL.modifiers.Preflight = true
			}
		}
	case 62:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:333
		{
			{
				mmVAL.modifiers.Volatile = true
			}
		}
	case 63:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:338
		{
			{
				mmVAL.bindings = &BindStms{NewAstNode(mmDollar[0].loc, mmDollar[0].locmap), []*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 64:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:340
		{
			{
				mmDollar[1].bindings.List = append(mmDollar[1].bindings.List, mmDollar[2].binding)
				mmVAL.bindings = mmDollar[1].bindings
			}
		}
	case 65:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:348
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 66:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:350
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 67:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:352
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 68:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:354
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 69:
		mmDollar = mmS[mmpt-0 : mmpt+1]
		//line src/martian/syntax/grammar.y:358
		{
			{
				mmVAL.bindings = &BindStms{NewAstNode(mmDollar[0].loc, mmDollar[0].locmap), []*BindStm{}, map[string]*BindStm{}}
			}
		}
	case 70:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:360
		{
			{
				mmDollar[1].bindings.List = append(mmDollar[1].bindings.List, mmDollar[2].binding)
				mmVAL.bindings = mmDollar[1].bindings
			}
		}
	case 71:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:368
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, mmDollar[3].exp, false, ""}
			}
		}
	case 72:
		mmDollar = mmS[mmpt-8 : mmpt+1]
		//line src/martian/syntax/grammar.y:370
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 73:
		mmDollar = mmS[mmpt-7 : mmpt+1]
		//line src/martian/syntax/grammar.y:372
		{
			{
				mmVAL.binding = &BindStm{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), mmDollar[1].val, &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[5].exps}, true, ""}
			}
		}
	case 74:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:377
		{
			{
				mmVAL.exps = append(mmDollar[1].exps, mmDollar[3].exp)
			}
		}
	case 75:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:379
		{
			{
				mmVAL.exps = []Exp{mmDollar[1].exp}
			}
		}
	case 76:
		mmDollar = mmS[mmpt-5 : mmpt+1]
		//line src/martian/syntax/grammar.y:384
		{
			{
				mmDollar[1].kvpairs[unquote(mmDollar[3].val)] = mmDollar[5].exp
				mmVAL.kvpairs = mmDollar[1].kvpairs
			}
		}
	case 77:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:389
		{
			{
				mmVAL.kvpairs = map[string]Exp{unquote(mmDollar[1].val): mmDollar[3].exp}
			}
		}
	case 78:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:394
		{
			{
				mmVAL.exp = mmDollar[1].exp
			}
		}
	case 79:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:396
		{
			{
				mmVAL.exp = mmDollar[1].exp
			}
		}
	case 80:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:400
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[2].exps}
			}
		}
	case 81:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:402
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: mmDollar[2].exps}
			}
		}
	case 82:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:404
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindArray, Value: []Exp{}}
			}
		}
	case 83:
		mmDollar = mmS[mmpt-2 : mmpt+1]
		//line src/martian/syntax/grammar.y:406
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: map[string]interface{}{}}
			}
		}
	case 84:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:408
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: mmDollar[2].kvpairs}
			}
		}
	case 85:
		mmDollar = mmS[mmpt-4 : mmpt+1]
		//line src/martian/syntax/grammar.y:410
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindMap, Value: mmDollar[2].kvpairs}
			}
		}
	case 86:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:412
		{
			{ // Lexer guarantees parseable float strings.
				f, _ := strconv.ParseFloat(mmDollar[1].val, 64)
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindFloat, Value: f}
			}
		}
	case 87:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:417
		{
			{ // Lexer guarantees parseable int strings.
				i, _ := strconv.ParseInt(mmDollar[1].val, 0, 64)
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindInt, Value: i}
			}
		}
	case 88:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:422
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindString, Value: unquote(mmDollar[1].val)}
			}
		}
	case 89:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:424
		{
			{
				mmVAL.exp = mmDollar[1].exp
			}
		}
	case 90:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:426
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindNull, Value: nil}
			}
		}
	case 91:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:431
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindBool, Value: true}
			}
		}
	case 92:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:433
		{
			{
				mmVAL.exp = &ValExp{Node: NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), Kind: KindBool, Value: false}
			}
		}
	case 93:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:437
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindCall, mmDollar[1].val, mmDollar[3].val}
			}
		}
	case 94:
		mmDollar = mmS[mmpt-1 : mmpt+1]
		//line src/martian/syntax/grammar.y:439
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindCall, mmDollar[1].val, "default"}
			}
		}
	case 95:
		mmDollar = mmS[mmpt-3 : mmpt+1]
		//line src/martian/syntax/grammar.y:441
		{
			{
				mmVAL.exp = &RefExp{NewAstNode(mmDollar[1].loc, mmDollar[1].locmap), KindSelf, mmDollar[3].val, ""}
			}
		}
	}
	goto mmstack /* stack new state and value */
}

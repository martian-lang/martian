// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

// Martian tokenizer. Simple regexp-based implementation.

package syntax

import (
	"regexp"
	re_syntax "regexp/syntax"
	"unicode"
	"unicode/utf8"
)

const defaultOutName = "default"

// re matches text to produce token.
type rule func([]byte) ([]byte, int)

func bytesPrefixString(b []byte, s string) []byte {
	if len(b) < len(s) {
		return nil
	} else if len(b) > len(s) && re_syntax.IsWordChar(rune(b[len(s)])) {
		return nil
	}
	for i, r := range s {
		if b[i] != byte(r) {
			return nil
		}
	}
	return b[:len(s):len(s)]
}

// leadingSpace returns the bytes from the beginning of b which are unicode
// whitespace characters.
func leadingSpace(b []byte) ([]byte, int) {
	i := 0
	for ; i < len(b); i++ {
		if cb := b[i]; cb < utf8.RuneSelf {
			// ASCII fastpath
			switch cb {
			case '\t', '\n', '\v', '\f', '\r', ' ':
			default:
				return b[:i:i], SKIP
			}
		} else {
			r, s := utf8.DecodeRune(b[i:])
			if r == utf8.RuneError || !unicode.IsSpace(r) {
				return b[:i:i], SKIP
			}
			i += s - 1
		}
	}
	return b[:i:i], SKIP
}

// Python-style comments, capture until end of line.
func tokCommentRule(b []byte) ([]byte, int) {
	if len(b) == 0 {
		return nil, 0
	}
	if b[0] != '#' {
		return nil, INVALID
	}
	i := 1
	for i < len(b) {
		if r, s := utf8.DecodeRune(b[i:]); r == utf8.RuneError {
			return b[:i:i], COMMENT
		} else {
			i += s
			if r == '\n' {
				return b[:i:i], COMMENT
			}
		}
	}
	return b[:i:i], COMMENT
}

// Match keywords and punctuation based on the next character in the source.
func keywordToken(b []byte) ([]byte, int) {
	if len(b) > 0 {
		r := b[0]
		switch r {
		case '=', ':', ';', '.', ',', '(', ')', '{', '}', '[', ']', '<', '>':
			// Puctuation marks
			return b[:1:1], int(r)
		case '"':
			// No token other than a literal string starts with "
			return tokStringRule(b)
		case '#':
			// Comment token
			return tokCommentRule(b)
		case '\t', '\n', '\v', '\f', '\r', ' ':
			return leadingSpace(b)
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
			// Numeric tokens
			if v, id := tokFloatRule(b); len(v) > 0 {
				return v, id
			}
			return tokIntRule(b)
		case '_':
			return tokIdRule(b)

		// keywords
		case '@':
			// No token other than @include starts with @
			return bytesPrefixString(b, `@include`), INCLUDE_DIRECTIVE
		// keywords where no other keyword shares the same first character:
		case 'a':
			return bytesPrefixString(b, `as`), AS
		case 'b':
			return bytesPrefixString(b, KindBool), BOOL
		case 'c':
			if v := bytesPrefixString(b, `call`); len(v) > 0 {
				return v, CALL
			}
			return bytesPrefixString(b, abr_compiled), COMPILED
		case 'd':
			if v := bytesPrefixString(b, defaultOutName); len(v) > 0 {
				return v, DEFAULT
			}
			return bytesPrefixString(b, disabled), DISABLED
		case 'e':
			return bytesPrefixString(b, abr_exec), EXEC
		case 'f':
			if v := bytesPrefixString(b, `false`); len(v) > 0 {
				return v, FALSE
			}
			if v := bytesPrefixString(b, `filetype`); len(v) > 0 {
				return v, FILETYPE
			}
			if v := bytesPrefixString(b, KindFloat); len(v) > 0 {
				return v, FLOAT
			}
		case 'i':
			if v := bytesPrefixString(b, `in`); len(v) > 0 {
				return v, IN
			}
			if v := bytesPrefixString(b, KindInt); len(v) > 0 {
				return v, INT
			}
		case 'l':
			return bytesPrefixString(b, local), LOCAL
		case 'm':
			if v := bytesPrefixString(b, KindMap); len(v) > 0 {
				return v, MAP
			}
			return tokMemRule(b)
		case 'n':
			return bytesPrefixString(b, KindNull), NULL
		case 'o':
			return bytesPrefixString(b, `out`), OUT
		case 'p':
			if v := bytesPrefixString(b, KindPath); len(v) > 0 {
				return v, PATH
			}
			if v := bytesPrefixString(b, `pipeline`); len(v) > 0 {
				return v, PIPELINE
			}
			if v := bytesPrefixString(b, preflight); len(v) > 0 {
				return v, PREFLIGHT
			}
			return bytesPrefixString(b, abr_python), PY
		case 'r':
			if v := bytesPrefixString(b, `retain`); len(v) > 0 {
				return v, RETAIN
			}
			return bytesPrefixString(b, `return`), RETURN
		case 's':
			if v := bytesPrefixString(b, KindSelf); len(v) > 0 {
				return v, SELF
			}
			if v := bytesPrefixString(b, `special`); len(v) > 0 {
				return v, SPECIAL
			}
			if v := bytesPrefixString(b, KindSplit); len(v) > 0 {
				return v, SPLIT
			}
			if v := bytesPrefixString(b, `src`); len(v) > 0 {
				return v, SRC
			}
			if v := bytesPrefixString(b, `stage`); len(v) > 0 {
				return v, STAGE
			}
			if v := bytesPrefixString(b, strict); len(v) > 0 {
				return v, STRICT
			}
			if v := bytesPrefixString(b, KindString); len(v) > 0 {
				return v, STRING
			}
			return bytesPrefixString(b, `struct`), STRUCT
		case 't':
			if v := bytesPrefixString(b, `threads`); len(v) > 0 {
				return v, THREADS
			}
			return bytesPrefixString(b, `true`), TRUE
		case 'u':
			return bytesPrefixString(b, `using`), USING
		case 'v':
			if v := bytesPrefixString(b, volatile); len(v) > 0 {
				return v, VOLATILE
			}
			return tokVMemRule(b)
		}
		if r > utf8.RuneSelf {
			// Non-ASCII space
			return leadingSpace(b)
		}
	}
	return nil, 0
}

func regexpRule(exp string, tokid int) rule {
	re := regexp.MustCompile(exp)
	return func(b []byte) ([]byte, int) {
		return re.Find(b), tokid
	}
}

var (
	// double-quoted strings with escaping.
	tokStringRule = regexpRule(
		`^"(?:[^\\"]|`+ // non-escape sequences
			`\\(?:`+
			`[abfnrtv\\"]|`+ // standard escapes
			`[0-7]{3}|`+ // octal-encoded ascii
			`x[[:xdigit:]]{2}|`+ // one-byte unicode
			`u[[:xdigit:]]{4}|`+ // two-byte unicode
			`U[[:xdigit:]]{8}`+ // four-byte unicode
			`))*"`,
		LITSTRING,
	)
	tokFloatRule = regexpRule(`^-?\d+(:?(?:\.\d+)?[eE][+-]?|\.)\d+\b`, NUM_FLOAT)
	tokIntRule   = regexpRule(`^-?0*\d{1,19}\b`, NUM_INT)
	tokMemRule   = regexpRule(`^mem_?gb\b`, MEM_GB)
	tokVMemRule  = regexpRule(`^vmem_?gb\b`, VMEM_GB)

	// Identifiers for filetypes, stages, etc.
	tokIdRule = regexpRule(`^_?[[:alpha:]]\w*\b`, ID)
)

func nextToken(head []byte) (int, []byte) {
	val, tokid := keywordToken(head)
	if len(val) > 0 {
		return tokid, val
	}
	val, tokid = tokIdRule(head)
	if len(val) > 0 {
		return tokid, val
	}
	return INVALID, nil
}

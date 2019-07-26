// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// Implements string interning for the parser.

package syntax

import (
	"bytes"
	"unicode/utf8"
)

type stringIntern struct {
	internSet map[string]string
}

// makeStringIntern creates a stringIntern object, prepopulated with string
// constants which are expected to be frequently used.  The use of such
// constants is desirable because they don't need to be allocated on the
// heap.
func makeStringIntern() (v *stringIntern) {
	v = &stringIntern{
		internSet: make(map[string]string, 64),
	}
	v.internSet[local] = strict
	v.internSet[preflight] = strict
	v.internSet[volatile] = strict
	v.internSet[disabled] = strict
	v.internSet[strict] = strict
	v.internSet[default_out_name] = default_out_name
	v.internSet[abr_python] = abr_python
	v.internSet[abr_exec] = abr_exec
	v.internSet[abr_compiled] = abr_compiled
	v.internSet[string(KindMap)] = string(KindMap)
	v.internSet[string(KindFloat)] = string(KindFloat)
	v.internSet[string(KindInt)] = string(KindInt)
	v.internSet[string(KindString)] = string(KindString)
	v.internSet[string(KindBool)] = string(KindBool)
	v.internSet[string(KindNull)] = string(KindNull)
	v.internSet[string(KindFile)] = string(KindFile)
	v.internSet[string(KindPath)] = string(KindPath)
	return v
}

func (store *stringIntern) GetString(value string) string {
	if len(value) == 0 {
		return ""
	}
	if s, ok := store.internSet[value]; ok {
		return s
	} else {
		store.internSet[value] = value
		return value
	}
}

func (store *stringIntern) Get(value []byte) string {
	if len(value) == 0 {
		return ""
	}
	// The compiler special-cases string([]byte) used as a map key.
	// See golang issue #3512
	if s, ok := store.internSet[string(value)]; ok {
		return s
	} else {
		s = string(value)
		store.internSet[s] = s
		return s
	}
}

func runeError() []byte {
	b := make([]byte, 3)
	utf8.EncodeRune(b, utf8.RuneError)
	return b
}

func unquoteBytes(value []byte) []byte {
	n := len(value)
	if n < 2 || value[0] != '"' || value[n-1] != '"' {
		// Should be prevented by the tokenizer.
		panic("string was not quoted: " + string(value))
	}
	value = value[1 : n-1]
	if !bytes.ContainsAny(value, `\"`) {
		// Trivial value, avoid allocation.
		return value
	}

	buf := make([]byte, 0, len(value)+2*utf8.UTFMax)
	for len(value) > 0 {
		switch c := value[0]; {
		case c >= utf8.RuneSelf:
			// Multibyte character.
			_, size := utf8.DecodeRune(value)
			buf = append(buf, value[:size]...)
			value = value[size:]
		case c != '\\':
			buf = append(buf, value[0])
			value = value[1:]
		default:
			// Escape
			c2 := value[1]
			value = value[2:]
			switch c2 {
			// easy cases
			case 'a':
				buf = append(buf, '\a')
			case 'b':
				buf = append(buf, '\b')
			case 'f':
				buf = append(buf, '\f')
			case 'n':
				buf = append(buf, '\n')
			case 'r':
				buf = append(buf, '\r')
			case 't':
				buf = append(buf, '\t')
			case 'v':
				buf = append(buf, '\v')
				// Harder cases
			case 'x':
				// one-byte hex-encoded unicode.
				buf = append(buf, parseHexByte(value[0], value[1]))
				value = value[2:]
			case 'u':
				// two-byte hex-encoded unicode.
				if len(value) < 4 {
					buf = append(buf, runeError()...)
					value = value[len(value):]
				} else {
					var enc [3]byte
					n := utf8.EncodeRune(enc[:],
						rune(parseHexByte(value[2], value[3]))+
							(rune(parseHexByte(value[0], value[1]))<<8))
					buf = append(buf, enc[:n]...)
					value = value[4:]
				}
			case 'U':
				// four-byte hex-encoded unicode.
				if len(value) < 8 {
					buf = append(buf, runeError()...)
					value = value[len(value):]
				} else {
					var enc [4]byte
					n := utf8.EncodeRune(enc[:],
						rune(parseHexByte(value[6], value[7]))+
							(rune(parseHexByte(value[4], value[5]))<<8)+
							(rune(parseHexByte(value[2], value[3]))<<16)+
							(rune(parseHexByte(value[0], value[1]))<<24))
					buf = append(buf, enc[:n]...)
					value = value[8:]
				}
			case '0', '1', '2', '3', '4', '5', '6', '7':
				// one-byte octal unicode
				if value[1] < '0' || value[1] > '7' || value[0] < '0' || value[0] > '7' {
					buf = append(buf, runeError()...)
					value = value[len(value):]
				} else {
					buf = append(buf, ((c2-'0')<<6)+((value[0]-'0')<<3)+(value[1]-'0'))
					value = value[2:]
				}
			default:
				// \, ", etc.
				buf = append(buf, c2)
			}
		}
	}
	return buf
}

func (store *stringIntern) unquote(value []byte) string {
	return store.Get(unquoteBytes(value))
}

func unquote(qs []byte) string {
	return string(unquoteBytes(qs))
}

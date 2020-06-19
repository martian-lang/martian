// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package core

import "unicode/utf8"

// Append a quoted form of the string s to the given buffer, and return it.
//
// Handles escaping of \, ", and $, as well as standard escape sequences and
// octal characters.
func appendShellSafeQuote(buf []byte, s string) []byte {
	buf = append(buf, '"')
	for width := 0; len(s) > 0; s = s[width:] {
		r := rune(s[0])
		width = 1
		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRuneInString(s)
		}
		if width == 1 {
			switch r {
			case utf8.RuneError:
				buf = append(buf, '\\')
				buf = append(buf, '0'+s[0]>>6)
				buf = append(buf, '0'+((s[0]>>3)&7))
				buf = append(buf, '0'+(s[0]&7))
			// Stuff which should be escaped
			case '\\':
				buf = append(buf, `\\`...)
			case '"':
				buf = append(buf, `\"`...)
			case '$':
				buf = append(buf, `\$`...)
			// Escape sequences which bash knows about
			case '\a':
				buf = append(buf, `\a`...)
			case '\b':
				buf = append(buf, `\b`...)
			case '\n':
				buf = append(buf, `\n`...)
			case '\r':
				buf = append(buf, `\r`...)
			case '\t':
				buf = append(buf, `\t`...)
			case '\v':
				buf = append(buf, `\v`...)
			default:
				buf = append(buf, byte(r))
			}
		} else {
			var runeTmp [utf8.UTFMax]byte
			n := utf8.EncodeRune(runeTmp[:], r)
			buf = append(buf, runeTmp[:n]...)
		}
	}
	return append(buf, '"')
}

func shellSafeQuote(s string) string {
	buf := make([]byte, 0, len(s)+8)
	return string(appendShellSafeQuote(buf, s))
}

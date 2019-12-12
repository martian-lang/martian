// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// FormatExp returns an mro-formatted representation of an expression.
//
// Each line of formatted text will begin with the given prefix.
func FormatExp(exp Exp, prefix string) string {
	var buf strings.Builder
	exp.format(&buf, prefix)
	return buf.String()
}

func (e *ArrayExp) format(w stringWriter, prefix string) {
	if e.Value == nil {
		mustWriteString(w, "null")
		return
	}
	values := e.Value
	if len(values) == 0 {
		mustWriteString(w, "[]")
	} else if len(values) == 1 {
		// Place single-element arrays on a single line.
		mustWriteRune(w, '[')
		values[0].format(w, prefix)
		mustWriteRune(w, ']')
	} else {
		mustWriteString(w, "[\n")
		vindent := prefix + INDENT
		for _, val := range values {
			mustWriteString(w, vindent)
			val.format(w, vindent)
			mustWriteString(w, ",\n")
		}
		mustWriteString(w, prefix)
		mustWriteRune(w, ']')
	}
}
func (e *ArrayExp) GoString() string {
	if e == nil || e.Value == nil {
		return "null"
	}
	if len(e.Value) == 0 {
		return "[]"
	}
	var buf strings.Builder
	if _, err := buf.WriteRune('['); err != nil {
		panic(err)
	}
	for i, v := range e.Value {
		if i < 2 || i >= len(e.Value)-2 {
			if i != 0 {
				if _, err := buf.WriteRune(','); err != nil {
					panic(err)
				}
			}
			if v == nil {
				if _, err := buf.WriteString("null"); err != nil {
					panic(err)
				}
			} else {
				if _, err := buf.WriteString(v.GoString()); err != nil {
					panic(err)
				}
			}
		} else if i == 2 {
			if _, err := buf.WriteString(",..."); err != nil {
				panic(err)
			}
		}
	}
	if _, err := buf.WriteRune(']'); err != nil {
		panic(err)
	}
	return buf.String()
}

func (e *SplitExp) format(w stringWriter, prefix string) {
	if e.Value == nil {
		mustWriteString(w, "null")
		return
	}
	mustWriteString(w, "split ")
	e.Value.format(w, prefix)
}
func (e *SplitExp) GoString() string {
	if e == nil || e.Value == nil {
		return "null"
	}
	return "split " + e.Value.GoString()
}

func (e *MapExp) format(w stringWriter, prefix string) {
	if e.Value == nil {
		mustWriteString(w, "null")
	}
	if len(e.Value) > 0 {
		mustWriteString(w, "{\n")
		vindent := prefix + INDENT
		keys := make([]string, 0, len(e.Value))
		maxKeyLen := 0
		for key, val := range e.Value {
			keys = append(keys, key)
			if e.Kind == KindStruct {
				switch val.(type) {
				case *ArrayExp, *MapExp:
				default:
					if len(key) > maxKeyLen {
						maxKeyLen = len(key)
					}
				}
			}
		}
		sort.Strings(keys)
		for _, key := range keys {
			mustWriteString(w, vindent)
			if e.Kind != KindStruct {
				quoteString(w, key)
			} else {
				mustWriteString(w, key)
			}
			mustWriteString(w, `: `)
			v := e.Value[key]
			if e.Kind == KindStruct {
				switch v.(type) {
				case *ArrayExp, *MapExp:
				default:
					for i := len(key); i < maxKeyLen; i++ {
						mustWriteRune(w, ' ')
					}
				}
			}
			v.format(w, vindent)
			mustWriteString(w, ",\n")
		}
		mustWriteString(w, prefix)
		mustWriteRune(w, '}')
	} else {
		mustWriteString(w, "{}")
	}
}

func (e *MapExp) GoString() string {
	if e == nil || e.Value == nil {
		return "null"
	}
	if len(e.Value) == 0 {
		return "{}"
	}
	var buf strings.Builder
	if _, err := buf.WriteRune('{'); err != nil {
		panic(err)
	}
	keys := make([]string, 0, len(e.Value))
	for k := range e.Value {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		if i < 2 || i >= len(e.Value)-2 {
			if i != 0 {
				if _, err := buf.WriteRune(','); err != nil {
					panic(err)
				}
			}
			if e.Kind == KindMap {
				if _, err := buf.WriteRune('"'); err != nil {
					panic(err)
				}
			}
			if _, err := buf.WriteString(k); err != nil {
				panic(err)
			}
			if e.Kind == KindMap {
				if _, err := buf.WriteRune('"'); err != nil {
					panic(err)
				}
			}
			if _, err := buf.WriteRune(':'); err != nil {
				panic(err)
			}
			v := e.Value[k]
			if v == nil {
				if _, err := buf.WriteString("null"); err != nil {
					panic(err)
				}
			} else {
				if _, err := buf.WriteString(v.GoString()); err != nil {
					panic(err)
				}
			}
		} else if i == 2 {
			if _, err := buf.WriteString(",..."); err != nil {
				panic(err)
			}
		}
	}
	if _, err := buf.WriteRune('}'); err != nil {
		panic(err)
	}
	return buf.String()
}

func (e *IntExp) format(w stringWriter, _ string) {
	mustWriteString(w, strconv.FormatInt(e.Value, 10))
}

func (e *IntExp) GoString() string {
	if e == nil {
		return "null"
	}
	return strconv.FormatInt(e.Value, 10)
}
func (e *IntExp) String() string {
	return e.GoString()
}

func (e *FloatExp) format(w stringWriter, _ string) {
	var buf [68]byte
	mustWrite(w, strconv.AppendFloat(buf[:0], e.Value, 'g', -1, 64))
}

func (e *FloatExp) GoString() string {
	if e == nil {
		return "null"
	}
	return strconv.FormatFloat(e.Value, 'g', -1, 64)
}
func (e *FloatExp) String() string {
	return e.GoString()
}

func (e *StringExp) format(w stringWriter, _ string) {
	quoteString(w, e.Value)
}

func (e *StringExp) GoString() string {
	if e == nil {
		return "null"
	}
	if len(e.Value) < 24 {
		return `"` + e.Value + `"`
	}
	buf := make([]byte, 1, 21)
	buf[0] = '"'
	buf = append(buf, e.Value[:8]...)
	buf = append(buf, "..."...)
	buf = append(buf, e.Value[len(e.Value)-8:]...)
	buf = append(buf, '"')
	return string(buf)
}
func (e *StringExp) String() string {
	if e == nil {
		return "null"
	}
	return e.Value
}
func (e *BoolExp) format(w stringWriter, _ string) {
	mustWriteString(w, strconv.FormatBool(e.Value))
}

func (e *BoolExp) GoString() string {
	if e == nil {
		return "null"
	}
	return strconv.FormatBool(e.Value)
}
func (e *BoolExp) String() string {
	return e.GoString()
}
func (e *NullExp) format(w stringWriter, _ string) {
	mustWriteString(w, "null")
}

func (e *NullExp) GoString() string {
	return "null"
}
func (e *NullExp) String() string {
	return e.GoString()
}

func (e *RefExp) format(w stringWriter, prefix string) {
	if e.Kind == KindCall {
		mustWriteString(w, e.Id)
	} else {
		if e.Id == "" {
			mustWriteString(w, string(KindSelf))
			return
		}
		mustWriteString(w, "self.")
		mustWriteString(w, e.Id)
	}
	if e.OutputId != "" {
		mustWriteRune(w, '.')
		mustWriteString(w, e.OutputId)
	}
}

func (self *RefExp) GoString() string {
	if self == nil {
		return "null"
	}
	if self.Kind == KindSelf {
		if self.Id == "" {
			return string(KindSelf)
		}
		if self.OutputId == "" {
			return "self." + self.Id
		}
		var buf strings.Builder
		buf.Grow(len("self..") + len(self.Id) + len(self.OutputId))
		buf.WriteString("self.")
		buf.WriteString(self.Id)
		buf.WriteRune('.')
		buf.WriteString(self.OutputId)
		return buf.String()
	}
	if self.OutputId == "" {
		return self.Id
	}
	return self.Id + "." + self.OutputId
}

// QuoteString writes a string, quoted and escaped as json.
//
// The reason we don't just use json.Marshal here is because the default
// encoder html-escapes strings, and disabling that by using json.Encoder
// puts carriage returns at the end of the string, which is also bad for
// this use case.  Plus this way we can bypass a lot of reflection junk.
//
// This method is mostly copy/pasted from unexported go standard library
// json encoder implementation (see
// https://github.com/golang/go/blob/release-branch.go1.11/src/encoding/json/encode.go#L884)
func quoteString(w stringWriter, s string) {
	mustWriteByte(w, '"')
	const hex = "0123456789abcdef"
	start := 0
	for i := 0; i < len(s); {
		// Single-byte code points.
		if b := s[i]; b < utf8.RuneSelf {
			if b >= ' ' && b != '"' && b != '\\' {
				i++
				continue
			}
			if start < i {
				mustWriteString(w, s[start:i])
			}
			switch b {
			case '\\', '"':
				mustWriteByte(w, '\\')
				mustWriteByte(w, b)
			case '\n':
				mustWriteByte(w, '\\')
				mustWriteByte(w, 'n')
			case '\r':
				mustWriteByte(w, '\\')
				mustWriteByte(w, 'r')
			case '\t':
				mustWriteByte(w, '\\')
				mustWriteByte(w, 't')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				mustWriteString(w, `\u00`)
				mustWriteByte(w, hex[b>>4])
				mustWriteByte(w, hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		// Multi-byte code points.
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			// Transform invalid code points into unicode
			// "replacement character".
			if start < i {
				mustWriteString(w, s[start:i])
			}
			mustWriteString(w, `\ufffd`)
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				mustWriteString(w, s[start:i])
			}
			mustWriteString(w, `\u202`)
			mustWriteByte(w, hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		mustWriteString(w, s[start:])
	}
	mustWriteByte(w, '"')
}

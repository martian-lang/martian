// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"bytes"
	"encoding/json"
	"testing"
	"testing/quick"
)

func TestStringIntern(t *testing.T) {
	inter := makeStringIntern()
	// Use a large string key to avoid small-allocation coalescing.
	const keyString = `10000000000000000000000000000000000000000000000000000000000`
	keyBytes := []byte(keyString)
	inter.GetString(keyString)
	if n := testing.AllocsPerRun(100, func() {
		if y := inter.GetString(keyString); y != keyString {
			t.Errorf("Expected "+keyString+", got %s", y)
		}
	}); n != 0 {
		t.Errorf("String key lookup AllocsPerRun = %f, want 0", n)
	}
	if n := testing.AllocsPerRun(100, func() {
		if y := inter.Get(keyBytes); y != keyString {
			t.Errorf("Expected "+keyString+" from bytes, got %s", y)
		}
	}); n != 0 {
		t.Errorf("Bytes key lookup AllocsPerRun = %f, want 0", n)
	}
}

func TestUnquote(t *testing.T) {
	check := func(t *testing.T, input, expect string) {
		t.Helper()
		if s := unquote([]byte(input)); s != expect {
			t.Errorf("Expected: %q, got %q",
				expect, s)
		}
	}
	check(t,
		`"\"hey\" is\\\n\tfor \U0001f40es"`,
		"\"hey\" is\\\n\tfor \U0001f40es")
	check(t,
		`"\xf2Y\xbb\x8a,\xd0(\xf0\xff=\x8c\xbd"`,
		"\xf2Y\xbb\x8a,\xd0(\xf0\xff=\x8c\xbd")
	check(t, `"multibyte \"ဤ\" character"`, "multibyte \"\xe1\x80\xa4\" character")
	check(t, `"Octal is \167eird"`, "Octal is weird")
	check(t, `"Hex is \x6eormal"`, "Hex is normal")
	check(t, `"Hex is \x6Eormal"`, "Hex is normal")
	check(t, `"Hex is \u0146ormal"`, "Hex is \u0146ormal")
	check(t, `"We căn use anỿ valid utf-8 ☺"`, "We căn use anỿ valid utf-8 ☺")
	check(t, `"Case sensitivity is \U0001f4A9"`, "Case sensitivity is \U0001f4A9")
	check(t, `"Control\a\b\f\n\r\t\v \u2029 characters"`, "Control\a\b\f\n\r\t\v \u2029 characters")
	check(t, `"Invalid \u123"`, "Invalid \ufffd")
}

// Fuzz test for unquote.
func TestUnquoteFuzz(t *testing.T) {
	t.Parallel()
	if err := quick.CheckEqual(func(s string) string {
		return s
	}, func(s string) string {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.Encode(s)
		return unquote(buf.Bytes()[:buf.Len()-1])
	}, nil); err != nil {
		t.Error(err)
	}
}

// Fuzzer test for format/decode round trip.
func TestUnquoteFormat(t *testing.T) {
	t.Parallel()
	enc := func(s string) []byte {
		var buf bytes.Buffer
		quoteString(&buf, s)
		return buf.Bytes()
	}
	roundTrip := func(s string) []byte {
		return enc(unquote(enc(s)))
	}
	if err := quick.CheckEqual(enc, roundTrip, nil); err != nil {
		t.Error(err)
	}
	jsonEnc := func(s string) []byte {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.Encode(s)
		return buf.Bytes()[:buf.Len()-1]
	}
	if err := quick.CheckEqual(enc, jsonEnc, nil); err != nil {
		t.Error(err)
	}
	check := func(t *testing.T, s string) {
		t.Helper()
		if e, a := jsonEnc(s), enc(s); !bytes.Equal(e, a) {
			t.Errorf("Expected %q -> %q, got %q", s, e, a)
		}
	}
	check(t, "\"hey\" is\\\n\tfor \U0001f40es")
	check(t, "Control\a\b\f\n\r\t\v \u2029 characters")
	check(t, "Invalid character \x88\xee")
}

// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func Equal(t *testing.T, value, expected, message string) {
	t.Helper()
	if value != expected {
		t.Errorf("%s\nExpected: %q\nActual: %q",
			message, expected, value)
	}
}

// Tests that floating point expressions are correctly formatted.
func TestFormatFloatExpression(t *testing.T) {
	ve := FloatExp{
		valExp: valExp{Node: AstNode{SourceLoc{0, new(SourceFile)}, nil, nil}},
	}
	var buff strings.Builder

	ve.Value = 10.0
	ve.format(&buff, "")
	Equal(t, buff.String(), "10", "Preserve single zero after decimal.")
	buff.Reset()

	ve.Value = 10.05
	ve.format(&buff, "")
	Equal(t, buff.String(), "10.05", "Do not strip numbers ending in non-zero digit.")
	buff.Reset()

	ve.Value = 10.050
	ve.format(&buff, "")
	Equal(t, buff.String(), "10.05", "Strip single trailing zero.")
	buff.Reset()

	ve.Value = 10.050000000
	ve.format(&buff, "")
	Equal(t, buff.String(), "10.05", "Strip multiple trailing zeroes.")
	buff.Reset()

	ve.Value = 0.0000000005
	ve.format(&buff, "")
	Equal(t, buff.String(), "5e-10", "Handle exponential notation.")
	buff.Reset()

	ve.Value = 0.0005
	ve.format(&buff, "")
	Equal(t, buff.String(), "0.0005", "Handle low decimal floats.")
}

// Tests that integer expressions are correctly formatted.
func TestFormatIntExpression(t *testing.T) {
	ve := IntExp{
		valExp: valExp{Node: AstNode{SourceLoc{0, new(SourceFile)}, nil, nil}},
	}
	var buff strings.Builder

	ve.Value = 0
	ve.format(&buff, "")
	Equal(t, buff.String(), "0", "Format zero integer.")
	buff.Reset()

	ve.Value = 10
	ve.format(&buff, "")
	Equal(t, buff.String(), "10", "Format non-zero integer.")
	buff.Reset()

	ve.Value = 1000000
	ve.format(&buff, "")
	Equal(t, buff.String(), "1000000", "Preserve integer trailing zeroes.")
}

// Tests that string expressions are correctly formatted.
func TestFormatStringExpression(t *testing.T) {
	ve := StringExp{
		valExp: valExp{Node: AstNode{SourceLoc{0, new(SourceFile)}, nil, nil}},
	}
	var buff strings.Builder
	ve.Value = "blah"
	ve.format(&buff, "")
	Equal(t, buff.String(), "\"blah\"", "Double quote a string.")
	buff.Reset()

	ve.Value = `"blah"`
	ve.format(&buff, "")
	Equal(t, buff.String(), `"\"blah\""`, "Double quote a double-quoted string.")
}

// Tests that null expressions are correctly formatted.
func TestFormatNullExpression(t *testing.T) {
	ve := NullExp{
		valExp: valExp{Node: AstNode{SourceLoc{0, new(SourceFile)}, nil, nil}},
	}
	Equal(t, FormatExp(&ve, ""), "null", "Nil value is 'null'.")
}

func TestExpressionStringer(t *testing.T) {
	check := func(e Exp, expect string) {
		t.Helper()
		if str, ok := e.(fmt.Stringer); !ok {
			t.Errorf("Expression of type %T does not implement fmt.Stringer", e)
		} else if s := str.String(); s != expect {
			t.Errorf("%T: %s != %s", e, s, expect)
		}
	}
	check(new(NullExp), "null")
	const s = `A potentially very long string, which would be truncated
	           when using GoString, which is intended for debugging.`
	check(&StringExp{Value: s}, s)
	check(new(BoolExp), "false")
	check(&IntExp{Value: 4}, "4")
	check(&FloatExp{Value: 3.14}, "3.14")
}

func TestMarshalJsonExpression(t *testing.T) {
	var parser Parser
	const src = `{
	"arr": [
		{
			"m": "n",
			"o": 1
		}
	],
	"b": true,
	"bar": "thing \u2029 ",
	"baz": 1.2,
	"empty_array": [],
	"empty_map": {},
	"empty_string": "",
	"foo": 1,
	"n": null
}`
	var buf bytes.Buffer
	if exp, err := parser.ParseValExp([]byte(src)); err != nil {
		t.Error(err)
	} else if b, err := exp.MarshalJSON(); err != nil {
		t.Error(err)
	} else if err := json.Indent(&buf, b, "", "\t"); err != nil {
		t.Error(err)
	} else if s := buf.String(); s != src {
		t.Errorf("expected: %s\ngot:      %s",
			src, s)
	}
}

func ExampleMapExp_GoString() {
	var parser Parser
	exp, err := parser.ParseValExp([]byte(`{
		"arr": [
		{
			"m": "n123456789012345678901234567890",
			"o": 1
		}
		],
	"b": true,
	"bar": "thing \u2029 ",
	"baz": 1.2,
	"empty_array": [],
	"empty_map": {},
	"empty_string": "",
	"foo": 1,
	"n": null
}`))
	if err != nil {
		panic(err)
	}
	fmt.Println(exp.GoString())
	exp, err = parser.ParseValExp([]byte(`{
	"int": 1,
	"thing": {
		foo: null,
	},
	"empty_array": [],
}`))
	if err != nil {
		panic(err)
	}
	fmt.Println(exp.GoString())
	// Output:
	// {"arr":[{"m":"n1234567...34567890","o":1}],"b":true,...,"foo":1,"n":null}
	// {"empty_array":[],"int":1,"thing":{foo:null}}
}

func ExampleArrayExp_GoString() {
	var parser Parser
	exp, err := parser.ParseValExp([]byte(`[
	true,
	"thing",
	1.2,
	[],
	{},
	1,
]`))
	if err != nil {
		panic(err)
	}
	fmt.Println(exp.GoString())
	// Output:
	// [true,"thing",...,{},1]
}

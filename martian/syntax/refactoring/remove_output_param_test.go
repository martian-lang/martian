// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/martian-lang/martian/martian/syntax"
)

func diffLine(t *testing.T, expected, actual string) {
	t.Helper()
	if expected == actual {
		t.Log("=", expected)
		return
	}
	if st, tt := strings.TrimSpace(expected), strings.TrimSpace(actual); st == tt {
		if strings.TrimLeftFunc(expected, unicode.IsSpace) == strings.TrimLeftFunc(actual, unicode.IsSpace) {
			lead := len(expected) - len(st)
			if len(st) > len(expected) {
				lead = 0
			}
			eLead := expected[:lead]
			lead = len(actual) - len(st)
			if len(st) > len(actual) {
				lead = 0
			}
			aLead := actual[:lead]
			t.Error("(leading whitespace differences)",
				strconv.QuoteToGraphic(eLead), "!=",
				strconv.QuoteToGraphic(aLead))
		} else if strings.TrimRightFunc(expected, unicode.IsSpace) ==
			strings.TrimRightFunc(actual, unicode.IsSpace) {
			if len(actual) > len(expected) {
				t.Error(st, "trailing whitespace differences >", strconv.QuoteToGraphic(actual))
			} else {
				t.Error(st, "trailing whitespace differences <", strconv.QuoteToGraphic(expected))
			}
		} else {
			t.Error("(whitespace differences)",
				strconv.QuoteToGraphic(expected), "!=",
				strconv.QuoteToGraphic(actual))
		}
	} else {
		t.Error(st, "!=", tt)
	}
}

func diff(t *testing.T, expected, actual string) {
	t.Helper()
	for i := strings.IndexRune(expected, '\n'); i >= 0; i = strings.IndexRune(expected, '\n') {
		j := strings.IndexRune(actual, '\n')
		if j < 0 {
			diffLine(t, expected[:i], actual)
			if len(expected) > i+1 {
				t.Error("<", expected[i+1:])
			}
			return
		}
		diffLine(t, expected[:i], actual[:j])
		actual = actual[j+1:]
		expected = expected[i+1:]
	}
	if expected != "" {
		j := strings.IndexRune(actual, '\n')
		if j >= 0 {
			diffLine(t, expected, actual[:j])
			actual = actual[j+1:]
		}
	}
	if expected == "" && actual != "" {
		t.Error(">", actual)
	} else if expected != "" && actual == "" {
		t.Error("<", expected)
	} else if expected != "" && actual != "" {
		diffLine(t, expected, actual)
	}
}

func TestRemoveOutputParam(t *testing.T) {
	var parser syntax.Parser
	_, file, _, _ := runtime.Caller(0)
	const src = `struct Foo(
    int  foo,
    file bar,
)

stage STAGE(
    in  int  foo,
    out bool disable,
    out Foo  foo,
    src comp "none",
) retain (
    foo,
)

pipeline MIDDLE(
    in  int        foo,
    out Foo        foo,
    out int        bar,
    out int[]      bars,
    out int[][]    bazs,
    out map<int[]> maz,
)
{
    call STAGE as FIRST(
        foo = self.foo,
    )

    call STAGE as SECOND(
        foo = FIRST.foo.foo,
    ) using (
        volatile = true,
        disabled = FIRST.disable,
    )

    map call STAGE as THIRD(
        foo = split [
            SECOND.foo.foo,
            self.foo,
        ],
    ) using (
        disabled = FIRST.disable,
    )

    return (
        foo = FIRST.foo,
        bar = SECOND.foo.foo,
        bars = THIRD.foo.foo,
        bazs = [
            [FIRST.foo.foo],
            [SECOND.foo.foo,1],
            THIRD.foo.foo,
        ],
        maz = {
            "blah": [FIRST.foo.foo],
        },
    )
}

pipeline TOP(
    in  int        foo,
    out Foo        foo,
    out Foo        foo2,
    out int        bar,
    out int[]      bars,
    out int[][]    bazs,
    out map<int[]> maz,
)
{
    call MIDDLE(
        foo = self.foo,
    )

    return (
        foo  = MIDDLE.foo,
        foo2 = {
            bar: "bar",
            foo: MIDDLE.foo.foo,
        },
        bar  = MIDDLE.bar,
        bars = MIDDLE.bars,
        bazs = MIDDLE.bazs,
        maz  = {
            "gone": MIDDLE.bars,
            "kept": [
                self.foo,
                MIDDLE.bar,
            ],
        },
    )

    retain (
        MIDDLE.foo,
    )
}
`
	srcBytes := []byte(src)
	_, _, ast, err := parser.ParseSourceBytes(srcBytes, file,
		nil, false)
	if err != nil {
		t.Fatal(err)
	}
	edit1, err := RemoveOutputParam(ast.Callables.Table["STAGE"],
		"foo", []*syntax.Ast{ast})
	if err != nil {
		t.Fatal(err)
	}
	if edit1 == nil {
		t.Fatal("Expected non-nil edit")
	}
	edit2, err := RemoveOutputParam(ast.Callables.Table["STAGE"],
		"disable", []*syntax.Ast{ast})
	if err != nil {
		t.Fatal(err)
	}
	if edit2 == nil {
		t.Fatal("Expected non-nil edit")
	}
	fmtAst, err := parser.UncheckedParse(srcBytes, file)
	if err != nil {
		t.Fatal(err)
	}
	if c, err := edit1.Apply(fmtAst); err != nil {
		t.Fatal(err)
	} else if c != 23 {
		t.Errorf("%d != 22", c)
	}
	if c, err := edit2.Apply(fmtAst); err != nil {
		t.Fatal(err)
	} else if c != 3 {
		t.Errorf("%d != 2", c)
	}
	const expected = `struct Foo(
    int  foo,
    file bar,
)

stage STAGE(
    in  int foo,
    src comp "none",
)

pipeline MIDDLE(
    in  int     foo,
    out int[][] bazs,
)
{
    call STAGE as FIRST(
        foo = self.foo,
    )

    call STAGE as SECOND(
        foo = null,
    ) using (
        volatile = true,
    )

    map call STAGE as THIRD(
        foo = split [self.foo],
    )

    return (
        bazs = [[1]],
    )
}

pipeline TOP(
    in  int        foo,
    out Foo        foo2,
    out int[][]    bazs,
    out map<int[]> maz,
)
{
    call MIDDLE(
        foo = self.foo,
    )

    return (
        foo2 = {
            bar: "bar",
            foo: null,
        },
        bazs = MIDDLE.bazs,
        maz  = {
            "kept": [self.foo],
        },
    )
}
`
	if s := fmtAst.Format(); s != expected {
		diff(t, expected, s)
	}
}

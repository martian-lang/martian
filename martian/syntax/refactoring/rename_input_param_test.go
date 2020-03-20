// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"runtime"
	"testing"

	"github.com/martian-lang/martian/martian/syntax"
)

func TestRenameInputParam(t *testing.T) {
	var parser syntax.Parser
	_, file, _, _ := runtime.Caller(0)
	const src = `
stage FOO(
    in  int  foo,
    in  file bar,
    out file foo,
    src comp "none",
)

pipeline PIPE(
    in  int    foo,
    in  file   bar,
    in  bool   disble,
    out file[] foo,
    out file   bar,
)
{
    call FOO(
        foo = self.foo,
        bar = self.bar,
    ) using (
        disabled = self.disble,
    )

    return (
        foo = [FOO.foo],
        bar = self.bar,
    )

    retain (
        FOO.foo,
        self.bar,
    )
}

call PIPE(
    foo    = 1,
    bar    = "baz",
    disble = false,
)
`
	srcBytes := []byte(src)
	_, _, ast, err := parser.ParseSourceBytes(srcBytes, file,
		nil, false)
	if err != nil {
		t.Fatal(err)
	}
	edit := RenameInput(ast.Callables.Table["PIPE"],
		"bar", "baz", []*syntax.Ast{ast})
	if edit == nil {
		t.Fatal("Expected non-nil edit")
	}
	if _, err := edit.Apply(ast); err != nil {
		t.Error(err)
	}
	fmtAst, err := parser.UncheckedParse(srcBytes, file)
	if err != nil {
		t.Fatal(err)
	}
	if c, err := edit.Apply(fmtAst); err != nil {
		t.Fatal(err)
	} else if c != 5 {
		t.Errorf("%d != 5", c)
	}
	edit = RenameInput(ast.Callables.Table["FOO"],
		"foo", "fuzz", []*syntax.Ast{ast})
	if edit == nil {
		t.Fatal("Expected non-nil edit")
	}
	if _, err := edit.Apply(ast); err != nil {
		t.Error(err)
	}
	if c, err := edit.Apply(fmtAst); err != nil {
		t.Fatal(err)
	} else if c != 2 {
		t.Errorf("%d != 2", c)
	}
	edit = RenameInput(ast.Callables.Table["PIPE"],
		"disble", "disable", []*syntax.Ast{ast})
	if edit == nil {
		t.Fatal("Expected non-nil edit")
	}
	if _, err := edit.Apply(ast); err != nil {
		t.Error(err)
	}
	if c, err := edit.Apply(fmtAst); err != nil {
		t.Fatal(err)
	} else if c != 3 {
		t.Errorf("%d != 3", c)
	}
	const expected = `stage FOO(
    in  int  fuzz,
    in  file bar,
    out file foo,
    src comp "none",
)

pipeline PIPE(
    in  int    foo,
    in  file   baz,
    in  bool   disable,
    out file[] foo,
    out file   bar,
)
{
    call FOO(
        fuzz = self.foo,
        bar  = self.baz,
    ) using (
        disabled = self.disable,
    )

    return (
        foo = [FOO.foo],
        bar = self.baz,
    )

    retain (
        FOO.foo,
        self.baz,
    )
}

call PIPE(
    foo     = 1,
    baz     = "baz",
    disable = false,
)
`
	if s := fmtAst.Format(); s != expected {
		diff(t, expected, s)
	}
}

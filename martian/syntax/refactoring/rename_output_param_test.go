// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"runtime"
	"testing"

	"github.com/martian-lang/martian/martian/syntax"
)

func TestRenameOutputParam(t *testing.T) {
	var parser syntax.Parser
	_, file, _, _ := runtime.Caller(0)
	const src = `
stage FOO(
    in  file foo,
    out file foo,
    out bool disbl,
    src comp "none",
)

pipeline PIPE(
    in  file foo,
    out file bar,
    out bool disble,
)
{
    call FOO as FOO1(
        foo = self.foo,
    )

    call FOO as FOO2(
        foo = FOO1.foo,
    ) using (
        disabled = FOO1.disbl,
    )

    return (
        bar    = FOO2.foo,
        disble = FOO1.disbl,
    )

    retain (
        FOO1.foo,
    )
}

call PIPE(
    foo = "foo",
)
`
	srcBytes := []byte(src)
	_, _, ast, err := parser.ParseSourceBytes(srcBytes, file,
		nil, false)
	if err != nil {
		t.Fatal(err)
	}
	edit := RenameOutput(ast.Callables.Table["FOO"],
		"disbl", "disable", []*syntax.Ast{ast})
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
	} else if c != 3 {
		t.Errorf("%d != 3", c)
	}
	edit = RenameOutput(ast.Callables.Table["FOO"],
		"foo", "fuzz", []*syntax.Ast{ast})
	if edit == nil {
		t.Fatal("Expected non-nil edit")
	}
	if _, err := edit.Apply(ast); err != nil {
		t.Error(err)
	}
	if c, err := edit.Apply(fmtAst); err != nil {
		t.Fatal(err)
	} else if c != 4 {
		t.Errorf("%d != 4", c)
	}
	edit = RenameOutput(ast.Callables.Table["PIPE"],
		"disble", "disable", []*syntax.Ast{ast})
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
	const expected = `stage FOO(
    in  file foo,
    out file fuzz,
    out bool disable,
    src comp "none",
)

pipeline PIPE(
    in  file foo,
    out file bar,
    out bool disable,
)
{
    call FOO as FOO1(
        foo = self.foo,
    )

    call FOO as FOO2(
        foo = FOO1.fuzz,
    ) using (
        disabled = FOO1.disable,
    )

    return (
        bar     = FOO2.fuzz,
        disable = FOO1.disable,
    )

    retain (
        FOO1.fuzz,
    )
}

call PIPE(
    foo = "foo",
)
`
	if s := fmtAst.Format(); s != expected {
		diff(t, expected, s)
	}
}

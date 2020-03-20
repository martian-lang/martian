// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package refactoring

import (
	"runtime"
	"testing"

	"github.com/martian-lang/martian/martian/syntax"
)

func TestRenameCallable(t *testing.T) {
	var parser syntax.Parser
	_, file, _, _ := runtime.Caller(0)
	const src = `
stage OLDNAME(
    in  int  foo,
    out int  bar,
    out file foo,
    src comp "none",
)

stage FOO(
    in  int[] foo,
    out file  foo,
    src comp  "none",
)

pipeline PIPE1(
    in  int    foo,
    out file[] foo,
)
{
    call OLDNAME(
        foo = self.foo,
    )

    call OLDNAME as FIRST(
        foo = OLDNAME.bar,
    )

    call FOO(
        foo = [
            self.foo,
            OLDNAME.bar,
            FIRST.bar,
        ],
    )

    return (
        foo = [
            FIRST.foo,
            OLDNAME.foo,
            FOO.foo,
        ],
    )

    retain (
        FIRST.foo,
        OLDNAME.foo,
        FOO.foo,
    )
}

pipeline PIPE2(
    in  int    foo,
    out file[] foo,
)
{
    call OLDNAME(
        foo = self.foo,
    )

    call OLDNAME as FIRST(
        foo = OLDNAME.bar,
    )

    call FOO as NEWNAME(
        foo = [
            self.foo,
            OLDNAME.bar,
            FIRST.bar,
        ],
    )

    return (
        foo = [
            FIRST.foo,
            OLDNAME.foo,
            NEWNAME.foo,
        ],
    )

    retain (
        FIRST.foo,
        OLDNAME.foo,
        NEWNAME.foo,
    )
}

call OLDNAME(
    foo = 1,
)
`
	srcBytes := []byte(src)
	_, _, ast, err := parser.ParseSourceBytes(srcBytes, file,
		nil, false)
	if err != nil {
		t.Fatal(err)
	}
	edit := RenameCallable(ast.Callables.Table["OLDNAME"],
		"NEWNAME", []*syntax.Ast{ast})
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
	} else if c != 10 {
		t.Errorf("%d != 10", c)
	}
	const expected = `stage NEWNAME(
    in  int  foo,
    out int  bar,
    out file foo,
    src comp "none",
)

stage FOO(
    in  int[] foo,
    out file  foo,
    src comp  "none",
)

pipeline PIPE1(
    in  int    foo,
    out file[] foo,
)
{
    call NEWNAME(
        foo = self.foo,
    )

    call NEWNAME as FIRST(
        foo = NEWNAME.bar,
    )

    call FOO(
        foo = [
            self.foo,
            NEWNAME.bar,
            FIRST.bar,
        ],
    )

    return (
        foo = [
            FIRST.foo,
            NEWNAME.foo,
            FOO.foo,
        ],
    )

    retain (
        FIRST.foo,
        NEWNAME.foo,
        FOO.foo,
    )
}

pipeline PIPE2(
    in  int    foo,
    out file[] foo,
)
{
    call NEWNAME as OLDNAME(
        foo = self.foo,
    )

    call NEWNAME as FIRST(
        foo = OLDNAME.bar,
    )

    call FOO as NEWNAME(
        foo = [
            self.foo,
            OLDNAME.bar,
            FIRST.bar,
        ],
    )

    return (
        foo = [
            FIRST.foo,
            OLDNAME.foo,
            NEWNAME.foo,
        ],
    )

    retain (
        FIRST.foo,
        OLDNAME.foo,
        NEWNAME.foo,
    )
}

call NEWNAME(
    foo = 1,
)
`
	if s := fmtAst.Format(); s != expected {
		diff(t, expected, s)
	}
}

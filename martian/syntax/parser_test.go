//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian formatter tests.
//

package syntax

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	SetEnforcementLevel(EnforceError)
	os.Exit(m.Run())
}

func testGood(t *testing.T, src string) *Ast {
	t.Helper()
	if ast, err := yaccParse(src, nil); err != nil {
		t.Fatalf("ParseError: unexpected token '%s' at line %d:%d in\n%s",
			err.token, err.loc, err.pos, err.src)
		return nil
	} else if err := ast.compile(nil, false); err != nil {
		t.Errorf("Failed to compile src: %v\n%s", err, err.Error())
		return nil
	} else {
		return ast
	}
}

func testBadGrammar(t *testing.T, src string) {
	t.Helper()
	if _, err := yaccParse(src, nil); err == nil {
		t.Error("Expected failure to parse, but got success.")
	}
}

func testBadCompile(t *testing.T, src string) {
	t.Helper()
	if ast, err := yaccParse(src, nil); err != nil {
		t.Fatalf("ParseError: unexpected token '%s' at line %d:%d in\n%s",
			err.token, err.loc, err.pos, err.src)
	} else if err := ast.compile(nil, false); err == nil {
		t.Error("Expected failure to compile.")
	}
}

func TestSimplePipe(t *testing.T) {
	if ast := testGood(t, `
# File comment

# Stage comment
# This describes the stage.
stage SUM_SQUARES(
    in  float[] values,
    # sum comment
    out float   sum,
    src py      "stages/sum_squares",
)

pipeline SUM_SQUARE_PIPELINE(
    in  float[] values,
    out float   sum,
)
{
    call SUM_SQUARES(
        values = self.values,
    )
    return (
        sum = SUM_SQUARES.sum,
    )
}

call SUM_SQUARE_PIPELINE(
    values = [1.0, 2.0, 3.0],
)
`); ast != nil {
		if s := ast.Callables.Table["SUM_SQUARES"]; s == nil {
			t.Error("No callable named SUM_SQUARES found")
		} else {
			if len(s.getNode().Comments) != 2 {
				t.Errorf("Incorrect stage comment count %d", len(s.getNode().Comments))
			} else {
				if s.getNode().Comments[0] != "# Stage comment" {
					t.Errorf("Expected comment '# Stage comment', got %s",
						s.getNode().Comments[0])
				}
				if s.getNode().Comments[1] != "# This describes the stage." {
					t.Errorf("Expected comment '# This describes the stage.', got %s",
						s.getNode().Comments[1])
				}
			}
			o := s.(*Stage).OutParams
			if len(o.Table) != 1 {
				t.Errorf("Incorrect out param count %d", len(o.Table))
			} else if p := o.Table["sum"]; p == nil {
				t.Error("No out param 'sum' found")
			} else {
				if len(p.getNode().Comments) != 1 {
					t.Errorf("Incorrect param comment count %d", len(p.getNode().Comments))
				} else {
					if p.getNode().Comments[0] != "# sum comment" {
						t.Errorf("Expected comment '# sum comment', got %s",
							p.getNode().Comments[0])
					}
				}
			}
		}
	}
}

func TestBinding(t *testing.T) {
	testGood(t, `
stage SUM_SQUARES(
    in  float[] values,
    in  int     threads,
    in  bool    local,
    out float   sum,
    src py      "stages/sum_squares",
)

stage REPORT(
    in  float[] values,
    in  float   sum,
    src py      "stages/report",
)

pipeline SUM_SQUARE_PIPELINE(
    in  float[] values,
    out float   sum,
)
{
    call SUM_SQUARES(
        values  = self.values,
        threads = 1,
        local   = false,
    )
    call REPORT(
        values = self.values,
        sum    = SUM_SQUARES.sum,
    )

    return (
        sum = SUM_SQUARES.sum,
    )
}

call SUM_SQUARE_PIPELINE(
    values = [1.0, 2.0, 3.0],
)
`)
}

func TestSubPipe(t *testing.T) {
	testGood(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
)

stage REPORT(
    in  float[] values,
    in  float   sum,
    src py      "stages/report",
)

pipeline STUFF(
    in float[] values,
    out float sum,
)
{
    call SUM_SQUARES(
        values = self.values,
    )
    return (
        sum = SUM_SQUARES.sum,
    )
}

pipeline SUM_SQUARE_PIPELINE(
    in  float[] values,
    out float   sum,
    out float   sum2,
)
{
    call STUFF(
        values = self.values,
    )
    call SUM_SQUARES(
        values = self.values,
    )
    call REPORT(
        values = self.values,
        sum    = SUM_SQUARES.sum,
    )

    return (
        sum = STUFF.sum,
        sum2 = SUM_SQUARES.sum,
    )
}

call SUM_SQUARE_PIPELINE(
    values = [1.0, 2.0, 3.0],
)
`)
}

func TestUnusedParam(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
	in  int     unused,
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
)

pipeline SUM_SQUARE_PIPELINE(
    in  float[] values,
    out float   sum,
)
{
    call SUM_SQUARES(
        values = self.values,
    )

    return (
        sum = SUM_SQUARES.sum,
    )
}

call SUM_SQUARE_PIPELINE(
    values = [1.0, 2.0, 3.0],
)
`)
}

func TestMissingReturn(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
)

pipeline SUM_SQUARE_PIPELINE(
    in  float[] values,
    out float   sum,
    out float   sum2,
)
{
    call SUM_SQUARES(
        values = self.values,
    )

    return (
        sum = SUM_SQUARES.sum,
    )
}

call SUM_SQUARE_PIPELINE(
    values = [1.0, 2.0, 3.0],
)
`)
}

func TestInvalidReturnBinding(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
)

pipeline SUM_SQUARE_PIPELINE(
    in  float[] values,
    out float   sum,
    out float   sum2,
)
{
    call SUM_SQUARES(
        values = self.values,
    )

    return (
        sum = STUFF.sum,
    )
}

call SUM_SQUARE_PIPELINE(
    values = [1.0, 2.0, 3.0],
)
`)
}

func TestSelfReturnBinding(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
)

pipeline SUM_SQUARE_PIPELINE(
    in  float[] values,
    out float   sum,
)
{
    call SUM_SQUARES(
        values = self.values,
    )

    return (
        sum = self.sum,
    )
}

call SUM_SQUARE_PIPELINE(
    values = [1.0, 2.0, 3.0],
)
`)
}

func TestTopoSort(t *testing.T) {
	if ast := testGood(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
)

stage REPORT(
    in  float[] values,
    in  float   sum,
    src py      "stages/report",
)

pipeline SUM_SQUARE_PIPELINE(
    in  float[] values,
    out float   sum,
)
{
    call REPORT(
        values = self.values,
        sum    = SUM_SQUARES.sum,
    )
    call SUM_SQUARES(
        values = self.values,
    )

    return (
        sum = SUM_SQUARES.sum,
    )
}

call SUM_SQUARE_PIPELINE(
    values = [1.0, 2.0, 3.0],
)
`); ast != nil {
		if len(ast.Pipelines) != 1 {
			t.Fatal("Incorrect pipeline count", len(ast.Pipelines))
		} else if calls := ast.Pipelines[0].Calls; len(calls) != 2 {
			t.Fatal("Incorrect call count", len(calls))
		} else if calls[0].Id != "SUM_SQUARES" {
			t.Error("Incorrect stage ordering.")
		}
	}
}

func TestArrayBind(t *testing.T) {
	testGood(t, `
filetype json;

stage ADD_KEY(
    in  string key,
    in  string value,
    in  json   start,
    out json   result,
    src py     "stages/add_key",
)

stage MERGE_JSON(
    in  json[] inputs,
    out json   result,
    src py     "stages/merge_json",
)

pipeline STUFF(
    in string key1,
    in string value1,
    out json outfile,
)
{
    call ADD_KEY as ADD_KEY1(
        key   = self.key1,
        value = self.value1,
        start = null,
    )
    call ADD_KEY as ADD_KEY2(
        key   = "key2",
        value = "value2",
        start = ADD_KEY1.result,
    )
    call ADD_KEY as ADD_KEY3(
        key   = "key3",
        value = "value3",
        start = ADD_KEY1.result,
    )
    call MERGE_JSON(
        inputs = [
            ADD_KEY2.result,
            ADD_KEY3.result,
        ],
    )
    return (
        outfile = MERGE_JSON.result,
    )
}
`)
}

func TestUserType(t *testing.T) {
	testGood(t, `
filetype goodness;

stage SUM_SQUARES(
    in  float[]  values,
    out goodness sum,
    src py       "stages/sum_squares",
)
`)
}

func TestFileAsUserType(t *testing.T) {
	testGood(t, `
filetype goodness;

stage SUM_SQUARES(
    in  float[]  values,
    out goodness sum,
    src py       "stages/sum_squares",
)

pipeline PIPE(
    in  float[] values,
    out file sum,
)
{
    call SUM_SQUARES(
        values = self.values,
    )
    return (
        sum = SUM_SQUARES.sum,
    )
}
`)
}

func TestIncompatibleUserType(t *testing.T) {
	testBadCompile(t, `
filetype goodness;
filetype badness;

stage SUM_SQUARES(
    in  float[]  values,
    out goodness sum,
    src py       "stages/sum_squares",
)

pipeline PIPE(
    in  float[] values,
    out badness sum,
)
{
    call SUM_SQUARES(
        values = self.values,
    )
    return (
        sum = SUM_SQUARES.sum,
    )
}
`)
}

func TestInvalidInType(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
    in  badness[] values,
    out float     sum,
    src py        "stages/sum_squares",
)
`)
}

func TestInvalidOutType(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out badness sum,
    src py      "stages/sum_squares",
)
`)
}

func TestResources(t *testing.T) {
	testGood(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
) using (
    threads = 2,
    mem_gb = 1,
)
`)
}

func TestBadMemGB(t *testing.T) {
	testBadGrammar(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
) using (
    threads = 2,
    mem_gb = 1.5,
)
`)
}

func TestBadThreads(t *testing.T) {
	testBadGrammar(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
) using (
    threads = 2.2,
    mem_gb = 1,
)
`)
}

func TestResourcesWithSplit(t *testing.T) {
	if ast := testGood(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
) split (
    in  int     foo,
    out int     bar,
) using (
    threads = 2,
    mem_gb = 3,
)
`); ast != nil {
		if len(ast.Stages) != 1 {
			t.Fatalf("Incorrect stage count %d", len(ast.Stages))
		} else if res := ast.Stages[0].Resources; res == nil {
			t.Fatal("No resources.")
		} else {
			if res.StrictVolatile {
				t.Error("Should not be strict volatile.")
			}
			if res.Threads != 2 {
				t.Errorf("Expected 2 threads, saw %d",
					res.Threads)
			}
			if res.MemGB != 3 {
				t.Errorf("Expected 3gb, saw %d",
					res.MemGB)
			}
		}
	}
}

func TestStrictVolatile(t *testing.T) {
	if ast := testGood(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
) using (
    threads = 2,
    mem_gb = 1,
    volatile = strict,
)
`); ast != nil {
		if len(ast.Stages) != 1 {
			t.Fatalf("Incorrect stage count %d", len(ast.Stages))
		} else if res := ast.Stages[0].Resources; res == nil {
			t.Fatal("No resources.")
		} else if !res.StrictVolatile {
			t.Error("Not volatile.")
		}
	}
}

func TestRetain(t *testing.T) {
	if ast := testGood(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    out file    report,
    src py      "stages/sum_squares",
) using (
    threads = 2,
    mem_gb = 1,
    volatile = strict,
) retain (
    report,
)
`); ast != nil {
		if len(ast.Stages) != 1 {
			t.Fatalf("Incorrect stage count %d", len(ast.Stages))
		} else if ret := ast.Stages[0].Retain; ret == nil {
			t.Fatal("No retain.")
		} else if len(ret.Params) != 1 {
			t.Errorf("Expected 1 retain, found %d",
				len(ret.Params))
		} else if ret.Params[0].Id != "report" {
			t.Errorf("Expected to retain 'report'.  Saw '%s' instead.",
				ret.Params[0].Id)
		}
	}
}

func TestRetainNonFile(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    out file    report,
    src py      "stages/sum_squares",
) using (
    threads = 2,
    mem_gb = 1,
    volatile = strict,
) retain (
    sum,
)
`)
}

func TestSplit(t *testing.T) {
	testGood(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
) split (
    in  int     foo,
    out int     bar,
)
`)
}

func TestSplitDuplicateIn(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
) split (
    in  int     values,
    out int     bar,
)
`)
}

func TestSplitDuplicateOut(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
) split (
    in  int     foo,
    out int     sum,
)
`)
}

// Check that there is an error if a call depends on itself directly.
func TestSelfBind(t *testing.T) {
	testBadCompile(t, `
stage SQUARES(
    in  float value,
    out float square,
    src py    "stages/square",
)

pipeline QUARTIC(
    out float quart,
)
{
    call SQUARES(
        value = SQUARES.square,
    )
    return (
        quart = SQUARES.square,
    )
}
`)
}

// Check that there is an error if a call depends on itself with one level of
// indirection.
func TestTransDep(t *testing.T) {
	testBadCompile(t, `
stage SQUARES(
    in  float value,
    out float square,
    src py    "stages/square",
)

pipeline POLY(
    out float quart,
)
{
    call SQUARES as S1(
        value = S2.square,
    )
    call SQUARES as S2(
        value = S1.square,
    )
    return (
        quart = S2.square,
    )
}
`)
}

// Check that there is an error if a call depends on itself with two levels of
// indirection.
func TestTransDep2(t *testing.T) {
	testBadCompile(t, `
stage SQUARES(
    in  float value,
    out float square,
    src py    "stages/square",
)

pipeline POLY(
    out float quart,
)
{
    call SQUARES as S1(
        value = S3.square,
    )
    call SQUARES as S2(
        value = S1.square,
    )
    call SQUARES as S3(
        value = S2.square,
    )
    return (
        quart = S3.square,
    )
}
`)
}

// Check that there is an error if a call depends on itself directly in an
// array.
func TestSelfBindArray(t *testing.T) {
	testBadCompile(t, `
stage SQUARES(
    in  float[][] values,
    out float     square,
    src py        "stages/square",
)

pipeline QUARTIC(
    out float quart,
)
{
    call SQUARES(
        values = [[1], [2, 3], [1, SQUARES.square]],
    )
    return (
        quart = SQUARES.square,
    )
}
`)
}

// Check that binding inside an array works.
func TestGoodBindArray(t *testing.T) {
	testGood(t, `
stage SQUARES(
    in  float[][] values,
    out float     square,
    src py        "stages/square",
)

pipeline QUARTIC(
	in  float value,
    out float quart,
)
{
    call SQUARES(
        values = [[1], [2, 3], [1, self.value]],
    )
    return (
        quart = SQUARES.square,
    )
}
`)
}

func TestMultiArray(t *testing.T) {
	testGood(t, `
stage SQUARES(
    in  int[][] values,
    out float   square,
    src py      "stages/square",
)

pipeline QUARTIC(
    out float quart,
)
{
    call SQUARES(
        values = [[1], [2, 3], [1, 4]],
    )
    return (
        quart = SQUARES.square,
    )
}
`)
}

func TestInconsistentArray(t *testing.T) {
	testBadCompile(t, `
stage SQUARES(
    in  int[][] values,
    out float   square,
    src py      "stages/square",
)

pipeline QUARTIC(
    out float quart,
)
{
    call SQUARES(
        values = [1, [2, 3], [1, 4]],
    )
    return (
        quart = SQUARES.square,
    )
}
`)
}

func TestWrongArrayDim(t *testing.T) {
	testBadCompile(t, `
stage SQUARES(
    in  int[][] values,
    out float   square,
    src py      "stages/square",
)

pipeline QUARTIC(
    out float quart,
)
{
    call SQUARES(
        values = [1, 2, 3],
    )
    return (
        quart = SQUARES.square,
    )
}
`)
}

func TestArrayType(t *testing.T) {
	testBadCompile(t, `
stage SQUARES(
    in  int[][] values,
    out float   square,
    src py      "stages/square",
)

pipeline QUARTIC(
    out float quart,
)
{
    call SQUARES(
        values = [[2, 3], [1, 4.2]],
    )
    return (
        quart = SQUARES.square,
    )
}
`)
}

func TestDuplicateInParam(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    in  string  values,
    out float   sum,
    src py      "stages/sum_squares",
)
`)
}

func TestDuplicateOutParam(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    out string  sum,
    src py      "stages/sum_squares",
)
`)
}

func TestSplitOut(t *testing.T) {
	testGood(t, `
stage SUM_SQUARES(
    in  float[] values,
    out string  sum,
    src py      "stages/sum_squares",
) split using (
    in  float value,
    out float square,
)
`)
}

func TestDuplicateSplitOut(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out string  sum,
    src py      "stages/sum_squares",
) split using (
    in  float value,
    out float square,
    out int   square,
)
`)
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out string  sum,
    src py      "stages/sum_squares",
) split using (
    in  float value,
    out float sum,
)
`)
}

func TestDuplicateCallable(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
)

pipeline SUM_SQUARES(
    in  float[] values,
    out float   sum,
)
{
    call SUM_SQUARES(
        values = self.values,
    )
    return (
        sum = SUM_SQUARES.sum,
    )
}
`)
}

func TestMissingCallable(t *testing.T) {
	testBadCompile(t, `
pipeline SUM_SQUARES_PIPE(
    in  float[] values,
    out float   sum,
)
{
    call SUM_SQUARES(
        values = self.values,
    )
    return (
        sum = SUM_SQUARES.sum,
    )
}
`)
}

func TestVolatile(t *testing.T) {
	if ast := testGood(t, `
stage SQUARE(
    in  int   value,
    out float square,
    src py    "stages/square",
)

pipeline SQ_PIPE(
    out float square,
)
{
    call volatile SQUARE(
        value = 1,
    )
    return (
        square = SQUARE.square,
    )
}
`); ast != nil {
		if mods := ast.Pipelines[0].Calls[0].Modifiers; mods == nil {
			t.Errorf("Nil mods")
		} else {
			if mods.Bindings != nil {
				t.Errorf("Expected non-bound volatile")
			}
			if !mods.Volatile {
				t.Errorf("Expected volatile")
			}
		}
	}
}

func TestPreflight(t *testing.T) {
	if ast := testGood(t, `
stage SQUARE(
    in  int   value,
    src py    "stages/square",
)

pipeline SQ_PIPE(
)
{
    call preflight SQUARE(
        value = 1,
    )
    return ()
}
`); ast != nil {
		if mods := ast.Pipelines[0].Calls[0].Modifiers; mods == nil {
			t.Errorf("Nil mods")
		} else {
			if mods.Bindings != nil {
				t.Errorf("Expected non-bound volatile")
			}
			if mods.Volatile {
				t.Errorf("Expected non volatile")
			}
		}
	}
}

func TestVolatilePreflight(t *testing.T) {
	if ast := testGood(t, `
stage SQUARE(
    in  int   value,
    src py    "stages/square",
)

pipeline SQ_PIPE(
)
{
    call preflight volatile SQUARE(
        value = 1,
    )
    return ()
}
`); ast != nil {
		if mods := ast.Pipelines[0].Calls[0].Modifiers; mods == nil {
			t.Errorf("Nil mods")
		} else {
			if mods.Bindings != nil {
				t.Errorf("Expected non-bound volatile")
			}
			if !mods.Volatile {
				t.Errorf("Expected volatile")
			}
			if mods.Local {
				t.Errorf("Expected non-local")
			}
			if !mods.Preflight {
				t.Errorf("Expected preflight")
			}
		}
	}
}

func TestVolatilePreflight2(t *testing.T) {
	if ast := testGood(t, `
stage SQUARE(
    in  int   value,
    src py    "stages/square",
)

pipeline SQ_PIPE(
)
{
    call SQUARE(
        value = 1,
    ) using (
        volatile  = true,
        preflight = true,
    )
    return ()
}
`); ast != nil {
		if mods := ast.Pipelines[0].Calls[0].Modifiers; mods == nil {
			t.Errorf("Nil mods")
		} else {
			if !mods.Volatile {
				t.Errorf("Expected volatile")
			}
			if mods.Local {
				t.Errorf("Expected non-local")
			}
			if !mods.Preflight {
				t.Errorf("Expected preflight")
			}
		}
	}
}

func TestDisable(t *testing.T) {
	if ast := testGood(t, `
stage SQUARE(
    in  int   value,
    src py    "stages/square",
)

pipeline SQ_PIPE(
    in bool disable_square,
)
{
    call SQUARE(
        value = 1,
    ) using (
        disabled  = self.disable_square,
        volatile  = true,
        preflight = false,
    )
    return ()
}
`); ast != nil {
		if mods := ast.Pipelines[0].Calls[0].Modifiers; mods == nil {
			t.Errorf("Nil mods")
		} else {
			if !mods.Volatile {
				t.Errorf("Expected volatile")
			}
			if mods.Local {
				t.Errorf("Expected non-local")
			}
			if mods.Preflight {
				t.Errorf("Expected non-preflight")
			}
			if mods.Bindings == nil {
				t.Errorf("Expected bindings.")
			} else if dis := mods.Bindings.Table[disabled]; dis == nil {
				t.Errorf("Expected disable binding.")
			}
		}
	}
}

func TestBadDisable(t *testing.T) {
	testBadCompile(t, `
stage SQUARE(
    in  int   value,
    src py    "stages/square",
)

pipeline SQ_PIPE(
    in int disable_square,
)
{
    call SQUARE(
        value = 1,
    ) using (
        disabled  = self.disable_square,
        volatile  = true,
        preflight = false,
    )
    return ()
}
`)
}

func TestBadDisable2(t *testing.T) {
	testBadGrammar(t, `
stage SQUARE(
    in  int   value,
    src py    "stages/square",
)

pipeline SQ_PIPE()
{
    call SQUARE(
        value = 1,
    ) using (
        disabled  = true,
        volatile  = true,
        preflight = false,
    )
    return ()
}
`)
}

func TestTopCall(t *testing.T) {
	testGood(t, `
stage SQUARE(
    in  int   value,
    src py    "stages/square",
)

pipeline SQ_PIPE(
    in int value,
)
{
    call SQUARE(
        value = self.value,
    )
    return ()
}

call SQ_PIPE(
    value = 1,
)
`)
}

func TestDisableTopCall(t *testing.T) {
	testBadGrammar(t, `
stage SQUARE(
    in  int   value,
    src py    "stages/square",
)

pipeline SQ_PIPE(
    in int value,
)
{
    call SQUARE(
        value = self.value,
    )
    return ()
}

call SQ_PIPE(
    value = 1,
) using (
    disabled = true,
)
`)
}

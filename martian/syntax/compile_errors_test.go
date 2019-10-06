// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"fmt"
	"strings"
	"testing"
)

// Checks that the source can be parsed but does not compile.
func testBadCompile(t *testing.T, src, expect string) string {
	t.Helper()
	if ast, err := yaccParse([]byte(src), new(SourceFile), makeStringIntern()); err != nil {
		t.Fatal(err.Error())
		return ""
	} else if err := ast.compile(); err == nil {
		t.Error("Expected failure to compile.")
		return ""
	} else {
		msg := err.Error()
		if !strings.Contains(msg, expect) {
			t.Errorf("Expected %q, got %q", expect, msg)
		}
		return msg
	}
}

func TestMissingStageParam(t *testing.T) {
	t.Parallel()
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
`, "ArgumentNotSuppliedError")
}

func TestMissingBoundCall(t *testing.T) {
	t.Parallel()
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
        values = MISSING.values,
    )

    return (
        sum = SUM_SQUARES.sum,
    )
}

call SUM_SQUARE_PIPELINE(
    values = [1.0, 2.0, 3.0],
)
`, "ScopeNameError")
}

func TestMissingReturn(t *testing.T) {
	t.Parallel()
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
`, "ArgumentNotSuppliedError")
}

func TestExtraReturn(t *testing.T) {
	t.Parallel()
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
        sum  = SUM_SQUARES.sum,
        sum2 = 3,
    )
}

call SUM_SQUARE_PIPELINE(
    values = [1.0, 2.0, 3.0],
)
`, "ArgumentError")
}

func TestReturnBadArray(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float[] sum,
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
`, "TypeMismatchError")
}

func TestReturnBadNotArray(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
)

pipeline SUM_SQUARE_PIPELINE(
    in  float[] values,
    out float[] sum,
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
`, "TypeMismatchError")
}

func TestReturnWrongArray(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float[] sum,
    src py      "stages/sum_squares",
)

pipeline SUM_SQUARE_PIPELINE(
    in  float[]   values,
    out float[][] sum,
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
`, "TypeMismatchError")
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
`, "ScopeNameError")
}

func TestSelfReturnBinding(t *testing.T) {
	t.Parallel()
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
`, "ScopeNameError")
}

func TestIncompatibleUserType(t *testing.T) {
	t.Parallel()
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
`, "TypeMismatchError")
}

func TestIncompatibleUserTypeIn(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
filetype goodness;
filetype badness;

stage SUM_SQUARES(
    in  goodness values,
    out goodness sum,
    src py       "stages/sum_squares",
)

pipeline PIPE(
    in  badness  values,
    out goodness sum,
)
{
    call SUM_SQUARES(
        values = self.values,
    )
    return (
        sum = SUM_SQUARES.sum,
    )
}
`, "TypeMismatchError")
}

func TestInvalidInType(t *testing.T) {
	testBadCompile(t, `
stage SUM_SQUARES(
    in  badness[] values,
    out float     sum,
    src py        "stages/sum_squares",
)
`, "TypeError")
}

func TestInvalidOutType(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out badness sum,
    src py      "stages/sum_squares",
)
`, "TypeError")
}

func TestRetainNonFile(t *testing.T) {
	t.Parallel()
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
`, "RetainParamError")
}

func TestRetainPipelineNonFile(t *testing.T) {
	t.Parallel()
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
)

pipeline SQUARES(
    in  float[] values,
    out float sum,
)
{
    call SUM_SQUARES(
        values = self.values,
    )

    return (
        sum = SUM_SQUARES.sum,
    )

    retain (
        SUM_SQUARES.sum,
    )
}
`, "RetainParamError")
}

func TestRetainMissing(t *testing.T) {
	t.Parallel()
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
    sum2,
)
`, "RetainParamError")
}

func TestSplitDuplicateIn(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
) split (
    in  int     values,
    out int     bar,
)
`, "DuplicateNameError")
}

func TestSplitDuplicateSplitIn(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
) split (
    in  int     value,
    in  int     value,
    out int     bar,
)
`, "DuplicateNameError")
}

func TestSplitDuplicateOut(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src py      "stages/sum_squares",
) split (
    in  int     foo,
    out int     sum,
)
`, "DuplicateNameError")
}

// Check that there is an error if a call depends on itself directly.
func TestSelfBind(t *testing.T) {
	t.Parallel()
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
`, "CyclicDependencyError")
}

// Check that there is an error if a call binds a parameter twice.
func TestDuplicateBinding(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SQUARES(
    in  float value,
    out float square,
    src py    "stages/square",
)

pipeline QUARTIC(
	in  float value,
    out float quart,
)
{
    call SQUARES(
        value = self.value,
		value = 5,
    )
    return (
        quart = SQUARES.square,
    )
}
`, "DuplicateBinding")
}

// Check that there is an error if a call binds a return parameter twice.
func TestDuplicateReturnBinding(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SQUARES(
    in  float value,
    out float square,
    src py    "stages/square",
)

pipeline QUARTIC(
	in  float value,
    out float quart,
)
{
    call SQUARES(
        value = self.value,
    )
    return (
        quart = SQUARES.square,
        quart = SQUARES.square,
    )
}
`, "DuplicateBinding")
}

// Check that there is an error if a has both old- and new-style modifiers.
func TestConflictingModifiers(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SQUARES(
    in  float value,
    out float square,
    src py    "stages/square",
)

pipeline QUARTIC(
    in  float value,
    out float quart,
)
{
    call volatile SQUARES(
        value = self.value,
    ) using (
        volatile = true,
    )

    return (
        quart = SQUARES.square,
    )
}
`, "ConflictingModifiers")
	testBadCompile(t, `
stage SQUARES(
    in  float value,
    out float square,
    src py    "stages/square",
)

pipeline QUARTIC(
    in  float value,
    out float quart,
)
{
    call preflight SQUARES(
        value = self.value,
    ) using (
        preflight = true,
    )

    return (
        quart = SQUARES.square,
    )
}
`, "ConflictingModifiers")
	testBadCompile(t, `
stage SQUARES(
    in  float value,
    out float square,
    src py    "stages/square",
)

pipeline QUARTIC(
    in  float value,
    out float quart,
)
{
    call local SQUARES(
        value = self.value,
    ) using (
        local = true,
    )

    return (
        quart = SQUARES.square,
    )
}
`, "ConflictingModifiers")
}

// Check that there is an error if a call depends on itself with one level of
// indirection.
func TestTransDep(t *testing.T) {
	t.Parallel()
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
`, "CyclicDependencyError")
}

// Check that there is an error if a call depends on itself with two levels of
// indirection.
func TestTransDep2(t *testing.T) {
	t.Parallel()
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
`, "CyclicDependencyError")
}

// Check that there is an error if a call depends on itself directly in an
// array.
func TestSelfBindArray(t *testing.T) {
	t.Parallel()
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
`, "CyclicDependencyError")
}

// Check that there is an error if pipeline calls itself (infinite recursion).
func TestPipelineRecursion(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
pipeline QUARTIC(
    in  float value,
    out float quart,
)
{
    call QUARTIC as SQUARES(
        value = self.value,
    )
    return (
        quart = SQUARES.square,
    )
}
`, "RecursiveCallError")
}

// Check that there is an error if pipeline calls itself indirectly.
func TestPipelineIndirectRecursion(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
pipeline QUARTIC(
    in  float value,
    out float quart,
)
{
    call CUBIC as SQUARES(
        value = self.value,
    )
    return (
        quart = SQUARES.cube,
    )
}

pipeline CUBIC(
    in  float value,
    out float cube,
)
{
    call QUARTIC as SQUARES(
        value = self.value,
    )

    return (
        cube = SQUARES.quart,
    )
}

call QUARTIC(
    value = 1.0,
)
`, "ArgumentError")
}

// Check that pipelines cannot be called as local, preflight, or volatile.
func TestPipelineUnsuportedMods(t *testing.T) {
	t.Parallel()
	src := `
stage SQUARES(
    in  float value,
    out float square,
    src py    "stages/square",
)

pipeline SQUARES_PIPE(
    in  float value,
    out float square,
)
{
    call SQUARES(
        value = self.value,
    )
    return (
        square = SQUARES.square,
    )
}

pipeline QUARTIC(
    in  float value,
    out float quart,
)
{
    call SQUARES(
        value = self.value,
    )

	call SQUARES_PIPE(
		value = SQUARES.value,
	) using (
		%s = true,
	)

    return (
        quart = SQUARES_PIPE.square,
    )
}
`
	testBadCompile(t, fmt.Sprintf(src, "volatile"),
		"UnsupportedTagError")
	testBadCompile(t, fmt.Sprintf(src, "local"),
		"UnsupportedTagError")
	testBadCompile(t, fmt.Sprintf(src, "preflight"),
		"UnsupportedTagError")
}

// Check that binding an array with incorrect dimension fails.
func TestBindArrayWrong(t *testing.T) {
	t.Parallel()
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
        values = [1],
    )
    return (
        quart = SQUARES.square,
    )
}
`, "TypeMismatchError")
}

// Check that null as an array value works.
func TestBindArrayWrongType(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SQUARES(
    in  int[] values,
    out float square,
    src py    "stages/square",
)

pipeline QUARTIC(
    out float quart,
)
{
    call SQUARES(
        values = "foo",
    )
    return (
        quart = SQUARES.square,
    )
}
`, "TypeMismatchError")
}

// Check that binding an array to a scaler input fails.
func TestBindUnexpectedArray(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SQUARES(
    in  float values,
    out float square,
    src py    "stages/square",
)

pipeline QUARTIC(
    out float quart,
)
{
    call SQUARES(
        values = [1],
    )
    return (
        quart = SQUARES.square,
    )
}
`, "TypeMismatchError")
}

func TestInconsistentArray(t *testing.T) {
	t.Parallel()
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
`, "TypeMismatchError")
}

func TestWrongArrayDim(t *testing.T) {
	t.Parallel()
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
`, "TypeMismatchError")
}

func TestArrayType(t *testing.T) {
	t.Parallel()
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
`, "TypeMismatchError")
}

func TestDuplicateInParam(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    in  string  values,
    out float   sum,
    src py      "stages/sum_squares",
)
`, "DuplicateNameError")
}

func TestDuplicateOutParam(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    out string  sum,
    src py      "stages/sum_squares",
)
`, "DuplicateNameError")
}

func TestDuplicateSplitOut(t *testing.T) {
	t.Parallel()
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
`, "DuplicateNameError")
	testBadCompile(t, `
stage SUM_SQUARES(
    in  float[] values,
    out string  sum,
    src py      "stages/sum_squares",
) split using (
    in  float value,
    out float sum,
)
`, "DuplicateNameError")
}

func TestDuplicateCallable(t *testing.T) {
	t.Parallel()
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
`, "DuplicateNameError")
}

func TestMissingCallable(t *testing.T) {
	t.Parallel()
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
`, "ScopeNameError")
}

func TestBadDisable(t *testing.T) {
	t.Parallel()
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
`, "TypeMismatchError")
}

// Tests that preflight inputs cannot be bound to other calls in the pipeline.
func TestPreflightBadDepends(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage PREFLIGHT(
    in  int value,
    src py  "stages/preflight",
)

stage COMPUTE(
    in  int value,
    out int result,
    src py  "stages/compute",
)

pipeline THING(
    in  int  value,
    out int  result,
)
{
    call COMPUTE(
        value = self.value,
    )

    call PREFLIGHT(
        value = COMPUTE.result,
    ) using (
        preflight = true,
    )

    return (
        result = COMPUTE.result,
    )
}

call THING(
    value        = 1,
)
`, "PreflightBindingError")
}

// Tests that one cannot disable a preflight based on the output of
// another stage.
func TestDisablePreflightBad(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage PICK_DISABLE(
    in  int  value,
    out bool result,
    src py   "stages/pick",
)

stage PREFLIGHT(
    in  int value,
    src py  "stages/preflight",
)

stage COMPUTE(
    in  int value,
    out int result,
    src py  "stages/compute",
)

pipeline THING(
    in  int value,
    out int result,
)
{
    call PICK_DISABLE(
        value = self.value,
    )

    call PREFLIGHT(
        value = self.value,
    ) using (
        disabled  = PICK_DISABLE.result,
        preflight = true,
    )

    call COMPUTE(
        value = self.value,
    )

    return (
        result = COMPUTE.result,
    )
}

call THING(
    value = 1,
)
`, "PreflightBindingError")
}

func TestDuplicateCall(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage COMPUTE(
    in  int value,
    out int result,
    src py  "stages/compute",
)

pipeline THING(
    in  int  value,
    out int  result,
)
{
    call COMPUTE(
        value = self.value,
    )

    call COMPUTE(
        value = self.value,
    )

    return (
        result = COMPUTE.result,
    )
}

call THING(
    value        = 1,
)
`, "DuplicateCallError")
}

func TestMissingPipelineParam(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage COMPUTE(
    in  int value,
    out int result,
    src py  "stages/compute",
)

pipeline THING(
    in  int value1,
    in  int value2,
    out int result,
)
{
    call COMPUTE as COMPUTE1(
        value = self.value1,
    )

    call COMPUTE as COMPUTE2(
        value = self.value2,
    )

    return (
        result = COMPUTE2.result,
    )
}

call THING(
    value1 = 1,
)
`, "ArgumentNotSuppliedError")
}

func TestUnusedParam(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage COMPUTE(
    in  int value,
    out int result,
    src py  "stages/compute",
)

pipeline THING(
    in  int value1,
    in  int value2,
    out int result,
)
{
    call COMPUTE (
        value = self.value1,
    )

    return (
        result = COMPUTE.result,
    )
}

call THING(
    value1 = 1,
	value2 = 2,
)
`, "UnusedInputError")
}

// Checks that the top-level call cannot be preflight.
func TestDisableTopCall(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SQUARE(
    in  int   value,
    src py    "stages/square",
)

call SQUARE(
    value = 1,
) using (
    preflight = true,
)
`, "UnsupportedTagError")
}

// Checks that there is an error when calling a non-existent pipeline.
func TestMissingTopCall(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
call SQUARE(
    value = 1,
)
`, "ScopeNameError")
}

func TestDuplicateStructOutNames(t *testing.T) {
	t.Parallel()
	const src = `
struct X(
    file foo "" "foo.txt",
	file bar "" "foo.%s",
)
`
	testGood(t, fmt.Sprintf(src, "csv"))
	testBadCompile(t, fmt.Sprintf(src, "txt"),
		"DuplicateNameError")
}

func TestBadStructBinding(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
struct DEST(
    int x,
)

stage SOURCE(
    in  int x,
    out int y,
    src py "foo",
)

pipeline CALLING(
    in  int x,
    out DEST result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = {
            x: SOURCE.x,
        },
    )
}
`, "NoSuchOutputError")
	testBadCompile(t, `
struct DEST(
    int x,
)

stage SOURCE(
    in  int x,
    out int y,
    src py "foo",
)

pipeline CALLING(
    in  int x,
    out DEST result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = SOURCE,
    )
}
`, "no member x")
	testGood(t, `
struct DEST(
    int y,
)

stage SOURCE(
    in  int x,
    out int y,
    src py "foo",
)

pipeline CALLING(
    in  int x,
    out DEST result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = SOURCE,
    )
}
`)
	testBadCompile(t, `
struct DEST(
    int[] y,
)

stage SOURCE(
    in  int x,
    out int y,
    src py "foo",
)

pipeline CALLING(
    in  int x,
    out DEST result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = SOURCE,
    )
}
`, "differing array dimensions")
	testBadCompile(t, `
struct DEST(
    map<int> y,
)

stage SOURCE(
    in  int x,
    out int y,
    src py "foo",
)

pipeline CALLING(
    in  int x,
    out DEST result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = SOURCE,
    )
}
`, "not a map")
	testBadCompile(t, `
struct DEST(
    map<int[]> y,
)

stage SOURCE(
    in  int      x,
    out map<int> y,
    src py       "foo",
)

pipeline CALLING(
    in  int x,
    out DEST result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = SOURCE,
    )
}
`, "differing inner array dimensions")
	testBadCompile(t, `
struct DEST(
    int y,
)

stage SOURCE(
    in  int   x,
    out int[] y,
    src py    "foo",
)

pipeline CALLING(
    in  int x,
    out DEST result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = SOURCE,
    )
}
`, "differing array dimensions")
	testBadCompile(t, `
struct DEST(
    int y,
)

stage SOURCE(
    in  int      x,
    out map<int> y,
    src py       "foo",
)

pipeline CALLING(
    in  int x,
    out DEST result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = SOURCE,
    )
}
`, "unexpected map")
	testBadCompile(t, `
struct DEST(
    int y,
)

stage SOURCE(
    in  int    x,
    out DEST[] y,
    src py     "foo",
)

pipeline CALLING(
    in  int x,
    out DEST result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = SOURCE.y,
    )
}
`, "binding is an array")
	testBadCompile(t, `
struct DEST(
    int y,
)

stage SOURCE(
    in  int       x,
    out map<DEST> y,
    src py        "foo",
)

pipeline CALLING(
    in  int x,
    out DEST result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = SOURCE.y,
    )
}
`, "binding is a map")
	testBadCompile(t, `
struct DEST(
    int y,
)

stage SOURCE(
    in  int  x,
    out DEST y,
    src py   "foo",
)

pipeline CALLING(
    in  int x,
    out int result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = SOURCE.y.x,
    )
}
`, "could not evaluate SOURCE.y.x")
	testBadCompile(t, `
struct DEST(
    int y,
)

stage SOURCE(
    in  int  x,
    out DEST y,
    src py   "foo",
)

pipeline CALLING(
    in  DEST x,
    out int  result,
)
{
    call SOURCE(
        x = self.x.x,
    )

    return (
        result = SOURCE.y.y,
    )
}
`, "could not evaluate self.x.x")
}

func TestBadArrayBinding(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SOURCE(
    in  int     x,
    out float[] y,
    src py "foo",
)

pipeline CALLING(
    in  int   x,
    out int[] result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = SOURCE.y,
    )
}
`, "float cannot be assigned to int")
}

func TestBadMapBinding(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage SOURCE(
    in  int        x,
    out map<float> y,
    src py         "foo",
)

pipeline CALLING(
    in  int      x,
    out map<int> result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = SOURCE.y,
    )
}
`, "float cannot be assigned to int")
	testBadCompile(t, `
stage SOURCE(
    in  int x,
    out int y,
    src py  "foo",
)

pipeline CALLING(
    in  int      x,
    out map<int> result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = SOURCE.y,
    )
}
`, "not a typed map")
	testBadCompile(t, `
stage SOURCE(
    in  int   x,
    out int   y,
	out float z,
    src py    "foo",
)

pipeline CALLING(
    in  int      x,
    out map<int> result,
)
{
    call SOURCE(
        x = self.x,
    )

    return (
        result = {
			"y": SOURCE.y,
			"z": SOURCE.z,
		},
    )
}
`, "float cannot be assigned to int")
}

// Tests that typed maps which would be directories in final outs reject keys
// which are not legal posix file names.
func TestTypedMapKeys(t *testing.T) {
	t.Parallel()
	const pipeline = `
filetype json;

stage STAGE(
    in  json i,
    out json o,
    src py   "foo.py",
)

pipeline BAR(
    in  json      i,
    out map<json> o,
)
{
    call STAGE(
        i = self.i,
    )

    return(
        o = {
            "%s": STAGE.o,
        },
    )
}
`
	testGood(t, fmt.Sprintf(pipeline, "s2"))
	testBadCompile(t,
		fmt.Sprintf(pipeline, "a/s2"),
		"'/' is not allowed")
	testBadCompile(t,
		fmt.Sprintf(pipeline, ""),
		"empty string")
	testBadCompile(t,
		fmt.Sprintf(pipeline, ".."),
		"reserved name")
	testBadCompile(t,
		fmt.Sprintf(pipeline, `\x00`),
		"null character")
	testBadCompile(t,
		fmt.Sprintf(pipeline,
			`0123456789012345678901234567890123456789012345678901234567890123`+
				`0123456789012345678901234567890123456789012345678901234567890123`+
				`0123456789012345678901234567890123456789012345678901234567890123`+
				`0123456789012345678901234567890123456789012345678901234567890123`),
		"too long")
}

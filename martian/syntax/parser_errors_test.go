// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"testing"
)

// Check that a typo in a keyword causes the parser to fail.
func TestBadSyntax(t *testing.T) {
	testBadGrammar(t, `
# File comment

# Stage comment
# This describes the stage.
stage SUM_SQUARES(
    in  float[] values,
    # sum comment
    osut float   sum,
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
    values = [10.0, 2.0e1, 3.0e+1, 400.0e-1, 5e1, 6e+1, 700e-1],
)
`)
}

// Checks that non-integer mem_gb is a parse failure.
func TestBadMemGB(t *testing.T) {
	t.Parallel()
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

// Checks that non-integer threads is a parse failure.
func TestBadThreads(t *testing.T) {
	t.Parallel()
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

// Tests that disabled cannot be set to a constant value.
func TestConstDisable(t *testing.T) {
	t.Parallel()
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

// Checks that the top-level call cannot be disabled.
func TestPreflightTopCall(t *testing.T) {
	t.Parallel()
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

// Checks that there can be only one top-level call.
func TestDuplicateTopCall(t *testing.T) {
	t.Parallel()
	testBadGrammar(t, `
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

    return (
        result = COMPUTE.result,
    )
}

call THING(
    value        = 1,
)

call THING(
    value        = 2,
)
`)
}

// Check ban on nested maps.
func TestBadMapSyntax(t *testing.T) {
	t.Parallel()
	testBadGrammar(t, `
struct STRUCT_1(
    map<map<int>>   i1,
)
`)
	testBadGrammar(t, `
struct STRUCT_1(
    map<map>   i1,
)
`)
}

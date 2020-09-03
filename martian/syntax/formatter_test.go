//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Martian formatter tests.
//

package syntax

import (
	"io/ioutil"
	"testing"
)

func fmtTestSrc() []byte {
	srcb, err := ioutil.ReadFile("testdata/formatter_test.mro")
	if err != nil {
		panic(err)
	}
	return srcb
}

func TestFormatCommentedSrc(t *testing.T) {
	src := string(fmtTestSrc())
	if formatted, err := Format(src, "test", false, nil); err != nil {
		t.Errorf("Format error: %v", err)
	} else if formatted != src {
		diffLines(src, formatted, t)
	}
}

func BenchmarkFormat(b *testing.B) {
	srcFile := new(SourceFile)
	if ast, err := yaccParse(fmtTestSrc(),
		srcFile, makeStringIntern()); err != nil {
		b.Error(err)
	} else {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ast.format(false)
		}
	}
}

func TestFormatTopoSort(t *testing.T) {
	const src = `pipeline PIPELINE(
    in  int input,
    out int output1,
    out int output2,
)
{
    call STAGE_3(
        in1 = self.input,
        in2 = STAGE_2.output,
    )

    call STAGE as STAGE_1(
        input = self.input,
    )

    call STAGE as STAGE_2(
        input = self.input,
    )

    call STAGE as STAGE_4(
        input = STAGE_2.output,
    )

    call STAGE_3 as STAGE_5(
        in1 = self.input,
        in2 = STAGE_2.output,
    )

    return (
        output1 = STAGE_3.output,
        output2 = STAGE_5.output,
    )
}
`
	const expected = `pipeline PIPELINE(
    in  int input,
    out int output1,
    out int output2,
)
{
    call STAGE as STAGE_1(
        input = self.input,
    )

    call STAGE as STAGE_2(
        input = self.input,
    )

    call STAGE_3(
        in1 = self.input,
        in2 = STAGE_2.output,
    )

    call STAGE as STAGE_4(
        input = STAGE_2.output,
    )

    call STAGE_3 as STAGE_5(
        in1 = self.input,
        in2 = STAGE_2.output,
    )

    return (
        output1 = STAGE_3.output,
        output2 = STAGE_5.output,
    )
}
`
	if formatted, err := Format(src, "test", false, nil); err != nil {
		t.Errorf("Format error: %v", err)
	} else if formatted != expected {
		diffLines(expected, formatted, t)
	}
}

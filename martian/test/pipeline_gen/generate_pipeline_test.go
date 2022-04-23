//go:build bazel
// +build bazel

package pipeline_gen

import "testing"

func TestGeneratePipeline(t *testing.T) {
	src := GeneratePipeline([]float64{1, 2, 3}, false)
	const expected = `@include "pipeline_stages.mro"

call SUM_SQUARE_PIPELINE(
    values     = [
        1,
        2,
        3,
    ],
    disable_sq = false,
)
`
	if src != expected {
		t.Error(src, "!=", expected)
	}
}

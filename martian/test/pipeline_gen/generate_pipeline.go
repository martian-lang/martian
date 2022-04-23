//go:build bazel
// +build bazel

// Package pipeline_gen is used to test the bazel rules for mro2go.
//
// It cannot be compiled without bazel.
package pipeline_gen

import "github.com/martian-lang/martian/martian/syntax/ast_builder"

func GeneratePipeline(values []float64, disableSq bool) string {
	return ast_builder.MakeCallAst(&SumSquarePipelineArgs{
		Values:    values,
		DisableSq: disableSq,
	}).Format()
}

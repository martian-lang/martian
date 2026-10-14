// Copyright (c) 2026 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"testing"
)

// A pipeline called in a map call over a split whose keys are only known at
// runtime, with a subpipeline pruned at compile time by a literal `disabled`.
const prunedMergeSrc = `
stage GEN(
    out map<int> keys,
    src py       "stages/gen",
)

stage WORK(
    in  int x,
    out int y,
    src py  "stages/work",
)

pipeline SUB(
    in  int x,
    out int y,
)
{
    call WORK(
        x = self.x,
    )
    return (
        y = WORK.y,
    )
}

pipeline INNER(
    in  int  x,
    in  bool skip,
    out int  y,
    out int  z,
)
{
    call WORK as KEEP(
        x = self.x,
    )
    call SUB(
        x = self.x,
    ) using (
        disabled = self.skip,
    )
    return (
        y = KEEP.y,
        z = SUB.y,
    )
}

pipeline OUTER(
    in  bool          skip,
    out map<int>      ys,
    out map<int>      zs,
)
{
    call GEN()
    map call INNER(
        x    = split GEN.keys,
        skip = self.skip,
    )
    return (
        ys = INNER.y,
        zs = INNER.z,
    )
}

call OUTER(
    skip = true,
)
`

func resolvedOutput(t *testing.T, src, name string) Exp {
	t.Helper()
	_, _, ast, err := ParseSourceBytes([]byte(src), "merge_pruned_test.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ast.MakeCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	outs := graph.ResolvedOutputs()
	m, ok := outs.Exp.(*MapExp)
	if !ok {
		t.Fatalf("outputs are %T, not a map", outs.Exp)
	}
	e, ok := m.Value[name]
	if !ok {
		t.Fatalf("no output %q in %s", name, FormatExp(outs.Exp, ""))
	}
	return e
}

// The merged output of a pruned subpipeline in a map call keeps one entry per
// key of the split, and the keys are only known at runtime, so the merge must
// be resolved when the pipeline runs rather than treated as a literal.
func TestMergeOfPrunedSubpipelineNeedsRuntimeKeys(t *testing.T) {
	t.Run("kept", func(t *testing.T) {
		e := resolvedOutput(t, prunedMergeSrc, "ys")
		t.Log(FormatExp(e, ""))
		if !e.HasRef() {
			t.Error("merge of a live stage should have a reference")
		}
	})
	t.Run("pruned", func(t *testing.T) {
		e := resolvedOutput(t, prunedMergeSrc, "zs")
		t.Logf("%T %s", e, FormatExp(e, ""))
		if !e.HasRef() {
			t.Error("merge of a pruned subpipeline over runtime keys must be " +
				"resolved at runtime, but is treated as a literal")
		}
	})
}

// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

// Test for SplitExp.

package syntax

import "testing"

// Tests that SplitExp.BindingPath has a correct Source.
func TestSplitExpBindingSource(t *testing.T) {
	ast := testGood(t, `
pipeline MAPPED(
	in  int thingy,
	out int thing,
)
{
	return (
		thing = self.thingy,
	)
}

pipeline MAPPER(
	in  int[]    thingies,
	out MAPPED[] things,
)
{
	map call MAPPED(
		thingy = split self.thingies,
	)

	return (
		things = MAPPED,
	)
}

map call MAPPER(
	thingies = split [
		[
			1,
		],
		[
			2,
			3,
			4,
		],
	],
)
`)
	if ast == nil {
		return
	}
	graph, err := ast.MakeCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	mappedCall := graph.GetChildren()[0]
	thingy := mappedCall.ResolvedInputs()["thingy"].Exp.(*SplitExp)
	if thingy.Source.KnownLength() {
		t.Error("Not supposed to be known length.")
	}
	fork := map[*CallStm]CollectionIndex{
		graph.Call(): arrayIndex(0),
	}
	boundThingy, err := thingy.BindingPath("", fork)
	if err != nil {
		t.Fatal(err)
	}
	split := boundThingy.(*SplitExp).Source
	if !split.KnownLength() {
		t.Error("expected known length for", split.GoString(),
			"backing", boundThingy.GoString())
	} else if split.ArrayLength() != 1 {
		t.Errorf("%d != 1 backing %s", split.ArrayLength(),
			boundThingy.GoString())
	}
	fork[graph.Call()] = arrayIndex(1)
	boundThingy, err = thingy.BindingPath("", fork)
	if err != nil {
		t.Fatal(err)
	}
	split = boundThingy.(*SplitExp).Source
	if !split.KnownLength() {
		t.Error("expected known length for", split.GoString(),
			"backing", boundThingy.GoString())
	} else if split.ArrayLength() != 3 {
		t.Errorf("%d != 3 backing %s", split.ArrayLength(),
			boundThingy.GoString())
	}
}

func TestNestedStaticForks(t *testing.T) {
	ast := testGood(t, `
	stage STAGE(
		in  bool   truth,
		in  string str,
		in  int    num,
		src comp   "mock",
	)
	
	pipeline INNER(
		in bool   truth,
		in string str,
		in int[]  nums,
	)
	{
		map call STAGE(
			truth = self.truth,
			str   = self.str,
			num   = split self.nums,
		)
	
		return ()
	}
	
	pipeline OUTER(
		in bool     truth,
		in string[] strs,
		in int[]    nums,
	)
	{
		map call INNER(
			truth = self.truth,
			str   = split self.strs,
			nums  = self.nums,
		)
	
		return ()
	}
	
	map call OUTER(
		truth = split {
			"first":  true,
			"second": false,
		},
		strs  = split {
			"first": [
				"foo",
			],
			"second": [
				"bar",
				"baz",
			]
		},
		nums  = split {
			"first": [1],
			"second": [
				2,
				3,
			],
		},
	)`)
	if ast == nil {
		return
	}
	graph, err := ast.MakePipelineCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	if forks := graph.Children[0].GetChildren()[0].ForkRoots(); len(forks) != 3 {
		t.Errorf("Expected 3 forks, got %d", len(forks))
	} else {
		keys := forks[0].Split().Source.Keys()
		if len(keys) != 2 {
			t.Errorf("Number of keys %d != 2", len(keys))
		}
		if _, ok := keys["first"]; !ok {
			t.Error("missing key first")
		}
		if _, ok := keys["second"]; !ok {
			t.Error("missing key second")
		}
		if forks[1].Split().Source.KnownLength() {
			t.Error("The length shouldn't be known yet.")
		}
		if forks[2].Split().Source.KnownLength() {
			t.Error("The length shouldn't be known yet.")
		}
	}
}

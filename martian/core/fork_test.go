package core

import (
	"fmt"
	"testing"

	"github.com/martian-lang/martian/martian/syntax"
)

func TestMakeForkIds(t *testing.T) {
	_, _, ast, err := syntax.ParseSourceBytes([]byte(`
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
	truth = split [
		true,
		false,
	],
	strs  = [
		"bar",
		"baz",
	],
	nums  = [
		2,
		3,
	],
)
`), "static_forks.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ast.MakePipelineCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	forks := graph.Children[0].GetChildren()[0].ForkRoots()
	var ids ForkIdSet
	ids.MakeForkIds(forks, &ast.TypeTable)
	mkId := func(i, j, k arrayIndexFork) ForkId {
		return ForkId{
			&ForkSourcePart{
				Split: forks[0].Split(),
				Id:    &i,
			},
			&ForkSourcePart{
				Split: forks[1].Split(),
				Id:    &j,
			},
			&ForkSourcePart{
				Split: forks[2].Split(),
				Id:    &k,
			},
		}
	}
	expect := []ForkId{
		mkId(0, 0, 0),
		mkId(1, 0, 0),
		mkId(0, 1, 0),
		mkId(1, 1, 0),
		mkId(0, 0, 1),
		mkId(1, 0, 1),
		mkId(0, 1, 1),
		mkId(1, 1, 1),
	}
	if len(ids.List) != 8 {
		t.Errorf("expected %d ids, got %d", len(expect), len(ids.List))
	}
	for i, id := range ids.List {
		if !id.Equal(expect[i]) {
			t.Errorf("expected %v, got %v", expect[i].GoString(), id.GoString())
		}
	}
	for i, id := range ids.List {
		if s, err := id.ForkIdString(); err != nil {
			t.Error(err)
		} else if s != fmt.Sprintf("fork%d", i) {
			t.Errorf("expected fork%d, got %s", i, s)
		}
	}
}

// Test that fork IDs are correctly constructed in the case where the number
// of forks is known statically but where the number of forks in some nesting
// levels is different depending on the chosen parent fork.
func TestMakeNonUniformForkIds(t *testing.T) {
	_, _, ast, err := syntax.ParseSourceBytes([]byte(`
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
	truth = split [
		true,
		false,
	],
	strs  = split [
		[
			"foo",
		],
		[
			"bar",
			"baz",
		]
	],
	nums  = split [
		[1],
		[
			2,
			3,
		],
	],
)
`), "static_forks.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ast.MakePipelineCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	forks := graph.Children[0].GetChildren()[0].ForkRoots()
	var ids ForkIdSet
	ids.MakeForkIds(forks, &ast.TypeTable)
	innerSources := graph.Inputs["nums"].Exp.(*syntax.SplitExp).Value.(*syntax.ArrayExp).Value
	if len(innerSources) != 2 {
		t.Fatalf("%d != 2", len(innerSources))
	}
	mkId := func(i, j, k arrayIndexFork) ForkId {
		return ForkId{
			&ForkSourcePart{
				Split: forks[0].Split(),
				Id:    &i,
			},
			&ForkSourcePart{
				Split: forks[1].Split(),
				Id:    &j,
			},
			&ForkSourcePart{
				Split: forks[2].Split(),
				Id:    &k,
			},
		}
	}
	expect := []ForkId{
		mkId(0, 0, 0),
		mkId(1, 0, 0),
		mkId(1, 1, 0),
		mkId(1, 0, 1),
		mkId(1, 1, 1),
	}
	if len(ids.List) != 5 {
		t.Errorf("expected %d ids, got %d", len(expect), len(ids.List))
	}
	for i, id := range ids.List {
		if !id.Equal(expect[i]) {
			t.Errorf("expected %v, got %v", expect[i].GoString(), id.GoString())
		}
	}
	expectStr := [...]string{
		"fork0",
		"fork1_fork0_fork0",
		"fork1_fork1_fork0",
		"fork1_fork0_fork1",
		"fork1_fork1_fork1",
	}
	for i, id := range ids.List {
		if s, err := id.ForkIdString(); err != nil {
			t.Error(err)
		} else if s != expectStr[i] {
			t.Errorf("expected %s, got %s for %s", expectStr[i], s, id.GoString())
		}
	}
}

// Test that fork IDs are correctly constructed in the case where the number
// of forks is known statically but where the number of forks in some nesting
// levels is different depending on the chosen parent fork.
func TestMakeNonUniformMapForkIds(t *testing.T) {
	_, _, ast, err := syntax.ParseSourceBytes([]byte(`
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
)
`), "static_forks.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ast.MakePipelineCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	forks := graph.Children[0].GetChildren()[0].ForkRoots()
	var ids ForkIdSet
	ids.MakeForkIds(forks, &ast.TypeTable)
	innerSources := graph.Inputs["nums"].Exp.(*syntax.SplitExp).Value.(*syntax.MapExp).Value
	if len(innerSources) != 2 {
		t.Fatalf("%d != 2", len(innerSources))
	}
	mkId := func(i mapKeyFork, j, k arrayIndexFork) ForkId {
		return ForkId{
			&ForkSourcePart{
				Split: forks[0].Split(),
				Id:    &i,
			},
			&ForkSourcePart{
				Split: forks[1].Split(),
				Id:    &j,
			},
			&ForkSourcePart{
				Split: forks[2].Split(),
				Id:    &k,
			},
		}
	}
	expect := [...]ForkId{
		mkId("first", 0, 0),
		mkId("second", 0, 0),
		mkId("second", 1, 0),
		mkId("second", 0, 1),
		mkId("second", 1, 1),
	}
	if len(ids.List) != 5 {
		t.Errorf("expected %d ids, got %d", len(expect), len(ids.List))
	}
	for i, id := range ids.List {
		if !id.Equal(expect[i]) {
			t.Errorf("expected %v, got %v", expect[i].GoString(), id.GoString())
		}
	}
	expectStr := [...]string{
		"fork_first/fork0",
		"fork_second/fork0_fork0",
		"fork_second/fork1_fork0",
		"fork_second/fork0_fork1",
		"fork_second/fork1_fork1",
	}
	for i, id := range ids.List {
		if s, err := id.ForkIdString(); err != nil {
			t.Error(err)
		} else if s != expectStr[i] {
			t.Errorf("expected %s, got %s for %s", expectStr[i], s, id.GoString())
		}
	}
}

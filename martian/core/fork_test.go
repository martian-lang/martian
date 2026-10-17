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

// Tests that a map-keyed fork part following an array-indexed part is
// included in the fork ID string.  Previously the map key was dropped,
// so the forks for different keys got the same ID and the same directory.
func TestArrayThenMapForkIdString(t *testing.T) {
	_, _, ast, err := syntax.ParseSourceBytes([]byte(`
stage STAGE(
	in  int num,
	src comp "mock",
)

pipeline INNER(
	in int x,
)
{
	map call STAGE(
		num = split {
			"p": self.x,
			"q": 2,
		},
	)

	return ()
}

map call INNER(
	x = split [
		1,
		2,
	],
)
`), "array_then_map.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ast.MakePipelineCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	var ids ForkIdSet
	ids.MakeForkIds(graph.GetChildren()[0].ForkRoots(), &ast.TypeTable)
	expectStr := [...]string{
		"fork0/fork_p",
		"fork1/fork_p",
		"fork0/fork_q",
		"fork1/fork_q",
	}
	if len(ids.List) != len(expectStr) {
		t.Fatalf("expected %d ids, got %d", len(expectStr), len(ids.List))
	}
	for i, id := range ids.List {
		if s, err := id.ForkIdString(); err != nil {
			t.Error(err)
		} else if s != expectStr[i] {
			t.Errorf("expected %s, got %s for %s", expectStr[i], s, id.GoString())
		}
	}
}

// Tests that journal updates are matched to forks by ID, rather than by their
// position in the node's fork list, which differs from their IDs after forks
// are expanded at runtime.
func TestGetForkByJournalIndex(t *testing.T) {
	_, _, ast, err := syntax.ParseSourceBytes([]byte(`
stage STAGE(
	in  int num,
	src comp "mock",
)

call STAGE(
	num = 1,
)
`), "get_fork.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ast.MakeCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	node := Node{call: graph}
	// The order in which forks are added when expanding a 3x2 set of forks
	// where the first dimension isn't known statically.
	for _, id := range []string{
		"fork0", "fork3", "fork1", "fork2", "fork4", "fork5", "fork2%2Ffork_a",
	} {
		node.forks = append(node.forks, &Fork{
			fqname: graph.GetFqid() + "." + id,
		})
	}
	for _, c := range []struct {
		index string
		want  int
	}{
		{"0", 0},
		{"1", 2},
		{"2", 3},
		{"3", 1},
		{"5", 5},
		{"2%2Ffork_a", 6},
		{"_x", -1},
	} {
		f := node.getFork(c.index)
		if c.want < 0 {
			if f != nil {
				t.Errorf("fork%s: expected nil, got %s", c.index, f.fqname)
			}
		} else if f != node.forks[c.want] {
			got := "nil"
			if f != nil {
				got = f.fqname
			}
			t.Errorf("fork%s: expected %s, got %s",
				c.index, node.forks[c.want].fqname, got)
		}
	}
}

// Tests that fork IDs ending in an empty part, for a source which turned out
// to be empty or disabled at runtime, keep the indices of the parts before it.
// Previously, those indices were dropped, so the empty forks for every nonzero
// instance of an enclosing statically mapped pipeline got the ID "", and
// all shared the node's own directory.
func TestEmptyForkIdString(t *testing.T) {
	_, _, ast, err := syntax.ParseSourceBytes([]byte(`
stage MAKE(
	in  int   n,
	out int[] xs,
	src comp  "mock",
)

stage USE(
	in  int  n,
	src comp "mock",
)

pipeline LEAF(
	in int n,
)
{
	call MAKE(
		n = self.n,
	)

	map call USE(
		n = split MAKE.xs,
	)

	return ()
}

pipeline MID(
	in int n,
)
{
	call MAKE(
		n = self.n,
	)

	map call LEAF(
		n = split MAKE.xs,
	)

	return ()
}

map call MID(
	n = split [
		1,
		2,
		3,
	],
)
`), "empty_fork.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ast.MakePipelineCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	// MID -> LEAF -> USE
	use := graph.GetChildren()[1].GetChildren()[1]
	roots := use.ForkRoots()
	if len(roots) != 3 {
		t.Fatalf("expected 3 fork roots, got %d", len(roots))
	}
	arr := func(i, n int) *ForkSourcePart {
		id := arrayIndexFork(i)
		return &ForkSourcePart{Id: &id, Range: arrayLengthRange(n)}
	}
	empty := func() *ForkSourcePart {
		return &ForkSourcePart{Id: emptyFork{}, Range: arrayLengthRange(0)}
	}
	for _, c := range []struct {
		name   string
		expect string
		id     ForkId
	}{
		{
			name:   "first parent empty leaf",
			id:     ForkId{arr(0, 3), arr(0, 1), empty()},
			expect: "fork0",
		},
		{
			name:   "second parent empty leaf",
			id:     ForkId{arr(1, 3), arr(0, 1), empty()},
			expect: "fork1",
		},
		{
			name:   "third parent empty leaf",
			id:     ForkId{arr(2, 3), arr(0, 1), empty()},
			expect: "fork2",
		},
		{
			name:   "second parent empty middle",
			id:     ForkId{arr(1, 3), empty(), empty()},
			expect: "fork1",
		},
		{
			name:   "variable length middle empty leaf",
			id:     ForkId{arr(1, 3), arr(2, 3), empty()},
			expect: "fork1_fork2",
		},
		{
			name:   "variable length middle",
			id:     ForkId{arr(1, 3), arr(2, 3), arr(1, 2)},
			expect: "fork1_fork2_fork1",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			for i, part := range c.id {
				part.Split = roots[i].Split()
			}
			if s, err := c.id.ForkIdString(); err != nil {
				t.Error(err)
			} else if s != c.expect {
				t.Errorf("expected %q, got %q for %s",
					c.expect, s, c.id.GoString())
			}
		})
	}
}

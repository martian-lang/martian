// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"
)

func TestCallGraphNodeTypeMarshalText(t *testing.T) {
	if KindStage == KindPipeline {
		t.Error("node types are not distinct")
	}
	for _, k := range []CallGraphNodeType{
		KindStage,
		KindPipeline,
	} {
		b, err := k.MarshalText()
		if err != nil {
			t.Error(err)
		}
		var k2 CallGraphNodeType
		if err := k2.UnmarshalText(b); err != nil {
			t.Error(err)
		}
		if k2 != k {
			t.Errorf("%v != %v", k2, k)
		}
	}
}

func TestAstMakeCallGraph(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/resolve_test.mro")
	if err != nil {
		t.Fatal(err)
	}
	_, _, ast, err := ParseSourceBytes(src, "resolve_test.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ast.MakeCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	if graph.GetFqid() != "POINT_MAPPER" {
		t.Error(graph.GetFqid(), " != POINT_MAPPER")
	}
	nodes := graph.NodeClosure()
	outputs := graph.ResolvedOutputs()
	if out := FormatExp(outputs.Exp, ""); out != `{
    result: {
        "1": POINT_MAPPER.PIPE1.POINT_MAKER,
        "2": POINT_MAPPER.PIPE2.POINT_MAKER,
        "3": POINT_MAPPER.PIPE3.POINT_MAKER,
    },
    xs: {
        "1": [
            POINT_MAPPER.PIPE1.POINT_MAKER.point.x,
            5,
        ],
        "2": [
            POINT_MAPPER.PIPE2.POINT_MAKER.point.x,
            7,
        ],
        "3": [
            POINT_MAPPER.PIPE3.POINT_MAKER.point.x,
            3,
        ],
    },
}` {
		t.Errorf("Incorrect pipeline output:\n%s", out)
	}
	if n := nodes["POINT_MAPPER.PIPE1.POINT_MAKER"]; n == nil {
		t.Error("No bound node for POINT_MAPPER.PIPE1.POINT_MAKER")
	} else if ins := FormatExp(n.ResolvedInputs()["points"].Exp, ""); ins != `[
    {
        x: 5,
        y: 6,
    },
    {
        x: 1,
        y: 2,
    },
]` {
		t.Errorf("Incorrect inputs to POINT_MAPPER.PIPE1.POINT_MAKER:\n%s", ins)
	}
	if n := nodes["POINT_MAPPER.POINT_USER"]; n == nil {
		t.Error("No bound node for POINT_MAPPER.POINT_USER")
	} else {
		if ins := FormatExp(n.ResolvedInputs()["points"].Exp, ""); ins != `[{
    x: 5,
    y: 6,
}]` {
			t.Errorf(
				"Incorrect inputs POINT_MAPPER.PIPE1.POINT_MAKER.points:\n%s",
				ins)
		}
		if ins := FormatExp(n.ResolvedInputs()["point"].Exp, ""); ins != `{
    x: 5,
    y: 6,
}` {
			t.Errorf(
				"Incorrect inputs POINT_MAPPER.PIPE1.POINT_MAKER.point:\n%s",
				ins)
		}
		if ins := FormatExp(n.ResolvedInputs()["xs"].Exp, ""); ins != `[
    POINT_MAPPER.PIPE1.POINT_MAKER.point.x,
    5,
]` {
			t.Errorf(
				"Incorrect inputs POINT_MAPPER.PIPE1.POINT_MAKER.xs:\n%s",
				ins)
		}
		if ins := FormatExp(n.ResolvedInputs()["ys"].Exp, ""); ins != `{
    "three": 10.2,
}` {
			t.Errorf(
				"Incorrect inputs POINT_MAPPER.PIPE1.POINT_MAKER.ys:\n%s",
				ins)
		}
		if ins := FormatExp(n.ResolvedInputs()["mpset"].Exp, ""); ins != `{
    "foo": {
        extra: "nope",
        point: {
            x: 3,
            y: 4,
        },
        points: [
            POINT_MAPPER.PIPE3.POINT_MAKER.point,
            {
                x: 3,
                y: 4,
            },
        ],
    },
}` {
			t.Errorf(
				"Incorrect inputs POINT_MAPPER.PIPE1.POINT_MAKER.mpset:\n%s",
				ins)
		}
	}
}

func TestCallGraphFindRefs(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/resolve_test.mro")
	if err != nil {
		t.Fatal(err)
	}
	_, _, ast, err := ParseSourceBytes(src, "resolve_test.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ast.MakeCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	refs := FindRefs(graph.ResolvedOutputs().Exp)
	if len(refs) != 3 {
		t.Errorf("Expected 3 bound stages, got %d", len(refs))
	}
	for _, s := range []string{
		"POINT_MAPPER.PIPE1.POINT_MAKER",
		"POINT_MAPPER.PIPE2.POINT_MAKER",
		"POINT_MAPPER.PIPE3.POINT_MAKER",
	} {
		if p, ok := refs[s]; !ok {
			t.Errorf("No reference to %s found", s)
		} else if len(p) != 2 {
			t.Errorf("Expected 2 binding paths, got %d", len(p))
		} else {
			if p[0] != "" && p[1] != "" {
				t.Errorf("Expected root binding for %s", s)
			}
			if p[0] != "point.x" && p[1] != "point.x" {
				t.Errorf("Expected binding %s.point.x", s)
			}
		}
	}
}

func TestResolvedBindingFindRefs(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/resolve_test.mro")
	if err != nil {
		t.Fatal(err)
	}
	_, _, ast, err := ParseSourceBytes(src, "resolve_test.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ast.MakeCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	ps := ast.TypeTable.Get(TypeId{Tname: "POINT_SET"})
	if ps == nil {
		t.Fatal("no POINT_SET type")
	}
	refs, err := graph.ResolvedOutputs().FindRefs(&ast.TypeTable)
	if err != nil {
		t.Error(err)
	}
	expected := []BoundReference{
		{
			Exp: &RefExp{
				Id: "POINT_MAPPER.PIPE1.POINT_MAKER",
			},
			Type: ps,
		},
		{
			Exp: &RefExp{
				Id: "POINT_MAPPER.PIPE2.POINT_MAKER",
			},
			Type: ps,
		},
		{
			Exp: &RefExp{
				Id: "POINT_MAPPER.PIPE3.POINT_MAKER",
			},
			Type: ps,
		},
		{
			Exp: &RefExp{
				Id:       "POINT_MAPPER.PIPE1.POINT_MAKER",
				OutputId: "point.x",
			},
			Type: &builtinInt,
		},
		{
			Exp: &RefExp{
				Id:       "POINT_MAPPER.PIPE2.POINT_MAKER",
				OutputId: "point.x",
			},
			Type: &builtinInt,
		},
		{
			Exp: &RefExp{
				Id:       "POINT_MAPPER.PIPE3.POINT_MAKER",
				OutputId: "point.x",
			},
			Type: &builtinInt,
		},
	}
	if len(refs) != len(expected) {
		b, err := json.MarshalIndent(refs, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(string(b))
		t.Errorf("Expected 6 refs, got %d", len(refs))
	}
	for i, expect := range expected {
		if refs[i].Exp.Id != expect.Exp.Id {
			t.Errorf("[%d].Id: %q != %q", i,
				refs[i].Exp.Id, expect.Exp.Id)
		}
		if refs[i].Exp.OutputId != expect.Exp.OutputId {
			t.Errorf("[%d].OutputId: %q != %q", i,
				refs[i].Exp.OutputId, expect.Exp.OutputId)
		}
		if refs[i].Type != expect.Type {
			t.Errorf("[%d].Type: %q != %q", i,
				refs[i].Type.GetId().str(), expect.Type.GetId().str())
		}
	}
}

func TestSerializeCallGraph(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/resolve_test.mro")
	if err != nil {
		t.Fatal(err)
	}
	_, _, ast, err := ParseSourceBytes(src, "resolve_test.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ast.MakeCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	var dest strings.Builder
	enc := json.NewEncoder(&dest)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(graph); err != nil {
		t.Fatal(err)
	}
	expectB, err := ioutil.ReadFile("testdata/resolve_test.json")
	if err != nil {
		t.Fatal(err)
	}
	expect := string(expectB)
	if s := dest.String(); expect != s {
		diffLines(expect, s, t)
	}
}

func TestSerializeMapCallGraph(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/map_call_test.mro")
	if err != nil {
		t.Fatal(err)
	}
	_, _, ast, err := ParseSourceBytes(src, "map_call_test.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ast.MakeCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	var dest strings.Builder
	enc := json.NewEncoder(&dest)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(graph); err != nil {
		t.Fatal(err)
	}
	expectB, err := ioutil.ReadFile("testdata/map_call_test.json")
	if err != nil {
		t.Fatal(err)
	}
	expect := string(expectB)
	if s := dest.String(); expect != s {
		diffLines(expect, s, t)
	}
}

func makeSplitMapExp(t *testing.T, src string) (*MapExp, MapCallSource) {
	t.Helper()
	var parser Parser
	if exp, err := parser.ParseValExp([]byte(src)); err != nil {
		t.Error(err)
	} else if m, ok := exp.(*MapExp); !ok {
		t.Errorf("Expected map, got %T", exp)
	} else {
		var src MapCallSource
		for k, v := range m.Value {
			if v, ok := v.(MapCallSource); ok &&
				v.CallMode() != ModeNullMapCall &&
				v.KnownLength() && src == nil {
				src = v
			}
			m.Value[k] = &SplitExp{
				Value:  v,
				Source: src,
			}
		}
		return m, src
	}
	return nil, nil
}

func TestMapInvertArray(t *testing.T) {
	if ma, src := makeSplitMapExp(t, `{
		a: [1,2,null,3,],
		b: [4,5,6,7,],
		c: null,
	}`); ma != nil {
		if a, err := unsplit(ma, make(map[MapCallSource]CollectionIndex, 1),
			ForkRootList{&CallGraphStage{Source: src}}); err != nil {
			t.Error(err)
		} else if aa, ok := a.(*ArrayExp); !ok {
			t.Errorf("Expected map, got %T", a)
		} else if len(aa.Value) != 4 {
			t.Errorf("Expected 4 elements, got %d", len(aa.Value))
		} else {
			var s strings.Builder
			aa.format(&s, "\t\t")
			const expect = `[
		    {
		        a: 1,
		        b: 4,
		        c: null,
		    },
		    {
		        a: 2,
		        b: 5,
		        c: null,
		    },
		    {
		        a: null,
		        b: 6,
		        c: null,
		    },
		    {
		        a: 3,
		        b: 7,
		        c: null,
		    },
		]`
			if s := s.String(); s != expect {
				diffLines(expect, s, t)
			}
		}
	}
	if ma, src := makeSplitMapExp(t, `{
		a: [1,],
		b: {"a": 1},
		c: null,
	}`); ma != nil {
		if aa, err := unsplit(ma, make(map[MapCallSource]CollectionIndex, 1),
			ForkRootList{&CallGraphStage{Source: src}}); err == nil {
			t.Error("expected failure")
			t.Log(FormatExp(aa, ""))
		}
	}
}

func TestMapInvertMap(t *testing.T) {
	if ma, src := makeSplitMapExp(t, `{
		a: {"x":1,"y":2,"z":null,},
		b: {"x":4,"y":5,"z":6,},
		c: null,
	}`); ma != nil {
		if a, err := unsplit(ma, make(map[MapCallSource]CollectionIndex, 1),
			ForkRootList{&CallGraphStage{Source: src}}); err != nil {
			t.Error(err)
		} else if aa, ok := a.(*MapExp); !ok {
			t.Errorf("Expected map, got %T", a)
		} else if len(aa.Value) != 3 {
			t.Errorf("Expected 3 elements, got %d", len(aa.Value))
		} else {
			var s strings.Builder
			aa.format(&s, "\t\t")
			const expect = `{
		    "x": {
		        a: 1,
		        b: 4,
		        c: null,
		    },
		    "y": {
		        a: 2,
		        b: 5,
		        c: null,
		    },
		    "z": {
		        a: null,
		        b: 6,
		        c: null,
		    },
		}`
			if s := s.String(); s != expect {
				diffLines(expect, s, t)
			}
		}
	}
	if ma, src := makeSplitMapExp(t, `{
		a: [1,],
		b: {"a": 1},
		c: null,
	}`); ma != nil {
		if aa, err := unsplit(ma, make(map[MapCallSource]CollectionIndex, 1),
			ForkRootList{&CallGraphStage{Source: src}}); err == nil {
			t.Error("expected failure")
			t.Log(FormatExp(aa, ""))
		}
	}
}

// Test a pipeline which uses a map call to reshape a struct of maps into a
// map of structs.
func TestResolveReshapeMap(t *testing.T) {
	t.Parallel()
	ast := testGood(t, `
pipeline METRIC1(
    out map<int> metric,
)
{
    return (
        metric = {
            "foo": 1,
            "bar": 2,
            "baz": 3,
        },
    )
}

pipeline METRIC2(
    out map<int> metric,
)
{
    return (
        metric = {
            "foo": 4,
            "bar": 5,
            "baz": 6,
        },
    )
}

pipeline METRIC(
    in  int a,
    in  int b,
    out int a,
    out int b,
)
{
    return (
        a = self.a,
        b = self.b,
    )
}

pipeline METRICS(
    out map<METRIC> metrics,
)
{
    call METRIC1()
    call METRIC2()

    map call METRIC(
        a = split METRIC1.metric,
        b = split METRIC2.metric,
    )

    return (
        metrics = METRIC,
    )
}

call METRICS()
`)
	if ast == nil {
		return
	}
	graph, err := ast.MakeCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	result := FormatExp(graph.ResolvedOutputs().Exp, "")
	expect := `{
    metrics: {
        "bar": {
            a: 2,
            b: 5,
        },
        "baz": {
            a: 3,
            b: 6,
        },
        "foo": {
            a: 1,
            b: 4,
        },
    },
}`
	if result != expect {
		diffLines(expect, result, t)
	}
}

func TestResolveDisableExp(t *testing.T) {
	result, err := resolveDisableExp(&BoolExp{Value: true}, nil)
	if err != nil {
		t.Error(err)
	}
	if len(result) != 1 {
		t.Errorf("len %d != 1", len(result))
	} else if b, ok := result[0].(*BoolExp); !ok {
		t.Errorf("Non-boolean %T %s", result[0], result[0].GoString())
	} else if !b.Value {
		t.Error("expected true")
	}

	result, err = resolveDisableExp(&SplitExp{
		Value: &ArrayExp{
			Value: []Exp{
				&BoolExp{Value: true},
				&BoolExp{Value: true},
			},
		},
	}, nil)
	if err != nil {
		t.Error(err)
	}
	if len(result) != 1 {
		t.Errorf("len %d != 1", len(result))
	} else if b, ok := result[0].(*BoolExp); !ok {
		t.Errorf("Non-boolean %T %s", result[0], result[0].GoString())
	} else if !b.Value {
		t.Error("expected true")
	}

	result, err = resolveDisableExp(&SplitExp{
		Value: &ArrayExp{
			Value: []Exp{
				&BoolExp{Value: false},
				&BoolExp{Value: false},
			},
		},
	}, nil)
	if err != nil {
		t.Error(err)
	}
	if len(result) != 0 {
		t.Errorf("len %d != 0", len(result))
	}

	result, err = resolveDisableExp(&SplitExp{
		Value: &ArrayExp{
			Value: []Exp{
				&BoolExp{Value: true},
				&BoolExp{Value: false},
			},
		},
	}, nil)
	if err != nil {
		t.Error(err)
	}
	if len(result) != 1 {
		t.Errorf("len %d != 1", len(result))
	} else if _, ok := result[0].(*SplitExp); !ok {
		t.Errorf("Non-split %T %s", result[0], result[0].GoString())
	}

	result, err = resolveDisableExp(&SplitExp{
		Value: &ArrayExp{
			Value: []Exp{
				&BoolExp{Value: true},
				&BoolExp{Value: true},
			},
		},
	}, result)
	if err != nil {
		t.Error(err)
	}
	if len(result) != 1 {
		t.Errorf("len %d != 1", len(result))
	} else if b, ok := result[0].(*BoolExp); !ok {
		t.Errorf("Non-boolean %T %s", result[0], result[0].GoString())
	} else if !b.Value {
		t.Error("expected true")
	}

	result, err = resolveDisableExp(&SplitExp{
		Value: &MapExp{
			Kind: KindMap,
			Value: map[string]Exp{
				"foo": &BoolExp{Value: true},
				"bar": &BoolExp{Value: true},
			},
		},
	}, nil)
	if err != nil {
		t.Error(err)
	}
	if len(result) != 1 {
		t.Errorf("len %d != 1", len(result))
	} else if b, ok := result[0].(*BoolExp); !ok {
		t.Errorf("Non-boolean %T %s", result[0], result[0].GoString())
	} else if !b.Value {
		t.Error("expected true")
	}
}

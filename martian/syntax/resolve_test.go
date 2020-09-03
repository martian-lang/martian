// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

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
		if ins := FormatExp(n.ResolvedInputs()["points"].Exp, ""); ins != `[
    {
        x: 5,
        y: 6,
    },
]` {
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

func TestAstMakeStageCallGraph(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/call_stage.mro")
	if err != nil {
		t.Fatal(err)
	}
	_, _, ast, err := ParseSourceBytes(src, "testdata/call_stage.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ast.MakePipelineCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAstMakeCallGraphMappedPipeline(t *testing.T) {
	_, _, ast, err := ParseSourceBytes([]byte(`
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
	map call MAPPED as FIRST(
		thingy = split self.thingies,
	)

	map call MAPPED as SECOND(
		thingy = split FIRST.thing,
	)

	return (
		things = SECOND,
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
`), "mapped_pipeline.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ast.MakeCallGraph("", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	outs := graph.ResolvedOutputs()
	if s := outs.Type.TypeId().str(); s != "MAPPER[]" {
		t.Errorf("%s != MAPPER[]", s)
	}
	const expected = `[
	    {
	        things: [
	            {
	                thing: 1,
	            },
	        ],
	    },
	    {
	        things: [
	            {
	                thing: 2,
	            },
	            {
	                thing: 3,
	            },
	            {
	                thing: 4,
	            },
	        ],
	    },
	]`
	if s := FormatExp(outs.Exp, "\t"); err != nil {
		t.Error(err)
	} else if s != expected {
		t.Errorf("%s != "+expected, s)
	}
}

func TestAstMakeCallGraphFailures(t *testing.T) {
	t.Parallel()
	// First check that assignment of struct to map works if it doesn't
	// contain a reference.
	ast := testGood(t, `
struct Foo(
	int foo,
)

stage BAZ(
	in  map  foo,
	out int  foo,
	out file bar,
	src comp "nope",
)

pipeline FINE(
	in  Foo foo,
	out int foo,
)
{
	call BAZ(
		foo = self.foo,
	)

	return (
		foo = BAZ.foo,
	)

	retain (
		BAZ.bar,
	)
}

pipeline ALSO_FINE(
	out int foo,
)
{
	call FINE(
		foo = {
			foo: 1,
		},
	)

	return(
		foo = FINE.foo,
	)
}

call ALSO_FINE()
`)
	if ast == nil {
		return
	}
	if _, err := ast.MakeCallGraph("", ast.Call); err != nil {
		t.Error(err)
	}

	ast = testGood(t, `
struct Foo(
	int foo,
)

stage BAR(
	out int  foo,
	out int  bar,
	src comp "nope",
)

stage BAZ(
	in  map  foo,
	out int  foo,
	src comp "nope",
)

pipeline FINE(
	in  Foo foo,
	out int foo,
)
{
	call BAZ(
		foo = self.foo,
	)

	return (
		foo = BAZ.foo,
	)
}

pipeline BROKEN(
	out int foo,
)
{
	call BAR()

	call FINE(
		foo = {
			foo: BAR.foo,
		},
	)

	return(
		foo = FINE.foo,
	)
}

call BROKEN()
`)
	if ast == nil {
		return
	}
	if _, err := ast.MakeCallGraph("", ast.Call); err == nil {
		t.Error("expected failure")
	} else if s := err.Error(); !strings.Contains(s, "contains reference") {
		t.Error("expected failure due to contains reference, got", s)
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
				refs[i].Type.TypeId().str(), expect.Type.TypeId().str())
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
	nodes := graph.NodeClosure()
	ones := nodes["SOME_STATIC.LEN.ONES"]
	if ones == nil {
		t.Error("Node SOME_STATIC.LEN.ONES not found")
	} else if forks := ones.ForkRoots(); len(forks) != 1 {
		t.Errorf("Expected 1 forks, got %d", len(forks))
	}
	ones = nodes["SOME_STATIC.VALUES1.LEN.ONES"]
	if ones == nil {
		t.Error("Node SOME_STATIC.VALUES1.LEN.ONES not found")
	} else if forks := ones.ForkRoots(); len(forks) != 2 {
		t.Errorf("Expected 2 forks, got %d", len(forks))
	} else if keys := forks[0].Split().Source.Keys(); len(keys) != 2 {
		t.Errorf("Outer fork key count %d != 2", len(keys))
	} else if forks[1].Split().Source.KnownLength() {
		t.Error("Inner split length should be unknown:",
			forks[1].Split().GoString(),
			"with map source",
			forks[1].Split().Source.GoString())
	}
	ones = nodes["SOME_STATIC.VALUES2.LEN.ONES"]
	if ones == nil {
		t.Error("Node SOME_STATIC.VALUES2.LEN.ONES not found")
	} else {
		inputs := ones.ResolvedInputs()
		for k, input := range inputs {
			brefs, err := input.FindRefs(&ast.TypeTable)
			if err != nil {
				t.Error("input", k, "bound reference error:", err)
			}
			refs := input.Exp.FindRefs()
			if len(refs) != 0 && len(brefs)+1 != len(refs) {
				t.Log(input.Exp.GoString())
				t.Errorf("%d + 1 != %d", len(brefs), len(refs))
			}
			for i, ref := range refs {
				if i < len(brefs) {
					if ref.Id != brefs[i].Exp.Id {
						t.Errorf("%d: %s != %s", i, ref.Id, brefs[i].Exp.Id)
					} else if ref.OutputId != brefs[i].Exp.OutputId {
						t.Errorf("%d: %s != %s", i, ref.GoString(), brefs[i].Exp.GoString())
					}
				} else if i < len(refs)-1 {
					t.Error("missing bound ref", ref.GoString())
				}
			}
			if len(brefs) > len(refs) {
				for _, ref := range brefs[len(refs):] {
					t.Error("missing ref", ref.GoString())
				}
			}
		}
	}
}

func TestSerializeDisableCallGraph(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/disable_pipeline.mro")
	if err != nil {
		t.Fatal(err)
	}
	_, _, ast, err := ParseSourceBytes(src, "testdata/disable_pipeline.mro", nil, false)
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
	expectB, err := ioutil.ReadFile("testdata/disable_pipeline.json")
	if err != nil {
		t.Fatal(err)
	}
	expect := string(expectB)
	if s := dest.String(); expect != s {
		diffLines(expect, s, t)
	}
}

func TestSerializeDisableBindingCallGraph(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/disable_bindings.mro")
	if err != nil {
		t.Fatal(err)
	}
	_, _, ast, err := ParseSourceBytes(src, "disable_bindings.mro", nil, false)
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
	expectB, err := ioutil.ReadFile("testdata/disable_bindings.json")
	if err != nil {
		t.Fatal(err)
	}
	expect := string(expectB)
	if s := dest.String(); expect != s {
		diffLines(expect, s, t)
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

func TestEmptyPipelineResolve(t *testing.T) {
	t.Parallel()
	ast := testGood(t, `
pipeline DUMMY(
	in  int foo,
	out int foo,
)
{
	return (
		foo = self.foo,
	)
}

call DUMMY(
	foo = 0,
)
`)
	if ast == nil {
		return
	}
	graph, err := ast.MakeCallGraph("ID.", ast.Call)
	if err != nil {
		t.Fatal(err)
	}
	var dest strings.Builder
	enc := json.NewEncoder(&dest)
	enc.SetEscapeHTML(false)
	enc.SetIndent("\t", "\t")
	if err := enc.Encode(graph); err != nil {
		t.Fatal(err)
	}
	const expect = `{
		"fqid": "ID.DUMMY",
		"inputs": {
			"foo": {
				"expression": 0,
				"type": "int"
			}
		},
		"outputs": {
			"expression": {
				"foo": 0
			},
			"type": "DUMMY"
		},
		"children": null
	}
`
	if s := dest.String(); expect != s {
		diffLines(expect, s, t)
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

func TestGraphNameLengthCheck(t *testing.T) {
	ast := testGood(t, `
stage VERY_DEEP_PIPELINE_STAGE(
	out int foo,
	src comp "nope",
)

pipeline VERY_DEEP_PIPELINE_PIPELINE1(
	out int foo,
)
{
	call VERY_DEEP_PIPELINE_STAGE()

	return (
		* = VERY_DEEP_PIPELINE_STAGE,
	)
}

pipeline VERY_DEEP_PIPELINE_PIPELINE2(
	out int foo,
)
{
	call VERY_DEEP_PIPELINE_PIPELINE1 as SUPER_DUPER_EXTRA_LONG_RIDICULOUS_VERY_DEEP_PIPELINE_PIPELINE()

	return (
		* = SUPER_DUPER_EXTRA_LONG_RIDICULOUS_VERY_DEEP_PIPELINE_PIPELINE,
	)
}


pipeline VERY_DEEP_PIPELINE_PIPELINE3(
	out int foo,
)
{
	call VERY_DEEP_PIPELINE_PIPELINE2 as SUPER_DUPER_EXTRA_LONG_RIDICULOUS_VERY_DEEP_PIPELINE_PIPELINE()

	return (
		* = SUPER_DUPER_EXTRA_LONG_RIDICULOUS_VERY_DEEP_PIPELINE_PIPELINE,
	)
}

pipeline VERY_DEEP_PIPELINE_PIPELINE4(
	out int foo,
)
{
	call VERY_DEEP_PIPELINE_PIPELINE3 as SUPER_DUPER_EXTRA_LONG_RIDICULOUS_VERY_DEEP_PIPELINE_PIPELINE()

	return (
		* = SUPER_DUPER_EXTRA_LONG_RIDICULOUS_VERY_DEEP_PIPELINE_PIPELINE,
	)
}

call VERY_DEEP_PIPELINE_PIPELINE4()
`)
	if ast == nil {
		return
	}
	g, err := ast.MakeCallGraph("irrelevant.", ast.Call)
	if err == nil {
		for len(g.GetChildren()) > 0 {
			g = g.GetChildren()[0]
		}
		t.Log(len(g.GetFqid()), ":", g.GetFqid())
		t.Error("Expected an error.")
	} else if !strings.Contains(err.Error(), "length of id string") {
		t.Errorf("Expected string too long error, got %q", err.Error())
	}
}

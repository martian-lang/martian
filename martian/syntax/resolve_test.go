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

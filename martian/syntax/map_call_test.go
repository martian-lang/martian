//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Tests for map call compilation.
//

package syntax

import "testing"

// Check that map call types are validated and that their return values are
// correct.
func TestSimpleMapCall(t *testing.T) {
	t.Parallel()
	testGood(t, `
stage THING(
    in  int  stuff,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  int[]    arr,
    in  map<int> maps,
    out int[]    a_result,
    out map<int> m_result,
)
{
    map call THING as THING1(
        stuff = split self.arr,
    )

    map call THING as THING2(
        stuff = split self.maps,
    )

    return(
        a_result = THING1.foo,
        m_result = THING2.foo,
    )
}
`)
	testGood(t, `
stage THING(
    in  int   stuff,
    out int[] foo,
    src comp  "nope",
)

pipeline THINGIFY(
    in  int[]      arr,
    in  map<int>   maps,
    out int[][]    a_result,
    out map<int[]> m_result,
)
{
    map call THING as THING1(
        stuff = split self.arr,
    )

    map call THING as THING2(
        stuff = split self.maps,
    )

    return(
        a_result = THING1.foo,
        m_result = THING2.foo,
    )
}
`)
	testGood(t, `
stage THING(
    in  int        stuff,
    out map<int[]> foo,
    src comp       "nope",
)

pipeline THINGIFY(
    in  int[]      arr,
    out map<int[]>[]    a_result,
)
{
    map call THING as THING1(
        stuff = split self.arr,
    )

    return(
        a_result = THING1.foo,
    )
}
`)
}

// Tests that various ways of splitting over multiple parameters works as
// expected.
func TestMapCallMergedArraySplits(t *testing.T) {
	t.Parallel()
	testGood(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    out int[] a_result,
)
{
    map call THING as THING1(
        stuff1 = split [
            1,
            2,
        ],
        stuff2 = split [
            1,
            2,
        ],
    )
    
    return (
        a_result = THING1.foo,
    )
}
`)
	testGood(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    in  int  stuff3,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  int[] arr,
    out int[] a_result,
)
{
    map call THING as THING1(
        stuff1 = split [
            1,
            2,
        ],
        stuff2 = split [
            1,
            2,
        ],
        stuff3 = split self.arr,
    )
    
    return (
        a_result = THING1.foo,
    )
}
`)
	testGood(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    out map<int> a_result,
)
{
map call THING as THING1(
    stuff1 = split {
        "b": 1,
        "a": 2,
    },
    stuff2 = split {
        "a": 1,
        "b": 2,
    },
)

return (
    a_result = THING1.foo,
)
}
`)
	testGood(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    in  int  stuff3,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    out map<int> a_result,
)
{
    map call THING as THING1(
        stuff1 = split {
            "b": 1,
            "a": 2,
        },
        stuff2 = split {
            "a": 1,
            "b": 2,
        },
        stuff3 = split {
            "a": 4,
            "b": 5,
        },
    )
    
    return (
        a_result = THING1.foo,
    )
}
`)
	testGood(t, `
stage THING(
    in  int  stuff,
    in  int  more_stuff,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  int[]    arr,
    in  map<int> maps,
    out int[]    a_result,
    out map<int> m_result,
)
{
    map call THING as THING1(
        stuff      = split self.arr,
        more_stuff = split [1,2,3],
    )

    map call THING as THING2(
        stuff      = split self.maps,
        more_stuff = split { "a": 1, "b": 2 },
    )

    return(
        a_result = THING1.foo,
        m_result = THING2.foo,
    )
}
`)
	testGood(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  int[]    arr,
    in  map<int> maps,
    out int[]    a_result,
    out map<int> m_result,
)
{
    map call THING as THING1(
        stuff1 = split [1,2,3],
        stuff2 = split self.arr,
    )

    map call THING as THING2(
        stuff1 = split { "a": 1, "b": 2 },
        stuff2 = split self.maps,
    )

    return(
        a_result = THING1.foo,
        m_result = THING2.foo,
    )
}
`)
}

// Tests various cases of invalid splits.
func TestMapCallBadSplits(t *testing.T) {
	t.Parallel()
	testBadGrammar(t, `
stage THING(
    in  int  stuff,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  int   val,
    out int[] a_result,
)
{
    map call THING as THING1(
        stuff = self.val,
    )

    return (
        a_result = THING1.foo,
    )
}
`)
	testBadCompile(t, `
stage THING(
    in  int  stuff,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  int   val,
    out int[] a_result,
)
{
    map call THING as THING1(
        stuff = split self.val,
    )

    return (
        a_result = THING1.foo,
    )
}
`, "binding is not a collection")
	testBadCompile(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    out int[] a_result,
)
{
    map call THING as THING1(
        stuff1 = split [
            1,
            2,
        ],
        stuff2 = split [
            1,
            2,
            3,
        ],
    )

    return (
        a_result = THING1.foo,
    )
}
`, "array length mismatch")
	testBadCompile(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    out map<int> a_result,
)
{
    map call THING as THING1(
        stuff1 = split {
            "b": 1,
            "c": 2,
        },
        stuff2 = split {
            "a": 1,
            "b": 2,
        },
    )
    
    return (
        a_result = THING1.foo,
    )
}
`, "map key missing")
	testBadCompile(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    out map<int> a_result,
)
{
    map call THING as THING1(
        stuff1 = split {
            "b": 1,
            "c": 2,
            "a": 3,
        },
        stuff2 = split {
            "a": 1,
            "b": 2,
        },
    )
    
    return (
        a_result = THING1.foo,
    )
}
`, "map length mismatch")
	testBadCompile(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  int[]    arr,
    in  map<int> maps,
    out int[]    a_result,
)
{
    map call THING(
        stuff1 = split self.maps,
        stuff2 = split self.arr,
    )

    return(
        a_result = THING.foo,
    )
}
`, "cannot split over both arrays and maps")
	testBadCompile(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  int[]    arr,
    in  map<int> maps,
    out int[]    a_result,
)
{
    map call THING(
        stuff1 = split self.arr,
        stuff2 = split self.maps,
    )
    
    return(
        a_result = THING.foo,
    )
}
`, "cannot split over both arrays and maps")
	testBadCompile(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    out int  foo,
    src comp "nope",
)

map call THING(
    stuff1 = split [1, 2],
    stuff2 = split {"a": 1},
)
`, "cannot split over both arrays and maps")
}

// Check that map call properties propagate through multiple calls in a
// pipeline.
func TestMapCallPropagate(t *testing.T) {
	t.Parallel()

	testGood(t, `
stage THING(
    in  int  stuff,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  int[]    arr,
    in  map<int> maps,
    out int[]    a_result,
    out map<int> m_result,
)
{
    map call THING as THING1(
        stuff = split self.arr,
    )

    map call THING as THING2(
        stuff = split self.maps,
    )
    
    map call THING as THING3(
        stuff = split THING1.foo,
    )
    
    map call THING as THING4(
        stuff = split THING2.foo,
    ) 

    return(
        a_result = THING3.foo,
        m_result = THING4.foo,
    )
}
`)
	testGood(t, `
stage THING1(
    in  int  stuff,
    out int  foo,
    src comp "nope",
)

stage THING2(
    in  int[] stuff,
    out int[] foo,
    src comp  "nope",
)

pipeline THINGIFY(
    in  int[] arr,
    out int[] a_result,
)
{
    map call THING1(
        stuff = split self.arr,
    )

    call THING2(
        stuff = THING1.foo,
    )

    return(
        a_result = THING2.foo,
    )
}
`)
	testGood(t, `
stage THING1(
    in  int  stuff,
    out int  foo,
    src comp "nope",
)

stage THING2(
    in  map<int> stuff,
    out map<int> foo,
    src comp     "nope",
)

pipeline THINGIFY(
    in  map<int> maps,
    out map<int> a_result,
)
{
    map call THING1(
        stuff = split self.maps,
    )

    call THING2(
        stuff = THING1.foo,
    )

    return(
        a_result = THING2.foo,
    )
}
`)
	testGood(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    in  int  stuff3,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  map<int> stuff,
    out map<int> result,
)
{
    map call THING as THING1(
        stuff1 = split self.stuff,
        stuff2 = split {"a": 1, "b": 2},
        stuff3 = split {"a": 3, "b": 4},
    )

    map call THING as THING2(
        stuff1 = split {"a": 5, "b": 6},
        stuff2 = split {"a": 7, "b": 8},
        stuff3 = split THING1.foo,
    )

    return(
        result = THING2.foo,
    )
}
`)
	testGood(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    in  int  stuff3,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  map<int> stuff,
    out map<int> result,
)
{
    map call THING as THING1(
        stuff1 = split self.stuff,
        stuff2 = split {"a": 1, "b": 2},
        stuff3 = split {"a": 3, "b": 4},
    )

    map call THING as THING2(
        stuff1 = split THING1.foo,
        stuff2 = split {"a": 5, "b": 6},
        stuff3 = split {"a": 7, "b": 8},
    )

    return(
        result = THING2.foo,
    )
}
`)
	testGood(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    in  int  stuff3,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  map<int> stuff1,
    in  map<int> stuff2,
    out map<int> result,
)
{
    map call THING as THING1(
        stuff1 = split self.stuff1,
        stuff2 = split self.stuff2,
        stuff3 = split {"a": 3, "b": 4},
    )

    map call THING as THING2(
        stuff1 = split self.stuff1,
        stuff2 = split self.stuff2,
        stuff3 = split THING1.foo,
    )

    return(
        result = THING2.foo,
    )
}
`)
	testGood(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    in  int  stuff3,
    out int  foo,
    src comp "nope",
)

stage OTHER_THING(
    in  THING stuff1,
    in  THING stuff2,
    in  int   stuff3,
    out int   foo,
    src comp  "nope",
)

pipeline THINGIFY(
    in  map<int> stuff1,
    in  map<int> stuff2,
    out map<int> result,
)
{
    map call THING as THING1(
        stuff1 = split self.stuff1,
        stuff2 = split self.stuff2,
        stuff3 = split {"a": 3, "b": 4},
    )

    map call THING as THING2(
        stuff1 = split self.stuff1,
        stuff2 = split self.stuff2,
        stuff3 = split {"a": 1, "b": 2},
    )

    map call OTHER_THING(
        stuff1 = split THING1,
        stuff2 = split THING2,
        stuff3 = split self.stuff2,
    )

    return(
        result = OTHER_THING.foo,
    )
}
`)
	testGood(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    in  int  stuff3,
    out int  foo,
    src comp "nope",
)

stage OTHER_THING(
    in  THING stuff1,
    in  THING stuff2,
    in  int   stuff3,
    out int   foo,
    src comp  "nope",
)

pipeline THINGIFY(
    in  map<int> stuff1,
    in  map<int> stuff2,
    out map<int> result,
)
{
    map call THING as THING1(
        stuff1 = split self.stuff1,
        stuff2 = split self.stuff2,
        stuff3 = split {"a": 3, "b": 4},
    )

    map call THING as THING2(
        stuff1 = split self.stuff1,
        stuff2 = split self.stuff2,
        stuff3 = split THING1.foo,
    )

    map call OTHER_THING(
        stuff1 = split THING1,
        stuff2 = split THING2,
        stuff3 = split self.stuff2,
    )

    return(
        result = OTHER_THING.foo,
    )
}
`)
	testGood(t, `
stage THING(
    in  int  stuff1,
    in  int  stuff2,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  int[]    arr,
    in  map<int> maps,
    out int[]    a_result,
    out map<int> m_result,
)
{
    map call THING as THING1(
        stuff1 = split self.arr,
        stuff2     = split [
            1,
            2,
        ],
    )

    map call THING as THING2(
        stuff1 = split self.maps,
        stuff2 = split {
            "a": 1,
            "b": 2,
        },
    )
    
    map call THING as THING3(
        stuff1 = split THING1.foo,
        stuff2 = split self.arr,
    )
    
    map call THING as THING4(
        stuff1 = split THING2.foo,
        stuff2 = split self.maps,
    ) 

    return(
        a_result = THING3.foo,
        m_result = THING4.foo,
    )
}
`)
}

// Check that map call properties propagate through multiple calls in a
// pipeline to cause failures where appropriate.
func TestMapCallBadPropagate(t *testing.T) {
	t.Parallel()
	testBadCompile(t, `
stage THING(
    in  int  stuff,
    in  int  more_stuff,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  int[] arr,
    out int[] a_result,
)
{
    map call THING as THING1(
        stuff      = split self.arr,
        more_stuff = split [
            1,
            2,
            3,
        ],
    )

    map call THING as THING2(
        stuff      = split THING1.foo,
        more_stuff = split { "a": 1, "b": 2 },
    )

    return(
        a_result = THING2.foo,
    )
}
`, "cannot split over both arrays and maps")
	// Check that array length propagates through refs.
	testBadCompile(t, `
stage THING(
    in  int  stuff,
    in  int  more_stuff,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    out int[] a_result,
)
{
    map call THING as THING1(
        stuff      = split [
            1,
            2,
            3,
        ],
        more_stuff = split [
            1,
            2,
            3,
        ],
    )

    map call THING as THING2(
        stuff      = split THING1.foo,
        more_stuff = split [
            1,
            2,
        ],
    )

    return(
        a_result = THING2.foo,
    )
}
`, "array length mismatch")
	testBadCompile(t, `
stage THING(
    in  int  stuff,
    in  int  more_stuff,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    out int[] a_result,
)
{
    map call THING as THING1(
        stuff      = split [
            1,
            2,
            3,
        ],
        more_stuff = split [
            1,
            2,
            3,
        ],
    )

    map call THING as THING2(
        stuff      = split [
            1,
            2,
        ],
        more_stuff = split THING1.foo,
    )

    return(
        a_result = THING2.foo,
    )
}
`, "array length mismatch")
	// Check that array length propagates through refs even if some of the
	// ref's split inputs are of unknown length.
	testBadCompile(t, `
stage THING(
    in  int  stuff,
    in  int  more_stuff,
    out int  foo,
    src comp "nope",
)

pipeline THINGIFY(
    in  int[] arr,
    out int[] a_result,
)
{
    map call THING as THING1(
        stuff      = split self.arr,
        more_stuff = split [
            1,
            2,
            3,
        ],
    )

    map call THING as THING2(
        stuff      = split THING1.foo,
        more_stuff = split [
            1,
            2,
        ],
    )

    return(
        a_result = THING2.foo,
    )
}
`, "array length mismatch")
}

// Tests that when a map call splits over both a source whose length is only
// known at runtime and one whose length is known statically, the known-length
// source is used for the call regardless of which order the splits appear in.
func TestMapCallUnifyKnownLengthSource(t *testing.T) {
	t.Parallel()
	const header = `
stage GEN(
    out int[]    arr,
    out map<int> m,
    src comp     "nope",
)

stage MAPPED(
    in  int  a,
    in  int  z,
    out int  c,
    src comp "nope",
)

stage ECHO(
    in  int  x,
    out int  c,
    src comp "nope",
)
`
	arrayPipeline := func(bindings string) string {
		return header + `
pipeline P(
    out MAPPED[] outs,
)
{
    call GEN()

    map call ECHO(
        x = split GEN.arr,
    )

    map call MAPPED(
` + bindings + `
    )

    return (
        outs = MAPPED,
    )
}

call P()
`
	}
	mapPipeline := func(bindings string) string {
		return header + `
pipeline P(
    out map<MAPPED> outs,
)
{
    call GEN()

    map call MAPPED(
` + bindings + `
    )

    return (
        outs = MAPPED,
    )
}

call P()
`
	}
	cases := []struct {
		name string
		src  string
		mode CallMode
	}{
		{
			name: "unknown first by line and name",
			src: arrayPipeline(`        a = split GEN.arr,
        z = split [1, 2, 3],`),
			mode: ModeArrayCall,
		},
		{
			name: "unknown first by line last by name",
			src: arrayPipeline(`        z = split GEN.arr,
        a = split [1, 2, 3],`),
			mode: ModeArrayCall,
		},
		{
			name: "known first by line",
			src: arrayPipeline(`        a = split [1, 2, 3],
        z = split GEN.arr,`),
			mode: ModeArrayCall,
		},
		{
			name: "same line",
			src:  arrayPipeline(`        a = split GEN.arr, z = split [1, 2, 3],`),
			mode: ModeArrayCall,
		},
		{
			name: "merge unknown first",
			src: arrayPipeline(`        a = split ECHO.c,
        z = split [1, 2, 3],`),
			mode: ModeArrayCall,
		},
		{
			name: "merge known first",
			src: arrayPipeline(`        a = split [1, 2, 3],
        z = split ECHO.c,`),
			mode: ModeArrayCall,
		},
		{
			name: "map unknown first",
			src: mapPipeline(`        a = split GEN.m,
        z = split {"x": 1, "y": 2},`),
			mode: ModeMapCall,
		},
		{
			name: "map known first",
			src: mapPipeline(`        a = split {"x": 1, "y": 2},
        z = split GEN.m,`),
			mode: ModeMapCall,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ast := testGood(t, c.src)
			if ast == nil {
				return
			}
			graph, err := ast.MakeCallGraph("", ast.Call)
			if err != nil {
				t.Fatal(err)
			}
			node, ok := graph.NodeClosure()["P.MAPPED"].(*CallGraphStage)
			if !ok {
				t.Fatal("could not find stage node P.MAPPED")
			}
			src := node.MapSource()
			if src == nil {
				t.Fatal("expected P.MAPPED to be mapped")
			}
			if m := src.CallMode(); m != c.mode {
				t.Errorf("call mode %v != %v", m, c.mode)
			}
			if !src.KnownLength() {
				t.Errorf("expected known length source, got %s", src.GoString())
			}
			switch c.mode {
			case ModeArrayCall:
				if l := src.ArrayLength(); l != 3 {
					t.Errorf("array length %d != 3", l)
				}
			case ModeMapCall:
				keys := src.Keys()
				if len(keys) != 2 || keys["x"] == nil || keys["y"] == nil {
					t.Errorf("expected keys x, y; got %v", keys)
				}
			}
			outs, ok := graph.ResolvedOutputs().Exp.(*MapExp)
			if !ok {
				t.Fatalf("expected struct output, got %s",
					FormatExp(graph.ResolvedOutputs().Exp, ""))
			}
			switch c.mode {
			case ModeArrayCall:
				if arr, ok := outs.Value["outs"].(*ArrayExp); !ok {
					t.Errorf("expected static array output, got %s",
						FormatExp(outs.Value["outs"], ""))
				} else if len(arr.Value) != 3 {
					t.Errorf("output length %d != 3", len(arr.Value))
				}
			case ModeMapCall:
				if m, ok := outs.Value["outs"].(*MapExp); !ok {
					t.Errorf("expected static map output, got %s",
						FormatExp(outs.Value["outs"], ""))
				} else if len(m.Value) != 2 {
					t.Errorf("output length %d != 2", len(m.Value))
				}
			}
		})
	}
}

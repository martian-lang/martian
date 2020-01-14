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

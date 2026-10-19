//
// Copyright (c) 2026 10X Genomics, Inc. All rights reserved.
//
// Tests for resolution of disabled calls and map calls over literal or
// runtime-disabled sources, by compiling call graphs from example mro.
//

package syntax

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Reads a file from the testdata directory.
func readTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Compile the source and build its call graph, converting panics into test
// failures so that one bad case doesn't hide the others.
func testGoodCallGraph(t *testing.T, src string) CallGraphNode {
	t.Helper()
	var graph CallGraphNode
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic while resolving call graph: %v\n%s", r, src)
			}
		}()
		_, _, ast, err := ParseSourceBytes([]byte(src), "test.mro", nil, false)
		if err != nil {
			t.Errorf("%v\n%s", err, src)
			return
		}
		g, err := ast.MakeCallGraph("", ast.Call)
		if err != nil {
			t.Errorf("%v\n%s", err, src)
			return
		}
		graph = g
	}()
	return graph
}

// Disabling calls with literals or with the outputs of other calls which are
// themselves disabled.
func TestResolveDisableLiterals(t *testing.T) {
	t.Parallel()
	defs := readTestdata(t, "disable_literals_defs.mro")
	for name, body := range map[string]string{
		"true stage":                          "disable_literals_true_stage.mro",
		"false stage":                         "disable_literals_false_stage.mro",
		"true pipeline":                       "disable_literals_true_pipeline.mro",
		"disabled by disabled output":         "disable_literals_disabled_by_disabled.mro",
		"disabled pipeline output as disable": "disable_literals_disabled_pipeline_output.mro",
	} {
		t.Run(name, func(t *testing.T) {
			testGoodCallGraph(t, defs+readTestdata(t, body))
		})
	}
}

// Map calls where the disable binding of a call inside the mapped pipeline is
// derived from the split, and a downstream map call splits over the output.
func TestResolveDisableSplitSources(t *testing.T) {
	t.Parallel()
	defs := readTestdata(t, "disable_split_defs.mro")
	for _, c := range []struct {
		name, inputs, args, split, outType string
	}{
		{
			name:    "array literal",
			split:   "[true, false]",
			outType: "int[]",
		},
		{
			name:    "all false array literal",
			split:   "[false, false]",
			outType: "int[]",
		},
		{
			name:    "all true array literal",
			split:   "[true, true]",
			outType: "int[]",
		},
		{
			name:    "single element array",
			split:   "[true]",
			outType: "int[]",
		},
		{
			name:    "array with reference",
			inputs:  "in bool b,",
			args:    "b = true,",
			split:   "[self.b, false]",
			outType: "int[]",
		},
		{
			name:    "array with only references",
			inputs:  "in bool b,",
			args:    "b = true,",
			split:   "[self.b, self.b]",
			outType: "int[]",
		},
		{
			name:    "array with null",
			split:   "[null, false]",
			outType: "int[]",
		},
		{
			name:    "null array reference",
			inputs:  "in bool[] bs,",
			args:    "bs = null,",
			split:   "self.bs",
			outType: "int[]",
		},
		{
			name:    "null map reference",
			inputs:  "in map<bool> bs,",
			args:    "bs = null,",
			split:   "self.bs",
			outType: "map<int>",
		},
		{
			name:    "array reference",
			inputs:  "in bool[] bs,",
			args:    "bs = [true, false],",
			split:   "self.bs",
			outType: "int[]",
		},
		{
			name:    "map literal",
			split:   `{"a": true, "b": false}`,
			outType: "map<int>",
		},
		{
			name:    "all false map literal",
			split:   `{"a": false, "b": false}`,
			outType: "map<int>",
		},
		{
			name:    "all true map literal",
			split:   `{"a": true, "b": true}`,
			outType: "map<int>",
		},
		{
			name:    "single key map literal",
			split:   `{"a": true}`,
			outType: "map<int>",
		},
		{
			name:    "map with reference",
			inputs:  "in bool b,",
			args:    "b = true,",
			split:   `{"a": true, "b": self.b}`,
			outType: "map<int>",
		},
		{
			name:    "map with only references",
			inputs:  "in bool b,",
			args:    "b = true,",
			split:   `{"a": self.b, "b": self.b}`,
			outType: "map<int>",
		},
		{
			name:    "map with null",
			split:   `{"a": null, "b": false}`,
			outType: "map<int>",
		},
		{
			name:    "map reference",
			inputs:  "in map<bool> bs,",
			args:    `bs = {"a": true, "b": false},`,
			split:   "self.bs",
			outType: "map<int>",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			testGoodCallGraph(t,
				defs+fmt.Sprintf(
					readTestdata(t, "disable_split_template.mro"),
					c.inputs, c.outType, c.split, c.args))
		})
	}
}

// Checks that the graph's resolved outputs serialize to the expected json.
func checkGraphOutputs(t *testing.T, graph CallGraphNode, expect string) {
	t.Helper()
	if graph == nil {
		return
	}
	b, err := json.Marshal(graph.ResolvedOutputs())
	if err != nil {
		t.Fatal(err)
	}
	if s := string(b); s != expect {
		t.Errorf("outputs %s != expected %s", s, expect)
	}
}

// Map calls over null sources, on both stages and pipelines.  These never
// run, but the result is an empty collection rather than null, since the calls
// themselves are not disabled.
func TestResolveMapCallEmptySources(t *testing.T) {
	t.Parallel()
	defs := readTestdata(t, "disable_empty_defs.mro")
	for _, c := range []struct {
		name, inType, outType, expect string
	}{
		{"array", "int[]", "int[]", "[]"},
		{"map", "map<int>", "map<int>", "{}"},
	} {
		expect := `{"expression":{"y":` + c.expect + `},"type":"TOP"}`
		t.Run(c.name+" stage", func(t *testing.T) {
			checkGraphOutputs(t,
				testGoodCallGraph(t,
					defs+fmt.Sprintf(
						readTestdata(t, "disable_empty_stage_template.mro"),
						c.inType, c.outType)),
				expect)
		})
		t.Run(c.name+" pipeline", func(t *testing.T) {
			checkGraphOutputs(t,
				testGoodCallGraph(t,
					defs+fmt.Sprintf(
						readTestdata(t, "disable_empty_pipeline_template.mro"),
						c.inType, c.outType)),
				expect)
		})
	}
}

// Map calls over empty sources which are also disabled.  The result is null if
// the call is disabled, and an empty collection otherwise.
func TestResolveMapCallEmptyDisabled(t *testing.T) {
	t.Parallel()
	defs := readTestdata(t, "disable_empty_defs.mro")
	for _, c := range []struct {
		name, inType, disabled, empty, disable, expect string
	}{
		{"array disabled", "int[]", "self.disable", "[]", "true", "null"},
		{"array enabled", "int[]", "self.disable", "[]", "false", "[]"},
		{"map disabled", "map<int>", "self.disable", "null", "true", "null"},
		{"map enabled", "map<int>", "self.disable", "{}", "false", "{}"},
		{
			"array runtime", "int[]", "F.b", "null", "false",
			`{"__disabled__":{"__reference__":"TOP.F.b"},"value":[]}`,
		},
		{
			"map runtime", "map<int>", "F.b", "{}", "false",
			`{"__disabled__":{"__reference__":"TOP.F.b"},"value":{}}`,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			checkGraphOutputs(t,
				testGoodCallGraph(t, defs+fmt.Sprintf(
					readTestdata(t, "disable_empty_disabled_template.mro"),
					c.inType, c.disabled, c.empty, c.disable)),
				`{"expression":{"y":`+c.expect+`},"type":"TOP"}`)
		})
	}
}

// Map calls over empty sources, inside a pipeline which is disabled at runtime,
// where the map call's own disabled binding is a literal.  The parent's
// runtime disable condition applies to the parent's outputs, but the call's
// own literal condition must still apply to the call's outputs.
func TestResolveMapCallEmptyDisabledNested(t *testing.T) {
	t.Parallel()
	defs := readTestdata(t, "disable_empty_defs.mro")
	for _, c := range []struct {
		name, inType, empty, disable, expect string
	}{
		{"array disabled", "int[]", "[]", "true", "null"},
		{"map disabled", "map<int>", "{}", "true", "null"},
		{"null disabled", "map<int>", "null", "true", "null"},
		{
			"array enabled", "int[]", "[]", "false",
			`{"__disabled__":{"__reference__":"TOP.F.b"},"value":[]}`,
		},
		{
			"map enabled", "map<int>", "{}", "false",
			`{"__disabled__":{"__reference__":"TOP.F.b"},"value":{}}`,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			checkGraphOutputs(t,
				testGoodCallGraph(t, defs+fmt.Sprintf(
					readTestdata(t, "disable_empty_nested_template.mro"),
					c.inType, c.empty, c.disable)),
				`{"expression":{"y":`+c.expect+`},"type":"TOP"}`)
		})
	}
}

// A stage with two map sources inside a mapped sub-pipeline, so that the
// sibling-matching and source-ordering code is exercised.
func TestResolveNestedMappedSources(t *testing.T) {
	t.Parallel()
	testGoodCallGraph(t, readTestdata(t, "disable_nested_sources.mro"))
}

func TestResolveNilReceivers(t *testing.T) {
	t.Parallel()
	var stage *CallGraphStage
	if r := stage.ResolvedInputs(); r != nil {
		t.Errorf("expected nil inputs, got %v", r)
	}
	if s := new(CallGraphStage).MapSource(); s != nil {
		t.Errorf("expected nil source, got %v", s)
	}
}

func TestSortedSplitListLess(t *testing.T) {
	t.Parallel()
	mk := func(file *SourceFile, line int) *SplitExp {
		return &SplitExp{
			valExp: valExp{
				Node: AstNode{Loc: SourceLoc{File: file, Line: line}},
			},
		}
	}
	a := &SourceFile{FileName: "a.mro"}
	b := &SourceFile{FileName: "b.mro"}
	list := sortedSplitList{
		mk(nil, 1),
		mk(b, 1),
		mk(a, 5),
		mk(a, 2),
		mk(nil, 0),
	}
	for _, c := range []struct {
		i, j   int
		expect bool
	}{
		{1, 2, false}, // b vs a
		{2, 1, true},  // a vs b
		{2, 3, false}, // same file, later line
		{3, 2, true},  // same file, earlier line
		{1, 0, true},  // file vs no file
		{0, 1, false}, // no file vs file
		{0, 4, false}, // no files, later line
		{4, 0, true},  // no files, earlier line
		{2, 2, false}, // same
	} {
		if got := list.Less(c.i, c.j); got != c.expect {
			t.Errorf("Less(%d, %d) = %v, expected %v",
				c.i, c.j, got, c.expect)
		}
	}
}

// A sub-pipeline which is disabled by a value derived from the split of the
// enclosing mapped pipeline, containing a map call, with a downstream map
// call over the sub-pipeline's output.
func TestResolveDisabledBySplit(t *testing.T) {
	t.Parallel()
	testGoodCallGraph(t, readTestdata(t, "disable_by_split.mro"))
}

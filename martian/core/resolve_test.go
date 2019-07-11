package core

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

func TestLazyArgumentMap_Path(t *testing.T) {
	_, _, ast, err := syntax.ParseSourceBytes([]byte(`
struct BAR(
    string c,
)

struct BAZ(
	int a,
	BAR b,
)

struct FOO(
	int   bar,
	BAZ[] baz,
)`), "example.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	lookup := &ast.TypeTable
	ty := lookup.Get(syntax.TypeId{Tname: "FOO", MapDim: 1})
	if ty == nil {
		t.Fatal("missing type")
	}

	var m LazyArgumentMap
	if err := json.Unmarshal([]byte(`{
		"foo1": {
			"bar": 0,
			"baz": [
				{
					"a": 1,
					"b": {
						"c": "2",
						"d": 8
					}
				},
				{
					"a": 3,
					"b": {
						"c": "4"
					},
					"c": "bar"
				},
				null
			]
		},
		"foo2": null
	}`), &m); err != nil {
		t.Fatal(err)
	}
	get := func(t *testing.T, dest syntax.Type, p string) string {
		t.Helper()
		if v, err := m.Path(p, ty, dest, lookup); err != nil {
			t.Error(err)
		} else if b, err := json.Marshal(v); err != nil {
			t.Error(err)
		} else {
			return string(b)
		}
		return ""
	}
	check := func(t *testing.T, dest syntax.Type, p, expect string) {
		t.Helper()
		if s := get(t, dest, p); s != expect {
			t.Errorf("%s != %s", s, expect)
		}
	}
	check(t, ty,
		"",
		`{"foo1":{"bar":0,"baz":[{"a":1,"b":{"c":"2"}},{"a":3,"b":{"c":"4"}},null]},"foo2":null}`)
	check(t, lookup.Get(syntax.TypeId{Tname: syntax.KindInt, MapDim: 1}),
		"bar",
		`{"foo1":0,"foo2":null}`)
	check(t, lookup.Get(syntax.TypeId{Tname: "BAZ", MapDim: 2}), "baz",
		`{"foo1":[{"a":1,"b":{"c":"2"}},{"a":3,"b":{"c":"4"}},null],"foo2":null}`)
	check(t, lookup.Get(syntax.TypeId{Tname: syntax.KindInt, MapDim: 2}),
		"baz.a",
		`{"foo1":[1,3,null],"foo2":null}`)
	check(t, lookup.Get(syntax.TypeId{Tname: "BAR", MapDim: 2}),
		"baz.b",
		`{"foo1":[{"c":"2"},{"c":"4"},null],"foo2":null}`)
	check(t, lookup.Get(syntax.TypeId{Tname: syntax.KindInt, MapDim: 2}),
		"baz.b.c",
		`{"foo1":["2","4",null],"foo2":null}`)

	ty = ty.(*syntax.TypedMapType).Elem
	if err := json.Unmarshal([]byte(`{
		"bar": 0,
		"baz": [
			{
				"a": 1,
				"b": {
					"c": "2",
					"d": 8
				}
			},
			{
				"a": 3,
				"b": {
					"c": "4"
				},
				"c": "bar"
			}
		]
	}`), &m); err != nil {
		t.Fatal(err)
	}
	check(t, ty,
		"",
		`{"bar":0,"baz":[{"a":1,"b":{"c":"2"}},{"a":3,"b":{"c":"4"}}]}`)
	check(t, lookup.Get(syntax.TypeId{Tname: syntax.KindInt}),
		"bar",
		"0")
	check(t, lookup.Get(syntax.TypeId{Tname: "BAZ"}),
		"baz",
		`[{"a":1,"b":{"c":"2"}},{"a":3,"b":{"c":"4"}}]`)
	check(t, lookup.Get(syntax.TypeId{Tname: syntax.KindInt, ArrayDim: 1}),
		"baz.a",
		`[1,3]`)
	check(t, lookup.Get(syntax.TypeId{
		Tname:    syntax.KindString,
		ArrayDim: 1,
		MapDim:   1,
	}),
		"baz.b",
		`[{"c":"2"},{"c":"4"}]`)
	check(t, lookup.Get(syntax.TypeId{
		Tname:    syntax.KindString,
		ArrayDim: 1,
	}),
		"baz.b.c",
		`["2","4"]`)
}

func TestLazyArgumentMap_jsonPath(t *testing.T) {
	var m LazyArgumentMap
	if err := json.Unmarshal([]byte(`{
		"foo1": {
			"bar": 0,
			"baz": [
				{
					"a": 1,
					"b": {
						"c": "2",
						"d": 8
					}
				},
				{
					"a": 3,
					"b": {
						"c": "4"
					},
					"c": "bar"
				},
				null
			]
		},
		"foo2": null
	}`), &m); err != nil {
		t.Fatal(err)
	}
	if b, err := m.jsonPath("foo1.baz.b.c").MarshalJSON(); err != nil {
		t.Error(err)
	} else if s := string(b); s != `["2","4",null]` {
		t.Errorf("'%s' != '8'", s)
	}
}

// Sets up a test pipestance.
func setupTestPipestance(t *testing.T, name string) (*Pipestance, string) {
	t.Helper()
	util.MockSignalHandlersForTest()
	psOuts, err := filepath.Abs(
		"testdata/test_" + name + "_struct_pipestance/outs")
	if err != nil {
		t.Fatal(err)
	}
	psPath := filepath.Dir(psOuts)
	psSrc := filepath.Join(psPath, "srcs")
	if err := os.RemoveAll(psPath); err != nil &&
		!os.IsNotExist(err) {
		t.Error(err)
	}
	if err := os.MkdirAll(psSrc, 0755); err != nil {
		t.Fatal(err)
	}

	src, err := ioutil.ReadFile("testdata/struct_pipeline.mro")
	if err != nil {
		t.Fatal(err)
	}

	// Set up minimal runtime without actually loading config.
	conf := DefaultRuntimeOptions()
	rt := Runtime{
		Config: &conf,
		LocalJobManager: &LocalJobManager{
			jobSettings: new(JobManagerSettings),
		},
	}
	rt.JobManager = rt.LocalJobManager
	_, _, pipestance, err := rt.instantiatePipeline(string(src),
		"testdata/struct_pipeline.mro",
		"test_struct_pipeline", psPath, nil,
		"none", nil, false, context.Background())
	if err != nil {
		os.RemoveAll(psPath)
		t.Fatal(err)
	}
	type creator struct {
		Bar   *int   `json:"bar"`
		File1 string `json:"file1"`
		File2 string `json:"file2"`
		File3 string `json:"file3"`
	}
	node := pipestance.node
	if c1 := node.top.allNodes[node.GetFQName()+".INNER.C1"]; c1 == nil {
		os.RemoveAll(psPath)
		t.Fatal("Could not get node for C1")
	} else {
		if err := c1.mkdirs(); err != nil {
			t.Error(err)
		}
		md := c1.forks[0].metadata
		b := 1
		outs := creator{
			Bar:   &b,
			File1: md.FilePath("file1"),
			File2: md.FilePath("file2"),
			File3: md.FilePath("file3"),
		}
		if err := md.Write(OutsFile, &outs); err != nil {
			t.Error(err)
		}
	}
	if c2 := node.top.allNodes[node.GetFQName()+".INNER.C2"]; c2 == nil {
		os.RemoveAll(psPath)
		t.Fatal("Could not get node for C2")
	} else {
		if err := c2.mkdirs(); err != nil {
			t.Error(err)
		}
		md := c2.forks[0].metadata
		outs := creator{
			File1: md.FilePath("file1"),
			File2: md.FilePath("file2"),
		}
		if err := md.Write(OutsFile, &outs); err != nil {
			t.Error(err)
		}
	}
	if consumer := node.top.allNodes[node.GetFQName()+".CONSUMER"]; consumer == nil {
		os.RemoveAll(psPath)
		t.Fatal("Could not get node for CONSUMER")
	} else {
		if err := consumer.mkdirs(); err != nil {
			t.Error(err)
		}
		if err := consumer.forks[0].metadata.Write(OutsFile, nil); err != nil {
			t.Error(err)
		}
	}
	return pipestance, psPath
}

// Compares the json form of a value to an expected one, after replacing
// all instances of psPath with an empty string, and including line-by-line
// information to make debugging easier.
func checkJsonOutput(t *testing.T, result json.Marshaler, psPath, expected string) {
	t.Helper()
	by, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		t.Fatal(err)
	}
	compareOutputText(t, expected,
		strings.Replace(string(by), psPath+"/", "", -1))
}

func TestResolvePipelineOutputs(t *testing.T) {
	pipestance, psPath := setupTestPipestance(t, "resolve_pipeline_outputs")
	defer func() {
		if !t.Failed() {
			os.RemoveAll(psPath)
		}
	}()
	if pipestance == nil {
		return
	}
	result, err := pipestance.node.resolvePipelineOutputs(nil)
	if err != nil {
		t.Fatal(err)
	}

	const expected = `{
	"bars": [
		null,
		3
	],
	"files1": [
		{
			"c1": "OUTER/INNER/C1/fork0/files/file1",
			"c2": "OUTER/INNER/C2/fork0/files/file1"
		},
		{
			"c1": "OUTER/INNER/C1/fork0/files/file1",
			"c2": null
		}
	],
	"inner": {
		"bar": {
			"bar": null,
			"file1": "OUTER/INNER/C2/fork0/files/file1",
			"file2": "OUTER/INNER/C2/fork0/files/file2",
			"file3": ""
		},
		"results1": {
			"c1": {
				"bar": 1,
				"file1": "OUTER/INNER/C1/fork0/files/file1",
				"file2": "OUTER/INNER/C1/fork0/files/file2",
				"file3": "OUTER/INNER/C1/fork0/files/file3"
			},
			"c2": {
				"bar": null,
				"file1": "OUTER/INNER/C2/fork0/files/file1",
				"file2": "OUTER/INNER/C2/fork0/files/file2",
				"file3": ""
			}
		},
		"results2": {
			"c1": {
				"bar": 1,
				"file1": "OUTER/INNER/C1/fork0/files/file1",
				"file2": "OUTER/INNER/C1/fork0/files/file2",
				"file3": "OUTER/INNER/C1/fork0/files/file3"
			},
			"c2": null
		}
	},
	"strs": [
		{
			"file1": "OUTER/INNER/C2/fork0/files/file1"
		},
		{
			"file1": "foo"
		}
	],
	"text": "OUTER/INNER/C2/fork0/files/file1",
	"texts": [
		"OUTER/INNER/C2/fork0/files/file2"
	]
}`
	checkJsonOutput(t, result, psPath, expected)
}

func TestResolveInputs(t *testing.T) {
	pipestance, psPath := setupTestPipestance(t, "resolve_inputs")
	defer func() {
		if !t.Failed() {
			os.RemoveAll(psPath)
		}
	}()
	if pipestance == nil {
		return
	}
	node := pipestance.node.top.allNodes[pipestance.node.GetFQName()+".CONSUMER"]
	if node == nil {
		t.Fatal("could not get node " + pipestance.node.GetFQName() + ".CONSUMER")
	}
	result, err := node.resolveInputs(nil)
	if err != nil {
		t.Fatal(err)
	}

	const expected = `{
	"c1": {
		"bar": null,
		"file1": "OUTER/INNER/C2/fork0/files/file1"
	},
	"files1": [
		"OUTER/INNER/C2/fork0/files/file2",
		"/some/path"
	],
	"files2": {
		"c1": "OUTER/INNER/C1/fork0/files/file2",
		"c2": "OUTER/INNER/C2/fork0/files/file2"
	}
}`
	checkJsonOutput(t, result, psPath, expected)
}

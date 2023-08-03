package core

import (
	"context"
	"encoding/json"
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

struct BazSubset(
	BAR b,
)

struct FooSubset(
	BazSubset[] baz,
)

struct Foos(
	FooSubset foo1,
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
	check(t, lookup.Get(syntax.TypeId{Tname: syntax.KindString, MapDim: 2}),
		"baz.b.c",
		`{"foo1":["2","4",null],"foo2":null}`)

	fooSubset := lookup.Get(syntax.TypeId{Tname: "FooSubset", MapDim: 1})
	if fooSubset == nil {
		t.Fatal("missing subset type")
	}
	check(t, fooSubset, "",
		`{"foo1":{"baz":[{"b":{"c":"2"}},{"b":{"c":"4"}},null]},"foo2":null}`)
	bazSubset := lookup.Get(syntax.TypeId{Tname: "BazSubset", MapDim: 2})
	if bazSubset == nil {
		t.Fatal("missing subset type")
	}
	check(t, bazSubset, "baz",
		`{"foo1":[{"b":{"c":"2"}},{"b":{"c":"4"}},null],"foo2":null}`)
	check(t, lookup.Get(syntax.TypeId{Tname: "Foos"}), "",
		`{"foo1":{"baz":[{"b":{"c":"2"}},{"b":{"c":"4"}},null]}}`)

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
	check(t, lookup.Get(syntax.TypeId{Tname: "BAZ", ArrayDim: 1}),
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
		`[{"c":"2","d":8},{"c":"4"}]`)
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

func TestJsonPath(t *testing.T) {
	result := jsonPath([]byte(`{
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
	}`), "foo1.baz.a")
	if b, err := json.Marshal(result); err != nil {
		t.Error(err)
	} else if string(b) != `[1,3,null]` {
		t.Errorf("%q != [1,3,null]", b)
	}
	if b, err := json.Marshal(jsonPath([]byte(`null`), "foo1.baz.a")); err != nil {
		t.Error(err)
	} else if string(b) != "null" {
		t.Errorf("%q != null", b)
	}
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

// Sets up a test pipestance.
func setupTestPipestance(t *testing.T, mro, name string) (*Pipestance, string) {
	t.Helper()
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

	src, err := os.ReadFile(mro)
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
	_, _, pipestance, err := rt.instantiatePipeline(src,
		mro,
		"test_struct_pipeline", psPath, nil,
		"none", nil, true, false, context.Background())
	if err != nil {
		os.RemoveAll(psPath)
		t.Fatal(err)
	}
	return pipestance, psPath
}

func touch(t *testing.T, md *Metadata, n string) string {
	t.Helper()
	fn := md.FilePath(n)
	if f, err := os.Create(fn); err != nil {
		t.Error(err)
	} else {
		defer f.Close()
		if _, err := f.WriteString(n); err != nil {
			t.Error(err)
		}
	}
	return fn
}

func setupSimpleStructPipestance(t *testing.T, name string) (*Pipestance, string) {
	t.Helper()
	pipestance, psPath := setupTestPipestance(t,
		"testdata/simple_struct_pipeline.mro", name)
	type foo struct {
		A string `json:"a"`
		B string `json:"b"`
		C string `json:"c"`
		D string `json:"d"`
	}
	type fooBar struct {
		Bar int `json:"bar"`
		Foo foo `json:"foo"`
	}
	type makeFooBar struct {
		FooBar fooBar `json:"foobar"`
	}
	node := pipestance.node
	if fb := node.top.allNodes[node.GetFQName()+".MAKEFOOBAR"]; fb == nil {
		os.RemoveAll(psPath)
		t.Fatal("Could not get node for MAKEFOOBAR")
	} else {
		if err := fb.mkdirs(); err != nil {
			t.Error(err)
		}
		md := fb.forks[0].metadata
		outs := makeFooBar{
			FooBar: fooBar{
				Bar: 1,
				Foo: foo{
					A: touch(t, md, "a"),
					B: touch(t, md, "b"),
					C: touch(t, md, "c"),
					D: touch(t, md, "d"),
				},
			},
		}
		if err := md.Write(OutsFile, &outs); err != nil {
			t.Error(err)
		}
	}
	return pipestance, psPath
}

func TestResolveSimplePipelineOutputs(t *testing.T) {
	pipestance, psPath := setupSimpleStructPipestance(t, "resolve_simple_outputs")
	defer func() {
		if !t.Failed() {
			os.RemoveAll(psPath)
		}
	}()
	if pipestance == nil {
		return
	}
	result, typ, err := pipestance.node.resolvePipelineOutputs(nil)
	if err != nil {
		t.Fatal(err)
	}
	const expected = `{
	"foobar": {
		"bar": 1,
		"foo": {
			"a": "HELLO/MAKEFOOBAR/fork0/files/a",
			"b": "HELLO/MAKEFOOBAR/fork0/files/b",
			"c": "HELLO/MAKEFOOBAR/fork0/files/c",
			"d": "HELLO/MAKEFOOBAR/fork0/files/d"
		}
	}
}`
	if s, ok := typ.(*syntax.StructType); !ok {
		t.Errorf("%T != struct", typ)
	} else if s.Id != "HELLO" {
		t.Errorf("%s != HELLO", s.Id)
	}
	checkJsonOutput(t, result, psPath, expected)
	// Now check that post-process won't mangle it too badly.
	fork := pipestance.node.forks[0]
	if err := fork.metadata.Write(OutsFile, result); err != nil {
		t.Error(err)
	}
	var buf strings.Builder
	util.SetPrintLogger(&buf)
	if err := fork.postProcess(context.Background()); err != nil {
		t.Error(err)
	}
	util.SetPrintLogger(&devNull)
	const expectSummary = `Outputs:
- foobar:
    foo:
      a: outs/foobar/foo/a.csv
      b: outs/foobar/foo/b.csv
      c: outs/foobar/foo/c.csv
      d: outs/foobar/foo/d.csv
    bar: 1`
	compareOutputText(t, expectSummary,
		strings.Replace(buf.String(), psPath+"/", "", -1))
	if b, err := fork.metadata.readRawBytes(OutsFile); err != nil {
		t.Error(err)
	} else {
		const expectedUpdated = `{
	"foobar": {
		"bar": 1,
		"foo": {
			"a": "outs/foobar/foo/a.csv",
			"b": "outs/foobar/foo/b.csv",
			"c": "outs/foobar/foo/c.csv",
			"d": "outs/foobar/foo/d.csv"
		}
	}
}`
		checkJsonOutput(t, json.RawMessage(b), psPath, expectedUpdated)
	}
}

func setupTestStructPipestance(t *testing.T, name string) (*Pipestance, string) {
	t.Helper()
	pipestance, psPath := setupTestPipestance(t,
		"testdata/struct_pipeline.mro", name)
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
			File1: touch(t, md, "file1"),
			File2: touch(t, md, "file2"),
			File3: touch(t, md, "file3"),
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
			File1: touch(t, md, "file1"),
			File2: touch(t, md, "file2"),
			File3: md.FilePath("file3"), // file that does not exist.
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

func TestResolvePipelineOutputs(t *testing.T) {
	pipestance, psPath := setupTestStructPipestance(t, "resolve_pipeline_outputs")
	defer func() {
		if !t.Failed() {
			os.RemoveAll(psPath)
		}
	}()
	if pipestance == nil {
		return
	}
	result, typ, err := pipestance.node.resolvePipelineOutputs(nil)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := typ.(*syntax.StructType); !ok {
		t.Errorf("%T != struct", typ)
	} else if s.Id != "OUTER" {
		t.Errorf("%s != OUTER", s.Id)
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
			"file3": "OUTER/INNER/C2/fork0/files/file3"
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
				"file3": "OUTER/INNER/C2/fork0/files/file3"
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
	"many": {
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
			"file3": "OUTER/INNER/C2/fork0/files/file3"
		}
	},
	"one": {
		"bar": null,
		"file1": "OUTER/INNER/C2/fork0/files/file1",
		"file2": "OUTER/INNER/C2/fork0/files/file2",
		"file3": "OUTER/INNER/C2/fork0/files/file3"
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
	// Now check that post-process won't mangle it too badly.
	fork := pipestance.node.forks[0]
	if err := fork.metadata.Write(OutsFile, result); err != nil {
		t.Error(err)
	}
	var buf strings.Builder
	util.SetPrintLogger(&buf)
	if err := fork.postProcess(context.Background()); err != nil {
		t.Error(err)
	}
	util.SetPrintLogger(&devNull)
	const expectSummary = `Outputs:
- one text file: outs/text.txt
- inner:
    bar:
      bar:       null
      file1:     outs/text.txt
      file2:     outs/inner/bar/file2
      help text: null
    results1:    {
      c1:
        bar:       1
        file1:     outs/inner/results1/c1/file1.txt
        file2:     outs/inner/results1/c1/file2
        help text: outs/inner/results1/c1/output_name.file
      c2:
        bar:       null
        file1:     outs/text.txt
        file2:     outs/inner/bar/file2
        help text: null
    }
    description: {
      c1:
        bar:       1
        file1:     outs/inner/results1/c1/file1.txt
        file2:     outs/inner/results1/c1/file2
        help text: outs/inner/results1/c1/output_name.file
      c2: null
    }
- files1:        [
    0: {
      c1: outs/inner/results1/c1/file1.txt
      c2: outs/text.txt
    }
    1: {
      c1: outs/inner/results1/c1/file1.txt
      c2: null
    }
  ]
- some ints:     [null,3]
- strs:          [
    {"file1":"OUTER/INNER/C2/fork0/files/file1"}
    {"file1":"foo"}
  ]
- some files:    [
    0: outs/inner/bar/file2
  ]
- one:
    bar:       null
    file1:     outs/text.txt
    file2:     outs/inner/bar/file2
    help text: null
- many files:    {
    c1:
      bar:       1
      file1:     outs/inner/results1/c1/file1.txt
      file2:     outs/inner/results1/c1/file2
      help text: outs/inner/results1/c1/output_name.file
    c2:
      bar:       null
      file1:     outs/text.txt
      file2:     outs/inner/bar/file2
      help text: null
  }`
	compareOutputText(t, expectSummary,
		strings.Replace(buf.String(), psPath+"/", "", -1))
	if b, err := fork.metadata.readRawBytes(OutsFile); err != nil {
		t.Error(err)
	} else {
		const expectedUpdated = `{
	"bars": [
		null,
		3
	],
	"files1": [
		{
			"c1": "outs/inner/results1/c1/file1.txt",
			"c2": "outs/text.txt"
		},
		{
			"c1": "outs/inner/results1/c1/file1.txt",
			"c2": null
		}
	],
	"inner": {
		"bar": {
			"bar": null,
			"file1": "outs/text.txt",
			"file2": "outs/inner/bar/file2",
			"file3": null
		},
		"results1": {
			"c1": {
				"bar": 1,
				"file1": "outs/inner/results1/c1/file1.txt",
				"file2": "outs/inner/results1/c1/file2",
				"file3": "outs/inner/results1/c1/output_name.file"
			},
			"c2": {
				"bar": null,
				"file1": "outs/text.txt",
				"file2": "outs/inner/bar/file2",
				"file3": null
			}
		},
		"results2": {
			"c1": {
				"bar": 1,
				"file1": "outs/inner/results1/c1/file1.txt",
				"file2": "outs/inner/results1/c1/file2",
				"file3": "outs/inner/results1/c1/output_name.file"
			},
			"c2": null
		}
	},
	"many": {
		"c1": {
			"bar": 1,
			"file1": "outs/inner/results1/c1/file1.txt",
			"file2": "outs/inner/results1/c1/file2",
			"file3": "outs/inner/results1/c1/output_name.file"
		},
		"c2": {
			"bar": null,
			"file1": "outs/text.txt",
			"file2": "outs/inner/bar/file2",
			"file3": null
		}
	},
	"one": {
		"bar": null,
		"file1": "outs/text.txt",
		"file2": "outs/inner/bar/file2",
		"file3": null
	},
	"strs": [
		{
			"file1": "OUTER/INNER/C2/fork0/files/file1"
		},
		{
			"file1": "foo"
		}
	],
	"text": "outs/text.txt",
	"texts": [
		"outs/inner/bar/file2"
	]
}`
		checkJsonOutput(t, json.RawMessage(b), psPath, expectedUpdated)
	}
}

func TestResolveInputs(t *testing.T) {
	pipestance, psPath := setupTestStructPipestance(t, "resolve_inputs")
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
	mapped, result, err := node.resolveInputs(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(mapped) != 0 {
		t.Error("should not be a map call")
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

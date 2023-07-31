//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Martian runtime tests.
//

package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

func ExampleBuildCallSource() {
	src, _ := BuildCallSource(
		"STAGE_NAME",
		MakeMarshalerMap(map[string]interface{}{
			"input1": []int{1, 2},
			"input2": "foo",
			"input3": json.RawMessage(`{"foo":"bar"}`),
		}),
		nil,
		&syntax.Stage{
			Node: syntax.NewAstNode(syntax.SourceLoc{
				Line: 15,
				File: &syntax.SourceFile{
					FileName: "foo.mro",
					FullPath: "/path/to/foo.mro",
				},
			}),
			Id: "STAGE_NAME",
			InParams: &syntax.InParams{
				List: []*syntax.InParam{
					{
						Tname: syntax.TypeId{
							Tname:    syntax.KindInt,
							ArrayDim: 1,
						},
						Id: "input1",
					},
					{
						Tname: syntax.TypeId{Tname: syntax.KindString},
						Id:    "input2",
					},
					{
						Tname: syntax.TypeId{Tname: syntax.KindMap},
						Id:    "input3",
					},
				},
			},
		},
		syntax.NewTypeLookup(),
		nil)
	fmt.Println(src)
	// Output:
	// @include "foo.mro"
	//
	// call STAGE_NAME(
	//     input1 = [
	//         1,
	//         2,
	//     ],
	//     input2 = "foo",
	//     input3 = {
	//         "foo": "bar",
	//     },
	// )
}

type nullWriter struct{}

func (*nullWriter) Write(b []byte) (int, error) {
	return len(b), nil
}
func (*nullWriter) WriteString(b string) (int, error) {
	return len(b), nil
}

var devNull nullWriter

func TestMain(m *testing.M) {
	syntax.SetEnforcementLevel(syntax.EnforceError)
	// Disable logging here, because otherwise the race detector can get unhappy
	// when running parallel tests.
	util.SetPrintLogger(&devNull)
	util.LogTeeWriter(&devNull)
	os.Exit(m.Run())
}

// Very basic invoke test.
func TestInvoke(t *testing.T) {
	invokeTest(`
stage SUM_SQUARES(
    in  float[] values,
    in  int     threads,
    in  bool    local,
    out float   sum,
    src comp    "stages/sum_squares",
)

stage REPORT(
    in  float[] values,
    in  float   sum,
    src exec    "stages/report",
)

pipeline SUM_SQUARE_PIPELINE(
    in  float[] values,
    out float   sum,
)
{
    call SUM_SQUARES(
        values  = self.values,
        threads = 1,
        local   = false,
    )
    call REPORT(
        values = self.values,
        sum    = SUM_SQUARES.sum,
    )

    return (
        sum = SUM_SQUARES.sum,
    )
}

call SUM_SQUARE_PIPELINE(
    values = [1.0, 2.0, 3.0],
)
`, t)
}

func invokeTest(src string, t *testing.T) {
	t.Helper()
	if d, err := os.MkdirTemp("", "pipestance"); err != nil {
		t.Error(err)
	} else {
		defer os.RemoveAll(d)
		t.Log("Invoking pipestance in ", d)
		pdir := util.RelPath("..")
		if d, err := os.Open(pdir); err != nil {
			t.Skip(err)
		} else {
			// hold open the directory so it doesn't disappear on us.
			defer d.Close()
		}
		t.Log("Runtime directory is ", pdir)
		jobPath := path.Join(pdir, "jobmanagers")
		if _, err := os.Stat(jobPath); os.IsNotExist(err) {
			t.Log("Creating ", jobPath)
			// test harness runs in temp dir.  Need to make a fake config.json.
			if err := os.MkdirAll(jobPath, 0777); err != nil {
				t.Skip(err)
			}
			if d, err := os.Open(jobPath); err != nil {
				t.Skip(err)
			} else {
				defer d.Close()
			}
			defer os.RemoveAll(jobPath)
		} else if err != nil {
			t.Skip(err)
		}
		cfg := path.Join(jobPath, "config.json")
		if _, err := os.Stat(cfg); os.IsNotExist(err) {
			t.Log("Creating ", cfg)
			if err := os.WriteFile(cfg, []byte(`{
  "settings": {
    "threads_per_job": 1,
    "memGB_per_job": 1,
    "thread_envs": []
  },
  "jobmodes": {}
}`), 0666); err != nil {
				t.Log(err)
			}
			defer os.Remove(cfg)
		} else if err != nil {
			t.Log(err)
		}
		opts := DefaultRuntimeOptions()
		util.SetupSignalHandlers()
		rt, err := opts.NewRuntime()
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Runtime instantiated.")
		if ps, err := rt.InvokePipeline(src,
			path.Join(d, "src.mro"), "test",
			path.Join(d, "test"), nil, "1.0.0",
			make(map[string]string), nil); err != nil {
			t.Error(err)
		} else if ps == nil {
			t.Errorf("nil pipestance")
		} else if _, err := os.Stat(path.Join(d, "test")); err != nil {
			t.Error(err)
		} else {
			ps.Unlock()
		}
	}
}

func TestInvokeEmpty(t *testing.T) {
	invokeTest(`
stage FOO (
    in  int  val,
    out int  val,
    src comp "foo",
)

pipeline CALL_FOO(
    in  int val,
    out int val,
)
{
    call FOO(
        val = self.val,
    )
    return (
        val = FOO.val,
    )
}

pipeline MAP_FOO(
    in  int[] vals,
    out int[] vals,
)
{
    map call CALL_FOO(
        val = split self.vals,
    )

    return (
        vals = CALL_FOO.val,
    )
}

call MAP_FOO(
    vals = [],
)
`, t)
}

func TestGetCallableFrom(t *testing.T) {
	callable, _, err := GetCallableFrom("MY_STAGE",
		path.Join("stages.mro"), []string{"testdata"})
	if err != nil {
		t.Error(err)
	} else if callable.GetId() != "MY_STAGE" {
		t.Errorf("Expected MY_STAGE, got %q", callable.GetId())
	} else if callable.File().FileName != "stages.mro" {
		t.Errorf("Expected stages.mro, got %q", callable.File().FileName)
	}
	callable, _, err = GetCallableFrom("MY_STAGE",
		path.Join("sub/stages.mro"), []string{"testdata"})
	if err != nil {
		t.Error(err)
	} else if callable.GetId() != "MY_STAGE" {
		t.Errorf("Expected MY_STAGE, got %q", callable.GetId())
	} else if callable.File().FileName != "sub/stages.mro" {
		t.Errorf("Expected stages.mro, got %q", callable.File().FileName)
	}
}

func TestGetCallable(t *testing.T) {
	atd, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	mropaths := []string{"testdata", "testdata/", atd, atd + "/"}
	for i := range mropaths {
		callable, _, err := GetCallable(mropaths[i:], "MY_STAGE", false)
		if err != nil {
			t.Error(err)
		} else if callable.GetId() != "MY_STAGE" {
			t.Errorf("Expected MY_STAGE, got %q", callable.GetId())
		} else if callable.File().FileName != "stages.mro" {
			t.Errorf("Expected stages.mro, got %q (MROPATH: %s)",
				callable.File().FileName,
				strings.Join(mropaths[i:], ":"))
		}
	}
}

func TestInvocationDataFromSource(t *testing.T) {
	const src = `@include "testdata/mock_stages.mro"

map call MOCK_PHASER_SVCALLER(
    sample_def = [
        {
            "gem_group": null,
            "lanes": null,
            "read_path": "testdata/manager/bclprocessor",
            "sample_indices": [
                "AAAGCATA",
                "TCCAATAA",
            ],
        },
    ],
    downsample = {
        "subsample_rate": 1,
    },
    unused     = split [
        1,
        2,
    ],
)
`
	invocationData, err := InvocationDataFromSource([]byte(src), []string{""})
	if err != nil {
		t.Error(err)
	}
	if invocationData == nil {
		t.Fatal("no invocation data returned")
	}
	if len(invocationData.Args) != 3 {
		t.Error("incorrect argument count ", len(invocationData.Args))
	}
	if invocationData.Call != "MOCK_PHASER_SVCALLER" {
		t.Errorf("Expected call MOCK_PHASER_SVCALLER, got %s",
			invocationData.Call)
	}
	if len(invocationData.SplitArgs) != 1 {
		t.Errorf("Expected 1 split arg, got %d", len(invocationData.SplitArgs))
	}
	if len(invocationData.SplitArgs) > 0 &&
		invocationData.SplitArgs[0] != "unused" {
		t.Errorf("expected 'unused' as split arg, got %s",
			invocationData.SplitArgs[0])
	}
	if invocationData.Include != "testdata/mock_stages.mro" {
		t.Errorf("expected include 'testdata/mock_stages.mro', got %q",
			invocationData.Include)
	}
	if s, err := invocationData.BuildCallSource([]string{""}); err != nil {
		t.Error(err)
	} else if s != src {
		t.Errorf("incorrect source:\n%s", s)
	}
}

func ExampleBuildDataForAst() {
	ast := syntax.Ast{
		Includes: []*syntax.Include{{Value: "mro/pipeline.mro"}},
		Call: &syntax.CallStm{
			Id:    "AS_PIPELINE",
			DecId: "PIPELINE",
			Bindings: &syntax.BindStms{
				List: []*syntax.BindStm{
					{
						Id: "bag",
						Exp: &syntax.MapExp{
							Kind: syntax.KindMap,
							Value: map[string]syntax.Exp{
								"key": &syntax.IntExp{Value: 2},
							},
						},
					},
					{
						Id: "config",
						Exp: &syntax.MapExp{
							Kind: syntax.KindStruct,
							Value: map[string]syntax.Exp{
								"value": &syntax.BoolExp{Value: true},
							},
						},
					},
					{
						Id:  "sample_id",
						Exp: &syntax.StringExp{Value: "sample"},
					},
				},
			},
		},
	}
	invocation, err := BuildDataForAst(&ast)
	if err != nil {
		panic(err)
	}
	b, _ := json.MarshalIndent(invocation, "", "\t")
	fmt.Println(string(b))
	// Output:
	// {
	// 	"call": "PIPELINE",
	// 	"args": {
	// 		"bag": {
	// 			"key": 2
	// 		},
	// 		"config": {
	// 			"value": true
	// 		},
	// 		"sample_id": "sample"
	// 	},
	//	"mro_file": "mro/pipeline.mro"
	// }
}

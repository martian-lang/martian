//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

// This is more or less an integration test for what mrp does.

package core

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path"
	"reflect"
	"runtime/trace"
	"strings"
	"testing"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

// Remove any buffered items from the channel.
func flushChannel(c <-chan struct{}) {
	for {
		select {
		case <-c:
		default:
			return
		}
	}
}

func loopBody(t *testing.T, pipestance *Pipestance) (bool, bool) {
	ctx, task := trace.NewTask(context.Background(), "update")
	defer task.End()
	pipestance.RefreshState(ctx)

	// Check for completion states.
	state := pipestance.GetState(ctx)
	if state == Complete || state == DisabledState {
		return true, false
	} else if state == Failed {
		_, _, _, log, _, errPaths := pipestance.GetFatalError()
		t.Error(log)
		for _, p := range errPaths {
			t.Log("Errors in", p)
			b, err := os.ReadFile(p)
			if err != nil {
				t.Error(err)
			} else {
				t.Error(string(b))
			}
		}
		return true, false
	} else {
		// Check job heartbeats.
		pipestance.CheckHeartbeats(ctx)

		// Step all nodes.
		progress := pipestance.StepNodes(ctx)
		return false, progress
	}
}

type testLogger struct {
	t *testing.T
}

func (t testLogger) Write(b []byte) (int, error) {
	t.t.Helper()
	t.t.Log(string(bytes.TrimSpace(b)))
	return len(b), nil
}

func (t testLogger) WriteString(b string) (int, error) {
	t.t.Helper()
	t.t.Log(strings.TrimSpace(b))
	return len(b), nil
}

// Runs the pipestance for the given mro file to completion, and returns it
// along with the directory it ran in.  The directory is removed when the
// test completes.
func runTestPipestance(t *testing.T, mroPath string) (*Pipestance, string) {
	t.Helper()
	data, err := os.ReadFile(mroPath)
	if err != nil {
		t.Fatal(err)
	}
	util.SetPrintLogger(testLogger{t: t})
	defer util.SetPrintLogger(&devNull)
	rtOpts := DefaultRuntimeOptions()
	rt := Runtime{
		Config: &rtOpts,
	}
	rt.jobConfig = &JobManagerJson{
		JobSettings: &JobManagerSettings{
			ThreadsPerJob: 1,
			MemGBPerJob:   1,
			ExtraVmemGB:   1,
			ThreadEnvs:    []string{"GOMAXPROCS"},
		},
	}
	rt.LocalJobManager, err = NewLocalJobManager(4,
		4, 16,
		true,
		false,
		false,
		rt.jobConfig)
	if err != nil {
		t.Fatal(err)
	}
	rt.JobManager = rt.LocalJobManager
	psdir, err := os.MkdirTemp("", "TestPipestanceRun")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(psdir) })
	t.Log("Starting pipestance in", psdir)
	pipestance, err := rt.InvokePipeline(string(data),
		mroPath, t.Name(),
		psdir, []string{"testdata"}, "<none>", nil, nil)
	if err != nil {
		t.Fatal("Invoking pipeline:", err)
	}
	pipestance.LoadMetadata(context.Background())

	ti := time.NewTimer(0)
	if !ti.Stop() {
		<-ti.C
	}
	for {
		flushChannel(rt.LocalJobManager.Done())
		done, hadProgress := loopBody(t, pipestance)

		if done {
			break
		}

		if !hadProgress {
			// Wait for a either stepSecs or until a local job finishes.
			ti.Reset(time.Second)
			select {
			case <-ti.C:
			case <-rt.LocalJobManager.Done():
				if !ti.Stop() {
					<-ti.C
				}
			}
		}
	}

	// Test that serializing the state works correctly with a canceled context.
	// Coverage from this call is going to be racy, of course, but it's very
	// difficult to get coverage more reliably here.
	// This essentially simulates the effect of an API client querying the
	// state while the pipestance is running.
	ch := make(chan interface{}, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Microsecond)
	go func() { ch <- pipestance.SerializePerf(ctx) }()
	go func() { ch <- pipestance.SerializeState(ctx) }()
	<-ch
	<-ch
	cancel()
	return pipestance, psdir
}

// Tests actually running a pipestance.
//
// The reason this test exists, rather than simply relying on the end-to-end
// integration tests, is mainly to be able to see code coverage.  It's also
// very fast because of the trivial stage code.
func TestPipestanceRun(t *testing.T) {
	pipestance, psdir := runTestPipestance(t, "testdata/map_call_edge_cases.mro")

	nodeInfos := pipestance.SerializeState(context.Background())
	if len(nodeInfos) != 27 {
		t.Errorf("node count %d != 27", len(nodeInfos))
	}
	outs, err := os.ReadFile(path.Join(psdir, "TOP",
		defaultFork,
		OutsFile.FileName()))
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(outs))
	dec.DisallowUnknownFields()
	type Inputs1 struct {
		T1 string `json:"thing1"`
		T2 int    `json:"thing2"`
		T3 int    `json:"thing3"`
	}
	type Inputs2 struct {
		Inputs []Inputs1 `json:"inputs"`
		Thing  bool      `json:"thing"`
	}
	type Inputs4 struct {
		Outs []*struct {
			Blah  []*string `json:"blah"`
			Thing int       `json:"thing"`
		} `json:"outs"`
	}
	var outputs struct {
		Outputs *struct {
			ExpectNull1 []Inputs2 `json:"expect_null1"`
			ExpectNull2 []Inputs2 `json:"expect_null2"`
			Output      *struct {
				Result string `json:"result"`
			} `json:"output"`
			I1 map[string]*Inputs4 `json:"intermediate1"`
			I2 []*Inputs4          `json:"intermediate2"`
			I3 []*Inputs4          `json:"intermediate3"`
			I4 map[string]*Inputs4 `json:"intermediate4"`
			I5 []*Inputs4          `json:"intermediate5"`
		} `json:"outputs"`
	}
	t.Log(string(outs))
	if err := dec.Decode(&outputs); err != nil {
		t.Error(err)
	}
	if outputs.Outputs == nil {
		t.Fatal("outputs was null")
	}
	if outputs.Outputs.ExpectNull1 != nil {
		t.Error("expected null output 1")
	}
	if outputs.Outputs.ExpectNull2 != nil {
		t.Error("expected null output 2")
	}
	if outputs.Outputs.Output == nil {
		t.Error("outputs.output was null")
	} else if outputs.Outputs.Output.Result != "thingy" {
		t.Errorf(`%q != "thingy"`, outputs.Outputs.Output.Result)
	}
	if len(outputs.Outputs.I1) != 1 {
		t.Errorf("incorrect length %d != 1 for intermediate1",
			len(outputs.Outputs.I1))
	} else if len(outputs.Outputs.I1["foo"].Outs) != 1 {
		t.Errorf("incorrect length %d != 1 for intermediate1.foo",
			len(outputs.Outputs.I1["foo"].Outs))
	}
	if len(outputs.Outputs.I2) != 2 {
		t.Errorf("incorrect length %d != 2 for intermediate2",
			len(outputs.Outputs.I2))
	} else {
		if len(outputs.Outputs.I2[0].Outs) != 1 {
			t.Errorf("incorrect length %d != 1 for intermediate2[0].foo",
				len(outputs.Outputs.I2[0].Outs))
		}
		if len(outputs.Outputs.I2[1].Outs) != 1 {
			t.Errorf("incorrect length %d != 1 for intermediate2[1].foo",
				len(outputs.Outputs.I2[1].Outs))
		}
	}
	if len(outputs.Outputs.I3) != 1 {
		t.Errorf("incorrect length %d != 1 for intermediate3",
			len(outputs.Outputs.I3))
	} else if len(outputs.Outputs.I3[0].Outs) != 1 {
		t.Errorf("incorrect length %d != 1 for intermediate3.foo",
			len(outputs.Outputs.I3[0].Outs))
	}
	if len(outputs.Outputs.I4) != 3 {
		t.Errorf("incorrect length %d != 2 for intermediate4",
			len(outputs.Outputs.I4))
	} else {
		if b := outputs.Outputs.I4["thing1"].Outs; len(b) != 2 {
			t.Errorf("incorrect length %d != 1 for intermediate4[thing1].outs",
				len(b))
		}
		if b := outputs.Outputs.I4["thing2"].Outs; len(b) != 1 {
			t.Errorf("incorrect length %d != 1 for intermediate4[thing2].outs",
				len(b))
		} else if b[0] != nil {
			t.Error("expected thing2 to be nil")
		}
		if b := outputs.Outputs.I4["thing3"].Outs; len(b) != 0 {
			t.Errorf("incorrect length %d != 0 for intermediate4[thing3].outs",
				len(b))
		}
	}
	if len(outputs.Outputs.I5) != 1 {
		t.Errorf("incorrect length %d != 1 for intermediate5",
			len(outputs.Outputs.I5))
	} else if b := outputs.Outputs.I5[0].Outs; len(b) != 1 {
		t.Errorf("incorrect length %d != 1 for intermediate5[0].outs",
			len(b))
	}
}

// Tests running a pipeline where a map call contains a disabled sub-pipeline,
// so that the merge of the pruned fork has to be resolved.
func TestPipestanceRunPrunedMerge(t *testing.T) {
	_, psdir := runTestPipestance(t, "testdata/pruned_merge.mro")
	outs, err := os.ReadFile(path.Join(psdir, "OUTER",
		defaultFork,
		OutsFile.FileName()))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(outs))
	var outputs struct {
		Ys map[string]int  `json:"ys"`
		Zs map[string]*int `json:"zs"`
	}
	if err := json.Unmarshal(outs, &outputs); err != nil {
		t.Fatal(err)
	}
	if len(outputs.Ys) != 2 ||
		outputs.Ys["a"] != 1 || outputs.Ys["b"] != 2 {
		t.Errorf("unexpected ys: %v", outputs.Ys)
	}
	if len(outputs.Zs) != 2 {
		t.Errorf("expected zs to be the same length as Ys (2), got %d", len(outputs.Zs))
	}
	if outputs.Zs["a"] != nil || outputs.Zs["b"] != nil {
		t.Errorf("expected all outputs for zs to be null, got %v",
			outputs.Zs)
	}
}

// Tests running a pipeline where a map call over a sub-pipeline contains a
// stage which is disabled by a value that is available before the split
// source is, and the merged output of that stage is consumed downstream.
func TestPipestanceRunMapCallDisabledInner(t *testing.T) {
	_, psdir := runTestPipestance(t, "testdata/map_call_disabled_inner.mro")
	outs, err := os.ReadFile(path.Join(psdir, "TOP",
		defaultFork,
		OutsFile.FileName()))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(outs))
	var outputs struct {
		UsedA map[string]*int `json:"used_a"`
	}
	if err := json.Unmarshal(outs, &outputs); err != nil {
		t.Fatal(err)
	}
	if len(outputs.UsedA) != 2 {
		t.Errorf("expected used_a to have 2 keys, got %v", outputs.UsedA)
	}
	for _, k := range []string{"x", "y"} {
		if v, ok := outputs.UsedA[k]; !ok {
			t.Errorf("missing key %s in used_a", k)
		} else if v != nil {
			t.Errorf("expected used_a[%s] to be null, got %d", k, *v)
		}
	}
}

// Tests running a pipeline where a map call splits over the output of a map
// call inside a sub-pipeline which is disabled by a value determined at
// runtime.  When the sub-pipeline is disabled, its output is null, so the
// downstream map call has no forks, and its output is an empty collection.
func TestPipestanceRunMapCallDisabledPipeline(t *testing.T) {
	for _, c := range []struct {
		expect any
		mro    string
	}{
		{
			mro:    "testdata/map_call_disabled_pipeline.mro",
			expect: map[string]any{"x": 1.0, "y": 2.0},
		},
		{
			mro:    "testdata/map_call_disabled_pipeline_static.mro",
			expect: map[string]any{"x": 1.0, "y": 2.0},
		},
		{
			mro:    "testdata/map_call_disabled_pipeline_on.mro",
			expect: map[string]any{},
		},
		{
			mro:    "testdata/map_call_disabled_pipeline_array_on.mro",
			expect: []any{},
		},
	} {
		t.Run(path.Base(c.mro), func(t *testing.T) {
			_, psdir := runTestPipestance(t, c.mro)
			outs, err := os.ReadFile(path.Join(psdir, "TOP",
				defaultFork,
				OutsFile.FileName()))
			if err != nil {
				t.Fatal(err)
			}
			t.Log(string(outs))
			var outputs struct {
				Result any `json:"result"`
			}
			if err := json.Unmarshal(outs, &outputs); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(outputs.Result, c.expect) {
				t.Errorf("result %v != expected %v", outputs.Result, c.expect)
			}
		})
	}
}

// Tests running pipelines where a map call splits over the output of a map
// call inside a sub-pipeline which is disabled by a value determined at
// runtime, in cases involving nested collections.
func TestPipestanceRunMapCallDisabledPipelineNested(t *testing.T) {
	for _, c := range []struct {
		expect map[string]any
		mro    string
	}{
		{
			mro: "testdata/map_call_disabled_pipeline_map_of_arrays.mro",
			expect: map[string]any{
				"result_on": map[string]any{},
				"result_off": map[string]any{
					"x": []any{1.0, 2.0},
					"y": []any{3.0},
				},
			},
		},
		{
			mro: "testdata/map_call_disabled_pipeline_array_of_maps.mro",
			expect: map[string]any{
				"result_on": []any{},
				"result_off": []any{
					map[string]any{"x": 1.0},
					map[string]any{"y": 2.0},
				},
			},
		},
		{
			mro: "testdata/map_call_disabled_pipeline_mapped_parent.mro",
			expect: map[string]any{
				"result_first_on": []any{
					map[string]any{},
					map[string]any{"x": 2.0},
				},
				"result_last_on": []any{
					map[string]any{"x": 1.0},
					map[string]any{},
				},
			},
		},
		{
			mro: "testdata/map_call_disabled_pipeline_mapped_parent_keys.mro",
			expect: map[string]any{
				"result_first_on": []any{
					map[string]any{},
					map[string]any{"y": 2.0},
				},
				"result_last_on": []any{
					map[string]any{"x": 1.0},
					map[string]any{},
				},
			},
		},
		{
			mro: "testdata/map_call_mapped_pipeline_keys.mro",
			expect: map[string]any{
				"result": []any{
					map[string]any{"x": 1.0},
					map[string]any{"y": 2.0},
				},
			},
		},
	} {
		t.Run(path.Base(c.mro), func(t *testing.T) {
			_, psdir := runTestPipestance(t, c.mro)
			outs, err := os.ReadFile(path.Join(psdir, "TOP",
				defaultFork,
				OutsFile.FileName()))
			if err != nil {
				t.Fatal(err)
			}
			t.Log(string(outs))
			var outputs map[string]any
			if err := json.Unmarshal(outs, &outputs); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(outputs, c.expect) {
				t.Errorf("result %v != expected %v", outputs, c.expect)
			}
		})
	}
}

// Tests running pipelines which exercise fork expansion for map calls over
// literal sources, partially matched nested sources, and sources which are
// disabled by a value that is part of an enclosing split.
func TestPipestanceRunMapCallForkExpansion(t *testing.T) {
	for _, c := range []struct {
		expect map[string]any
		mro    string
	}{
		{
			mro: "testdata/map_call_nested_partial_parts.mro",
			expect: map[string]any{
				"result": []any{
					map[string]any{
						"a": []any{1.0, 2.0},
						"b": []any{3.0},
					},
					map[string]any{
						"c": []any{4.0},
					},
				},
			},
		},
		{
			mro: "testdata/map_call_literal_sources.mro",
			expect: map[string]any{
				"arr": []any{
					[]any{1.0, 2.0},
					[]any{2.0, 2.0},
					[]any{3.0, 2.0},
				},
				"one": []any{
					map[string]any{"k": 1.0},
					map[string]any{"k": 2.0},
					map[string]any{"k": 3.0},
				},
				"two": []any{
					map[string]any{"a": 1.0, "b": 5.0},
					map[string]any{"a": 2.0, "b": 5.0},
					map[string]any{"a": 3.0, "b": 5.0},
				},
			},
		},
		{
			mro: "testdata/map_call_static_literals.mro",
			expect: map[string]any{
				"multi":  map[string]any{"a": 1.0, "b": 2.0},
				"single": map[string]any{"only": 7.0},
				"arr":    []any{4.0, 5.0},
				"nested": []any{
					map[string]any{"p": 1.0, "q": 2.0},
					map[string]any{"p": 2.0, "q": 2.0},
				},
			},
		},
		{
			mro: "testdata/map_call_disabled_by_split.mro",
			expect: map[string]any{
				"result": []any{
					map[string]any{},
					map[string]any{"x": 1.0, "y": 2.0},
				},
			},
		},
	} {
		t.Run(path.Base(c.mro), func(t *testing.T) {
			_, psdir := runTestPipestance(t, c.mro)
			outs, err := os.ReadFile(path.Join(psdir, "TOP",
				defaultFork,
				OutsFile.FileName()))
			if err != nil {
				t.Fatal(err)
			}
			t.Log(string(outs))
			var outputs map[string]any
			if err := json.Unmarshal(outs, &outputs); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(outputs, c.expect) {
				t.Errorf("result %v != expected %v", outputs, c.expect)
			}
		})
	}
}

// Tests running a pipeline where a map call over the output of a
// runtime-disabled sub-pipeline is inside a statically mapped pipeline, and
// the sub-pipeline is disabled in more than one nonzero instance.  The empty
// forks for those instances must have distinct IDs, rather than all sharing
// the node's directory.
func TestPipestanceRunMapCallDisabledPipelineMultiEmpty(t *testing.T) {
	pipestance, psdir := runTestPipestance(t,
		"testdata/map_call_disabled_pipeline_mapped_parent_multi.mro")
	for _, node := range pipestance.allNodes() {
		paths := make(map[string]string, len(node.forks))
		for _, fork := range node.forks {
			if fork.path == node.path {
				t.Errorf("fork %s of %s has the node's own path",
					fork.forkId.GoString(), node.GetFQName())
			}
			if other, ok := paths[fork.path]; ok {
				t.Errorf("forks %s and %s of %s share path %s",
					other, fork.forkId.GoString(),
					node.GetFQName(), fork.path)
			}
			paths[fork.path] = fork.forkId.GoString()
		}
	}
	outs, err := os.ReadFile(path.Join(psdir, "TOP",
		defaultFork,
		OutsFile.FileName()))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(outs))
	var outputs map[string]any
	if err := json.Unmarshal(outs, &outputs); err != nil {
		t.Fatal(err)
	}
	expect := map[string]any{
		"result_first_on": []any{
			map[string]any{"x": 1.0},
			map[string]any{},
			map[string]any{},
		},
		"result_middle_on": []any{
			map[string]any{},
			map[string]any{"x": 2.0},
			map[string]any{},
		},
	}
	if !reflect.DeepEqual(outputs, expect) {
		t.Errorf("result %v != expected %v", outputs, expect)
	}
}

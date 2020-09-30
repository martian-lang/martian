//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

// This is more or less an integration test for what mrp does.

package core

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
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
			b, err := ioutil.ReadFile(p)
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

// Tests actually running a pipestance.
//
// The reason this test exists, rather than simply relying on the end-to-end
// integration tests, is mainly to be able to see code coverage.  It's also
// very fast because of the trivial stage code.
func TestPipestanceRun(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/map_call_edge_cases.mro")
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
	rt.LocalJobManager = NewLocalJobManager(4,
		4, 16,
		true,
		false,
		false,
		rt.jobConfig)
	rt.JobManager = rt.LocalJobManager
	psdir, err := ioutil.TempDir("", "TestPipestanceRun")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(psdir)
	t.Log("Starting pipestance in", psdir)
	pipestance, err := rt.InvokePipeline(string(data),
		"testdata/map_call_edge_cases.mro", path.Base(psdir),
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
	outs, err := os.Open(path.Join(psdir, "TOP",
		defaultFork,
		OutsFile.FileName()))
	if err != nil {
		t.Fatal(err)
	}
	defer outs.Close()
	dec := json.NewDecoder(outs)
	dec.DisallowUnknownFields()
	type Inputs4 struct {
		Outs []*struct {
			Blah  []*string `json:"blah"`
			Thing int       `json:"thing"`
		} `json:"outs"`
	}
	var outputs struct {
		Outputs *struct {
			Output *struct {
				Result string `json:"result"`
			} `json:"output"`
			I1 map[string]*Inputs4 `json:"intermediate1"`
			I2 []*Inputs4          `json:"intermediate2"`
			I3 []*Inputs4          `json:"intermediate3"`
			I4 map[string]*Inputs4 `json:"intermediate4"`
		} `json:"outputs"`
	}
	if err := dec.Decode(&outputs); err != nil {
		t.Error(err)
	}
	outs.Close()
	if outputs.Outputs == nil {
		t.Fatal("outputs was null")
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
			len(outputs.Outputs.I1))
	}
	if len(outputs.Outputs.I2) != 2 {
		t.Errorf("incorrect length %d != 2 for intermediate2",
			len(outputs.Outputs.I2))
	} else {
		if len(outputs.Outputs.I2[0].Outs) != 1 {
			t.Errorf("incorrect length %d != 1 for intermediate2[0].foo",
				len(outputs.Outputs.I1))
		}
		if len(outputs.Outputs.I2[1].Outs) != 1 {
			t.Errorf("incorrect length %d != 1 for intermediate2[1].foo",
				len(outputs.Outputs.I1))
		}
	}
	if len(outputs.Outputs.I3) != 1 {
		t.Errorf("incorrect length %d != 1 for intermediate3",
			len(outputs.Outputs.I1))
	} else if len(outputs.Outputs.I3[0].Outs) != 1 {
		t.Errorf("incorrect length %d != 1 for intermediate3.foo",
			len(outputs.Outputs.I1))
	}
	if len(outputs.Outputs.I4) != 2 {
		t.Errorf("incorrect length %d != 2 for intermediate4",
			len(outputs.Outputs.I2))
	} else {
		if len(outputs.Outputs.I4["thing1"].Outs) != 1 {
			t.Errorf("incorrect length %d != 1 for intermediate2[0].foo",
				len(outputs.Outputs.I1))
		}
		if b := outputs.Outputs.I4["thing2"].Outs; len(b) != 1 {
			t.Errorf("incorrect length %d != 1 for intermediate2[1].foo",
				len(outputs.Outputs.I1))
		} else if b[0] != nil {
			t.Error("expected thing2 to be nil")
		}
	}
}

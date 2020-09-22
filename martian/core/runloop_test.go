//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

// This is more or less an integration test for what mrp does.

package core

import (
	"bytes"
	"context"
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
			return
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
}

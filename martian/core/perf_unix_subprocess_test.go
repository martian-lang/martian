// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.
// +build freebsd linux netbsd openbsd solaris

package core

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

func runGetProcessVsize(t *testing.T, vsize, rss string) (ObservedMemory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python", "testdata/vsize.py", vsize, rss)
	cmd.SysProcAttr = util.Pdeathsig(&syscall.SysProcAttr{}, syscall.SIGKILL)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := cmd.Wait(); err != nil {
			t.Error(err)
		}
	}()
	// Wait for the test script to set up.  It's supposed signal that state
	// by closing its stdout.
	if _, err := ioutil.ReadAll(stdout); err != nil {
		t.Error(err)
	}
	mem, err := GetRunningMemory(cmd.Process.Pid)
	if err := stdin.Close(); err != nil {
		t.Error(err)
	}
	return mem, err
}

// Run a python script that uses 1GB of vmem and 10MB of rss (plus python
// overhead) and confirm that the measured usage is as expected.
func TestGetProcessVsize(t *testing.T) {
	t.Parallel()
	if mem, err := runGetProcessVsize(t, "1048576", "10240"); err != nil {
		t.Error(err)
	} else {
		if rss := mem.RssKb(); rss < 10240 {
			t.Errorf("Expected at least 10240kb rss usage, got %d", rss)
		} else if rss > 10*10240 {
			t.Errorf("Expected 10240kb rss usage, plus overhead, got %d", rss)
		}
		if vmem := mem.VmemKb(); vmem < 1048576 {
			t.Errorf("Expected at least 1048576kb vmem usage, got %d", vmem)
		}
	}
}

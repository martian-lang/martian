// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package main

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

func Test_waitChildren(t *testing.T) {
	dl, ok := t.Deadline()
	if !ok {
		dl = time.Now().Add(time.Second * 5)
	}
	dl = dl.Add(-time.Millisecond * 200)
	ctx, cancel := context.WithDeadline(context.Background(), dl)
	defer cancel()
	setSubreaper()
	ctx1, cancel1 := context.WithTimeout(ctx, time.Second/2)
	defer cancel1()
	// Launch a shell process to create an orphan.
	cmd := exec.CommandContext(ctx1, "sh", "-c", "echo annie&")
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
	var buf bytes.Buffer
	util.LogTeeWriter(&buf)
	// Wait for grandchild to terminate.
	waitChildren()
	for buf.Len() == 0 {
		select {
		case <-ctx1.Done():
			t.Fatal("no messages")
		case <-time.After(time.Millisecond):
			waitChildren()
		}
	}
	cancel1()
	output := buf.String()
	if !strings.Contains(output, "orphaned child process") {
		t.Errorf("Expected message about orphaned child process, got %q", output)
	} else if !strings.HasSuffix(strings.TrimSpace(output), "status 0") {
		t.Error(output)
	}

	ctx2, cancel2 := context.WithTimeout(ctx, time.Second/2)
	defer cancel2()
	cmd = exec.CommandContext(ctx2, "sh", "-c", "setsid sh -c 'kill $$'&")
	buf.Reset()
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}

	waitChildren()
	// Wait for grandchild to terminate.
	for buf.Len() == 0 {
		select {
		case <-ctx2.Done():
			t.Fatal("no messages")
		case <-time.After(time.Millisecond):
			waitChildren()
		}
	}
	cancel2()
	output = buf.String()
	if !strings.Contains(output, "orphaned child process") {
		t.Errorf("Expected message about orphaned child process, got %q", output)
	} else if !strings.HasSuffix(strings.TrimSpace(output), "signal terminated") {
		t.Error(output)
	}
}

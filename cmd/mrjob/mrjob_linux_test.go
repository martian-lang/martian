// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

func Test_reportChildren(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	util.LogTeeWriter(&buf)
	defer func() {
		util.LOGGER = nil
	}()
	if !reportChildren() {
		t.Error("Expected to find child process.")
	}
	output := string(bytes.TrimSpace(buf.Bytes()))
	if !strings.HasSuffix(output, "(sleep 5) is still running (state S).") &&
		!strings.HasSuffix(output, "(sleep 5) is still running (state R).") {
		t.Errorf("expected (sleep 5) is still running (state S or R), got\n%s",
			output)
	} else {
		t.Log(output)
	}
	_ = cmd.Process.Kill()
	buf.Reset()
	time.Sleep(100 * time.Millisecond)
	if !reportChildren() {
		t.Error("Expected to find child process.")
	}
	output = string(bytes.TrimSpace(buf.Bytes()))
	if !strings.HasSuffix(output, "is still running (state Z).") {
		t.Errorf("expected is still running (state Z), got\n%s",
			output)
	} else {
		t.Log(output)
	}
	_ = cmd.Wait()
	if reportChildren() {
		t.Error("Didn't expect a child process.")
	}
}

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
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Error("Error running command:", err)
	}
	t.Log("child PID:", cmd.ProcessState.Pid())
	var buf bytes.Buffer
	util.LogTeeWriter(&buf)
	defer func() {
		util.LOGGER = nil
	}()
	// Wait for grandchild to terminate.
	for waitChildren() {
		if err := ctx1.Err(); err != nil {
			t.Error(err)
			break
		}
		time.Sleep(time.Millisecond)
	}
	if waitChildren() {
		t.Error("Not all child jobs finished.")
	}
	cancel1()
	output := string(bytes.TrimSpace(buf.Bytes()))
	if !strings.Contains(output, "orphaned child process") {
		t.Errorf("Expected message about orphaned child process, got\n%s", output)
	} else if !strings.HasSuffix(output, "status 0") {
		t.Errorf("Expected to see status 0, but got\n%s", output)
	} else {
		t.Log(output)
	}

	ctx2, cancel2 := context.WithTimeout(ctx, time.Second/2)
	defer cancel2()
	cmd = exec.CommandContext(ctx2, "sh", "-c", "setsid sh -c 'kill $$'&")
	cmd.Stderr = os.Stderr
	buf.Reset()
	if err := cmd.Run(); err != nil {
		t.Error("Error running command:", err)
	}
	t.Log("child PID:", cmd.ProcessState.Pid())

	// Wait for grandchild to terminate.
	for waitChildren() {
		if err := ctx2.Err(); err != nil {
			t.Error(err)
			break
		}
		time.Sleep(time.Millisecond)
	}
	cancel2()
	output = string(bytes.TrimSpace(buf.Bytes()))
	if !strings.Contains(output, "orphaned child process") {
		t.Errorf("Expected message about orphaned child process, got\n%s", output)
	} else if !strings.HasSuffix(output, "signal terminated") {
		t.Errorf("Expected to see signal terminated, but got\n%s", output)
	} else {
		t.Log(output)
	}

	ctx3, cancel3 := context.WithTimeout(ctx, time.Second/2)
	defer cancel3()
	cmd = exec.CommandContext(ctx3, "sh", "-c",
		`setsid bash -c 'setsid sh -c "echo foo&"&'&`)
	cmd.Stderr = os.Stderr
	buf.Reset()
	if err := cmd.Run(); err != nil {
		t.Error("Error running command:", err)
	}
	t.Log("child PID:", cmd.ProcessState.Pid())

	// Wait for grandchild to terminate.
	for waitChildren() {
		if err := ctx3.Err(); err != nil {
			t.Error(err)
			break
		}
		time.Sleep(time.Millisecond)
	}
	cancel3()
	output = string(bytes.TrimSpace(buf.Bytes()))
	if !strings.Contains(output, "orphaned child process") {
		t.Errorf("Expected message about orphaned child process, got\n%s", output)
	} else if strings.Count(output, "\n") < 2 {
		t.Errorf("Expected to see 3 child process terminations, but got\n%s", output)
	} else {
		t.Log(output)
	}
}

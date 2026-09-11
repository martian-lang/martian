// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

func Test_reportChildren(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sleep", "5")
	cmd.Stderr = os.Stderr
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
	if !regexp.MustCompile(
		`\(sleep(?: 5)?\) is still running \(state [SR]\).$`).MatchString(output) {
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

// orphanChainScript is a shell script which creates a chain of processes,
// each of which blocks until the test tells it to exit.
//
// Each process forks the next one in the chain before blocking on its own
// file descriptor, and the test releases the processes in order from the
// outermost inwards, so every process in the chain is guaranteed to be alive
// when its parent exits.  That is what makes the test deterministic: a
// process which exits before its parent may be reaped by that parent (shells
// reap their own background jobs), in which case it never gets reparented to
// the subreaper and this test would never see it.
const orphanChainScript = `#!/bin/sh
# usage: sh chain.sh <depth> <fd> [kill]
#
# Forks a chain of <depth> processes.  The process at the top of the chain
# waits for the write end of the pipe on file descriptor <fd> to be closed,
# the next one waits on <fd>+1, and so on.  If "kill" is given, the innermost
# process terminates itself with a signal rather than exiting normally.
depth="$1"
fd="$2"
if [ "$depth" -gt 1 ]; then
	sh "$0" "$((depth - 1))" "$((fd + 1))" "$3" &
fi
# Block until the test closes the write end of the pipe on this descriptor.
eval "read line <&$fd" || true
if [ "$depth" -eq 1 ] && [ "$3" = kill ]; then
	kill $$
	# Just in case the signal doesn't arrive promptly, don't exit normally.
	sleep 10
	exit 1
fi
`

// startOrphanChain runs orphanChainScript from a process which exits
// immediately, so that the chain is orphaned and reparented to this process,
// which is a child subreaper.
//
// It returns the write ends of the pipes the chain is blocked on; closing the
// first one allows the outermost process to exit, and so on.
func startOrphanChain(ctx context.Context, t *testing.T,
	script string, depth int, action string) []*os.File {
	t.Helper()
	cmd := exec.CommandContext(ctx, "sh", "-c", `sh "$0" "$1" 3 "$2" &`,
		script, strconv.Itoa(depth), action)
	cmd.Stderr = os.Stderr
	pipes := make([]*os.File, 0, depth)
	readers := make([]*os.File, 0, depth)
	defer func() {
		for _, r := range readers {
			r.Close()
		}
	}()
	for i := 0; i < depth; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal("Error creating pipe:", err)
		}
		readers = append(readers, r)
		// The child sees these as file descriptors 3, 4, ...
		cmd.ExtraFiles = append(cmd.ExtraFiles, r)
		pipes = append(pipes, w)
	}
	if err := cmd.Run(); err != nil {
		t.Error("Error running command:", err)
	}
	t.Log("child PID:", cmd.ProcessState.Pid())
	return pipes
}

// releaseOrphan closes w, which allows one process in the chain to exit, and
// then waits for waitChildren to collect it.
//
// It returns the log message which waitChildren produced for that process.
func releaseOrphan(ctx context.Context, t *testing.T,
	buf *bytes.Buffer, w *os.File) string {
	t.Helper()
	buf.Reset()
	if err := w.Close(); err != nil {
		t.Error("Error closing pipe:", err)
	}
	for {
		more := waitChildren()
		if s := string(bytes.TrimSpace(buf.Bytes())); s != "" {
			return s
		}
		if !more {
			t.Error("Child process terminated without being reported.")
			return ""
		}
		if err := ctx.Err(); err != nil {
			t.Error(err)
			return ""
		}
		time.Sleep(time.Millisecond)
	}
}

// cleanupOrphans releases any processes which are still running, so that a
// failed test case doesn't leak child processes into the next one.
func cleanupOrphans(ctx context.Context, t *testing.T, pipes []*os.File) {
	t.Helper()
	for _, w := range pipes {
		// Ignore errors, since these are usually already closed.
		_ = w.Close()
	}
	for waitChildren() {
		if err := ctx.Err(); err != nil {
			t.Error("Child processes did not all terminate:", err)
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func Test_waitChildren(t *testing.T) {
	dl, ok := t.Deadline()
	if !ok {
		dl = time.Now().Add(time.Second * 5)
	}
	dl = dl.Add(-time.Millisecond * 200)
	ctx, cancel := context.WithDeadline(t.Context(), dl)
	defer cancel()
	setSubreaper()
	script := filepath.Join(t.TempDir(), "chain.sh")
	if err := os.WriteFile(script, []byte(orphanChainScript), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	util.LogTeeWriter(&buf)
	defer func() {
		util.LOGGER = nil
	}()

	t.Run("exit", func(t *testing.T) {
		pipes := startOrphanChain(ctx, t, script, 1, "")
		defer cleanupOrphans(ctx, t, pipes)
		if !waitChildren() {
			t.Error("Expected an orphaned child process.")
		}
		output := releaseOrphan(ctx, t, &buf, pipes[0])
		if !strings.Contains(output, "orphaned child process") {
			t.Errorf("Expected message about orphaned child process, got\n%s", output)
		} else if !strings.HasSuffix(output, "status 0") {
			t.Errorf("Expected to see status 0, but got\n%s", output)
		} else {
			t.Log(output)
		}
		if waitChildren() {
			t.Error("Not all child jobs finished.")
		}
	})

	t.Run("signal", func(t *testing.T) {
		pipes := startOrphanChain(ctx, t, script, 1, "kill")
		defer cleanupOrphans(ctx, t, pipes)
		output := releaseOrphan(ctx, t, &buf, pipes[0])
		if !strings.Contains(output, "orphaned child process") {
			t.Errorf("Expected message about orphaned child process, got\n%s", output)
		} else if !strings.HasSuffix(output, "signal terminated") {
			t.Errorf("Expected to see signal terminated, but got\n%s", output)
		} else {
			t.Log(output)
		}
		if waitChildren() {
			t.Error("Not all child jobs finished.")
		}
	})

	t.Run("chain", func(t *testing.T) {
		const depth = 3
		pipes := startOrphanChain(ctx, t, script, depth, "")
		defer cleanupOrphans(ctx, t, pipes)
		// Each process in the chain is reparented to this process as its
		// parent exits, so they should be reported one at a time, in order.
		for i, w := range pipes {
			output := releaseOrphan(ctx, t, &buf, w)
			if !strings.Contains(output, "orphaned child process") {
				t.Errorf("Expected message about orphaned child process %d, got\n%s",
					i+1, output)
			} else if strings.Count(output, "\n") != 0 {
				t.Errorf("Expected exactly one termination message for process %d, got\n%s",
					i+1, output)
			} else if !strings.HasSuffix(output, "status 0") {
				t.Errorf("Expected to see status 0, but got\n%s", output)
			} else {
				t.Log(output)
			}
			if remaining := waitChildren(); remaining != (i+1 < depth) {
				t.Errorf("After %d of %d child processes terminated, waitChildren returned %v",
					i+1, depth, remaining)
			}
		}
	})
}

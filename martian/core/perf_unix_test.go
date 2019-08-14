//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//
// Converts system rusage into our structures.
//

package core

import (
	"os"
	"testing"
)

func TestGetUserProcessCount(t *testing.T) {
	if c, err := GetUserProcessCount(); err != nil {
		t.Error(err)
	} else if c < 2 {
		t.Errorf("Expected at least 2 processes for this uid, got %d.", c)
	} else {
		t.Log("Current user process count", c)
	}
}

func TestGetProcessTreeMemory(t *testing.T) {
	io := make(map[int]*IoAmount)
	if mem1, err := GetProcessTreeMemory(os.Getpid(), true, io); err != nil {
		t.Error(err)
	} else if mem2, err := GetProcessTreeMemory(os.Getppid(), true, io); err != nil {
		t.Skip(err)
	} else if mem1.Rss >= mem2.Rss {
		t.Errorf("Tree including parent had %d, while this process had %d",
			mem2.RssKb(), mem1.RssKb())
	}
}

func BenchmarkGetProcessTreeMemory(b *testing.B) {
	pid := os.Getppid()
	io := make(map[int]*IoAmount)
	for n := 0; n < b.N; n++ {
		if _, err := GetProcessTreeMemory(pid, true, io); err != nil {
			b.Error(err)
		}
	}
}

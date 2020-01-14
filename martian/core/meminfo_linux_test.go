// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package core

import "testing"

func TestMemInfo(t *testing.T) {
	var m MemInfo
	if err := m.Get(); err != nil {
		t.Fatal(err)
	}
	if m.Total <= 0 {
		t.Errorf("Expected nonzero total memory, got %d", m.Total)
	}
	if m.ActualFree <= 0 {
		t.Errorf("Expected nonzero available memory, got %d", m.ActualFree)
	} else if m.ActualFree > m.Total {
		t.Errorf("Free %d > total %d", m.ActualFree, m.Total)
	}
}

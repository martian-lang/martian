// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"testing"
)

func TestStringIntern(t *testing.T) {
	inter := makeStringIntern()
	// Use a large string key to avoid small-allocation coalescing.
	const keyString = `10000000000000000000000000000000000000000000000000000000000`
	keyBytes := []byte(keyString)
	inter.GetString(keyString)
	if n := testing.AllocsPerRun(100, func() {
		if y := inter.GetString(keyString); y != keyString {
			t.Errorf("Expected "+keyString+", got %s", y)
		}
	}); n != 0 {
		t.Errorf("String key lookup AllocsPerRun = %f, want 0", n)
	}
	if n := testing.AllocsPerRun(100, func() {
		if y := inter.Get(keyBytes); y != keyString {
			t.Errorf("Expected "+keyString+" from bytes, got %s", y)
		}
	}); n != 0 {
		t.Errorf("Bytes key lookup AllocsPerRun = %f, want 0", n)
	}
}

// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

package util

import (
	"os"
	"strconv"
	"testing"
)

func TestCountDirNames(t *testing.T) {
	t.Parallel()
	if d, err := os.MkdirTemp("", "count_test"); err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(d)
		const count = 30
		for i := 0; i < count; i++ {
			if f, err := os.CreateTemp(d, strconv.Itoa(i)); err == nil {
				if _, err := f.WriteString(strconv.Itoa(i)); err != nil {
					t.Error(err)
				}
				if err := f.Close(); err != nil {
					t.Error(err)
				}
			} else {
				t.Fatal(err)
			}
		}
		if df, err := os.Open(d); err != nil {
			t.Fatal(err)
		} else {
			if c, err := CountDirNames(int(df.Fd())); err != nil {
				t.Error(err)
			} else if c != count {
				t.Errorf("Expected %d files, found %d", count, c)
			}
		}
	}
}

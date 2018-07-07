// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

package util

import (
	"io/ioutil"
	"os"
	"strconv"
	"testing"
)

func TestCountDirNames(t *testing.T) {
	t.Parallel()
	if d, err := ioutil.TempDir("", "count_test"); err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(d)
		const count = 30
		for i := 0; i < count; i++ {
			if f, err := ioutil.TempFile(d, strconv.Itoa(i)); err == nil {
				f.Write([]byte(strconv.Itoa(i)))
				f.Close()
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

// Copyright (c) 2022 10X Genomics, Inc. All rights reserved.

package core

import (
	"errors"
	"os"
	"path"
	"testing"
)

func Test_writeAtomic(t *testing.T) {
	dir := t.TempDir()
	file := path.Join(dir, "hello.txt")
	if err := writeAtomic(file, []byte("hello world")); err != nil {
		t.Error("writing file", err)
	}
	if data, err := os.ReadFile(file); err != nil {
		t.Error("reading back", err)
	} else if string(data) != "hello world" {
		t.Errorf("%q != 'hello world'", data)
	}
	if err := writeAtomic(path.Join(dir, "hello.txt", "subdir"), nil); err == nil {
		t.Error("expected error writing to subdirectory of file path")
	}
	if err := os.Remove(file); err != nil {
		t.Error("cleanup error", err)
	}
	if files, err := os.ReadDir(dir); err != nil {
		t.Error("reading dir", err)
	} else if len(files) != 0 {
		t.Error("leftover garbage files", files)
	}
	if err := writeAtomic(path.Join(dir, "not_exist", "hello.txt"), nil); err == nil {
		t.Error("expected error writing to non-existent directory.")
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Error("expected ENOENT, got", err)
	}
	if err := os.Mkdir(file, 0755); err != nil {
		t.Error(err)
	}
	if err := writeAtomic(file, []byte("hello world")); err == nil {
		t.Error("expected error opening file, which is actually a directory.")
	}
}

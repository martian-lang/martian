// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func ExampleWalk() {
	if root, err := ioutil.TempDir("", "examplewalk"); err != nil {
		fmt.Println("Failed to create tempdir:", err)
		return
	} else {
		defer os.RemoveAll(root)
		if err := os.MkdirAll(path.Join(root, "a", "b", "c", "d"), 0777); err != nil {
			fmt.Println(err)
		}
		if err := os.Symlink(path.Join(root, "a", "b", "c", "d"),
			path.Join(root, "a", "b", "c", "ds")); err != nil {
			fmt.Println(err)
		}
		if err := os.MkdirAll(path.Join(root, "a", "b", "e", "f"), 0777); err != nil {
			fmt.Println(err)
		}
		if err := os.MkdirAll(path.Join(root, "a", "b", "g"), 0777); err != nil {
			fmt.Println(err)
		}
		if err := ioutil.WriteFile(path.Join(root, "a.txt"), []byte("Test string"), 0666); err != nil {
			fmt.Println(err)
		}
		if err := ioutil.WriteFile(path.Join(root, "b.txt"), []byte("Test string2"), 0666); err != nil {
			fmt.Println(err)
		}
		if err := ioutil.WriteFile(path.Join(root, "a", "c.txt"), []byte("Test string33"), 0666); err != nil {
			fmt.Println(err)
		}
		if err := ioutil.WriteFile(path.Join(root, "a", "d.txt"), []byte("Test string444"), 0666); err != nil {
			fmt.Println(err)
		}
		if err := ioutil.WriteFile(path.Join(root, "a", "b", "g", "e.txt"), []byte("Test string5555"), 0666); err != nil {
			fmt.Println(err)
		}
		if err := ioutil.WriteFile(path.Join(root, "a", "b", "g", "f.txt"), []byte("Test string66666"), 0666); err != nil {
			fmt.Println(err)
		}
		if err := ioutil.WriteFile(path.Join(root, "a", "b", "e", "f", "g.txt"), []byte("Test string777777"), 0666); err != nil {
			fmt.Println(err)
		}

		if err := Walk(root, func(p string, info os.FileInfo, err error) error {
			if !strings.HasPrefix(p, root) {
				fmt.Println("Path was not rooted:", p)
			}
			if err != nil {
				fmt.Printf("Error reading %s:\n%v\n", strings.TrimPrefix(p, root), err)
			} else {
				if info.IsDir() {
					fmt.Printf("Directory '%s'\n", strings.TrimPrefix(p, root))
				} else if info.Mode()&os.ModeSymlink != 0 {
					fmt.Printf("Symlink '%s'\n", strings.TrimPrefix(p, root))
				} else {
					fmt.Printf("File '%s' had %d bytes\n", strings.TrimPrefix(p, root), info.Size())
				}
			}
			if path.Base(p) == "f" {
				fmt.Println("Skipping", path.Base(p))
				return filepath.SkipDir
			} else {
				return err
			}
		}); err != nil {
			fmt.Println(err)
		}

		if err := Walk(path.Join(root, "b"), func(p string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("Correctly expected error reading %s\n",
					strings.TrimPrefix(p, root))
			} else {
				fmt.Printf("File '%s' had %d bytes\n", p, info.Size())
			}
			return err
		}); err != nil {
			fmt.Println("Correctly expected error")
		}
		// Unordered Output:
		// Directory ''
		// Directory '/a'
		// File '/a.txt' had 11 bytes
		// File '/b.txt' had 12 bytes
		// Directory '/a/b'
		// File '/a/c.txt' had 13 bytes
		// File '/a/d.txt' had 14 bytes
		// Directory '/a/b/c'
		// Directory '/a/b/e'
		// Directory '/a/b/g'
		// Directory '/a/b/c/d'
		// Symlink '/a/b/c/ds'
		// Directory '/a/b/e/f'
		// File '/a/b/g/e.txt' had 15 bytes
		// File '/a/b/g/f.txt' had 16 bytes
		// Skipping f
		// Correctly expected error reading /b
		// Correctly expected error
	}
}

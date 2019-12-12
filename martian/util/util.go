//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian miscellaneous utilities.
//

// Packae util includes various utility methods.
//
// These utilities are frequently Martian-specific but do not depend on
// Martian runtime infrastructure.
package util // import "github.com/martian-lang/martian/martian/util"

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func getExeBasePath() string {
	if base := os.Getenv("MARTIAN_BASE"); base != "" {
		return base
	} else {
		if exe, err := os.Executable(); err != nil {
			panic(err)
		} else {
			if exe, err := filepath.EvalSymlinks(exe); err != nil {
				panic(err)
			} else {
				return path.Dir(exe)
			}
		}
	}
}

var exeBasePath string

func RelPath(p string) string {
	if exeBasePath == "" {
		exeBasePath = getExeBasePath()
	}
	return path.Join(exeBasePath, p)
}

func Mkdir(p string) error {
	if err := os.Mkdir(p, 0777); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func MkdirAll(p string) error {
	return os.MkdirAll(p, 0777)
}

func ParseMroPath(mroPath string) []string {
	return strings.Split(mroPath, ":")
}

func FormatMroPath(mroPaths []string) string {
	return strings.Join(mroPaths, ":")
}

type dirSizeCounter struct {
	numFiles uint
	numBytes uint64
}

func (c *dirSizeCounter) dirSizeWalker(_ string, info os.FileInfo, err error) error {
	if err == nil {
		c.numBytes += uint64(info.Size())
		c.numFiles++
	}
	return nil
}

func GetDirectorySize(paths []string) (uint, uint64) {
	var count dirSizeCounter
	for _, path := range paths {
		Walk(path, count.dirSizeWalker)
	}
	return count.numFiles, count.numBytes
}

// SearchPaths searches through searchPaths for the first path such that
// path.Join(p, fname) points to an existing file, and returns that joined
// path, or false if no such path is present.
func SearchPaths(fname string, searchPaths []string) (string, bool) {
	for _, searchPath := range searchPaths {
		fpath := path.Join(searchPath, fname)
		if _, err := os.Stat(fpath); !os.IsNotExist(err) {
			return fpath, true
		}
	}
	return "", false
}

// FindUniquePath searches through searchPaths for a path such that
// path.Join(p, fname) points to an existing file.  If more than one
// path in searchPaths satisfies that requirement, an error is returned
// indicating the ambiguity.  If no such path exists, an error satisfying
// os.IsNotExist will be returned.
func FindUniquePath(fname string, searchPaths []string) (string, error) {
	var resolved string
	for _, searchPath := range searchPaths {
		fpath := path.Join(searchPath, fname)
		if !path.IsAbs(fpath) {
			if p, err := filepath.Abs(fpath); err == nil {
				fpath = p
			}
		}
		if _, err := os.Stat(fpath); err == nil {
			if resolved == "" {
				resolved = fpath
			} else if resolved != fpath {
				var buf strings.Builder
				buf.Grow(len("ambiguous paths: '' could refer to '' or ''") +
					len(fname) + len(resolved) + len(fpath))
				buf.WriteString("ambiguous paths: '")
				buf.WriteString(fname)
				buf.WriteString("' could refer to '")
				buf.WriteString(resolved)
				buf.WriteString("' or '")
				buf.WriteString(fpath)
				return resolved, errors.New(buf.String())
			}
		}
	}
	if resolved == "" {
		_, err := os.Stat(fname)
		if err != nil {
			return "", err
		} else {
			return filepath.Abs(fname)
		}
	} else {
		return resolved, nil
	}
}

func ArrayToString(data []interface{}) []string {
	list := []string{}
	for _, i := range data {
		if value, ok := i.(string); ok {
			list = append(list, value)
		}
	}
	return list
}

func ValidateID(id string) error {
	if ok, _ := regexp.MatchString("^(\\d|\\w|-)+$", id); !ok {
		return &MartianError{fmt.Sprintf("Invalid name: %s (only numbers, letters, dash, and underscore allowed)", id)}
	}
	return nil
}

const TIMEFMT = "2006-01-02 15:04:05"

func Timestamp() string {
	return time.Now().Format(TIMEFMT)
}

func Pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// Gets the number of digits required to display a given integer in base 10.
// Optimizes for the common cases.
func WidthForInt(max int) int {
	if max < 0 {
		return 1 + WidthForInt(-max)
	} else if max < 10 {
		return 1
	} else if max < 100 {
		return 2
	} else if max < 1000 {
		return 3
	} else if max < 10000 {
		return 4
	} else if max < 100000 {
		return 5
	} else {
		return 1 + int(math.Log10(float64(max)))
	}
}

// Atoi returns parses the utf8-encoded bytes of a decimal string of native
// size.  Use this instead of strconv.Atoi to avoid copying bytes to string.
func Atoi(s []byte) (int64, error) {
	// capture and remove sign, if any
	neg := false
	if s[0] == '+' {
		s = s[1:]
	} else if s[0] == '-' {
		neg = true
		s = s[1:]
	}

	const cutoff = int64(^uint64(0)>>1) / 10

	var v int64
	for _, b := range s {
		if v >= cutoff {
			return cutoff, fmt.Errorf("Overflow parsing %s", string(s))
		} else if '0' <= b && b <= '9' {
			v = 10*v + int64(b-'0')
		} else {
			return v, fmt.Errorf("Invalid character in int %s", string(s))
		}
	}
	if neg {
		return -v, nil
	} else {
		return v, nil
	}
}

func FormatEnv(envs map[string]string) []string {
	l := []string{}
	for key, value := range envs {
		l = append(l, fmt.Sprintf("%s=%s", key, value))
	}
	return l
}

func MergeEnv(envs map[string]string) []string {
	e := map[string]string{}

	// Get base environment and convert to dictionary
	for _, env := range os.Environ() {
		envList := strings.SplitN(env, "=", 2)
		key, value := envList[0], envList[1]
		e[key] = value
	}

	// Set relevant environment variables
	for key, value := range envs {
		e[key] = value
	}

	return FormatEnv(e)
}

func EnvRequire(reqs [][]string, log bool) map[string]string {
	e := map[string]string{}
	for _, req := range reqs {
		val := os.Getenv(req[0])
		if len(val) == 0 {
			fmt.Println("Please set the following environment variables:")
			for _, req := range reqs {
				if len(os.Getenv(req[0])) == 0 {
					fmt.Printf("export %s=%s", req[0], req[1])
				}
			}
			os.Exit(1)
		}
		e[req[0]] = val
		if log {
			LogInfo("environ", "%s=%s", req[0], val)
		}
	}
	return e
}

func ParseTagsOpt(opt string) []string {
	tags := strings.Split(opt, ",")
	for _, tag := range tags {
		tagList := strings.Split(tag, ":")
		if len(tagList) != 2 {
			PrintInfo("options", "TagError: Tag '%s' does not <key>:<value> format", tag)
			os.Exit(1)
		}
		if len(tagList[0]) == 0 {
			PrintInfo("options", "TagError: Tag '%s' has empty key", tag)
			os.Exit(1)
		}
		if len(tagList[1]) == 0 {
			PrintInfo("options", "TagError: Tag '%s' has empty value", tag)
			os.Exit(1)
		}
	}
	return tags
}

func Readdirnames(readPath string) (names []string, err error) {
	dir, err := os.Open(readPath)
	if err != nil {
		return nil, err
	}
	names, err = dir.Readdirnames(0)
	dir.Close()
	return names, err
}

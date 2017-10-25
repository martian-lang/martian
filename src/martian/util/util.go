//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian miscellaneous utilities.
//

// Packae util includes various utility methods.
//
// These utilities are frequently Martian-specific but do not depend on
// Martian runtime infrastructure.
package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/martian-lang/docopt.go"
)

func RelPath(p string) string {
	base := os.Getenv("MARTIAN_BASE")
	if base != "" {
		return path.Join(base, p)
	} else {
		if exe, err := os.Executable(); err != nil {
			panic(err)
		} else {
			if exe, err := filepath.EvalSymlinks(exe); err != nil {
				panic(err)
			} else {
				return path.Join(path.Dir(exe), p)
			}
		}
	}
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

func MakeJSON(data interface{}) string {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}

func ParseMroPath(mroPath string) []string {
	return strings.Split(mroPath, ":")
}

func FormatMroPath(mroPaths []string) string {
	return strings.Join(mroPaths, ":")
}

func MakeTag(key string, value string) string {
	return fmt.Sprintf("%s:%s", key, value)
}

func ParseTag(tag string) (string, string) {
	tagList := strings.Split(tag, ":")
	if len(tagList) < 2 {
		return "", tag
	}
	return tagList[0], tagList[1]
}

func GetDirectorySize(paths []string) (uint, uint64) {
	var numFiles uint = 0
	var numBytes uint64 = 0
	for _, path := range paths {
		filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
			if err == nil {
				numBytes += uint64(info.Size())
				numFiles++
			}
			return nil
		})
	}
	return numFiles, numBytes
}

func SearchPaths(fname string, searchPaths []string) (string, bool) {
	for _, searchPath := range searchPaths {
		fpath := path.Join(searchPath, fname)
		if _, err := os.Stat(fpath); !os.IsNotExist(err) {
			return fpath, true
		}
	}
	return "", false
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

func Render(dir string, tname string, data interface{}) string {
	tmpl, err := template.New(tname).Delims("[[", "]]").ParseFiles(RelPath(path.Join("..", dir, tname)))
	if err != nil {
		return err.Error()
	}
	var doc bytes.Buffer
	err = tmpl.Execute(&doc, data)
	if err != nil {
		return err.Error()
	}
	return doc.String()
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

func GetFilenameWithSuffix(dir string, fname string) string {
	suffix := 0
	names, err := Readdirnames(dir)
	if err != nil {
		return fmt.Sprintf("%s-%d", fname, 0)
	}
	re := regexp.MustCompile(fmt.Sprintf("^%s-(\\d+)$", fname))
	for _, name := range names {
		if m := re.FindStringSubmatch(name); m != nil {
			infoSuffix, _ := strconv.Atoi(m[1])
			if suffix <= infoSuffix {
				suffix = infoSuffix + 1
			}
		}
	}
	return fmt.Sprintf("%s-%d", fname, suffix)
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
					fmt.Println("export", req[0]+"="+req[1])
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

func ParseMroFlags(opts map[string]interface{}, doc string, martianOptions []string, martianArguments []string) {
	// Parse doc string for accepted arguments
	r := regexp.MustCompile("--\\w+")
	s := r.FindAllString(doc, -1)
	if s == nil {
		s = []string{}
	}

	allowedOptions := map[string]bool{}
	for _, allowedOption := range s {
		allowedOptions[allowedOption] = true
	}
	// Remove unallowed options
	newMartianOptions := []string{}
	for allowedOption := range allowedOptions {
		for _, option := range martianOptions {
			if strings.HasPrefix(option, allowedOption) {
				newMartianOptions = append(newMartianOptions, option)
				break
			}
		}
	}
	newMartianOptions = append(newMartianOptions, martianArguments...)
	defopts, err := docopt.Parse(doc, newMartianOptions, false, "", true, false)
	if err != nil {
		LogInfo("environ", "EnvironError: MROFLAGS environment variable has incorrect format\n")
		fmt.Println(doc)
		os.Exit(1)
	}
	for id, defval := range defopts {
		// Only use options
		if !strings.HasPrefix(id, "--") {
			continue
		}
		if val, ok := opts[id].(bool); (ok && !val) || (!ok && opts[id] == nil) {
			opts[id] = defval
		}
	}
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

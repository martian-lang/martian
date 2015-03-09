//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian miscellaneous utilities.
//
package core

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/10XDev/osext"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/docopt/docopt.go"
)

func max(x int, y int) int {
	if x > y {
		return x
	}
	return y
}

func RelPath(p string) string {
	folder, _ := osext.ExecutableFolder()
	return path.Join(folder, p)
}

func mkdir(p string) {
	os.Mkdir(p, 0755)
}

func mkdirAll(p string) {
	os.MkdirAll(p, 0755)
}

func MakeJSON(data interface{}) string {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err.Error()
	}
	return string(bytes)
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

func searchPaths(fname string, searchPaths []string) (string, bool) {
	for _, searchPath := range searchPaths {
		fpath := path.Join(searchPath, fname)
		if _, err := os.Stat(fpath); !os.IsNotExist(err) {
			return fpath, true
		}
	}
	return "", false
}

func cartesianProduct(valueSets []interface{}) []interface{} {
	perms := []interface{}{[]interface{}{}}
	for _, valueSet := range valueSets {
		newPerms := []interface{}{}
		for _, perm := range perms {
			for _, value := range valueSet.([]interface{}) {
				perm := perm.([]interface{})
				newPerm := make([]interface{}, len(perm))
				copy(newPerm, perm)
				newPerm = append(newPerm, value)
				newPerms = append(newPerms, newPerm)
			}
		}
		perms = newPerms
	}
	return perms
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
			LogInfo("options", "TagError: Tag '%s' does not <key>:<value> format", tag)
			os.Exit(1)
		}
		if len(tagList[0]) == 0 {
			LogInfo("options", "TagError: Tag '%s' has empty key", tag)
			os.Exit(1)
		}
		if len(tagList[1]) == 0 {
			LogInfo("options", "TagError: Tag '%s' has empty value", tag)
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
	for allowedOption, _ := range allowedOptions {
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
		if val, ok := opts[id].(bool); (ok && val == false) || (!ok && opts[id] == nil) {
			opts[id] = defval
		}
	}
}

func ReadTar(tarPath string, filePath string) (string, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return "", &TarError{tarPath, filePath}
		}
		if err != nil {
			return "", err
		}
		if hdr.Name == filePath {
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, tr); err != nil {
				return "", err
			}
			return buf.String(), nil
		}
	}
}

func UnpackTar(tarPath string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		filePath := path.Join(path.Dir(tarPath), hdr.Name)
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, tr); err != nil {
			return err
		}
		mkdirAll(path.Dir(filePath))
		if err := ioutil.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
			return err
		}
	}
}

func CreateTar(tarPath string, filePaths []string) error {
	f, err := os.Create(tarPath)
	if err != nil {
		return err
	}

	tw := tar.NewWriter(f)
	for _, filePath := range filePaths {
		bytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(path.Dir(tarPath), filePath)
		hdr := &tar.Header{
			Name: relPath,
			Size: int64(len(bytes)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(bytes); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return nil
	}

	f.Close()
	return nil
}

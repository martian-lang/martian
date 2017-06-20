//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian miscellaneous utilities.
//
package core

import (
	"archive/zip"
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
	"strconv"
	"strings"
	"time"

	"github.com/10XDev/docopt.go"
)

func max(x int, y int) int {
	if x > y {
		return x
	}
	return y
}

func RelPath(p string) string {
	base := os.Getenv("MARTIAN_BASE")
	if base != "" {
		return path.Join(base, p)
	} else {
		folder, _ := osext.ExecutableFolder()
		return path.Join(folder, p)
	}
}

func mkdir(p string) error {
	if err := os.Mkdir(p, 0777); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func mkdirAll(p string) error {
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

func GetFilenameWithSuffix(dir string, fname string) string {
	suffix := 0
	infos, _ := ioutil.ReadDir(dir)
	re := regexp.MustCompile(fmt.Sprintf("^%s-(\\d+)$", fname))
	for _, info := range infos {
		if m := re.FindStringSubmatch(info.Name()); m != nil {
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
		if val, ok := opts[id].(bool); (ok && val == false) || (!ok && opts[id] == nil) {
			opts[id] = defval
		}
	}
}

func ReadZip(zipPath string, filePath string) (string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.Name == filePath {
			in, err := f.Open()
			if err != nil {
				return "", err
			}
			defer in.Close()

			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, in); err != nil {
				return "", err
			}
			return buf.String(), nil
		}
	}

	return "", &ZipError{zipPath, filePath}
}

func Unzip(zipPath string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, f := range zr.File {
		filePath := path.Join(path.Dir(zipPath), f.Name)
		mkdirAll(path.Dir(filePath))

		in, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.Create(filePath)
		if err != nil {
			return err
		}

		if _, err := io.Copy(out, in); err != nil {
			return err
		}

		in.Close()
		out.Close()
	}

	return nil
}

func CreateZip(zipPath string, filePaths []string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for _, filePath := range filePaths {
		info, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		if info.IsDir() {
			continue
		}

		relPath, _ := filepath.Rel(path.Dir(zipPath), filePath)
		out, err := zw.Create(relPath)
		if err != nil {
			return err
		}

		in, err := os.Open(filePath)
		if err != nil {
			return err
		}

		if _, err := io.Copy(out, in); err != nil {
			return err
		}

		in.Close()
	}
	if err := zw.Close(); err != nil {
		return err
	}

	return nil
}

func SearchPipestanceParams(pipestance *Ast, what string) interface{} {
	b1 := pipestance.Call.Bindings.Table[what]
	if b1 == nil {
		return nil
	} else {
		return b1.Exp.(*ValExp).Value
	}
}

/*
 * Compute a "partially" Qualified stage name. This is a fully qualified name
 * (ID.pipestance.pipe.pipe.pipe.....stage) with the initial ID and pipestance
 * trimmed off. This allows for comparisons between different pipestances with
 * the same (or similar) shapes.
 */
func partiallyQualifiedName(n string) string {
	count := 0
	for i := 0; i < len(n); i++ {
		if n[i] == '.' {
			count++
		}
		if count == 2 {
			return n[i+1:]
		}
	}
	return ""
}

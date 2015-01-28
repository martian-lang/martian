//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian miscellaneous utilities.
//
package core

import (
	"fmt"
	"github.com/10XDev/osext"
	"os"
	"os/signal"
	"path"
	"regexp"
	"strings"
	"syscall"
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

func SetupSignalHandlers() {
	// Handle CTRL-C and kill.
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, syscall.SIGTERM)
	go func() {
		<-sigchan
		os.Exit(1)
	}()
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
			fmt.Println("Please set the following environment variables:\n")
			for _, req := range reqs {
				if len(os.Getenv(req[0])) == 0 {
					fmt.Println("export", req[0], "=", req[1])
				}
			}
			os.Exit(1)
		}
		e[req[0]] = val
		if log {
			LogInfo("environ", "%s = %s", req[0], val)
		}
	}
	return e
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

//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Margo
//
package core

import (
	"github.com/10XDev/osext"
	"os"
	"path"
	"time"
)

func mkdir(p string) {
	err := os.Mkdir(p, 0755)
	if err != nil {
		panic(err.Error())
	}
}

func RelPath(p string) string {
	folder, _ := osext.ExecutableFolder()
	return path.Join(folder, p)
}

func idemMkdir(p string) {
	os.Mkdir(p, 0755)
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

const TIMEFMT = "2006-01-02 15:04:05"

func Timestamp() string {
	return time.Now().Format(TIMEFMT)
}

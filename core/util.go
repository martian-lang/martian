//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Margo
//
package core

import (
	"os"
	"time"
)

func mkdir(p string) {
	err := os.Mkdir(p, 0700)
	if err != nil {
		panic(err.Error())
	}
}

func idemMkdir(p string) {
	os.Mkdir(p, 0700)
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

func Timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

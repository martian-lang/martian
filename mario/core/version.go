//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario versioning facility.
//
package core

import (
	"fmt"
)

var __PKGVER__ string = ""
var __MODVER__ string = "<module version not embedded>"

func GetVersionPackage() string {
	return __PKGVER__
}

func GetVersionModule() string {
	return __MODVER__
}

func GetVersion() string {
	if __PKGVER__ == "" {
		return GetVersionModule()
	}
	return fmt.Sprintf("%s (%s)", GetVersionPackage(), GetVersionModule())
}

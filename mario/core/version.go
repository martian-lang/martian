//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario versioning facility.
//
package core

var __VERSION__ string = "<version not embedded>"

func GetVersion() string {
	return __VERSION__
}

// Copyright (c) 2021 10X Genomics, Inc. All rights reserved.

//go:build !linux
// +build !linux

// Stubs for non-linux builds.

package main

// setSubreaper generic stub, which does nothing.
func setSubreaper() {}

// Copyright (c) 2022 10X Genomics, Inc. All rights reserved.

// Stubs for windows builds.

package main

// syncFile generic stub, which does nothing.
func syncFile(string) {}

// reportChildren generic stub which always returns false.
func reportChildren() bool {
	return false
}

// waitChildren eneric stub which always returns false.
func waitChildren() bool {
	return false
}

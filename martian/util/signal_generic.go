//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Martian signal handler.
//

//go:build !linux
// +build !linux

package util

import (
	"os"
)

// After a call to SetupSignalHandlers, these signals will be handled
// by waiting for all pending critical sections to complete, running
// all registered handlers, and then exiting with return code 1
var HANDLED_SIGNALS = [...]os.Signal{
	os.Interrupt,
}

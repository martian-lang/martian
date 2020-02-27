//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Martian signal handler.
//

package util

import (
	"os"
	"syscall"
)

// After a call to SetupSignalHandlers, these signals will be handled
// by waiting for all pending critical sections to complete, running
// all registered handlers, and then exiting with return code 1
var HANDLED_SIGNALS = [...]os.Signal{
	os.Interrupt,
	syscall.SIGHUP,
	syscall.SIGTERM,
	syscall.SIGUSR1,
	syscall.SIGUSR2,
}

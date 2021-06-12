// Copyright (c) 2021 10X Genomics, Inc. All rights reserved.

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// setSubreamer attempts to set PR_SET_CHILD_SUBREAPER to collect rusage
// information from subprocesses launched by job but orphaned before the job
// completes.
func setSubreaper() {
	if err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0); err != nil {
		fmt.Fprintln(os.Stderr, "Warning: unable to set PR_SET_CHILD_SUBREAPER:",
			err,
			"\nChild memory statistics may be incomplete.")
	}
}

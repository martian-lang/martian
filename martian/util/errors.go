//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian error types.
//

package util

import (
	"fmt"
	"os"
	"time"
)

//
// Martian Errors
//

// MartianError
type MartianError struct {
	Msg string
}

func (self *MartianError) Error() string {
	return self.Msg
}

// ZipError
type ZipError struct {
	ZipPath  string
	FilePath string
}

func (self *ZipError) Error() string {
	return fmt.Sprintf("ZipError: %s does not exist in %s", self.FilePath, self.ZipPath)
}

// End the process if err is not nil.  Because this method waits up to one
// minute for critical sections to end, it should not be called from inside
// a critical section.
func DieIf(err error) {
	if err != nil {
		fmt.Println()
		fmt.Println(err.Error())
		fmt.Println()
		Suicide()
		// We don't want to return, but if someone ran this from inside a
		// critical section that's also bad.
		time.Sleep(time.Minute)
		os.Exit(1)
	}
}

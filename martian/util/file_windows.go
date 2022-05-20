// Copyright (c) 2022 10X Genomics, Inc. All rights reserved.

package util

import (
	"io/fs"
)

// GetFileOwner on windows always returns false.
func GetFileOwner(fs.FileInfo) (uint32, uint32, bool) {
	return 0, 0, false
}

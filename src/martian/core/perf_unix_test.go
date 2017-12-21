//
// Copyright (c) 201 10X Genomics, Inc. All rights reserved.
//
// Converts system rusage into our structures.
//

package core

import (
	"testing"
)

func TestGetUserProcessCount(t *testing.T) {
	if c, err := GetUserProcessCount(); err != nil {
		t.Error(err)
	} else if c < 2 {
		t.Errorf("Expected at least 2 processes for this uid, got %d.", c)
	} else {
		t.Log("Current user process count", c)
	}
}

// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

package core

import "testing"

func TestLoadAverage(t *testing.T) {
	var la LoadAverage
	if err := la.Get(); err != nil {
		t.Fatal(err)
	}
	if la.One <= 0 {
		t.Errorf("one-minute loadavg should be nonzero, was %f", la.One)
	}
	if la.Five <= 0 {
		t.Errorf("one-minute loadavg should be nonzero, was %f", la.One)
	}
	if la.Fifteen <= 0 {
		t.Errorf("one-minute loadavg should be nonzero, was %f", la.One)
	}
}

// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

// +build !linux

// Stubs for non-linux OS.

package core

import "errors"

type LoadAverage struct {
	One, Five, Fifteen float64
}

func (*LoadAverage) Get() error {
	return errors.New("not supported")
}

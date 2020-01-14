// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

//  Utility method to parse /proc/loadavg

package core

import (
	"bytes"
	"io/ioutil"
	"strconv"
	"strings"
)

type LoadAverage struct {
	One, Five, Fifteen float64
}

func (la *LoadAverage) Get() error {
	b, err := ioutil.ReadFile("/proc/loadavg")
	if err != nil {
		// Ignore errors.
		return nil
	}
	fields := strings.Fields(string(bytes.TrimSpace(b)))
	la.One, err = strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return err
	}
	la.Five, err = strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return err
	}
	la.Fifteen, err = strconv.ParseFloat(fields[2], 64)
	return err
}

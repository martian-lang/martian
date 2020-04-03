// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatGB(t *testing.T) {
	var buf strings.Builder
	// Check that parsing the rounded-up value will result in the same
	// rounded-up value.
	for _, v := range [...]float32{
		1.0 / 1024,
		2.0 / 1024,
		0.001,
		0.05,
		0.62499,
		0.625,
		0.62599,
		2.5,
		1,
		4,
		10.2,
		2000.1,
	} {
		v = roundUpTo(v, 1024)
		formatGB(&buf, v)
		if r := roundUpTo(parseFloat32([]byte(buf.String())), 1024); r != v {
			t.Errorf("%g != %g", r, v)
		}
		buf.Reset()
	}
	// Check that decimal fractions aren't change inappropriately.
	for _, v := range [...]float32{
		0.001,
		0.05,
		0.624,
		0.625,
		0.626,
		2.5,
		1,
		4,
		10.2,
		2000.1,
	} {
		formatGB(&buf, roundUpTo(v, 1024))
		if r := buf.String(); r != fmt.Sprintf("%g", v) {
			t.Errorf("%s != %g", r, v)
		}
		buf.Reset()
	}
}

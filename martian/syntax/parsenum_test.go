// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"math"
	"testing"
)

type parseIntCase struct {
	s string
	i int64
}

var intTests = [...]parseIntCase{
	{"0", 0},
	{"-0", 0},
	{"1", 1},
	{"-1", -1},
	{"12345", 12345},
	{"-12345", -12345},
	{"012345", 12345},
	{"-012345", -12345},
	{"98765432100", 98765432100},
	{"-98765432100", -98765432100},
	{"9223372036854775807", 1<<63 - 1},
	{"-9223372036854775807", -(1<<63 - 1)},
	{"-9223372036854775808", -1 << 63},
}

func TestParseInt(t *testing.T) {
	for _, it := range intTests {
		if v := parseInt([]byte(it.s)); v != it.i {
			t.Errorf("%d != %d", it.i, v)
		}
		if it.s[0] != '-' {
			if v := parseInt([]byte("+" + it.s)); v != it.i {
				t.Errorf("+%d != %d", it.i, v)
			}
		}
	}
}

func TestParseFloat(t *testing.T) {
	check := func(s string, f float64) {
		t.Helper()
		if f2 := parseFloat([]byte(s)); f2 != f {
			t.Errorf("Expected %s -> %g, got %g",
				s, f, f2)
		}
	}
	check(`0.0`, 0)
	check(`0.1`, 0.1)
	check(`0e0`, 0)
	check(`0e10`, 0)
	check(`2e01`, 20)
	check(`1e0`, 1)
	check(`10e0`, 10)
	check(`01e0`, 1)
	check(`0E0`, 0)
	check(`0.1e0`, .1)
	check(`0.0E0`, 0)
	check(`1e+0`, 1)
	check(`0E+0`, 0)
	check(`0.1e+1`, 1)
	check(`0.0E+0`, 0)
	check(`1e-1`, .1)
	check(`1E-0`, 1)
	check(`0.1e-0`, .1)
	check(`1.0E-0`, 1)
	check(`-0.0`, 0)
	check(`-1e0`, -1)
	check(`-1e10`, -10000000000)
	check(`-0e01`, 0)
	check(`-1e0`, -1)
	check(`-10e0`, -10)
	check(`-01e0`, -1)
	check(`-0E0`, 0)
	check(`-0.1e3`, -100)
}

func BenchmarkParseFloat(b *testing.B) {
	v := [][]byte{
		[]byte(`0.0001`),
		[]byte(`0.001e-1`),
		[]byte(`1e-4`),
		[]byte(`10.0e-5`),
		[]byte(`1.0e-4`),
	}
	z := [][]byte{
		[]byte(`0.0`),
		[]byte(`-0.0`),
		[]byte(`0.0e3`),
		[]byte(`0.0e+3`),
		[]byte(`-0.0e3`),
		[]byte(`-0.0e+3`),
		[]byte(`000000.0`),
		[]byte(`-0.000000`),
		[]byte(`0.0e30`),
		[]byte(`0.0e+003`),
		[]byte(`-0.0000e3`),
		[]byte(`-000000.0e+0300`),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range v {
			if f := parseFloat(s); f != 0.0001 {
				b.Errorf("Expected 0.0001, got %g", f)
			}
		}
		for _, s := range z {
			if f := parseFloat(s); f != 0.0 {
				b.Errorf("Expected 0.0, got %g", f)
			}
		}
	}
}

func TestRoundUpTo(t *testing.T) {
	for _, v := range [...]float32{
		0.001,
		0.05,
		2.5,
		1,
		4,
		10.2,
	} {
		expected := int(math.Ceil(float64(v) * 1024))
		actualF := roundUpTo(v, 1024)
		actual := int(actualF * 1024)
		if actual != expected {
			t.Errorf("Expected %g == %d MB, got %d MB from %g",
				v, expected, actual, actualF)
		}
	}
}

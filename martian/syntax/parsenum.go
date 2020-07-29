// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// Parse methods for byte strings, with fast paths

package syntax

import (
	"math"
	"strconv"
)

// parseInt parses bytes as a 64-bit signed decimal integer.
//
// Reimplementing this method avoids the overhead of copying the byte array to
// a string for use with strconv.ParseInt, and also hard-codes the fast path.
//
// Panics on invalid input, since the tokenizer is supposed to guarantee
// valid input.
func parseInt(s []byte) int64 {
	if len(s) == 0 {
		panic("Empty string can't be parsed as int.")
	}
	neg := false
	if s[0] == '+' {
		s = s[1:]
	} else if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	if len(s) == 0 {
		panic("Sign alone can't be parsed as int.")
	}
	const cutoff = 1 << 63
	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			panic("invalid character in int string " + string(s))
		}
		n1 := 10*n + uint64(c-'0')
		if n1 < n || n1 > cutoff || (!neg && n1 == cutoff) {
			panic("integer overflow parsing " + string(s))
		}
		n = n1
	}
	if neg {
		return -int64(n)
	} else {
		return int64(n)
	}
}

// parseFloat parses bytes as a 64-bit float.
//
// Unlike with parseInt, there are a lot more edge cases for floats so this
// just eats the overhead of copying to string and calling the standard library,
// except in a few common cases.
func parseFloat(s []byte) float64 {
	f, err := strconv.ParseFloat(string(s), 64)
	if err != nil {
		panic(err)
	}
	return f
}

// parseFloat parses bytes as a 32-bit float.
//
// Unlike with parseInt, there are a lot more edge cases for floats so this
// just eats the overhead of copying to string and calling the standard library,
// except in a few common cases.
func parseFloat32(s []byte) float32 {
	f, err := strconv.ParseFloat(string(s), 32)
	if err != nil {
		panic(err)
	}
	return float32(f)
}

// roundUpTo rounds a value away from zero to the nearest 1/granularity.
//
// For example, roundUpTo(0.0001, 100) -> 0.01.
func roundUpTo(value float32, granularity float64) float32 {
	if value > 0 {
		return float32(math.Ceil(float64(value)*granularity) / granularity)
	} else if value < 0 {
		return float32(math.Floor(float64(value)*granularity) / granularity)
	}
	return 0
}

func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	default:
		panic(string(append([]byte("Invalid character "), c)))
	}
}

func parseHexByte(c0, c1 byte) byte {
	return (unhex(c0) << 4) + unhex(c1)
}

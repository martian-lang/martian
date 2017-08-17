//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian formatter tests.
//

package syntax

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFormatValueExpression(t *testing.T) {
	ve := ValExp{
		Node:  AstNode{0, "", ""},
		Kind:  "float",
		Value: 0,
	}

	//
	// Format float ValExps.
	//
	ve.Kind = "float"

	ve.Value = 10.0
	assert.Equal(t, ve.format(""), "10", "Preserve single zero after decimal.")

	ve.Value = 10.05
	assert.Equal(t, ve.format(""), "10.05", "Do not strip numbers ending in non-zero digit.")

	ve.Value = 10.050
	assert.Equal(t, ve.format(""), "10.05", "Strip single trailing zero.")

	ve.Value = 10.050000000
	assert.Equal(t, ve.format(""), "10.05", "Strip multiple trailing zeroes.")

	ve.Value = 0.0000000005
	assert.Equal(t, ve.format(""), "5e-10", "Handle exponential notation.")

	ve.Value = 0.0005
	assert.Equal(t, ve.format(""), "0.0005", "Handle low decimal floats.")

	//
	// Format int ValExps.
	//
	ve.Kind = "int"

	ve.Value = 0
	assert.Equal(t, ve.format(""), "0", "Format zero integer.")

	ve.Value = 10
	assert.Equal(t, ve.format(""), "10", "Format non-zero integer.")

	ve.Value = 1000000
	assert.Equal(t, ve.format(""), "1000000", "Preserve integer trailing zeroes.")

	//
	// Format string ValExps.
	//
	ve.Kind = "string"

	ve.Value = "blah"
	assert.Equal(t, ve.format(""), "\"blah\"", "Double quote a string.")

	ve.Value = "\"blah\""
	assert.Equal(t, ve.format(""), "\"\"blah\"\"", "Double quote a double-quoted string.")

	//
	// Format nil ValExps.
	//
	ve.Value = nil
	assert.Equal(t, ve.format(""), "null", "Nil value is 'null'.")
}

//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian formatter tests.
//

package core

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFormatValueExpression(t *testing.T) {
	ve := ValExp{
		node:  AstNode{0, ""},
		kind:  "float",
		value: 0,
	}

	//
	// Format float ValExps.
	//
	ve.kind = "float"

	ve.value = 10.0 // MARIO-205: 10.0 mrf'ed into 10.
	assert.Equal(t, ve.format(), "10.0", "Preserve single zero after decimal.")

	ve.value = 10.05
	assert.Equal(t, ve.format(), "10.05", "Do not strip numbers ending in non-zero digit.")

	ve.value = 10.050
	assert.Equal(t, ve.format(), "10.05", "Strip single trailing zero.")

	ve.value = 10.050000000
	assert.Equal(t, ve.format(), "10.05", "Strip multiple trailing zeroes.")

	//
	// Format int ValExps.
	//
	ve.kind = "int"

	ve.value = 0
	assert.Equal(t, ve.format(), "0", "Format zero integer.")

	ve.value = 10
	assert.Equal(t, ve.format(), "10", "Format non-zero integer.")

	ve.value = 1000000
	assert.Equal(t, ve.format(), "1000000", "Preserve integer trailing zeroes.")

	//
	// Format string ValExps.
	//
	ve.kind = "string"

	ve.value = "blah"
	assert.Equal(t, ve.format(), "\"blah\"", "Double quote a string.")

	ve.value = "\"blah\""
	assert.Equal(t, ve.format(), "\"\"blah\"\"", "Double quote a double-quoted string.")

	//
	// Format nil ValExps.
	//
	ve.value = nil
	assert.Equal(t, ve.format(), "null", "Nil value is 'null'.")
}

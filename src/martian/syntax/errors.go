//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian error types.
//
package syntax

import (
	"fmt"
	"strings"
)

// PreprocessError
type PreprocessError struct {
	files []string
}

func (self *PreprocessError) Error() string {
	return fmt.Sprintf("@include file not found: %s", strings.Join(self.files, ", "))
}

// AstError
type AstError struct {
	global *Ast
	Node   *AstNode
	Msg    string
}

func (self *AstError) Error() string {
	return fmt.Sprintf("MRO %s at %s:%d.", self.Msg, self.Node.Fname, self.Node.Loc)
}

// ParseError
type ParseError struct {
	token string
	fname string
	loc   int
}

func (self *ParseError) Error() string {
	return fmt.Sprintf("MRO ParseError: unexpected token '%s' at %s:%d.", self.token, self.fname, self.loc)
}

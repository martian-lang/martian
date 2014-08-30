//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Margo
//
package core

import (
	"fmt"
	"os"
	"strings"
)

//
// Mario Errors
//

// MarioError
type MarioError struct {
	Msg string
}

func (self *MarioError) Error() string {
	return self.Msg
}

// PipestanceExistsError
type PipestanceExistsError struct {
	psid string
}

func (self *PipestanceExistsError) Error() string {
	return fmt.Sprintf("PipestanceExistsError: '%s'.", self.psid)
}

// PreprocessError
type PreprocessError struct {
	files []string
}

func (self *PreprocessError) Error() string {
	return fmt.Sprintf("@include file not found: %s", strings.Join(self.files, ", "))
}

// AstError
type AstError struct {
	global  *Ast
	locable Locatable
	msg     string
}

func (self *AstError) Error() string {
	return fmt.Sprintf("MRO %s at %s:%d.", self.msg,
		self.global.locmap[self.locable.Loc()].fname,
		self.global.locmap[self.locable.Loc()].loc)
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

func DieIf(err error) {
	if err != nil {
		fmt.Println()
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

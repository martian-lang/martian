//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Margo
//
package core

import (
	"fmt"
	"os"
)

//
// Mario Errors
//
type MarioError struct {
	Msg string
}

func (self *MarioError) Error() string {
	return self.Msg
}

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
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

package main

import (
	"fmt"
)

//
// Mario Errors
//
type MarioError struct {
	global  *Ast
	locable Locatable
    msg     string
}

func (self *MarioError) Error() string {
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
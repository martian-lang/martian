package main

import (
	"fmt"
	"reflect"
)

//
// Mario Errors
//
type MarioError struct {
	fname string
	loc   int
}

func (self *MarioError) Format(obj interface{}, msg string) string {
	return fmt.Sprintf("MRO %s: %s at %s:%d.", reflect.TypeOf(obj).Name(), msg, self.fname, self.loc)
}

type ParseError struct {
	err   MarioError
	token string
}

func (self *ParseError) Error() string {
	return self.err.Format(*self, fmt.Sprintf("Unexpected token '%s'", self.token))
}
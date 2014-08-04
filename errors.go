package main

import (
	"fmt"
	"reflect"
)

//
// Mario Errors
//
type MarioError struct {
	locmap []FileLoc
	loc    int
}

func (self *MarioError) f(obj interface{}, msg string) string {
	return fmt.Sprintf("MRO %s: %s at %s:%d.", reflect.TypeOf(obj).Name(), msg, 
		self.locmap[self.loc].fname, self.locmap[self.loc].loc)
}

type ParseError struct {
	e     MarioError
	token string
}

func (self *ParseError) Error() string {
	return self.e.f(*self, fmt.Sprintf("Unexpected token '%s'", self.token))
}

type DuplicateNameError struct {
	e    MarioError
    kind string 
	id   string
}

func (self *DuplicateNameError) Error() string {
	return self.e.f(*self, fmt.Sprintf("%s '%s' was previously declared; duplicate encountered", self.kind, self.id))
}
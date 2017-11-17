//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian error types.
//

package syntax

import (
	"bytes"
	"fmt"
	"strings"
)

// PreprocessError
type PreprocessError struct {
	files    []string
	messages []string
}

func (self *PreprocessError) Error() string {
	msg := strings.Join(self.messages, "\n")
	if len(self.files) > 0 {
		if len(self.messages) > 0 {
			msg += "\n"
		}
		msg += fmt.Sprintf("@include file not found: %s", strings.Join(self.files, ", "))
	}
	return msg
}

// AstError
type AstError struct {
	global *Ast
	Node   *AstNode
	Msg    string
}

func (self *AstError) Error() string {
	line := fmt.Sprintf("MRO %s at %s:%d.", self.Msg, self.Node.Fname, self.Node.Loc)
	for _, inc := range self.Node.IncludeStack {
		line += fmt.Sprintf("\n\tincluded from %s", inc)
	}
	return line
}

// ParseError
type ParseError struct {
	token        string
	fname        string
	loc          int
	includeStack []string
}

func (self *ParseError) Error() string {
	line := fmt.Sprintf("MRO ParseError: unexpected token '%s' at %s:%d.", self.token, self.fname, self.loc)
	for _, inc := range self.includeStack {
		line += fmt.Sprintf("\n\tincluded from %s", inc)
	}
	return line
}

type ErrorList []error

func (self ErrorList) Error() string {
	var buf bytes.Buffer
	for i, err := range self {
		if i != 0 {
			buf.WriteRune('\n')
		}
		buf.WriteString(err.Error())
	}
	return buf.String()
}

// Collapse the error list down, and remove any nil errors.
// Returns nil if the list is empty.
func (self ErrorList) If() error {
	if len(self) > 0 {
		errs := make(ErrorList, 0, len(self))
		for _, err := range self {
			if err != nil {
				if list, ok := err.(ErrorList); ok {
					err = list.If()
				}
			}
			if err != nil {
				if list, ok := err.(ErrorList); ok {
					errs = append(errs, list...)
				} else {
					errs = append(errs, err)
				}
			}
		}
		if len(errs) > 1 {
			return errs
		} else if len(errs) == 1 {
			return errs[0]
		}
	}
	return nil
}

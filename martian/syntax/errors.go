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

type errorWriter interface {
	writeTo(w stringWriter)
}

// AstError
type AstError struct {
	global *Ast
	Node   *AstNode
	Msg    string
}

func (self *AstError) writeTo(w stringWriter) {
	w.WriteString("MRO ")
	w.WriteString(self.Msg)
	w.WriteString("\n    at ")
	self.Node.Loc.writeTo(w, "        ")
}

func (self *AstError) Error() string {
	var buff strings.Builder
	buff.Grow(len("MRO \n    at sourcename.mro:100 included from sourcename.mro:10") + len(self.Msg))
	self.writeTo(&buff)
	return buff.String()
}

type FileNotFoundError struct {
	loc  SourceLoc
	name string
}

func (err *FileNotFoundError) writeTo(w stringWriter) {
	w.WriteString("File '")
	w.WriteString(err.name)
	w.WriteString("' not found (included from ")
	err.loc.writeTo(w,
		"                 ")
	w.WriteRune(')')
}

func (err *FileNotFoundError) Error() string {
	var buff strings.Builder
	buff.Grow(len("File 'sourcename.mro' not found (included from sourcename.mro:10)"))
	err.writeTo(&buff)
	return buff.String()
}

type DuplicateCallError struct {
	First  *CallStm
	Second *CallStm
}

func (err *DuplicateCallError) writeTo(w stringWriter) {
	w.WriteString("Cannot have more than one top-level call.\n    First call: ")
	w.WriteString(err.First.Id)
	w.WriteString(" at ")
	err.First.Node.Loc.writeTo(w, "        ")
	w.WriteString("\n    Next call: ")
	w.WriteString(err.Second.Id)
	w.WriteString(" at ")
	err.Second.Node.Loc.writeTo(w, "        ")
}

func (err *DuplicateCallError) Error() string {
	var buff strings.Builder
	buff.Grow(len("Cannot have more than one top-level call.\n    First call: CALL_NAME at sourcefile.mro:100\n    Next call: SECOND_CALL at sourcefile.mro:200"))
	err.writeTo(&buff)
	return buff.String()
}

type wrapError struct {
	innerError error
	loc        SourceLoc
}

func (err *wrapError) writeTo(w stringWriter) {
	w.WriteString("MRO ")
	if ew, ok := err.innerError.(errorWriter); ok {
		ew.writeTo(w)
	} else {
		w.WriteString(err.innerError.Error())
	}
	w.WriteString("\n    at ")
	err.loc.writeTo(w,
		"        ")
}

func (err *wrapError) Error() string {
	var buff strings.Builder
	buff.Grow(len("MRO os.FileError: cannot access...\n    at sourcename.mro:100 included from sourcename.mro:10"))
	err.writeTo(&buff)
	return buff.String()
}

func (loc *SourceLoc) writeTo(w stringWriter, indent string) {
	if loc.File == nil ||
		loc.File.FullPath == "" && len(loc.File.IncludedFrom) == 0 {
		fmt.Fprintf(w, "line %d", loc.Line)
	} else if len(loc.File.IncludedFrom) == 0 {
		w.WriteString(loc.File.FullPath)
		fmt.Fprintf(w, ":%d", loc.Line)
	} else if len(loc.File.IncludedFrom) == 1 {
		fmt.Fprintf(w, "%s:%d\n%s    included from ",
			loc.File.FullPath, loc.Line,
			indent)
		loc.File.IncludedFrom[0].writeTo(w, indent)
	} else {
		newIndent := indent + "    "
		fmt.Fprintf(w, "%s:%d included from:",
			loc.File.FullPath, loc.Line)
		for i, inc := range loc.File.IncludedFrom {
			fmt.Fprintf(w, "\n%s[%d] ", newIndent, i)
			inc.writeTo(w, newIndent)
		}
	}
}

func (loc *SourceLoc) String() string {
	var buff strings.Builder
	buff.Grow(len("sourcename.mro:100 included from sourcename.mro:10"))
	loc.writeTo(&buff, "")
	return buff.String()
}

// ParseError
type ParseError struct {
	token string
	loc   SourceLoc
}

func (err *ParseError) writeTo(w stringWriter) {
	fmt.Fprintf(w, "MRO ParseError: unexpected token '%s' at ", err.token)
	err.loc.writeTo(w, "")
}

func (self *ParseError) Error() string {
	var buff strings.Builder
	self.writeTo(&buff)
	return buff.String()
}

type ErrorList []error

func (self ErrorList) Error() string {
	var buf strings.Builder
	for i, err := range self {
		if i != 0 {
			buf.WriteRune('\n')
		}
		if ew, ok := err.(errorWriter); ok {
			ew.writeTo(&buf)
		} else {
			buf.WriteString(err.Error())
		}
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

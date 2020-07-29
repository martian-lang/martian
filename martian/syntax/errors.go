//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Martian error types.
//

package syntax

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func mustWrite(w io.Writer, b []byte) {
	if _, err := w.Write(b); err != nil {
		panic(err)
	}
}

func mustWriteByte(w stringWriter, b byte) {
	if err := w.WriteByte(b); err != nil {
		panic(err)
	}
}

func mustWriteRune(w stringWriter, r rune) {
	if _, err := w.WriteRune(r); err != nil {
		panic(err)
	}
}

func mustWriteString(w stringWriter, s string) {
	if _, err := w.WriteString(s); err != nil {
		panic(err)
	}
}

type errorWriter interface {
	writeTo(w stringWriter)
}

// AstError contains information about an error in parsing or compiling an Ast.
type AstError struct {
	global *Ast
	Node   *AstNode
	Msg    string
}

func (err *AstError) writeTo(w stringWriter) {
	mustWriteString(w, "MRO ")
	mustWriteString(w, err.Msg)
	mustWriteString(w, "\n    at ")
	err.Node.Loc.writeTo(w, "        ")
}

func (err *AstError) Error() string {
	var buff strings.Builder
	buff.Grow(len("MRO \n    at sourcename.mro:100 included from sourcename.mro:10") + len(err.Msg))
	err.writeTo(&buff)
	return buff.String()
}

type FileNotFoundError struct {
	loc   SourceLoc
	name  string
	inner error
	paths string
}

func (err *FileNotFoundError) writeTo(w stringWriter) {
	mustWriteString(w, "File '")
	mustWriteString(w, err.name)
	if err.inner == nil || os.IsNotExist(err.inner) {
		if err.paths != "" {
			mustWriteString(w, "' not found in ")
			mustWriteString(w, err.paths)
			mustWriteString(w, " (included from ")
		} else {
			mustWriteString(w, "' not found (included from ")
		}
	} else {
		mustWriteString(w, "' could not be resolved: ")
		if ew, ok := err.inner.(errorWriter); ok {
			ew.writeTo(w)
		} else {
			mustWriteString(w, err.inner.Error())
		}
		mustWriteString(w, "\n                 (included from ")
	}
	err.loc.writeTo(w,
		"                 ")
	mustWriteRune(w, ')')
}

func (err *FileNotFoundError) Error() string {
	var buff strings.Builder
	buff.Grow(len("File '' not found in  (included from sourcename.mro:1000)") +
		len(err.name) + len(err.paths))
	err.writeTo(&buff)
	return buff.String()
}

func (err *FileNotFoundError) Unwrap() error {
	if err == nil {
		return err
	}
	return err.inner
}

type DuplicateCallError struct {
	First  *CallStm
	Second *CallStm
}

func (err *DuplicateCallError) writeTo(w stringWriter) {
	mustWriteString(w, "Cannot have more than one top-level call.\n    First call: ")
	mustWriteString(w, err.First.Id)
	mustWriteString(w, " at ")
	err.First.Node.Loc.writeTo(w, "        ")
	mustWriteString(w, "\n    Next call: ")
	mustWriteString(w, err.Second.Id)
	mustWriteString(w, " at ")
	err.Second.Node.Loc.writeTo(w, "        ")
}

func (err *DuplicateCallError) Error() string {
	var buff strings.Builder
	buff.Grow(len(
		`Cannot have more than one top-level call.
    First call: CALL_NAME at sourcefile.mro:100
    Next call: SECOND_CALL at sourcefile.mro:200`))
	err.writeTo(&buff)
	return buff.String()
}

type wrapError struct {
	innerError error
	loc        SourceLoc
}

func (err *wrapError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.innerError
}

func (err *wrapError) writeTo(w stringWriter) {
	if ew, ok := err.innerError.(*wrapError); ok {
		ew.writeTo(w)
	} else if ew, ok := err.innerError.(errorWriter); ok {
		mustWriteString(w, "MRO ")
		ew.writeTo(w)
	} else {
		mustWriteString(w, err.innerError.Error())
	}
	mustWriteString(w, "\n    at ")
	err.loc.writeTo(w,
		"        ")
}

func (err *wrapError) Error() string {
	var buff strings.Builder
	buff.Grow(len(`MRO os.FileError: cannot access...
    at sourcename.mro:100 included from sourcename.mro:10`))
	err.writeTo(&buff)
	return buff.String()
}

func (loc *SourceLoc) writeTo(w stringWriter, indent string) {
	if loc.File == nil ||
		loc.File.FullPath == "" && len(loc.File.IncludedFrom) == 0 {
		fmt.Fprintf(w, "line %d", loc.Line)
	} else if len(loc.File.IncludedFrom) == 0 {
		mustWriteString(w, loc.File.FullPath)
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

// ParseError contains information about errors during parsing.
type ParseError struct {
	token string
	loc   SourceLoc
}

func (err *ParseError) writeTo(w stringWriter) {
	fmt.Fprintf(w, "MRO ParseError: unexpected token '%s' at ", err.token)
	err.loc.writeTo(w, "")
}

func (err *ParseError) Error() string {
	var buff strings.Builder
	err.writeTo(&buff)
	return buff.String()
}

// An ErrorList can be used to collect and return multiple errors.
//
// Example usage:
//
//     func foo(things []int) error {
//         var errs ErrorList
//         for _, thing := range things {
//             if err := bar(thing); err != nil {
//                 errs = append(errs, err)
//             }
//         }
//         return errs.If()  // Important!
//    }
//
// It is very important to note that the .If() in the return statement
// is crucial.  See https://golang.org/doc/faq#nil_error.
type ErrorList []error

func (errList ErrorList) Error() string {
	var buf strings.Builder
	for i, err := range errList {
		if i != 0 || len(errList) > 1 {
			if _, err := buf.WriteString("\n\t"); err != nil {
				panic(err)
			}
		}
		if ew, ok := err.(errorWriter); ok {
			ew.writeTo(&buf)
		} else {
			mustWriteString(&buf, err.Error())
		}
	}
	return buf.String()
}

func (errList ErrorList) writeTo(buf stringWriter) {
	for i, err := range errList {
		if i != 0 || len(errList) > 1 {
			mustWriteString(buf, "\n\t")
		}
		if ew, ok := err.(errorWriter); ok {
			ew.writeTo(buf)
		} else {
			mustWriteString(buf, err.Error())
		}
	}
}

// If flattens an ErrorList, appending any contained ErrorLists and removing nil
// elements.  If the result is a single element, that element is returned
// without a slice wrapper.  If there are no elements, nil is returned.
//
// Note that this it is critical to use this method when returning type error,
// as returning a nil-valued ErrorList as an error will result in a non-nil
// error.  See https://golang.org/doc/faq#nil_error
func (errList ErrorList) If() error {
	// Trim nils off the start/end.
	for len(errList) > 0 && errList[0] == nil {
		errList = errList[1:]
	}
	for len(errList) > 1 && errList[len(errList)-1] == nil {
		errList = errList[:len(errList)-1]
	}
	// Common case, if there's only one error in the list, return it without
	// allocating a new list.
	if len(errList) == 1 {
		err := errList[0]
		if list, ok := err.(ErrorList); ok {
			return list.If()
		}
		return err
	} else if len(errList) > 0 {
		var errs ErrorList
		for i, err := range errList {
			if list, ok := err.(ErrorList); ok {
				err = list.If()
			}
			if err != nil {
				if list, ok := err.(ErrorList); ok {
					if errs == nil {
						errs = list
					} else {
						errs = append(errs, list...)
					}
				} else {
					if errs == nil {
						errs = make(ErrorList, 0, len(errList)-i)
					}
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

// As returns true if any error in the list satisfies errors.As(err, target).
//
// Target will be set to the first error in the list for which this is true.
func (errs ErrorList) As(target interface{}) bool {
	for _, err := range errs {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}

// As returns true if any error in the list satisfies errors.Is(err, target).
func (errs ErrorList) Is(target error) bool {
	for _, err := range errs {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// Returns the first non-nil error in the list, or nil.
func (errs ErrorList) Unwrap() error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

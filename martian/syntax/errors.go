//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Martian error types.
//

package syntax

import (
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

// AstError
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
	buff.Grow(len("Cannot have more than one top-level call.\n    First call: CALL_NAME at sourcefile.mro:100\n    Next call: SECOND_CALL at sourcefile.mro:200"))
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
	mustWriteString(w, "MRO ")
	if ew, ok := err.innerError.(errorWriter); ok {
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
	buff.Grow(len("MRO os.FileError: cannot access...\n    at sourcename.mro:100 included from sourcename.mro:10"))
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

// ParseError
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

// Collapse the error list down, and remove any nil errors.
// Returns nil if the list is empty.
func (errList ErrorList) If() error {
	if len(errList) > 0 {
		errs := make(ErrorList, 0, len(errList))
		for _, err := range errList {
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

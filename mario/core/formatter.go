//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario Formatting
//

package core

import (
	"fmt"
	"io/ioutil"
	"regexp"
	//"strings"
)

func FormatFile(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	src := string(data)
	re := regexp.MustCompile("^\\s*@include\\s+\"")
	src = re.ReplaceAllString(src, "#@include \"")
	global, mmli := yaccParse(src)
	if mmli != nil { // mmli is an mmLexInfo struct
		return "", &ParseError{mmli.token, filename, mmli.loc}
	}

	return global.format(), err
}

func (self *Ast) format() string {
	fsrc := ""

	// filetype declarations.
	for _, filetype := range self.filetypes {
		fsrc += fmt.Sprintf("filetype %s;\n", filetype.Id)
	}
	fsrc += "\n"

	// callables.
	for _, callable := range self.callables.list {
		switch c := callable.(type) {
		case *Stage:
			fsrc += fmt.Sprintf("stage %s\n", c.GetId())
		case *Pipeline:
			fsrc += fmt.Sprintf("pipeline %s\n", c.GetId())
		}
		fsrc += "\n"
	}

	return fsrc
}

package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

//
// Semantic Checking Helpers
//
type Checker struct {
	ast    *Ast
	locmap []FileLoc
}

func (self *Checker) checkIds(idables []Idable) error {
	table := map[string]bool{}
	for _, idable := range idables {
		if _, ok := table[idable.ID()]; ok {
			return &DuplicateNameError{MarioError{self.locmap, idable.Node().loc}, idable.ID()}
		}
		table[idable.ID()] = true
	}
	return nil
}

func (self *Checker) checkParameters(callable Callable) {
	idables := []Idable{}
	for _, param := range callable.Params() {
		fmt.Println(param)
		idables = append(idables, param)
	}
//	self.checkIds()
}

func (self *Checker) checkSemantics() error {
	// Build type table, starting with builtins. Duplicates allowed.
	types := []string{"string", "int", "float", "bool", "path", "file"}
	for _, filetype := range self.ast.filetypes {
		types = append(types, filetype.id)
	}
	typeTable := map[string]bool{}
	for _, t := range types {
		typeTable[t] = true
	}

	// Register stage and pipeline declarations and check for duplicate names.
	callableTable := map[string]Callable{}
	idables := []Idable{}
	for _, callable := range ast.callables {
		idables = append(idables, callable)
		callableTable[callable.ID()] = callable
		self.checkParameters(callable)
	}
	if err := self.checkIds(idables); err != nil {
		return err
	}

	return nil
}

//
// Package Exports
//
func ParseString(src string, locmap []FileLoc) (*Ast, error) {
	ast, err := yaccParse(src)
	if err != nil { // err is an mmLexInfo struct
		return nil, &ParseError{MarioError{locmap, err.loc}, err.token}
	}
	checker := Checker{ast, locmap}
	if err := checker.checkSemantics(); err != nil {
		return nil, err
	}
	return ast, nil
}

func ParseFile(filename string) (string, *Ast, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", nil, err
	}
	postsrc, locmap := preprocess(string(data), filename)
	//printSourceMap(postsrc, locmap)
	ast, err := ParseString(postsrc, locmap)
	return postsrc, ast, err
}

func main() {
	_, _, err := ParseFile("../pipelines/src/mro/analytics_phasing.mro")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	/*
		data, _ := ioutil.ReadFile("stages.mro")
		ast, err := Parse(string(data))
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		fmt.Println(len(ast.Decs))
		for _, dec := range ast.Decs {
			if filetypeDec, ok := dec.(*FileTypeDec); ok {
				fmt.Println(filetypeDec.id)
			}

			if stageDec, ok := dec.(*StageDec); ok {
				fmt.Println(stageDec.id)
				for _, param := range stageDec.params {
					fmt.Println(param)
				}
				if stageDec.splitter != nil {
					for _, param := range stageDec.splitter {
						fmt.Println(param)
					}
				}
			}

			if pipelineDec, ok := dec.(*PipelineDec); ok {
				fmt.Println(pipelineDec.id)
				for _, param := range pipelineDec.params {
					fmt.Println(param)
				}
				for _, call := range pipelineDec.calls {
					fmt.Println(call)
					for _, binding := range call.bindings {
						fmt.Println(binding.id, binding.exp)
					}
				}
				for _, binding := range pipelineDec.ret.bindings {
					fmt.Println(binding.id, binding.exp)
				}
			}
		}
	*/
}

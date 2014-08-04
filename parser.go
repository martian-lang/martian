package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

//
// Semantic Checking Helpers
//
func indexById(decs []Dec, decTable map[string]Dec, locmap []FileLoc) error {
	for _, dec := range decs {
		if _, ok := decTable[dec.ID()]; ok {
			return &DuplicateNameError{MarioError{locmap, dec.Node().loc}, dec.ID()}
		}
		decTable[dec.ID()] = dec
	}
	return nil
}
func checkSemantics(ast *Ast, locmap []FileLoc) error {
	// Build type table, starting with builtins. Duplicates allowed.
	typeTable := map[string]bool{
		"int": true, "string": true, "float": true,
		"bool": true, "path": true, "file": true,
	}

	// Put decs into separate lists.
	filetypeDecList := []Dec{}
	stageDecList := []Dec{}
	pipelineDecList := []Dec{}
	for _, dec := range ast.Decs {
		switch dec := dec.(type) {
		default:
		case *FileTypeDec:
			filetypeDecList = append(filetypeDecList, dec)
			typeTable[dec.id] = true
		case *StageDec:
			stageDecList = append(stageDecList, dec)
		case *PipelineDec:
			pipelineDecList = append(pipelineDecList, dec)
		}
	}

	// Register stage and pipeline declarations and check for duplicate names.
	decTable := map[string]Dec{}
	if err := indexById(append(stageDecList, pipelineDecList...), decTable, locmap); err != nil {
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
	if err := checkSemantics(ast, locmap); err != nil {
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

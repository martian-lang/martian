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
    global *Ast
    locmap []FileLoc
}

func (scope *CallScope) check(locmap []FileLoc) error {    
    for _, callable := range scope.callables {
        if _, ok := scope.table[callable.Id()]; ok {
            return &DuplicateNameError{MarioError{locmap, callable.Node().loc}, "stage or pipeline", callable.Id()}
        }
        scope.table[callable.Id()] = callable
    }
    return nil
}

func (scope *ParamScope) check(locmap []FileLoc) error {
    for _, param := range scope.params {
        if _, ok := scope.table[param.Id()]; ok {
            return &DuplicateNameError{MarioError{locmap, param.Node().loc}, "parameter", param.Id()}
        }
        scope.table[param.Id()] = param
    }
    return nil
}

func (self *Checker) checkSemantics() error {
    // Build type table, starting with builtins. Duplicates allowed.
    types := []string{"string", "int", "float", "bool", "path", "file"}
    for _, filetype := range self.global.filetypes {
        types = append(types, filetype.id)
    }
    typeTable := map[string]bool{}
    for _, t := range types {
        typeTable[t] = true
    }

    // Check for duplicate names amongst callables.
    if err := self.global.callScope.check(self.locmap); err != nil {
        return err
    }

    for _, stage := range self.global.stages {
        fmt.Println(stage.id)
        if err := stage.inParams.check(self.locmap); err != nil {
            return err
        }
        if err := stage.outParams.check(self.locmap); err != nil {
            return err
        }
    }

    return nil
}

//
// Package Exports
//
func ParseString(src string, locmap []FileLoc) (*Ast, error) {
    global, err := yaccParse(src)
    if err != nil { // err is an mmLexInfo struct
        return nil, &ParseError{MarioError{locmap, err.loc}, err.token}
    }
    checker := Checker{global, locmap}
    if err := checker.checkSemantics(); err != nil {
        return nil, err
    }
    return global, nil
}

func ParseFile(filename string) (string, *Ast, error) {
    data, err := ioutil.ReadFile(filename)
    if err != nil {
        return "", nil, err
    }
    postsrc, locmap := preprocess(string(data), filename)
    //printSourceMap(postsrc, locmap)
    global, err := ParseString(postsrc, locmap)
    return postsrc, global, err
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

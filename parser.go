package main

import (
    "fmt"
    "io/ioutil"
    "os"
)

//
// Semantic Checking Helpers
//
func (global *Ast) Error(locable Locatable, msg string) error {
    return &MarioError{global, locable, msg}
}

func (scope *CallScope) check(global *Ast) error {    
    for _, callable := range scope.callables {
        // Check for duplicates
        if _, ok := scope.table[callable.Id()]; ok {
            return global.Error(callable,
                fmt.Sprintf("DuplicateNameError: stage or pipeline '%s' was previously declared when encountered", callable.Id()))
        }
        scope.table[callable.Id()] = callable
    }
    return nil
}

func (scope *ParamScope) check(global *Ast) error {
    for _, param := range scope.params {
        // Check for duplicates
        if _, ok := scope.table[param.Id()]; ok {
            return global.Error(param,
                fmt.Sprintf("DuplicateNameError: parameter '%s' was previously declared when encountered", param.Id()))
        }
        scope.table[param.Id()] = param

        if _, ok := global.typeTable[param.Tname()]; !ok {
            return global.Error(param,
                fmt.Sprintf("TypeError: undefined type '%s'", param.Tname()))
        }

    }
    return nil
}

func (global *Ast) check() error {
    // Build type table, starting with builtins. Duplicates allowed.
    types := []string{"string", "int", "float", "bool", "path", "file"}
    for _, filetype := range global.filetypes {
        types = append(types, filetype.id)
    }
    for _, t := range types {
        global.typeTable[t] = true
    }

    // Check for duplicate names amongst callables.
    if err := global.callScope.check(global); err != nil {
        return err
    }

    for _, stage := range global.stages {
        if err := stage.inParams.check(global); err != nil {
            return err
        }
        if err := stage.outParams.check(global); err != nil {
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
        return nil, &ParseError{err.token, locmap[err.loc].fname, locmap[err.loc].loc}
    }

    global.locmap = locmap
    
    if err := global.check(); err != nil {
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

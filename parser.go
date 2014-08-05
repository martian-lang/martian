package main

import (
    "fmt"
    "io/ioutil"
    "os"
)

//
// Semantic Checking Helpers
//
func (global *Ast) err(locable Locatable, msg string) error {
    return &MarioError{global, locable, msg}
}

func (scope *CallableScope) check(global *Ast) error {    
    for _, callable := range scope.callables {
        // Check for duplicates
        if _, ok := scope.table[callable.Id()]; ok {
            return global.err(callable, fmt.Sprintf("DuplicateNameError: stage or pipeline '%s' was already declared when encountered again", callable.Id()))
        }
        scope.table[callable.Id()] = callable
    }
    return nil
}

func (scope *ParamScope) check(global *Ast) error {
    for _, param := range scope.params {
        // Check for duplicates
        if _, ok := scope.table[param.Id()]; ok {
            return global.err(param, fmt.Sprintf("DuplicateNameError: parameter '%s' was already declared when encountered again", param.Id()))
        }
        scope.table[param.Id()] = param

        // Check that types exist.
        if _, ok := global.typeTable[param.Tname()]; !ok {
            return global.err(param, fmt.Sprintf("TypeError: undefined type '%s'", param.Tname()))
        }

    }
    return nil
}

func (binding *Binding) Type() string {
    exp := binding.exp
    /*
    # Handle scalar types.
    if valueExp.kind in ['int','float','string','bool','path','null']
        return valueExp.kind

    # Array: [ 1, 2 ]
    if valueExp.kind == 'array'
        lastType = 'null'
        for elem in valueExp.value
            elemType = resolveExpType(elem, decTable, paramTable)
            # Make sure types of all array elements match.
            if lastType and (not checkTypeMatch(elemType, lastType))
                throw new MixedTypeArrayError(valueExp.loc)
            lastType = elemType
        return lastType

    # Param: self.myparam
    if valueExp.kind == 'self'
        param = paramTable[valueExp.id]
        if not param?
            throw new ScopeNameError('param', valueExp.id, valueExp.loc)
        if param.mode != 'in'
            throw new BindingInputError(valueExp.id, valueExp.loc)        
        return param.type

    # Call: STAGE.myoutparam or STAGE
    if valueExp.kind == 'call'
        # Check referenced segment is actually called in this scope.
        dec = decTable[valueExp.id]
        if not dec?
            throw new ScopeNameError('segment', valueExp.id, valueExp.loc)
    */
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
    if err := global.callableScope.check(global); err != nil {
        return err
    }

    // Check stage declarations.
    for _, stage := range global.stages {
        // Check in parameters.
        if err := stage.inParams.check(global); err != nil {
            return err
        }
        // Check out parameters.
        if err := stage.outParams.check(global); err != nil {
            return err
        }
        // Check split parameters.
        if stage.splitParams != nil {
            if err := stage.splitParams.check(global); err != nil {
                return err
            }
        }
    }

    // Check pipeline declarations.
    for _, pipeline := range global.pipelines {
        // Check in parameters.
        if err := pipeline.inParams.check(global); err != nil {
            return err
        }
        // Check out parameters.
        if err := pipeline.outParams.check(global); err != nil {
            return err
        }

        // Check for duplicate calls.
        for _, call := range pipeline.calls {
            if _, ok := pipeline.callableTable[call.id]; ok {
               return global.err(call, fmt.Sprintf("DuplicateCallError: '%s' was already called when encountered again", call.id))
            }
            // Check we're calling something declared.
            callable, ok := global.callableScope.table[call.id]
            if !ok {
                return global.err(call, fmt.Sprintf("ScopeNameError: '%s' is not defined in this scope", call.id))
            }
            // Save the valid callables for this scope.
            pipeline.callableTable[call.id] = callable

            // Check the bindings
            for _, binding := range call.bindings {
                // Collect bindings by id so we can check that all params are bound.
                call.bindingTable[binding.id] = binding

                // Make sure the bound-to id is a declared input parameter of the callable.
                if _, ok := callable.InParams().table[binding.id]; !ok {
                    return global.err(call, fmt.Sprintf("ArgumentError: '%s' is not an input parameter of '%s'", binding.id, call.id))
                }

                // Typecheck the binding and cache the type.
                valueType := binding.Type()
                //resolveExpType(binding.valueExp, decTable, pipelineParamTable, typeTable)
                //if not checkTypeMatch(param.type, valueType)
                //    throw new TypeMismatchError(bindStm.id, param.type, valueType, bindStm.loc)
                //bindStm.type = param.type

            }
            // Check that all input params of the callable are bound.   
            for _, param := range callable.InParams().params {
                if _, ok := call.bindingTable[param.Id()]; !ok {
                    return global.err(call, fmt.Sprintf("ArgumentNotSuppliedError: no argument supplied for parameter '%s'", param.Id()))
                }                    
            }
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

package main

import (
	"io/ioutil"
	"fmt"
)

func main() {
	data, _ := ioutil.ReadFile("stages.mro")
	ast := Parse(string(data))
	fmt.Println(len(ast.Decs))
	for _, dec := range ast.Decs {
		filetypeDec, ok := dec.(*FileTypeDec)
		if ok {
			fmt.Println(filetypeDec.id)
		}

		stageDec, ok := dec.(*StageDec)
		if ok {
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

		pipelineDec, ok := dec.(*PipelineDec)
		if ok {
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

}

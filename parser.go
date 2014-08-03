package main

import (
	"io/ioutil"
	"fmt"
	"os"
)

type FileLoc struct {
	fname string
	n int
}

func parse(src string, locmap map[int]FileLoc) *Ptree {
	return nil
}

func main() {
	data, _ := ioutil.ReadFile("stages.mro")
	ptree, err := Parse(string(data))
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println(len(ptree.Decs))
	for _, dec := range ptree.Decs {
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

}

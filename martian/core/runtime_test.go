//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian runtime tests.
//

package core

import (
	"encoding/json"
	"fmt"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

func MockRuntime() *Runtime {
	util.ENABLE_LOGGING = false // Disable core.LogInfo calls in Runtime
	return NewRuntime("local", "disable", "disable", "")
}

func ExampleBuildCallSource() {
	src, _ := BuildCallSource([]string{"bar1.mro, bar2.mro"},
		"STAGE_NAME",
		map[string]interface{}{
			"input1": []int{1, 2},
			"input2": "foo",
			"input3": json.RawMessage(`{"foo":"bar"}`),
		},
		nil,
		&syntax.Stage{
			Node: syntax.NewAstNode(15, &syntax.SourceFile{
				FileName: "foo.mro",
				FullPath: "/path/to/foo.mro",
			}),
			Id: "STAGE_NAME",
			InParams: &syntax.Params{
				List: []syntax.Param{
					&syntax.InParam{
						Tname:    "int",
						ArrayDim: 1,
						Id:       "input1",
					},
					&syntax.InParam{
						Tname: "string",
						Id:    "input2",
					},
					&syntax.InParam{
						Tname: "map",
						Id:    "input3",
					},
				},
			},
		})
	fmt.Println(src)
	// Output:
	// @include "foo.mro"
	//
	// call STAGE_NAME(
	//     input1 = [
	//         1,
	//         2
	//     ],
	//     input2 = "foo",
	//     input3 = {
	//         "foo": "bar"
	//     },
	// )
}

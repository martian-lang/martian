// Copyright (c) 2021 10X Genomics, Inc. All rights reserved.

package ast_builder

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/martian-lang/martian/martian/syntax"
)

func ExampleBindings() {
	bindings, err := Bindings(map[string]interface{}{
		"value1": 1,
		"value2": map[uint]string{1: "bar"},
		"value3": nil,
		"value4": json.RawMessage(`{"a": 2}`),
		"value5": new(struct {
			V1 bool              `json:"v1,string"`
			V2 uint              `json:"v2,string"`
			V3 map[string]string `json:"v3"`
		}),
	})
	if err != nil {
		panic(err)
	}
	ast := syntax.Ast{
		Call: &syntax.CallStm{
			DecId:    "STAGE",
			Id:       "STAGE",
			Bindings: bindings,
		},
	}
	fmt.Println(ast.Format())

	// Output:
	// call STAGE(
	//     value1 = 1,
	//     value2 = {
	//         "1": "bar",
	//     },
	//     value3 = null,
	//     value4 = {
	//         "a": 2,
	//     },
	//     value5 = {
	//         v1: "false",
	//         v2: "0",
	//         v3: null,
	//     },
	// )
}

func TestBadKeys(t *testing.T) {
	type KeyType struct {
		Value int
	}
	if _, err := ValExp(
		map[string]interface{}{
			"foo": map[KeyType]string{
				{}: "",
			},
		}); err == nil {
		t.Error("expected error")
	}
}

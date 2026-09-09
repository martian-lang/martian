// Copyright (c) 2021 10X Genomics, Inc. All rights reserved.

package ast_builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/martian-lang/martian/martian/syntax"
)

func ExampleStructType() {
	type InnerStruct struct {
		Value int
	}
	type MyStruct struct {
		Time           time.Time
		UntypedMap1    map[string]interface{}
		UntypedMap2    map[string]json.RawMessage
		MapVal         map[time.Time]InnerStruct `json:"int_map" mro_help:"a bunch of ints"`
		ByteMap        map[string][]byte         `json:"byte_map1"`
		FileByteMap    map[string][][]byte       `json:"byte_map2" mro_type:"csv"`
		StringVal      string                    `json:"str1"`
		UntypedMap3    []map[string]map[string]string
		Csvs           [][]string `json:"csvs" mro_type:"csv" mro_out:"csv_files"`
		unexported     int
		Omitted        int `json:"-"`
		IntAsStringVal int `json:"str2,string"`
		Float          float32
		FloatStr       float32 `json:"float_str,string"`
		BoolVal        bool
		Byte           int8
	}
	val := MyStruct{
		StringVal:      "foo",
		IntAsStringVal: 1,
		Time:           time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
		Omitted:        2,
		unexported:     3,
		MapVal:         map[time.Time]InnerStruct{time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC): {Value: 1}},
		Csvs: [][]string{
			{"foo"},
			nil,
			{"bar", "baz"},
		},
		ByteMap: map[string][]byte{"foo": []byte("bar"), "baz": nil},
		FileByteMap: map[string][][]byte{
			"foo": {[]byte("bar"), nil},
			"baz": nil,
		},
		UntypedMap1: map[string]interface{}{"foo": "bar", "baz": 2},
		UntypedMap2: nil,
		UntypedMap3: []map[string]map[string]string{
			{
				"foo": {"foo": "bar", "baz": ""},
			},
		},
		Float:    4,
		FloatStr: 5,
		BoolVal:  true,
		Byte:     6,
	}
	st1, err := StructType(reflect.TypeOf(InnerStruct{}))
	if err != nil {
		panic(err)
	}
	st2, err := StructType(reflect.TypeOf(val))
	if err != nil {
		panic(err)
	}
	bindings, err := Bindings(val)
	if err != nil {
		panic(err)
	}
	ast := syntax.Ast{
		UserTypes:   []*syntax.UserType{{Id: "csv"}},
		StructTypes: []*syntax.StructType{st1, st2},
		Call: &syntax.CallStm{
			DecId:    "STAGE",
			Id:       "ALIAS",
			Bindings: bindings,
		},
	}
	fmt.Println(ast.Format())

	// Output:
	// filetype csv;
	//
	// struct InnerStruct(
	//     int Value,
	// )
	//
	// struct MyStruct(
	//     string           Time,
	//     map              UntypedMap1,
	//     map              UntypedMap2,
	//     map<InnerStruct> int_map     "a bunch of ints",
	//     map<string>      byte_map1,
	//     map<csv[]>       byte_map2,
	//     string           str1,
	//     map[]            UntypedMap3,
	//     csv[][]          csvs        ""                "csv_files",
	//     string           str2,
	//     float            Float,
	//     string           float_str,
	//     bool             BoolVal,
	//     int              Byte,
	// )
	//
	// call STAGE as ALIAS(
	//     Time        = "2021-01-02T03:04:05Z",
	//     UntypedMap1 = {
	//         "baz": 2,
	//         "foo": "bar",
	//     },
	//     UntypedMap2 = null,
	//     int_map     = {
	//         "2021-01-02T03:04:05Z": {
	//             Value: 1,
	//         },
	//     },
	//     byte_map1   = {
	//         "baz": null,
	//         "foo": "YmFy",
	//     },
	//     byte_map2   = {
	//         "baz": null,
	//         "foo": [
	//             "YmFy",
	//             null,
	//         ],
	//     },
	//     str1        = "foo",
	//     UntypedMap3 = [
	//         {
	//             "foo": {
	//                 "baz": "",
	//                 "foo": "bar",
	//             },
	//         },
	//     ],
	//     csvs        = [
	//         ["foo"],
	//         null,
	//         [
	//             "bar",
	//             "baz",
	//         ],
	//     ],
	//     str2        = "1",
	//     Float       = 4,
	//     float_str   = "5",
	//     BoolVal     = true,
	//     Byte        = 6,
	// )
}

func TestEmbeddedStruct(t *testing.T) {
	type InnerStruct struct {
		Value1 int
	}
	type OuterStruct struct {
		Time   time.Time `mro_type:"time"`
		Value3 *InnerStruct
		Value4 *InnerStruct
		InnerStruct
		Value2 uint
	}
	val := OuterStruct{
		InnerStruct: InnerStruct{Value1: 1},
		Value2:      2,
		Value3:      new(InnerStruct),
	}
	st, err := StructType(reflect.TypeOf(val))
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := Bindings(&val)
	if err != nil {
		t.Fatal(err)
	}
	ast := syntax.Ast{
		StructTypes: []*syntax.StructType{st},
		Call: &syntax.CallStm{
			DecId:    "STAGE",
			Id:       "STAGE",
			Bindings: bindings,
		},
	}
	const expected = `struct OuterStruct(
    time        Time,
    InnerStruct Value3,
    InnerStruct Value4,
    int         Value1,
    int         Value2,
)

call STAGE(
    Time   = "0001-01-01T00:00:00Z",
    Value3 = {
        Value1: 0,
    },
    Value4 = null,
    Value1 = 1,
    Value2 = 2,
)
`
	result := ast.Format()
	if result != expected {
		t.Errorf("%s\n!=\n"+expected, result)
	}
}

func TestIllegalStruct(t *testing.T) {
	type HasInterface struct {
		Field interface{}
	}
	if _, err := StructType(reflect.TypeOf(new(HasInterface))); err == nil {
		t.Error("expected error")
	} else if !errors.Is(err, InterfaceTypeError) {
		t.Error("expected interface type error, got ", err)
	}
	type HasJson struct {
		Field json.RawMessage
	}
	if _, err := StructType(reflect.TypeOf(new(HasJson))); err == nil {
		t.Error("expected error")
	} else if !errors.Is(err, UnknownTypeError) {
		t.Error("expected unknown type error, got ", err)
	}
	type KeyType struct {
		Value int
	}
	type HasIllegalKey struct {
		Value []map[KeyType]string
	}
	if _, err := StructType(reflect.TypeOf(new(HasIllegalKey))); err == nil {
		t.Error("expected error")
	} else if !strings.Contains(err.Error(), "non-stringable key type") {
		t.Error("non-stringable key type error, got ", err)
	}
}

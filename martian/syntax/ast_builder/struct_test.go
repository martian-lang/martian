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
		StringVal      string `json:"str1"`
		IntAsStringVal int    `json:"str2,string"`
		Time           time.Time
		Omitted        int `json:"-"`
		unexported     int
		MapVal         map[time.Time]InnerStruct `json:"int_map" mro_help:"a bunch of ints"`
		Csvs           [][]string                `json:"csvs" mro_type:"csv" mro_out:"csv_files"`
		ByteMap        map[string][]byte         `json:"byte_map1"`
		FileByteMap    map[string][][]byte       `json:"byte_map2" mro_type:"csv"`
		UntypedMap1    map[string]interface{}
		UntypedMap2    map[string]json.RawMessage
		UntypedMap3    []map[string]map[string]string
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
	//     string           str1,
	//     string           str2,
	//     string           Time,
	//     map<InnerStruct> int_map     "a bunch of ints",
	//     csv[][]          csvs        ""                "csv_files",
	//     map<string>      byte_map1,
	//     map<csv[]>       byte_map2,
	//     map              UntypedMap1,
	//     map              UntypedMap2,
	//     map[]            UntypedMap3,
	//     float            Float,
	//     string           float_str,
	//     bool             BoolVal,
	//     int              Byte,
	// )
	//
	// call STAGE as ALIAS(
	//     str1        = "foo",
	//     str2        = "1",
	//     Time        = "2021-01-02T03:04:05Z",
	//     int_map     = {
	//         "2021-01-02T03:04:05Z": {
	//             Value: 1,
	//         },
	//     },
	//     csvs        = [
	//         ["foo"],
	//         null,
	//         [
	//             "bar",
	//             "baz",
	//         ],
	//     ],
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
	//     UntypedMap1 = {
	//         "baz": 2,
	//         "foo": "bar",
	//     },
	//     UntypedMap2 = null,
	//     UntypedMap3 = [
	//         {
	//             "foo": {
	//                 "baz": "",
	//                 "foo": "bar",
	//             },
	//         },
	//     ],
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
		InnerStruct
		Value2 uint
		Value3 *InnerStruct
		Value4 *InnerStruct
		Time   time.Time `mro_type:"time"`
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
    int         Value1,
    int         Value2,
    InnerStruct Value3,
    InnerStruct Value4,
    time        Time,
)

call STAGE(
    Value1 = 1,
    Value2 = 2,
    Value3 = {
        Value1: 0,
    },
    Value4 = null,
    Time   = "0001-01-01T00:00:00Z",
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

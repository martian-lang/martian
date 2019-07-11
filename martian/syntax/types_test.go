// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"fmt"
	"strings"
	"testing"
)

func TestTypeIdWrite(t *testing.T) {
	check := func(t *testing.T, id TypeId, expect string) {
		t.Helper()
		if s := id.String(); s != expect {
			t.Errorf(
				"array dim %d map dim %d type %s formatted as %q",
				id.ArrayDim, id.MapDim, id.Tname, s)
		}
		if s := id.strlen(); s != len(expect) {
			t.Errorf(
				"array dim %d map dim %d type %s incorrect length %d",
				id.ArrayDim, id.MapDim, id.Tname, s)
		}
		var buf strings.Builder
		id.writeTo(&buf)
		if s := buf.String(); s != expect {
			t.Errorf(
				"array dim %d map dim %d type %s wrote out as %q",
				id.ArrayDim, id.MapDim, id.Tname, s)
		}
	}
	check(t, TypeId{
		Tname:    "int",
		ArrayDim: 1,
	}, "int[]")
	check(t, TypeId{
		Tname:    "float",
		ArrayDim: 1,
		MapDim:   1,
	}, "map<float>[]")
	check(t, TypeId{
		Tname:  "bool",
		MapDim: 3,
	}, "map<bool[][]>")
	check(t, TypeId{
		Tname:    "complicated",
		ArrayDim: 4,
		MapDim:   5,
	}, "map<complicated[][][][]>[][][][]")
	check(t, TypeId{
		Tname:  "notMap",
		MapDim: 1,
	}, "map<notMap>")
}

func TestArrayTypeIsAssignableFrom(t *testing.T) {
	var ast Ast
	ast.TypeTable.init(1)
	lookup := &ast.TypeTable

	// array conversions
	floatArray := ArrayType{Elem: &builtinFloat, Dim: 1}
	if err := builtinFloat.IsAssignableFrom(&floatArray, lookup); err == nil {
		t.Error("conversion of array to singleton is not allowed")
	}
	intArray := ArrayType{Elem: &builtinInt, Dim: 1}
	if err := builtinFloat.IsAssignableFrom(&intArray, lookup); err == nil {
		t.Error("conversion of array to singleton is not allowed")
	}
	if err := floatArray.IsAssignableFrom(&floatArray, lookup); err != nil {
		t.Error(err)
	}
	if err := floatArray.IsAssignableFrom(&intArray, lookup); err != nil {
		t.Error(err)
	}
	if err := intArray.IsAssignableFrom(&floatArray, lookup); err == nil {
		t.Error("conversion of float[] to int[] is not allowed")
	}
	intArray2D := ArrayType{Elem: &builtinInt, Dim: 2}
	if err := intArray2D.IsAssignableFrom(&intArray, lookup); err == nil {
		t.Error("assigning int[][] to int[] is not allowed")
	}
	if err := intArray.IsAssignableFrom(&builtinInt, lookup); err == nil {
		t.Error("assigning int to int[] is not allowed")
	}
	if err := intArray.IsAssignableFrom(&builtinMap, lookup); err == nil {
		t.Error("assigning map to int[] is not allowed")
	}
}

func TestTypedMapTypeIsAssignableFrom(t *testing.T) {
	var ast Ast
	ast.TypeTable.init(1)
	lookup := &ast.TypeTable

	floatMap := TypedMapType{Elem: &builtinFloat}
	if err := builtinFloat.IsAssignableFrom(&floatMap, lookup); err == nil {
		t.Error("conversion of map to singleton is not allowed")
	}
	if err := floatMap.IsAssignableFrom(&floatMap, lookup); err != nil {
		t.Error(err)
	}
	intMap := TypedMapType{Elem: &builtinInt}
	if err := floatMap.IsAssignableFrom(&intMap, lookup); err != nil {
		t.Error(err)
	}
	if err := intMap.IsAssignableFrom(&floatMap, lookup); err == nil {
		t.Error("conversion of map<float> to map<int> is not allowed.")
	}
	intArray := ArrayType{Elem: &builtinInt, Dim: 1}
	if err := intMap.IsAssignableFrom(&intArray, lookup); err == nil {
		t.Error("conversion of int[] to map<int> is not allowed.")
	}

	// struct to map conversion
	structType := StructType{
		Id: "MY_STRUCT",
		Members: []*StructMember{
			&StructMember{
				Id: "my_field_1",
				Tname: TypeId{
					Tname: KindInt,
				},
			},
			&StructMember{
				Id: "my_field_2",
				Tname: TypeId{
					Tname: KindFloat,
				},
			},
		},
	}
	if err := structType.compile(&ast); err != nil {
		t.Error(err)
	}
	if err := ast.TypeTable.AddStructType(&structType); err != nil {
		t.Error(err)
	}
	if err := intMap.IsAssignableFrom(&structType, lookup); err == nil {
		t.Error("assigning a struct with float fields to an int map is not allowed.")
	}
	if err := floatMap.IsAssignableFrom(&structType, lookup); err != nil {
		t.Error(err)
	}
	if err := intMap.IsAssignableFrom(&builtinMap, lookup); err == nil {
		t.Error("assigning map to map<int> is not allowed")
	}
}

func TestArrayTypeIsValidExpression(t *testing.T) {
	floatArray := ArrayType{Elem: &builtinFloat, Dim: 2}
	if err := floatArray.IsValidExpression(&ArrayExp{
		Value: []Exp{
			new(NullExp),
			&ArrayExp{
				Value: []Exp{
					&FloatExp{Value: 1.5},
					&IntExp{Value: 4},
				},
			},
		},
	}, nil, nil); err != nil {
		t.Error(err)
	}
}

func TestTypedMapTypeIsValidExpression(t *testing.T) {
	floatMap := TypedMapType{Elem: &builtinFloat}
	if err := floatMap.IsValidExpression(&MapExp{
		Kind: KindMap,
		Value: map[string]Exp{
			"foo": &FloatExp{Value: 1.5},
			"bar": &IntExp{Value: 4},
		},
	}, nil, nil); err != nil {
		t.Error(err)
	}
}

func TestTypeIdMarshalText(t *testing.T) {
	id := TypeId{
		Tname:    "bam",
		ArrayDim: 1,
		MapDim:   3,
	}
	if b, err := id.MarshalText(); err != nil {
		t.Error(err)
	} else if s := string(b); s != "map<bam[][]>[]" {
		t.Errorf(`%q != "map<bam[][]>[]"`, s)
	}
}

func ExampleTypeId_GoString() {
	id := TypeId{
		Tname:    "bam",
		ArrayDim: 1,
		MapDim:   3,
	}
	fmt.Println(id.GoString())
	// Output:
	// map<bam[][]>[]
}

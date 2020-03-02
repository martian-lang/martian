// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package syntax

import "testing"

func TestBuiltinIsAssignableFrom(t *testing.T) {
	var ast Ast
	ast.TypeTable.init(1)
	lookup := &ast.TypeTable

	// int/float conversions
	if err := builtinFloat.IsAssignableFrom(&builtinFloat, lookup); err != nil {
		t.Error(err)
	}
	if err := builtinFloat.IsAssignableFrom(&builtinInt, lookup); err != nil {
		t.Error(err)
	}
	if err := builtinInt.IsAssignableFrom(&builtinInt, lookup); err != nil {
		t.Error(err)
	}
	if err := builtinInt.IsAssignableFrom(&builtinFloat, lookup); err == nil {
		t.Error("conversion of float to int is not allowed.")
	}

	// incompatible type
	if err := builtinFloat.IsAssignableFrom(&builtinString, lookup); err == nil {
		t.Error("conversion of string to float is not allowed")
	}
	if err := builtinFloat.IsAssignableFrom(&builtinMap, lookup); err == nil {
		t.Error("conversion of map to float is not allowed")
	}

	// user file type conversions

	user := UserType{Id: "foo"}
	if err := builtinFloat.IsAssignableFrom(&user, lookup); err == nil {
		t.Error("conversion of user file type to float is not allowed.")
	}
	if err := builtinFile.IsAssignableFrom(&user, lookup); err != nil {
		t.Error(err)
	}

	// map conversions

	floatMap := TypedMapType{Elem: &builtinFloat}
	if err := builtinMap.IsAssignableFrom(&floatMap, lookup); err != nil {
		t.Error(err)
	}

	// stringy conversions
	if err := builtinPath.IsAssignableFrom(&builtinString, lookup); err != nil {
		t.Error(err)
	}
	if err := builtinFile.IsAssignableFrom(&builtinString, lookup); err != nil {
		t.Error(err)
	}

	// struct to map conversion
	structType := StructType{
		Id: "MY_STRUCT",
		Members: []*StructMember{
			{
				Id: "my_field",
				Tname: TypeId{
					Tname: KindInt,
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
	if err := builtinMap.IsAssignableFrom(&structType, lookup); err != nil {
		t.Error(err)
	}
	if err := builtinInt.IsAssignableFrom(&structType, lookup); err == nil {
		t.Error("cannot assign struct to int")
	}
}

func TestBuiltinIsValidExpression(t *testing.T) {
	if err := builtinFloat.IsValidExpression(&FloatExp{Value: 1}, nil, nil); err != nil {
		t.Error(err)
	}
	if err := builtinInt.IsValidExpression(&FloatExp{Value: 1}, nil, nil); err != nil {
		t.Error(err)
	}
	if err := builtinFloat.IsValidExpression(&StringExp{Value: "1"}, nil, nil); err == nil {
		t.Error("cannot assign string to float")
	}
	if err := builtinString.IsValidExpression(&FloatExp{Value: 1}, nil, nil); err == nil {
		t.Error("cannot assign float to string")
	}
	if err := builtinString.IsValidExpression(&IntExp{Value: 1}, nil, nil); err == nil {
		t.Error("cannot assign int to string")
	}
	if err := builtinMap.IsValidExpression(&MapExp{
		Kind: KindMap,
		Value: map[string]Exp{
			"foo": &StringExp{Value: "bar"},
		},
	}, nil, nil); err != nil {
		t.Error(err)
	}
	if err := builtinMap.IsValidExpression(&MapExp{
		Kind: KindStruct,
		Value: map[string]Exp{
			"foo": &StringExp{Value: "bar"},
		},
	}, nil, nil); err == nil {
		t.Error("cannot assign struct to map")
	}
	if err := builtinMap.IsValidExpression(&MapExp{
		Kind: KindMap,
		Value: map[string]Exp{
			"foo": &RefExp{
				Kind: KindSelf,
				Id:   "foo"},
		},
	}, nil, nil); err == nil {
		t.Error("cannot assign map with references to untyped map")
	}
}

func TestBuiltinFilterJson(t *testing.T) {
	b, fatal, err := builtinInt.FilterJson([]byte(`1.0`), nil)
	if err == nil {
		t.Error("expected warning")
	}
	if fatal {
		t.Error("expected non-fatal")
	}
	if s := string(b); s != "1" {
		t.Errorf(`%q != "1"`, s)
	}
	_, fatal, err = builtinInt.FilterJson([]byte(`1.5`), nil)
	if err == nil {
		t.Error("expected warning")
	}
	if !fatal {
		t.Error("expected fatal")
	}
	b, fatal, err = builtinInt.FilterJson([]byte(` null`), nil)
	if err != nil {
		t.Error(err)
	}
	if fatal {
		t.Error("expected success")
	}
	if !isNullBytes(b) {
		t.Error(string(b))
	}
	b, fatal, err = builtinString.FilterJson([]byte(`"yo"`), nil)
	if err != nil {
		t.Error(err)
	}
	if fatal {
		t.Error("expected success")
	}
	if s := string(b); s != `"yo"` {
		t.Errorf(`%q != "yo"`, s)
	}
}

// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

package syntax

import "testing"

func TestMapIsAssignableFrom(t *testing.T) {
	structType1 := StructType{
		Id: "MY_STRUCT_1",
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
	structType2 := StructType{
		Id: "MY_STRUCT_2",
		Members: []*StructMember{
			&StructMember{
				Id: "my_field_1",
				Tname: TypeId{
					Tname: KindFloat,
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
	structType3 := StructType{
		Id: "NESTED_STRUCT_1",
		Members: []*StructMember{
			&StructMember{
				Id: "my_field_1",
				Tname: TypeId{
					Tname: structType1.Id,
				},
			},
			&StructMember{
				Id: "my_field_2",
				Tname: TypeId{
					Tname: structType1.Id,
				},
			},
		},
	}
	structType4 := StructType{
		Id: "NESTED_STRUCT_2",
		Members: []*StructMember{
			&StructMember{
				Id: "my_field_1",
				Tname: TypeId{
					Tname: structType2.Id,
				},
			},
			&StructMember{
				Id: "my_field_2",
				Tname: TypeId{
					Tname: structType2.Id,
				},
			},
		},
	}
	structType5 := StructType{
		Id: "MY_STRUCT_5",
		Members: []*StructMember{
			&StructMember{
				Id: "my_field_1",
				Tname: TypeId{
					Tname: KindFloat,
				},
			},
			&StructMember{
				Id: "my_field_2",
				Tname: TypeId{
					Tname: KindFloat,
				},
			},
			&StructMember{
				Id: "my_field_3",
				Tname: TypeId{
					Tname: KindFloat,
				},
			},
		},
	}
	structType6 := StructType{
		Id: "MY_STRUCT_6",
		Members: []*StructMember{
			&StructMember{
				Id: "my_field_1",
				Tname: TypeId{
					Tname: KindFloat,
				},
			},
			&StructMember{
				Id: "my_field_2",
				Tname: TypeId{
					Tname:    KindFloat,
					ArrayDim: 1,
				},
			},
		},
	}
	structType7 := StructType{
		Id: "MY_STRUCT_7",
		Members: []*StructMember{
			&StructMember{
				Id: "my_field_1",
				Tname: TypeId{
					Tname: KindFloat,
				},
			},
			&StructMember{
				Id: "my_field_2",
				Tname: TypeId{
					Tname:  KindFloat,
					MapDim: 1,
				},
			},
		},
	}
	ast := Ast{
		Callables: new(Callables),
		StructTypes: []*StructType{
			&structType1,
			&structType2,
			&structType3,
			&structType4,
			&structType5,
			&structType6,
			&structType7,
		},
	}
	lookup := &ast.TypeTable
	if err := ast.compileTypes(); err != nil {
		t.Error(err)
	}

	if err := structType1.IsAssignableFrom(&builtinMap, lookup); err == nil {
		t.Error("assigning map to struct is not allowed.")
	}
	if err := structType1.IsAssignableFrom(&structType1, lookup); err != nil {
		t.Error(err)
	}
	if err := structType2.IsAssignableFrom(&structType1, lookup); err != nil {
		t.Error(err)
	}
	if err := structType1.IsAssignableFrom(&structType2, lookup); err == nil {
		t.Error("coversion of float field to int is not allowed.")
	}

	if err := structType3.IsAssignableFrom(&structType3, lookup); err != nil {
		t.Error(err)
	}
	if err := structType4.IsAssignableFrom(&structType3, lookup); err != nil {
		t.Error(err)
	}
	if err := structType3.IsAssignableFrom(&structType4, lookup); err == nil {
		t.Error("coversion of float field to int is not allowed.")
	}
	if err := structType2.IsAssignableFrom(&structType5, lookup); err != nil {
		t.Error(err)
	}
	if err := structType5.IsAssignableFrom(&structType2, lookup); err == nil {
		t.Errorf("missing field")
	}
	if err := structType6.IsAssignableFrom(&structType2, lookup); err == nil {
		t.Errorf("array mismatch")
	}
	if err := structType2.IsAssignableFrom(&structType6, lookup); err == nil {
		t.Errorf("array mismatch")
	}
	if err := structType7.IsAssignableFrom(&structType2, lookup); err == nil {
		t.Errorf("map mismatch")
	}
	if err := structType2.IsAssignableFrom(&structType7, lookup); err == nil {
		t.Errorf("map mismatch")
	}
}

func TestStructTypeIsValidExpression(t *testing.T) {
	var ast Ast
	ast.TypeTable.init(2)

	// struct to map conversion
	structType1 := StructType{
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
	structType2 := StructType{
		Id: "NESTED_STRUCT",
		Members: []*StructMember{
			&StructMember{
				Id: "my_field_1",
				Tname: TypeId{
					Tname: structType1.Id,
				},
			},
			&StructMember{
				Id: "my_field_2",
				Tname: TypeId{
					Tname: structType1.Id,
				},
			},
		},
	}
	if err := structType1.compile(&ast); err != nil {
		t.Error(err)
	}
	if err := ast.TypeTable.AddStructType(&structType1); err != nil {
		t.Error(err)
	}
	if err := structType2.compile(&ast); err != nil {
		t.Error(err)
	}
	if err := ast.TypeTable.AddStructType(&structType2); err != nil {
		t.Error(err)
	}

	if err := structType1.IsValidExpression(&ArrayExp{
		Value: []Exp{
			new(NullExp),
			&ArrayExp{
				Value: []Exp{
					&FloatExp{Value: 1.5},
					&IntExp{Value: 4},
				},
			},
		},
	}, nil, &ast); err == nil {
		t.Error("assignment of array to struct is not allowed")
	}
	if err := structType2.IsValidExpression(&MapExp{
		Kind: KindStruct,
		Value: map[string]Exp{
			"my_field_1": &MapExp{
				Kind: KindStruct,
				Value: map[string]Exp{
					"my_field_1": &IntExp{Value: 1},
					"my_field_2": &FloatExp{Value: 4.5},
				},
			},
			"my_field_2": &MapExp{
				Kind: KindStruct,
				Value: map[string]Exp{
					"my_field_1": &IntExp{Value: 2},
					"my_field_2": &FloatExp{Value: 5.5},
				},
			},
		},
	}, nil, &ast); err != nil {
		t.Error(err)
	}
	if err := structType2.IsValidExpression(&MapExp{
		Kind: KindStruct,
		Value: map[string]Exp{
			"my_field_1": &MapExp{
				Kind: KindStruct,
				Value: map[string]Exp{
					"my_field_1": &IntExp{Value: 1},
					"my_field_2": &FloatExp{Value: 4.5},
				},
			},
			"my_field_2": new(NullExp),
		},
	}, nil, &ast); err != nil {
		t.Error(err)
	}

	if err := structType2.IsValidExpression(&MapExp{
		Kind: KindStruct,
		Value: map[string]Exp{
			"my_field_1": &MapExp{
				Kind: KindStruct,
				Value: map[string]Exp{
					"my_field_1": &IntExp{Value: 1},
					"my_field_2": &FloatExp{Value: 4.5},
				},
			},
			"my_field_2": &MapExp{
				Kind: KindMap,
				Value: map[string]Exp{
					"my_field_1": &IntExp{Value: 2},
					"my_field_2": &FloatExp{Value: 5.5},
				},
			},
		},
	}, nil, &ast); err == nil {
		t.Error("use of a map literal for a struct is not allowed")
	}
	if err := structType2.IsValidExpression(&MapExp{
		Kind: KindStruct,
		Value: map[string]Exp{
			"my_field_1": &MapExp{
				Kind: KindStruct,
				Value: map[string]Exp{
					"my_field_1": &IntExp{Value: 1},
					"my_field_2": &FloatExp{Value: 4.5},
				},
			},
			"my_field_2": &MapExp{
				Kind: KindStruct,
				Value: map[string]Exp{
					"my_field_1": &FloatExp{Value: 2.5},
					"my_field_2": &FloatExp{Value: 5.5},
				},
			},
		},
	}, nil, &ast); err == nil {
		t.Error("use of float literal for int field is not allowed.")
	}
	if err := structType2.IsValidExpression(&MapExp{
		Kind: KindStruct,
		Value: map[string]Exp{
			"my_field_1": &MapExp{
				Kind: KindStruct,
				Value: map[string]Exp{
					"my_field_1": &IntExp{Value: 1},
					"my_field_2": &FloatExp{Value: 4.5},
				},
			},
			"my_field_2": &FloatExp{Value: 5.5},
		},
	}, nil, &ast); err == nil {
		t.Error("use of float literal for struct field is not allowed.")
	}
	if err := structType2.IsValidExpression(&MapExp{
		Kind: KindStruct,
		Value: map[string]Exp{
			"my_field_1": &MapExp{
				Kind: KindStruct,
				Value: map[string]Exp{
					"my_field_1": &IntExp{Value: 1},
					"my_field_2": &FloatExp{Value: 4.5},
				},
			},
			"my_field_2": &MapExp{
				Kind: KindStruct,
				Value: map[string]Exp{
					"my_field_1": &FloatExp{Value: 2.5},
					"my_field_2": &FloatExp{Value: 5.5},
				},
			},
			"extra": new(NullExp),
		},
	}, nil, &ast); err == nil {
		t.Error("extra field is not allowed.")
	}
	if err := structType2.IsValidExpression(&MapExp{
		Kind: KindStruct,
		Value: map[string]Exp{
			"my_field_1": &MapExp{
				Kind: KindStruct,
				Value: map[string]Exp{
					"my_field_1": &IntExp{Value: 1},
					"my_field_2": &FloatExp{Value: 4.5},
				},
			},
		},
	}, nil, &ast); err == nil {
		t.Error("missing field is not allowed.")
	}
}

func TestStructTypeRedefinition(t *testing.T) {
	structType1 := StructType{
		Id: "MY_STRUCT_1",
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
	ast := Ast{
		Callables: new(Callables),
		StructTypes: []*StructType{
			&structType1,
			// Check equivalence allowed.
			&StructType{
				Id: "MY_STRUCT_1",
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
			},
		},
	}
	if err := ast.compileTypes(); err != nil {
		t.Error(err)
	}
	checkBad := func(st *StructType, msg string) {
		t.Helper()
		if err := st.compile(&ast); err != nil {
			t.Error(err)
		}
		if err := ast.TypeTable.AddStructType(st); err == nil {
			t.Error(msg)
		}
	}
	checkBad(&StructType{
		Id: "MY_STRUCT_1",
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
					Tname: KindInt,
				},
			},
		},
	}, "differing field types")
	checkBad(&StructType{
		Id: "MY_STRUCT_1",
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
			&StructMember{
				Id: "my_field_3",
				Tname: TypeId{
					Tname: KindFloat,
				},
			},
		},
	}, "extra field")
	checkBad(&StructType{
		Id: "MY_STRUCT_1",
		Members: []*StructMember{
			&StructMember{
				Id: "my_field_1",
				Tname: TypeId{
					Tname:    KindInt,
					ArrayDim: 1,
				},
			},
			&StructMember{
				Id: "my_field_2",
				Tname: TypeId{
					Tname: KindFloat,
				},
			},
		},
	}, "differing array dim")
	checkBad(&StructType{
		Id: "MY_STRUCT_1",
		Members: []*StructMember{
			&StructMember{
				Id: "my_field_1",
				Tname: TypeId{
					Tname:  KindInt,
					MapDim: 1,
				},
			},
			&StructMember{
				Id: "my_field_2",
				Tname: TypeId{
					Tname: KindFloat,
				},
			},
		},
	}, "differing map dim")
	checkBad(&StructType{
		Id: "MY_STRUCT_1",
		Members: []*StructMember{
			&StructMember{
				Id: "my_field_1",
				Tname: TypeId{
					Tname: KindInt,
				},
				OutName: "foo",
			},
			&StructMember{
				Id: "my_field_2",
				Tname: TypeId{
					Tname: KindFloat,
				},
			},
		},
	}, "differing outname")
}

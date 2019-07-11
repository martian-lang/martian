// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package core

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/martian-lang/martian/martian/syntax"
)

func TestArgumentMapValidateInputs(t *testing.T) {
	var def ChunkDef
	if err := json.Unmarshal([]byte(`{
		"__threads": 4,
		"__mem_gb": 3,
		"foo": 12,
		"bar": 1.2,
		"baz": { "fooz": "bars" },
		"bing": [1],
		"bath": "soap"
	}`), &def); err != nil {
		t.Errorf("Unmarshal failure: %v", err)
	}
	plist := []*syntax.InParam{
		{
			Id:    "foo",
			Tname: syntax.TypeId{Tname: syntax.KindInt},
		},
		{
			Id:    "bar",
			Tname: syntax.TypeId{Tname: syntax.KindFloat},
		},
		{
			Id:    "baz",
			Tname: syntax.TypeId{Tname: syntax.KindMap},
		},
		{
			Id: "bing",
			Tname: syntax.TypeId{
				Tname:    syntax.KindInt,
				ArrayDim: 1,
			},
		},
	}
	ptable := make(map[string]*syntax.InParam, len(plist))
	for _, p := range plist {
		ptable[p.GetId()] = p
	}
	params := syntax.InParams{
		Table: ptable,
		List:  plist,
	}
	lookup := syntax.NewTypeLookup()
	if err, msg := def.Args.ValidateInputs(lookup, &params); err == nil {
		t.Errorf("Expected error from extra param, got none.")
	} else if strings.TrimSpace(err.Error()) != "Unexpected parameter 'bath'" {
		t.Errorf(
			"Validation error: expected \""+
				"Unexpected parameter 'bath'"+
				"\", got \"%v\"",
			err)
	} else if msg != "" {
		t.Errorf("Didn't expect a soft error message, got %s", msg)
	}

	bath := &syntax.InParam{
		Id:    "bath",
		Tname: syntax.TypeId{Tname: syntax.KindString},
	}
	params.Table[bath.Id] = bath
	params.List = append(params.List, bath)
	if err, msg := def.Args.ValidateInputs(lookup, &params); err != nil {
		t.Errorf("Validation error: expected success, got %v", err)
	} else if msg != "" {
		t.Errorf("Didn't expect a soft error message, got %s", msg)
	}
	params.Table["bar"].Tname.Tname = syntax.KindInt
	if err, msg := def.Args.ValidateInputs(lookup, &params); err == nil {
		t.Errorf("Expected error from float, got none.")
	} else if e := "Expected int input parameter 'bar' value '1.2' " +
		"cannot be parsed as an integer"; strings.TrimSpace(err.Error()) != e {
		t.Errorf(
			"Validation error: expected\n\""+e+
				"\"\ngot\n\"%v\"",
			err)
	} else if msg != "" {
		t.Errorf("Didn't expect a soft error message, got %s", msg)
	}
	params.Table["bar"].Tname.Tname = syntax.KindFloat
	missing := &syntax.InParam{
		Id:    "miss",
		Tname: syntax.TypeId{Tname: syntax.KindString},
	}
	params.Table[missing.Id] = missing
	params.List = append(params.List, missing)
	if err, msg := def.Args.ValidateInputs(lookup, &params); err == nil {
		t.Errorf("Expected error from missing parameter, got none.")
	} else if strings.TrimSpace(err.Error()) != "Missing input parameter 'miss'" {
		t.Errorf(
			"Validation error: expected \""+
				"Missing input parameter 'miss'"+
				"\", got \"%v\"",
			err)
	} else if msg != "" {
		t.Errorf("Didn't expect a soft error message, got %s", msg)
	}
}

func TestArgumentMapValidateOutputs(t *testing.T) {
	var def ChunkDef
	if err := json.Unmarshal([]byte(`{
		"__threads": 4,
		"__mem_gb": 3,
		"foo": 12,
		"bar": 1.2,
		"baz": { "fooz": "bars" },
		"bing": [1],
		"bath": "soap"
	}`), &def); err != nil {
		t.Errorf("Unmarshal failure: %v", err)
	}
	plist := []*syntax.OutParam{
		{
			StructMember: syntax.StructMember{
				Id:    "foo",
				Tname: syntax.TypeId{Tname: syntax.KindInt},
			},
		},
		{
			StructMember: syntax.StructMember{
				Id:    "bar",
				Tname: syntax.TypeId{Tname: syntax.KindFloat},
			},
		},
		{
			StructMember: syntax.StructMember{
				Id:    "baz",
				Tname: syntax.TypeId{Tname: syntax.KindMap},
			},
		},
		{
			StructMember: syntax.StructMember{
				Id: "bing",
				Tname: syntax.TypeId{
					Tname:    syntax.KindInt,
					ArrayDim: 1,
				},
			},
		},
	}
	ptable := make(map[string]*syntax.OutParam, len(plist))
	for _, p := range plist {
		ptable[p.GetId()] = p
	}
	params := syntax.OutParams{
		Table: ptable,
		List:  plist,
	}

	lookup := syntax.NewTypeLookup()
	if err, alarms := def.Args.ValidateOutputs(lookup, &params); err != nil {
		t.Errorf("Expected pass from extra out param, got %v.",
			err)
	} else if strings.TrimSpace(alarms) != "Unexpected output 'bath'" {
		t.Errorf(
			"Validation error: expected \""+
				"Unexpected output 'bath'"+
				"\", got \"%s\"",
			alarms)
	}
	bath := &syntax.OutParam{
		StructMember: syntax.StructMember{
			Id:    "bath",
			Tname: syntax.TypeId{Tname: syntax.KindString},
		},
	}
	params.Table[bath.Id] = bath
	params.List = append(params.List, bath)
	if err, msg := def.Args.ValidateOutputs(lookup, &params); err != nil {
		t.Errorf("Validation error: expected success, got %v", err)
	} else if msg != "" {
		t.Errorf("Didn't expect a soft error message, got %s", msg)
	}
	params.Table["bar"].Tname.Tname = syntax.KindInt
	if err, msg := def.Args.ValidateOutputs(lookup, &params); err == nil {
		t.Error("Expected error from float, got none.")
		if msg != "" {
			t.Log(msg)
		}
	} else if e := "Expected int output value 'bar' value '1.2' " +
		"cannot be parsed as an integer"; strings.TrimSpace(err.Error()) != e {
		t.Errorf(
			"Validation error: expected\n\""+e+
				"\"\ngot\n\"%v\"",
			err)
	} else if msg != "" {
		t.Errorf("Didn't expect a soft error message, got %s", msg)
	}
	params.Table["bar"].Tname.Tname = syntax.KindFloat
	missing := &syntax.OutParam{
		StructMember: syntax.StructMember{
			Id:    "miss",
			Tname: syntax.TypeId{Tname: syntax.KindString},
		},
	}
	params.Table[missing.Id] = missing
	params.List = append(params.List, missing)
	if err, msg := def.Args.ValidateOutputs(lookup, &params); err == nil {
		t.Errorf("Expected error from missing parameter, got none.")
	} else if strings.TrimSpace(err.Error()) != "Missing output value 'miss'" {
		t.Errorf(
			"Validation error: expected \""+
				"Missing output value 'miss'"+
				"\", got \"%v\"",
			err)
	} else if msg != "" {
		t.Errorf("Didn't expect a soft error message, got %s", msg)
	}
}

func TestUnmarshalMarshallerMap(t *testing.T) {
	sb := []byte(`{
		"__threads": 4,
		"__mem_gb": 3,
		"foo": 12,
		"bar": 1.2,
		"baz": {
			"fooz": "bars"
		},
		"bing": [
			1
		],
		"bzzoot": "thing"
	}`)
	var m1 LazyArgumentMap
	var m2 MarshalerMap
	if err := json.Unmarshal(sb, &m1); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(sb, &m2); err != nil {
		t.Fatal(err)
	}
	if len(m1) != len(m2) {
		t.Errorf("len %d != len %d", len(m1), len(m2))
	}
	for k, v := range m2 {
		if b, ok := v.(json.RawMessage); !ok {
			t.Errorf("unexpected type %T", v)
		} else if !bytes.Equal(m1[k], b) {
			t.Errorf("%s != %s", string([]byte(b)), string([]byte(m1[k])))
		}
	}
	if err := json.Unmarshal(nullBytes, &m2); err != nil {
		t.Error(err)
	} else if m2 != nil {
		t.Error("expected nil")
	}
}

func TestLazyArgumentMapMarshal(t *testing.T) {
	var m LazyArgumentMap
	if b, err := json.Marshal(m); err != nil {
		t.Error(err)
	} else if !bytes.Equal(b, nullBytes) {
		t.Error("expected null")
	}
	m = make(LazyArgumentMap)
	if b, err := json.Marshal(m); err != nil {
		t.Error(err)
	} else if !bytes.Equal(b, []byte("{}")) {
		t.Error("expected null")
	}
	m["foo"] = nil
	if b, err := json.Marshal(m); err != nil {
		t.Error(err)
	} else if !bytes.Equal(b, []byte(`{"foo":null}`)) {
		t.Errorf(`%q != "{\"foo\":null}"`, string(b))
	}
}

func TestMarshalerMapMarshal(t *testing.T) {
	var m MarshalerMap
	if b, err := json.Marshal(m); err != nil {
		t.Error(err)
	} else if !bytes.Equal(b, nullBytes) {
		t.Error("expected null")
	}
	m = make(MarshalerMap)
	if b, err := json.Marshal(m); err != nil {
		t.Error(err)
	} else if !bytes.Equal(b, []byte("{}")) {
		t.Error("expected null")
	}
	m["foo"] = LazyArgumentMap{"bar": nullBytes}
	m["arr"] = nil
	if b, err := json.MarshalIndent(m, "\t", "\t"); err != nil {
		t.Error(err)
	} else if !bytes.Equal(b, []byte(`{
		"arr": null,
		"foo": {
			"bar": null
		}
	}`)) {
		t.Error("unexpected value", string(b))
	}
	if lm, err := m.ToLazyArgumentMap(); err != nil {
		t.Error(err)
	} else if b, err := json.MarshalIndent(lm, "\t", "\t"); err != nil {
		t.Error(err)
	} else if !bytes.Equal(b, []byte(`{
		"arr": null,
		"foo": {
			"bar": null
		}
	}`)) {
		t.Error("unexpected value", string(b))
	}
}

type toyStruct struct {
	Iface  interface{}
	Map    map[string]int
	Int    int
	Float  float64 `json:"n"`
	String string  `json:"s,omitempty"`
	IntP   *int
}

func TestMakeMarshalerMap(t *testing.T) {
	s := toyStruct{
		Int:   5,
		Float: 6,
	}
	m := MakeMarshalerMap(s)
	if len(m) != 5 {
		t.Errorf("Expected 5 elements, got %d", len(m))
	}
	check := func(m MarshalerMap, key string, value interface{}) {
		t.Helper()
		eb, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if v, ok := m[key]; !ok {
			t.Errorf("Missing key %s\t%s", key, string(b))
		} else if vb, err := v.MarshalJSON(); err != nil {
			t.Error(err)
		} else if !bytes.Equal(vb, eb) {
			t.Errorf("Incorrect value for %s: expected %s actual %s",
				key, string(eb), string(vb))
		}
	}
	check(m, "Iface", s.Iface)
	check(m, "Map", s.Map)
	check(m, "Int", s.Int)
	check(m, "n", s.Float)
	check(m, "IntP", s.IntP)
	s.String = "foo"
	m = MakeMarshalerMap(s)
	check(m, "s", s.String)
	m = MakeMarshalerMap(map[string]string{
		"foo": "bar",
	})
	check(m, "foo", "bar")
}

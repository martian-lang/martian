// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package core

import (
	"encoding/json"
	"github.com/martian-lang/martian/martian/syntax"
	"reflect"
	"strings"
	"testing"
)

func TestArgumentMapValidateInputs(t *testing.T) {
	var def LazyChunkDef
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
			Tname: "int",
		},
		{
			Id:    "bar",
			Tname: "float",
		},
		{
			Id:    "baz",
			Tname: "map",
		},
		{
			Id:       "bing",
			Tname:    "int",
			ArrayDim: 1,
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
	if err, msg := def.Args.ValidateInputs(&params); err == nil {
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
		Tname: "string",
	}
	params.Table[bath.Id] = bath
	params.List = append(params.List, bath)
	if err, msg := def.Args.ValidateInputs(&params); err != nil {
		t.Errorf("Validation error: expected success, got %v", err)
	} else if msg != "" {
		t.Errorf("Didn't expect a soft error message, got %s", msg)
	}
	params.Table["bar"].Tname = "int"
	if err, msg := def.Args.ValidateInputs(&params); err == nil {
		t.Errorf("Expected error from float, got none.")
	} else if strings.TrimSpace(err.Error()) !=
		"Expected int input parameter 'bar' with value \"1.2\" cannot be parsed as an integer." {
		t.Errorf(
			"Validation error: expected \""+
				"Expected int input parameter 'bar' with value \"1.2\" cannot be parsed as an integer."+
				"\", got \"%v\"",
			err)
	} else if msg != "" {
		t.Errorf("Didn't expect a soft error message, got %s", msg)
	}
	params.Table["bar"].Tname = "float"
	missing := &syntax.InParam{
		Id:    "miss",
		Tname: "string",
	}
	params.Table[missing.Id] = missing
	params.List = append(params.List, missing)
	if err, msg := def.Args.ValidateInputs(&params); err == nil {
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
	var def LazyChunkDef
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
			Id:    "foo",
			Tname: "int",
		},
		{
			Id:    "bar",
			Tname: "float",
		},
		{
			Id:    "baz",
			Tname: "map",
		},
		{
			Id:       "bing",
			Tname:    "int",
			ArrayDim: 1,
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

	if err, alarms := def.Args.ValidateOutputs(&params); err != nil {
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
		Id:    "bath",
		Tname: "string",
	}
	params.Table[bath.Id] = bath
	params.List = append(params.List, bath)
	if err, msg := def.Args.ValidateOutputs(&params); err != nil {
		t.Errorf("Validation error: expected success, got %v", err)
	} else if msg != "" {
		t.Errorf("Didn't expect a soft error message, got %s", msg)
	}
	params.Table["bar"].Tname = "int"
	if err, msg := def.Args.ValidateOutputs(&params); err == nil {
		t.Errorf("Expected error from float, got none.")
	} else if strings.TrimSpace(err.Error()) !=
		"Expected int output value 'bar' with value \"1.2\" cannot be parsed as an integer." {
		t.Errorf(
			"Validation error: expected \""+
				"Expected int output value 'bar' with value \"1.2\" cannot be parsed as an integer."+
				"\", got \"%v\"",
			err)
	} else if msg != "" {
		t.Errorf("Didn't expect a soft error message, got %s", msg)
	}
	params.Table["bar"].Tname = "float"
	missing := &syntax.OutParam{
		Id:    "miss",
		Tname: "string",
	}
	params.Table[missing.Id] = missing
	params.List = append(params.List, missing)
	if err, msg := def.Args.ValidateOutputs(&params); err == nil {
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

type toyStruct struct {
	Iface  interface{}
	Map    map[string]int
	Int    int
	Float  float64 `json:"n"`
	String string  `json:"s,omitempty"`
	IntP   *int
}

func TestMakeArgumentMap(t *testing.T) {
	s := toyStruct{
		Int:   5,
		Float: 6,
	}
	m := MakeArgumentMap(s)
	if len(m) != 5 {
		t.Errorf("Expected 5 elements, got %d", len(m))
	}
	check := func(m ArgumentMap, key string, value interface{}) {
		t.Helper()
		b, _ := json.Marshal(m)
		if v, ok := m[key]; !ok {
			t.Errorf("Missing key %s\t%s", key, string(b))
		} else if !reflect.DeepEqual(v, value) {
			t.Errorf("Incorrect value for %s: expected %v actual %v\n%s",
				key, value, v, string(b))
		}
	}
	check(m, "Iface", s.Iface)
	check(m, "Map", s.Map)
	check(m, "Int", s.Int)
	check(m, "n", s.Float)
	check(m, "IntP", s.IntP)
	s.String = "foo"
	m = MakeArgumentMap(s)
	check(m, "s", s.String)
	m = MakeArgumentMap(map[string]string{
		"foo": "bar",
	})
	check(m, "foo", "bar")
}

func TestArgumentMapDecode(t *testing.T) {
	check := func(expected ArgumentMap, actual interface{}) {
		t.Helper()
		if err := expected.Decode(actual); err != nil {
			t.Errorf("Error decoding: %v", err)
		}
	}
	s := toyStruct{}
	check(ArgumentMap{
		"Iface": map[string]string{"foo": "bar"},
		"s":     "baz",
	}, &s)
	if (s.Iface.(map[string]string))["foo"] != "bar" {
		t.Errorf("Incorrect foo in iface: %v", (s.Iface.(map[string]interface{}))["foo"])
	}
	if s.String != "baz" {
		t.Errorf("Incorrect String: %s", s.String)
	}
	checkMap := func(expected ArgumentMap, actual interface{}) {
		t.Helper()
		check(expected, actual)
		if be, err := json.Marshal(expected); err != nil {
			t.Errorf("Error encoding: %v", err)
		} else if ba, err := json.Marshal(actual); err != nil {
			t.Errorf("Error encoding: %v", err)
		} else if string(be) != string(ba) {
			t.Errorf("Incorrect decode: expected %s got %s", string(be), string(ba))
		}
	}
	checkMap(ArgumentMap{
		"foo": "bar",
	}, make(map[string]string))
	checkMap(ArgumentMap{
		"foo": 1,
	}, make(map[string]int))
	checkMap(ArgumentMap{
		"foo": 1,
	}, make(map[string]interface{}))
	m := ArgumentMap{
		"foo": "bar",
	}
	if err := m.Decode(make(map[string]int)); err == nil {
		t.Errorf("Expected error.")
	}
}

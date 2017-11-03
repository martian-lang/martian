// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package core

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestChunkDefMarshal(t *testing.T) {
	if b, err := json.Marshal(&ChunkDef{
		Resources: &JobResources{
			Threads: 3,
			MemGB:   2,
		},
		Args: ArgumentMap{
			"foo":  12,
			"bar":  1.2,
			"baz":  map[string]interface{}{"fooz": "bars"},
			"bath": "soap",
		},
	}); err != nil {
		t.Errorf("Marshaling failure %v", err)
	} else {
		def := make(map[string]interface{}, 6)
		if err := json.Unmarshal(b, &def); err != nil {
			t.Errorf("Unmarshaling failure %v", err)
		} else if len(def) != 6 {
			t.Errorf("Incorrect number of json keys: expected 6, got %d", len(def))
		} else {
			if v, ok := def["__threads"].(float64); !ok || v != 3.0 {
				t.Errorf("Incorrect threads: expected 3, got %v", def["__threads"])
			}
			if v, ok := def["__mem_gb"].(float64); !ok || v != 2.0 {
				t.Errorf("Incorrect mem_gb: expected 2, got %v", def["__mem_gb"])
			}
			if v, ok := def["foo"].(float64); !ok || v != 12.0 {
				t.Errorf("Incorrect foo: expected 12, got %v", def["foo"])
			}
			if v, ok := def["bar"].(float64); !ok || v != 1.2 {
				t.Errorf("Incorrect foo: expected 1.2, got %v", def["bar"])
			}
			if v, ok := def["baz"].(map[string]interface{}); !ok {
				t.Errorf("Incorrect foo: expected map, got %v", def["baz"])
			} else if s, ok := v["fooz"].(string); !ok || s != "bars" {
				t.Errorf("Incorrect foo: expected {\"fooz\":\"bars\"}, got %v", v)
			}
			if v, ok := def["bath"].(string); !ok || v != "soap" {
				t.Errorf("Incorrect foo: expected soap, got %v", def["bath"])
			}
		}
	}
}

func TestChunkDefUnmarshal(t *testing.T) {
	var def ChunkDef
	if err := json.Unmarshal([]byte(`{
		"__threads": 4,
		"__mem_gb": 3,
		"foo": 12,
		"bar": 1.2,
		"baz": { "fooz": "bars" },
		"bath": "soap"
	}`), &def); err != nil {
		t.Errorf("Unmarshal failure: %v", err)
	}
	if def.Resources == nil {
		t.Errorf("Expected resources, got nil.")
	} else {
		if def.Resources.Threads != 4 {
			t.Errorf("Incorrect threads: expected 3, got %d", def.Resources.Threads)
		}
		if def.Resources.MemGB != 3 {
			t.Errorf("Incorrect mem_gb: expected 3, got %d", def.Resources.MemGB)
		}
	}
	if len(def.Args) != 4 {
		t.Errorf("Incorrect number of args: expected 4, got %d", len(def.Args))
	}
	if v, ok := def.Args["foo"].(json.Number); !ok {
		t.Errorf("Incorrect type for foo: expected number 12, got %v", reflect.TypeOf(def.Args["foo"]))
	} else if n, err := v.Int64(); err != nil {
		t.Errorf("Error reading foo as int: %v", err)
	} else if n != 12 {
		t.Errorf("Incorrect foo: expected 12, got %d", n)
	}
	if v, ok := def.Args["bar"].(json.Number); !ok {
		t.Errorf("Incorrect type for bar: expected number 1.2, got %v", reflect.TypeOf(def.Args["bar"]))
	} else if n, err := v.Float64(); err != nil {
		t.Errorf("Error reading bar as float: %v", err)
	} else if n != 1.2 {
		t.Errorf("Incorrect bar: expected 1.2, got %f", n)
	}
	if v, ok := def.Args["baz"].(map[string]interface{}); !ok {
		t.Errorf("Incorrect foo: expected map, got %v", def.Args["baz"])
	} else if s, ok := v["fooz"].(string); !ok || s != "bars" {
		t.Errorf("Incorrect foo: expected {\"fooz\":\"bars\"}, got %v", v)
	}
	if v, ok := def.Args["bath"].(string); !ok || v != "soap" {
		t.Errorf("Incorrect foo: expected soap, got %v", def.Args["bath"])
	}
}

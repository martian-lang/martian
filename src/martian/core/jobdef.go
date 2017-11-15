//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//

// Data structure for keeping track of job resources and arguments.

package core

import (
	"encoding/json"
	"fmt"
	"martian/syntax"
	"martian/util"
)

// Defines resources used by a stage.
type JobResources struct {
	Threads int    `json:"__threads,omitempty"`
	MemGB   int    `json:"__mem_gb,omitempty"`
	Special string `json:"__special,omitempty"`
}

func (self *JobResources) updateFromArgs(args ArgumentMap) error {
	if args == nil {
		return nil
	}
	getInt := func(v interface{}, key string) (int, error) {
		switch n := v.(type) {
		case json.Number:
			if i, err := n.Int64(); err == nil {
				return int(i), nil
			} else if level := syntax.GetEnforcementLevel(); level == syntax.EnforceError {
				return int(i), err
			} else if f, err := n.Float64(); err != nil {
				return 0, err
			} else {
				if level == syntax.EnforceLog {
					util.LogInfo("runtime",
						"WARNING: value %v for %s was not of integer type",
						n, key)
				} else if level == syntax.EnforceAlarm {
					util.PrintInfo("runtime",
						"WARNING: value %v for %s was not of integer type",
						n, key)
				}
				return int(f), nil
			}
		case float64:
			if n != float64(int(n)) {
				return int(n), fmt.Errorf("%f is not an integer", n)
			} else {
				return int(n), nil
			}
		case int64:
			return int(n), nil
		case int:
			return n, nil
		default:
			return 0, fmt.Errorf("Expected integer for %s, found %v instead",
				key, v)
		}
	}
	if v, ok := args["__threads"]; ok {
		if n, err := getInt(v, "__threads"); err != nil {
			return err
		} else {
			self.Threads = n
		}
		delete(args, "__threads")
	}
	if v, ok := args["__mem_gb"]; ok {
		if n, err := getInt(v, "__mem_gb"); err != nil {
			return err
		} else {
			self.MemGB = n
		}
		delete(args, "__mem_gb")
	}
	if v, ok := args["__special"]; ok {
		if s, ok := v.(string); !ok {
			return fmt.Errorf("Expected string for __special, found %v instead", v)
		} else {
			self.Special = s
		}
		delete(args, "__special")
	}
	return nil
}

// Defines the resources and arguments of a chunk.
type ChunkDef struct {
	Resources *JobResources
	Args      ArgumentMap
}

func (self *ChunkDef) MergeArguments(bindings ArgumentMap) *ChunkDef {
	if bindings == nil || len(bindings) == 0 {
		return self
	}
	if self.Args == nil || len(self.Args) == 0 {
		return &ChunkDef{
			Resources: self.Resources,
			Args:      bindings,
		}
	} else {
		def := ChunkDef{
			Resources: self.Resources,
			Args:      make(ArgumentMap),
		}
		for key, value := range bindings {
			def.Args[key] = value
		}
		for key, value := range self.Args {
			def.Args[key] = value
		}
		return &def
	}
}

func (self *ChunkDef) Merge(bindings interface{}) *ChunkDef {
	if bindings == nil {
		return self
	}
	switch bindings := bindings.(type) {
	case map[string]interface{}:
		return self.Merge(ArgumentMap(bindings))
	case ArgumentMap:
		return self.MergeArguments(bindings)
	default:
		// Cross-serialize as if it were a map.
		return self.MergeArguments(MakeArgumentMap(bindings))
	}
}

func (self *ChunkDef) UnmarshalJSON(b []byte) error {
	args := self.Args
	if args == nil {
		args = make(ArgumentMap)
	}
	if err := json.Unmarshal(b, &args); err != nil {
		return err
	}
	self.Args = args
	if self.Resources != nil {
		return self.Resources.updateFromArgs(self.Args)
	} else {
		var res JobResources
		if err := res.updateFromArgs(self.Args); err != nil {
			return err
		}
		if res.Threads != 0 || res.MemGB != 0 || res.Special != "" {
			self.Resources = &res
		}
	}
	return nil
}

func (self *ChunkDef) MarshalJSON() ([]byte, error) {
	if self.Resources == nil {
		if self.Args == nil {
			return []byte("{}"), nil
		}
		return json.Marshal(self.Args)
	}
	args := make(map[string]interface{}, len(self.Args)+3)
	if b, err := json.Marshal(self.Resources); err != nil {
		return nil, err
	} else if err := json.Unmarshal(b, &args); err != nil {
		return nil, err
	}
	for k, v := range self.Args {
		args[k] = v
	}
	return json.Marshal(args)
}

type StageDefs struct {
	ChunkDefs []*ChunkDef   `json:"chunks"`
	JoinDef   *JobResources `json:"join,omitempty"`
}

func (self *StageDefs) UnmarshalJSON(b []byte) error {
	type stageDefsWeak struct {
		ChunkDefs []*ChunkDef `json:"chunks"`
		JoinDef   ArgumentMap `json:"join,omitempty"`
	}
	into := stageDefsWeak{
		ChunkDefs: self.ChunkDefs,
		JoinDef:   make(ArgumentMap),
	}
	if err := json.Unmarshal(b, &into); err != nil {
		return err
	}
	self.ChunkDefs = into.ChunkDefs
	if into.JoinDef != nil && len(into.JoinDef) > 0 {
		if self.JoinDef == nil {
			self.JoinDef = &JobResources{}
		}
		if err := self.JoinDef.updateFromArgs(into.JoinDef); err != nil {
			return err
		}
		if len(into.JoinDef) != 0 {
			return fmt.Errorf("Invalid parameter in join definition.")
		}
	}
	return nil
}

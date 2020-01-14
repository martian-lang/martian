//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

// Data structure for keeping track of job resources and arguments.

package core

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

// Defines resources used by a stage.
type JobResources struct {
	Threads int    `json:"__threads,omitempty"`
	MemGB   int    `json:"__mem_gb,omitempty"`
	VMemGB  int    `json:"__vmem_gb,omitempty"`
	Special string `json:"__special,omitempty"`
}

func (self *JobResources) ToLazyMap() LazyArgumentMap {
	r := make(LazyArgumentMap, 3)
	if self.Threads != 0 {
		r["__threads"] = json.RawMessage(strconv.Itoa(self.Threads))
	}
	if self.MemGB != 0 {
		r["__mem_gb"] = json.RawMessage(strconv.Itoa(self.MemGB))
	}
	if self.VMemGB != 0 {
		r["__vmem_gb"] = json.RawMessage(strconv.Itoa(self.VMemGB))
	}
	if self.Special != "" {
		r["__special"], _ = json.Marshal(self.Special)
	}
	return r
}

func (self *JobResources) updateFromLazyArgs(args LazyArgumentMap) error {
	if args == nil {
		return nil
	}
	getInt := func(v json.RawMessage, key string) (int, error) {
		var result int
		if err := json.Unmarshal(v, &result); err == nil {
			return result, nil
		} else if level := syntax.GetEnforcementLevel(); level == syntax.EnforceError {
			return result, err
		} else {
			var resultFloat float64
			if json.Unmarshal(v, &resultFloat) != nil {
				return result, err
			} else if level == syntax.EnforceLog {
				util.LogInfo("runtime",
					"WARNING: value %q for %s was not of integer type",
					v, key)
			} else if level == syntax.EnforceAlarm {
				util.PrintInfo("runtime",
					"WARNING: value %q for %s was not of integer type",
					v, key)
			}
			return int(resultFloat), nil
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
	if v, ok := args["__vmem_gb"]; ok {
		if n, err := getInt(v, "__vmem_gb"); err != nil {
			return err
		} else {
			self.VMemGB = n
		}
		delete(args, "__vmem_gb")
	}
	if v, ok := args["__special"]; ok {
		var s string
		if json.Unmarshal(v, &s) != nil {
			return fmt.Errorf("Expected string for __special, found %v instead", v)
		} else {
			self.Special = s
		}
		delete(args, "__special")
	}
	return nil

}

func (self *ChunkDef) mergeFromMarshaler(bindings MarshalerMap) *ChunkDef {
	if bindings == nil || len(bindings) == 0 {
		return self
	}
	def := ChunkDef{
		Resources: self.Resources,
		Args: make(LazyArgumentMap, func(a LazyArgumentMap, b int) int {
			if len(a) < b {
				return b
			} else {
				return len(a)
			}
		}(self.Args, len(bindings))),
	}
	for key, value := range bindings {
		if value == nil {
			def.Args[key] = nullBytes
		} else if b, ok := value.(json.RawMessage); ok {
			def.Args[key] = b
		} else if b, err := value.MarshalJSON(); err == nil {
			def.Args[key] = b
		} else {
			util.LogError(err, "runtime", "Error serializing bindings")
		}
	}
	for key, value := range self.Args {
		def.Args[key] = value
	}
	return &def
}

func (self *ChunkDef) mergeEagerArguments(bindings map[string]interface{}) *ChunkDef {
	if len(bindings) == 0 {
		return self
	}
	def := ChunkDef{
		Resources: self.Resources,
		Args: make(LazyArgumentMap, func(a LazyArgumentMap, b int) int {
			if a == nil || len(a) < b {
				return b
			} else {
				return len(a)
			}
		}(self.Args, len(bindings))),
	}
	for key, value := range bindings {
		if value == nil {
			def.Args[key] = nullBytes
		} else if b, ok := value.(json.RawMessage); ok {
			def.Args[key] = b
		} else if b, err := json.Marshal(value); err == nil {
			def.Args[key] = b
		} else {
			util.LogError(err, "runtime", "Error serializing bindings")
		}
	}
	for key, value := range self.Args {
		def.Args[key] = value
	}
	return &def
}

func (self *ChunkDef) UnmarshalJSON(b []byte) error {
	args := self.Args
	if args == nil {
		args = make(LazyArgumentMap)
	}
	if err := json.Unmarshal(b, &args); err != nil {
		return err
	}
	self.Args = args
	if self.Resources != nil {
		return self.Resources.updateFromLazyArgs(self.Args)
	} else {
		var res JobResources
		if err := res.updateFromLazyArgs(self.Args); err != nil {
			return err
		}
		if res.Threads != 0 || res.MemGB != 0 || res.VMemGB != 0 || res.Special != "" {
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
	args := self.Resources.ToLazyMap()
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
		ChunkDefs []*ChunkDef     `json:"chunks"`
		JoinDef   LazyArgumentMap `json:"join,omitempty"`
	}
	into := stageDefsWeak{
		ChunkDefs: self.ChunkDefs,
		JoinDef:   make(LazyArgumentMap),
	}
	if err := json.Unmarshal(b, &into); err != nil {
		return err
	}
	self.ChunkDefs = into.ChunkDefs
	if into.JoinDef != nil && len(into.JoinDef) > 0 {
		if self.JoinDef == nil {
			self.JoinDef = &JobResources{}
		}
		if err := self.JoinDef.updateFromLazyArgs(into.JoinDef); err != nil {
			return err
		}
		if len(into.JoinDef) != 0 {
			return fmt.Errorf("Invalid parameter in join definition.")
		}
	}
	return nil
}

func (self *ChunkDef) Merge(bindings interface{}) *ChunkDef {
	if bindings == nil {
		return self
	}
	switch bindings := bindings.(type) {
	case LazyArgumentMap:
		return self.MergeArguments(bindings)
	case map[string]interface{}:
		return self.mergeEagerArguments(bindings)
	case MarshalerMap:
		return self.mergeFromMarshaler(bindings)
	default:
		// Cross-serialize as if it were a map.
		return self.Merge(MakeMarshalerMap(bindings))
	}
}

// Defines the resources and arguments of a chunk.
type ChunkDef struct {
	Resources *JobResources
	Args      LazyArgumentMap
}

func (self *ChunkDef) MergeArguments(bindings LazyArgumentMap) *ChunkDef {
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
			Args:      make(LazyArgumentMap),
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

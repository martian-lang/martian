//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//

// Data structure for keeping track of job resources and arguments.

package core

import (
	"encoding/json"
)

// Defines resources used by a stage.
type JobResources struct {
	Threads int    `json:"__threads,omitempty"`
	MemGB   int    `json:"__mem_gb,omitempty"`
	Special string `json:"__special,omitempty"`
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
	} else {
		self.Args = args
	}
	res := false
	if _, ok := self.Args["__threads"]; ok {
		delete(self.Args, "__threads")
		res = true
	}
	if _, ok := self.Args["__mem_gb"]; ok {
		delete(self.Args, "__mem_gb")
		res = true
	}
	if _, ok := self.Args["__special"]; ok {
		delete(self.Args, "__special")
		res = true
	}
	if res {
		if self.Resources == nil {
			self.Resources = &JobResources{}
		}
		return json.Unmarshal(b, self.Resources)
	} else {
		return nil
	}
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

// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

/*
 * This implements a simple mechanism for per-stage property overrides in mrp.
 *
 * An overrides file might look like:
 * {
 *      "FULLY.QUALIFIED.STAGE.NAME": {
 *          "chunk.mem_gb": 17
 *        	"force_volatile: false,
 *  	},
 *      "FULLY.QUALIFIED": {
 *		    "mem_gb": 2,
 *		    "force_volatile" : true,
 * 	    },
 *	     "" : {
 *		    "force_volatile": false
 *      }
 * }
 *
 * This file sets the volatile flag to false for all stages. Except any substages of FULLY.QUALIFIED
 * (for which it is true) except for FULLY_QUALIFIED.STAGE.NAME for which it is false again.
 *
 */

package core

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/martian-lang/martian/martian/util"
)

// Compute a "partially" Qualified stage name. This is a fully qualified name
// (ID.pipestance.pipe.pipe.pipe.....stage) with the initial ID and pipestance
// trimmed off. This allows for comparisons between different pipestances with
// the same (or similar) shapes.
func partiallyQualifiedName(n string) string {
	for count := 0; count < 2 && len(n) > 0; n = n[1:] {
		if n[0] == '.' {
			count++
		}
	}
	return n
}

// StageOverride describes the runtime parameters for a stage which
// can be overridden.
type StageOverride struct {
	ForceVolatile *bool `json:"force_volatile,omitempty"`

	JoinThreads *float64     `json:"join.threads,omitempty"`
	JoinMem     *float64     `json:"join.mem_gb,omitempty"`
	JoinVMem    *float64     `json:"join.vmem_gb,omitempty"`
	JoinProfile *ProfileMode `json:"join.profile,omitempty"`

	ChunkThreads *float64     `json:"chunk.threads,omitempty"`
	ChunkMem     *float64     `json:"chunk.mem_gb,omitempty"`
	ChunkVMem    *float64     `json:"chunk.vmem_gb,omitempty"`
	ChunkProfile *ProfileMode `json:"chunk.profile,omitempty"`

	SplitThreads *float64     `json:"split.threads,omitempty"`
	SplitMem     *float64     `json:"split.mem_gb,omitempty"`
	SplitVMem    *float64     `json:"split.vmem_gb,omitempty"`
	SplitProfile *ProfileMode `json:"split.profile,omitempty"`
}

type PipestanceOverrides struct {
	filename         string
	overridesbystage map[string]*StageOverride
}

// Read the overrides file and produce a pipestance overrides object.
func ReadOverrides(path string) (*PipestanceOverrides, error) {
	pse := new(PipestanceOverrides)
	return pse, pse.Set(path)
}

func (*PipestanceOverrides) Type() string {
	return "pipestance overrides json"
}

// String returns the filename which was read by this overrides file.
func (pse *PipestanceOverrides) String() string {
	return pse.filename
}

// This is an alias for ReadFile, to  implement the flag.Value interface.
func (pse *PipestanceOverrides) Set(path string) error {
	return pse.ReadFile(path)
}

// The Set method loads and validates pipestance overrides from a file.
func (pse *PipestanceOverrides) ReadFile(path string) error {
	pse.filename = path

	if path == "" {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	err = dec.Decode(&pse.overridesbystage)

	if err != nil {
		return fmt.Errorf("decoding overrides content: %w", err)
	}

	util.Println("Loaded %v overrides from %v", len(pse.overridesbystage), path)
	return nil
}

func getParent(n string) string {
	for len(n) > 0 {
		if n[len(n)-1] == '.' {
			return n[:len(n)-1]
		}
		n = n[:len(n)-1]
	}
	return ""
}

// Compute the value to use for a stage's volatility, which might be overridden.
//
// node is the fully qualified node name
//
// def  is the default value to use if the value is not overridden
func (pse *PipestanceOverrides) GetForceVolatile(node string, def bool) bool {
	pqn := partiallyQualifiedName(node)
	for pqn != "" {
		so := pse.overridesbystage[pqn]
		if so == nil || so.ForceVolatile == nil {
			pqn = getParent(pqn)
		} else {
			util.LogInfo("overide", "At [force_volatile:%v] replace %v with %v",
				pqn, def, *so.ForceVolatile)
			return *so.ForceVolatile
		}
	}
	/* We didn't find any parent of node that existed and defined the key we're looking
	 * for. Give and use the default value.
	 */
	return def
}

// GetResources applies any resource overrides for the given node/phase to
// the given resource object.
func (pse *PipestanceOverrides) GetResources(node string, phase string, res *JobResources) {
	pqn := partiallyQualifiedName(node)
	res.Threads = pse.getThreads(pqn, phase, res.Threads)
	res.MemGB = pse.getMem(pqn, phase, res.MemGB)
	res.VMemGB = pse.getVMem(pqn, phase, res.VMemGB)
}

// Compute the value to use for a stage's thread reservation, which might be
// overridden.
//
// pqn is the partially qualified node name
//
// def  is the default value to use if the value is not overridden
func (pse *PipestanceOverrides) getThreads(pqn string, phase string, def int) int {
	for pqn != "" {
		val := pse.overridesbystage[pqn].GetThreads(phase)
		if val == nil {
			pqn = getParent(pqn)
		} else {
			util.LogInfo("overide", "At [%s.threads:%s] replace %d with %d",
				phase, pqn, def, int(*val))
			return int(*val)
		}
	}
	// We didn't find any parent of node that existed and defined the key we're looking
	// for. Give and use the default value.
	return def
}

// Compute the value to use for a stage's memory reservation, which might be
// overridden.
//
// pqn is the partially qualified node name
//
// def  is the default value to use if the value is not overridden
func (pse *PipestanceOverrides) getMem(pqn string, phase string, def int) int {
	for pqn != "" {
		val := pse.overridesbystage[pqn].GetMem(phase)
		if val == nil {
			pqn = getParent(pqn)
		} else {
			util.LogInfo("overide", "At [%s.mem_gb:%s] replace %d with %d",
				phase, pqn, def, int(*val))
			return int(*val)
		}
	}
	// We didn't find any parent of node that existed and defined the key we're looking
	// for. Give and use the default value.
	return def
}

// Compute the value to use for a stage's memory reservation, which might be
// overridden.
//
// pqn is the partially qualified node name
//
// def  is the default value to use if the value is not overridden
func (pse *PipestanceOverrides) getVMem(pqn string, phase string, def int) int {
	for pqn != "" {
		val := pse.overridesbystage[pqn].GetVMem(phase)
		if val == nil {
			pqn = getParent(pqn)
		} else {
			util.LogInfo("overide", "At [%s.vmem_gb:%s] replace %d with %d",
				phase, pqn, def, int(*val))
			return int(*val)
		}
	}
	// We didn't find any parent of node that existed and defined the key we're looking
	// for. Give and use the default value.
	return def
}

// Compute the value to use for a stage's profile mode, which might be
// overridden.
//
// |node| is the fully-qualified node name
//
// |def|  is the default value to use if the value is not overridden
func (pse *PipestanceOverrides) GetProfile(node string, phase string, def ProfileMode) ProfileMode {
	pqn := partiallyQualifiedName(node)
	for pqn != "" {
		val := pse.overridesbystage[pqn].GetProfile(phase)
		if val == nil {
			pqn = getParent(pqn)
		} else {
			util.LogInfo("overide", "At [%s.profile:%s] replace %s with %s",
				phase, pqn, def, *val)
			return *val
		}
	}
	// We didn't find any parent of node that existed and defined the key we're looking
	// for. Give and use the default value.
	return def
}

func (so *StageOverride) GetThreads(phase string) *float64 {
	if so == nil {
		return nil
	}
	switch phase {
	case STAGE_TYPE_SPLIT:
		return so.SplitThreads
	case STAGE_TYPE_CHUNK:
		return so.ChunkThreads
	case STAGE_TYPE_JOIN:
		return so.JoinThreads
	default:
		panic("invalid phase " + phase)
	}
}

func (so *StageOverride) GetMem(phase string) *float64 {
	if so == nil {
		return nil
	}
	switch phase {
	case STAGE_TYPE_SPLIT:
		return so.SplitMem
	case STAGE_TYPE_CHUNK:
		return so.ChunkMem
	case STAGE_TYPE_JOIN:
		return so.JoinMem
	default:
		panic("invalid phase " + phase)
	}
}

func (so *StageOverride) GetVMem(phase string) *float64 {
	if so == nil {
		return nil
	}
	switch phase {
	case STAGE_TYPE_SPLIT:
		return so.SplitVMem
	case STAGE_TYPE_CHUNK:
		return so.ChunkVMem
	case STAGE_TYPE_JOIN:
		return so.JoinVMem
	default:
		panic("invalid phase " + phase)
	}
}

func (so *StageOverride) GetProfile(phase string) *ProfileMode {
	if so == nil {
		return nil
	}
	switch phase {
	case STAGE_TYPE_SPLIT:
		return so.SplitProfile
	case STAGE_TYPE_CHUNK:
		return so.ChunkProfile
	case STAGE_TYPE_JOIN:
		return so.JoinProfile
	default:
		panic("invalid phase " + phase)
	}
}

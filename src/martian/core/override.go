// Copyright (c) 2016 10X Genomics, Inc. All rights reserved.

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
	"io/ioutil"
	"martian/util"
	"reflect"
)

/*
 * Compute a "partially" Qualified stage name. This is a fully qualified name
 * (ID.pipestance.pipe.pipe.pipe.....stage) with the initial ID and pipestance
 * trimmed off. This allows for comparisons between different pipestances with
 * the same (or similar) shapes.
 */
func partiallyQualifiedName(n string) string {
	count := 0
	for i := 0; i < len(n); i++ {
		if n[i] == '.' {
			count++
		}
		if count == 2 {
			return n[i+1:]
		}
	}
	return ""
}

type StageOverride map[string]interface{}

type PipestanceOverrides struct {
	overridesbystage map[string]StageOverride
}

/*
 * What are the expected types for elements in a stageoverride map. Note that
 * all JSON numeric types look like Float64s when we stick them in an interface.
 */
var LegalOverrideTypes map[string]reflect.Kind = map[string]reflect.Kind{
	"force_volatile": reflect.Bool,
	"join.threads":   reflect.Float64,
	"join.mem_gb":    reflect.Float64,
	"chunk.threads":  reflect.Float64,
	"chunk.mem_gb":   reflect.Float64,
	"split.threads":  reflect.Float64,
	"split.mem_gb":   reflect.Float64,
}

/*
 * Read the overrides file and produce a pipestance overrides object.
 */
func ReadOverrides(path string) (*PipestanceOverrides, error) {

	pse := new(PipestanceOverrides)

	pse.overridesbystage = make(map[string]StageOverride)

	if path == "" {
		return pse, nil
	}

	fdata, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(fdata, &(pse.overridesbystage))

	if err != nil {
		return nil, err
	}

	/*
	 * Validate semantic correctness of the overrides we just loaded.  We
	 * want to do this here so that future calls to GetOverride can safely
	 * cast the return value to the expected type. We'll catch any type
	 * errors here and prevent mrp from starting. We also want to keep the
	 * overrides untypes so that GetOverride can operate as a generic
	 * funciton for all possible types.
	 */

	for _, stage_override_data := range pse.overridesbystage {
		for override_key, data := range stage_override_data {

			val_kind, ok := LegalOverrideTypes[override_key]

			/* Can't refer to an unspecified override key */
			if !ok {
				return nil, fmt.Errorf("%v is not a legal override", override_key)
			}

			/* Overrides have to match to expected type */
			if reflect.ValueOf(data).Kind() != val_kind {
				return nil, fmt.Errorf("%v (%v) is the wrong type. Expected type is %v", override_key, data, val_kind)
			}
		}
	}

	util.Println("Loaded %v overrides from %v", len(pse.overridesbystage), path)
	return pse, nil
}

func getParent(node *Node) *Node {
	p := node.parent
	if p == nil {
		return nil
	} else {
		return p.getNode()
	}
}

/*
 * Compute the value to use for a stage option when that value might be overrided.
 * |node| is the Node object for the stage
 * |what| is the name of the override we're considering
 * |def|  is the default value to use if the value is not overridded
 */
func (self *PipestanceOverrides) GetOverride(node *Node, what string, def interface{}) interface{} {

	var so StageOverride

	/* Recursively search this node and its parents for a match. Use the most
	 * closely matching node.  Here the root node is represented by the empty string.
	 */
	for cur := node; cur != nil; cur = getParent(cur) {
		var exists bool
		pqn := partiallyQualifiedName(cur.fqname)
		so, exists = self.overridesbystage[pqn]
		if exists {
			val := so[what]
			if val != nil {
				/* If we found a node that exists *AND* it actually defines val,
				 * use it. Otherwise, backtrack another level and try again.
				 */
				util.LogInfo("override", "At [%v:%v] replace %v with %v",
					what, cur.fqname, def, val)
				return val
			}
		}
	}

	/* We didn't find any parent of node that existed and defined the key we're looking
	 * for. Give and use the default value.
	 */
	return def
}

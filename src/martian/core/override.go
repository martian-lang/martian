// Copyright (c) 2016 10X Genomics, Inc. All rights reserved.

/*
 * This implements a simple mechanism for per-stage property overrides in mrp.
 *
 * An overrides file might look like:
 * {
 *  	"FULLY.QUALIFIED.STAGE.NAME": {
 *        	"mem_gb": 17
 *        	"force_volatile: false,
 *  	},
 *	"FULLY.QUALIFIED": {
 *		"mem_gb": 2,
 *		"force_volatile" : true,
 *
 * 	},
 *	"" :{
 *		"force_volatile": false,
 *	}
 *
 * This file sets the volatile flag to false for all stages. Except any substages of FULLY.QUALIFIED
 * (for which it is true) except for FULLY_QUALIFIED.STAGE.NAME for which it is false again.
 *
 */

package core

import (
	"encoding/json"
	"io/ioutil"
)

type StageOverride map[string]interface{}

type PipestanceOverrides struct {
	overridesbystage map[string]StageOverride
}

/*
 * Read the overrides file and produce a pipestance overrides object.
 */
func ReadOverrides(path string) (*PipestanceOverrides, error) {

	pse := new(PipestanceOverrides)

	pse.overridesbystage = make(map[string]StageOverride)

	if path == "" {
		Println("NOPENOPENOPE")
		return pse, nil
	}

	fdata, err := ioutil.ReadFile(path)
	if err != nil {
		Println("UHOH: %v", err)
		return nil, err
	}

	err = json.Unmarshal(fdata, &(pse.overridesbystage))

	if err != nil {
		Println("UHOH: %v", err)
	}

	Println("Loaded %v overrides from %v", len(pse.overridesbystage), path)
	return pse, nil
}

func getparent(node *Node) *Node {

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

	/* Is this sane??? */
	if self == nil {
		return def
	}

	var so StageOverride

	/* Recursively search this node and its parents for a match. Use the most
	 * closely matching node.  Here the root node is represented by the empty string.
	 */
	for cur := node; cur != nil; cur = getparent(cur) {
		var exists bool
		pqn := partiallyQualifiedName(cur.fqname)
		so, exists = self.overridesbystage[pqn]
		if exists {
			val := so[what]
			if val != nil {
				/* If we found a node that exists *AND* it actually defines val,
				 * use it. Otherwise, backtrack another level and try again.
				 */
				Println("GETOVERRIDE[%v@%v]: replace %v with %v", what, node.fqname, def, val)
				return val
			}
		}
	}

	/* We didn't find any parent of node that existed and defined the key we're looking
	 * for. Give and use the default value.
	 */
	return def
}

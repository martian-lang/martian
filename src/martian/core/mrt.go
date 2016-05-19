// Copyright (c) 2016 10X Genomics, Inc. All rights reserved.

package core

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
        "fmt"
)

/*
 * PipestanceSetup defines the parameters we need to start a pipestance.
 * It encapsulates the argument to InvokePipstance and friends
 */
type PipestanceSetup struct {
	Srcpath        string   // Path to the mro invocation file
	Psid           string   // pipestance ID
	PipestancePath string   // Path to put this pipestance
	MroPaths       []string // Where to look for MROs
	MroVersion     string
	Envs           map[string]string
}

/*
 * This takes two pipestances and creates a map that associates nodes in
 * one pipestance with the nodes in the other. Nodes are associated if
 * they have the same name.
 */
func MapTwoPipestances(newp *Pipestance, oldp *Pipestance) map[*Node]*Node {

	m := make(map[*Node]*Node)
	mapR(newp.node, oldp.node, m)
	return m
}

/*
 * Helper function used by MapTwoPipestances that does the recursive enumeration
 * and mapping.
 */
func mapR(curnode *Node, oldRoot *Node, m map[*Node]*Node) {

	if curnode != nil {
		var oldNode *Node
		/*
		 * Find the name of |curnode| in |oldRoot| and assign it.
		 */
		oldRoot.FindNodeByName(partiallyQualifiedName(curnode.fqname), &oldNode)
		m[curnode] = oldNode

		/*
		 * Iterate over any subnodes (i.e. pipelines or stages) in this
		 * node.
		 */
		for _, subs := range curnode.subnodes {
			mapR(subs.getNode(), oldRoot, m)
		}
	}
}

/* This buids a set of symlinks from one pipestance to another. All of the nonblacklisted
 * stages (and sub-pipelines) that have a corresponding node will be linked.  We try to
 * link entire sub-pipelines when possible.
 */

func LinkDirectories(curnode *Node, oldRoot *Node, nodemap map[*Node]*Node) {
	linkDirectoriesR(curnode, oldRoot, nodemap)

}

func linkDirectoriesR(cur *Node, oldRoot *Node, nodemap map[*Node]*Node) {
	oldNode := nodemap[cur]
	if cur.kind == "stage" {
		/* Just try to link this stage. If we can't we just do nothing and let it
		 * get recomputed.
		 */
		if !cur.blacklistedFromMRT && oldNode != nil {
			Println("Link (stage) %v(%v) to %v(%v)", cur.name, cur.path, oldNode.name, oldNode.path)
			err := os.Symlink(oldNode.path, cur.path)
			if err != nil {
				panic(err)
			}
		}
	} else if cur.kind == "pipeline" {
		/* Try to link an entire pipeline */
		if !cur.blacklistedFromMRT && oldNode != nil {
			Println("Link (pipeline) %v(%v) to %v(%v)", cur.name, cur.path, oldNode.name, oldNode.path)
			err := os.Symlink(oldNode.path, cur.path)
			if err != nil {
				panic(err)
			}

		} else {
			/* If we can't (or shouldn't), we recurse and try to link its children */
			os.Mkdir(cur.path, 0777)
			for _, chld := range cur.subnodes {
				linkDirectoriesR(chld.getNode(), oldRoot, nodemap)
			}
		}
	}
}

/*
 * This markts a set of nodes as well as any nodes dependent on them as blacklisted.
 * A node is dependet another node if it uses data that it provides (is in postnodes) or
 * if it is a parent of that node.
 */
func (self *Pipestance) BlacklistMRTNodes(namesToBlacklist []string) error {
	for _, s := range namesToBlacklist {
		err := self.BlacklistMRTNode(s)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
 * Blacklist the node named |nameToBlacklist| as well as all of its descendents
 */
func (self *Pipestance) BlacklistMRTNode(nameToBlacklist string) error {
	var start *Node
	self.node.FindNodeByName(nameToBlacklist, &start)
	if start == nil {
		return errors.New("Your name doesn't exist")
	}
	TaintNode(start)
	return nil
}

/*
 * Recursively blacklist nodes.
 */
func TaintNode(root *Node) {
	if root.blacklistedFromMRT == false {
		Println("Invalidate: %v", root.name)
		root.blacklistedFromMRT = true

		/* If a stage or pipeline is tainted, its parent should also be tainted. */
		if root.parent != nil {
			TaintNode(root.parent.getNode())
		}

		/* Any stage that depends on this node must be tainted, too */
		for _, subs := range root.postnodes {
			TaintNode(subs.getNode())
		}
	}
}

func partiallyQualifiedName(n string) string {

        count := 0;
        for i := 0; i < len(n); i++ {
                if (n[i] == '.') {
                        count++;
                }
                if (count == 2) {
                        return n[i+1:len(n)]
                }
        }
        return n;
}

/*
 * Find a node given a name and store the node in *out.  If the name appears
 * multiple times in the pipeline, crash.
 */
func (n *Node) FindNodeByName(name string, out **Node) {
	if name == partiallyQualifiedName(n.fqname) || name == n.name{
		if *out != nil {
			panic(fmt.Sprintf("Name collision! %v at %v. Use a fully qualified name instead.", name, n.fqname))
		}
		*out = n
	} else {
		for _, subs := range n.subnodes {
			subs.getNode().FindNodeByName(name, out)
		}
	}
}

/*
 * This is the main entry point for "mrt".
 * newinfo corresponds to a new (non-existing) pipestance and oldinfo to an existing
 * pipestance.  Invalidate lists stages in the new pipestance that have code differences.
 *
 * We create a new pipestance directory and link every stage/pipeline from oldinfo
 * that we can. We explicitly don't link anything in |invalidate| or that derives from
 * anything in invalidate.
 *
 * After this runs, the new directory can be mrp'ed to run the new pipestance.
 */
func DoIt(newinfo *PipestanceSetup, oldinfo *PipestanceSetup, invalidate []string) {
	SetupSignalHandlers()

	/*
	 * Build runtime objects. We never actually use these but the interfaces
	 * to create pipestance objects require it.
	 */
	rtnew := NewRuntime("local", "disable", "disable", "2")
	rtold := NewRuntime("local", "disable", "disable", "2")

	if rtnew == nil {
		panic("Failed to allocate a runtime object.")
	}

	if rtold == nil {
		panic("Failed to allocate a runtime object.")
	}

	/* Setup the new pipestance */
	newcall, err := ioutil.ReadFile(newinfo.Srcpath)
	DieIf(err)

	psnew, err := rtnew.InvokePipeline(string(newcall),
		newinfo.Srcpath,
		newinfo.Psid,
		newinfo.PipestancePath,
		newinfo.MroPaths,
		newinfo.MroVersion,
		newinfo.Envs,
		[]string{})

	DieIf(err)

	/* Attach to the old pipestance */
	oldcall, err := ioutil.ReadFile(oldinfo.Srcpath)
	DieIf(err)

	psold, err := rtold.ReattachToPipestanceWithMroSrc(oldinfo.Psid,
		oldinfo.PipestancePath,
		string(oldcall),
		oldinfo.MroPaths,
		oldinfo.MroVersion,
		oldinfo.Envs,
		false,
		true)

	if err != nil {
		Println("COULD NOT ATTACH TO PIPESTANCE: %v", err)
		panic(err)
	}

	/* TODO: We should check a few things here:
	 * 1. Was the old pipestance built with no-vdr.
	 * 2. Is the old pipestance complete.
	 */

	/* Compute an association between nodes in the parallel pipestances */
	mapmap := MapTwoPipestances(psnew, psold)
	/* TODO:
	 * We should check for failures here. Failure to check for include
	 * 1. No nodes mapped between the two pipestances
	 * 2. Ambiguous maps. (This will cause a panic right now)
	 */

	/* Blacklist nodes in the newpipestance that have changed, as well as dependents
	 * of changed nodes.
	 */
	psnew.BlacklistMRTNodes(invalidate)

	/* Link directoroes in the new pipestance to the old pipestance, when possible */
	LinkDirectories(psnew.getNode(), psold.getNode(), mapmap)

	/*
	 * TODO:
	 * We need to close the new pipestance and delete the lock file.
	 */
}

func JM(x interface{}) string {
	m, err := json.Marshal(x)
	if err != nil {
		Println("JSON ERR: %v", err)
	}
	return string(m)
}

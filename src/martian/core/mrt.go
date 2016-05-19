// Copyright (c) 2016 10X Genomics, Inc. All rights reserved.

package core

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
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

	count := 0

	for _, x := range m {
		if x != nil {
			count++
		}
	}

	if count == 0 {
		panic("Failed to link any stages between the new and old pipeline. Sorry")
	}
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
		 * Find the name of |curnode| in |oldRoot| and assign it.  If we
		 * don't find it, we just assign m[curnode] to null which is fine.
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

/* This builds a set of symlinks from one pipestance to another. All of the nonblacklisted
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
			/* If we can't link the entire pipeline, make a directory for it and
			 * then recurse into its children and try to link them.
			 */
			os.Mkdir(cur.path, 0777)
			for _, chld := range cur.subnodes {
				linkDirectoriesR(chld.getNode(), oldRoot, nodemap)
			}
		}
	}
}

/*
 * This marks a set of nodes as well as any nodes dependent on them as blacklisted.
 * A node is dependent another node if it uses data that it provides (is in postnodes) or
 * if it is a parent of that node.
 */
func (self *Pipestance) BlacklistMRTNodes(namesToBlacklist []string, nodemap map[*Node]*Node) error {
	for _, s := range namesToBlacklist {
		var start *Node
		self.node.FindNodeByName(s, &start)
		if start == nil {
			return errors.New("Your name doesn't exist")
		}
		TaintNode(start, nodemap)
	}
	return nil
}

/*
 * Iterate over the entire tree and print the names of the nodes that have been blacklisted */
func ScanTree(root *Node) {

	if root.blacklistedFromMRT {
		Println("Invalidated: %v", root.name)
	}

	for _, s := range root.subnodes {
		ScanTree(s.getNode())
	}
}

/*
 * Recursively blacklist nodes.
 */
func TaintNode(root *Node, nodemap map[*Node]*Node) {
	if root.blacklistedFromMRT == false {
		//Println("Invalidate: %v", root.name)
		root.blacklistedFromMRT = true

		/* If a stage or pipeline is tainted, its parent should also be tainted. */
		if root.parent != nil {
			TaintNode(root.parent.getNode(), nodemap)
		}

		/* Any stage that depends on this node must be tainted, too */
		for _, subs := range root.postnodes {
			TaintNode(subs.getNode(), nodemap)
		}

		/* Since we have to redo *THIS* node, make sure that its dependencies
		 * have not been VDR'ed.  If they have, then blacklist them, too.
		 */
		VDRTaint(root, nodemap)
	}
}

/*
 * blacklist dependencies that have been VDR'ed.
 */
func VDRTaint(root *Node, nodemap map[*Node]*Node) {

	for _, s := range root.prenodes {
		sub := s.getNode()

		oldnode := nodemap[sub]
		if oldnode == nil {
			/* If we can't map this dependency into the old tree, move up one level
			 * and try again.
			 * This fails "soft" if we never find a mappable dependency we just give
			 * up and hope for the best.
			 */
			VDRTaint(sub, nodemap)
		} else {
			if oldnode.VDRMurdered() {
				/* If we found a match and it has been VDR'ed, we need to
				 * blacklist it. We also, have to walk a level up the tree and
				 * blacklist its parent and check its dependents.
				 *
				 * Conversely, the recursion stops when we find a match
				 * that has not been VDR'ed.
				 */
				sub.blacklistedFromMRT = true
				parent := sub.parent.getNode()

				/* XXX This step confuses me.  The tricky part is that
				 * the VDRTaint recursion goes "up" the tree but the
				 * TaintNode recursion goes down the tree. What if they meet?
				 * In that case, note that VDRTaint() will also be explicitly
				 * called on the parent node by TaintNode and it will do so
				 * before it calls VDRTaint on it.  This guarantees that the
				 * check below will never cause us to miss a node.
				 */
				if parent.blacklistedFromMRT == false {
					parent.blacklistedFromMRT = true
					VDRTaint(parent, nodemap)
				}

				/* Check sub's dependencies to see if they have been VDR'ed */
				VDRTaint(sub.getNode(), nodemap)
			}
		}
	}
}

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
			return n[i+1 : len(n)]
		}
	}
	return n
}

/*
 * Find a node by a name. |name| may be a "partially" qualified pipestance name
 * (see partiallyQualifiedName() above) or just a stage name.  If it is a stage name,
 * and that name occurs multiple times in the pipeline, we will panic().
 */
func (n *Node) FindNodeByName(name string, out **Node) {
	if name == partiallyQualifiedName(n.fqname) || name == n.name {
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
 * Return true if the data inside a node was VDR'ed.
 */
func (n *Node) VDRMurdered() bool {

	if len(n.forks) == 0 {
		Println("NO FORKS: %v", n.fqname)
	}
	anykilled := false
	for _, f := range n.forks {
		f.metadata.loadCache()
		var thiskilled = f.metadata.exists("vdrkill")

		if thiskilled {
			jsondata := f.metadata.read("vdrkill")

			/* TODO: Do some type checking here. */
			m := jsondata.(map[string]interface{})
			killcount := m["count"].(float64)

			if killcount > 0 {
				anykilled = true
			}
		} else {
			Println("%v Has no VDR record", n.name)

		}
	}
	return anykilled
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
func MRTBuildPipeline(newinfo *PipestanceSetup, oldinfo *PipestanceSetup, invalidate []string) {
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
	psnew.BlacklistMRTNodes(invalidate, mapmap)

	ScanTree(psnew.getNode())

	/* Link directoroes in the new pipestance to the old pipestance, when possible */
	LinkDirectories(psnew.getNode(), psold.getNode(), mapmap)

	psnew.Unlock()
}



package core

import (
	"os"
	"errors"
	"encoding/json"
	"io/ioutil"
)


type PSInfo struct {
	Srcpath string;
	Psid string;
	PipestancePath string;
	MroPaths []string
	MroVersion string
	Envs map[string]string
}

/*
 * This takes two pipestances and creates a map that associates nodes in
 * one pipestance with the nodes in the other. Nodes are associated if
 * they have the same name. 
 */
func MapTwoPipestances(newp * Pipestance, oldp * Pipestance) map[*Node]*Node {

	m := make(map[*Node]*Node);
	MapR(newp.node, oldp.node, m);
	return m;
}

func MapR(curnode * Node, oldRoot * Node, m map[*Node]*Node) {

	if (curnode != nil) {
		var oldNode *Node;
		oldRoot.FindNodeByName(curnode.name, &oldNode);
		m[curnode] = oldNode;

		for _, subs := range curnode.subnodes {
			MapR(subs.getNode(), oldRoot, m);
		}
	}
}


func LinkDirectories(curnode * Node, oldRoot * Node, nodemap map[*Node]*Node) {
	LinkDirectoriesR(curnode, oldRoot, nodemap);

}


/* This buids a set of symlinks from one pipestance to another. All of the nonblacklisted
 * stages (and sub-pipelines) that have a corresponding node will be linked.  We try to 
 * link entire sub-pipelines when possible.
 */
func LinkDirectoriesR(cur * Node, oldRoot * Node, nodemap map[*Node]*Node) {
	oldNode := nodemap[cur];
	if (cur.kind == "stage") {
		/* Just try to link this stage. If we can't we just do nothing and let it
		 * get recomputed.
		 */
		if (!cur.blacklistedFromMRT && oldNode != nil) {
			Println("Link (stage) %v(%v) to %v(%v)", cur.name, cur.path, oldNode.name, oldNode.path);
			err := os.Symlink(oldNode.path, cur.path);
			if (err != nil) {
				panic(err);
			}
		}
	}  else if (cur.kind == "pipeline"){
		/* Try to link an entire pipeline */
		if (!cur.blacklistedFromMRT && oldNode != nil) {
			Println("Link (pipeline) %v(%v) to %v(%v)", cur.name, cur.path, oldNode.name, oldNode.path);
			err := os.Symlink(oldNode.path, cur.path);
			if (err != nil) {
				panic(err);
			}

		} else {
			/* If we can't (or shouldn't), we recurse and try to link its children */
			os.Mkdir(cur.path, 0777);
			for _, chld := range cur.subnodes {
				LinkDirectoriesR(chld.getNode(), oldRoot, nodemap);
			}
		}
	}
}


/*
 * Mark all children of any nodes listed in |namesToBlacklist| as "blacklisted".
 * This will prevent their data from being autopopulated during the MRt autopopulate
 * stage. 
 *
 * NOTE: We assume a 1:1 correspondence of names and nodes. This isn't quite correct. 
 */
func (self * Pipestance) BlacklistMRTNodes(namesToBlacklist []string) error {
	for _, s := range namesToBlacklist {
		err := self.BlacklistMRTNode(s);
		if (err != nil) {
			return err;
		}
	}
	return nil;
}

/*
 * Blacklist the node named |nameToBlacklist| as well as all of its descendents
 */
func (self * Pipestance) BlacklistMRTNode(nameToBlacklist string) error {
	var start *Node;
	self.node.FindNodeByName(nameToBlacklist, &start);
	if (start == nil) {
		return errors.New("Your name doesn't exist");
	}
	TaintNode(start);
	return nil;
}


/*
 * Recursively blacklist nodes.
 */
func TaintNode(root * Node) {
	if (root.blacklistedFromMRT == false) {
		Println("Taint: %v", root.name);
		root.blacklistedFromMRT = true;

		/* If a stage or pipeline is tainted, its parent should also be tainted. */
		if root.parent != nil{
			Println("PARENT: of %v: %v", root.name, root.parent.getNode().name);
			TaintNode(root.parent.getNode());
		}
		
		/* Any stage that depends on this node must be tainted, too */
		for _, subs := range root.postnodes {
			Println("POSTNODE:of %v: %v", root.name, subs.getNode().name);
			TaintNode(subs.getNode());
		}
	}
}


func (n * Node) FindNodeByName(name string, out **Node) {
	if (name == n.name) {
		if (*out != nil) {
			panic("Name collision!");
		}
		*out = n;
	} else {
		for _, subs := range n.subnodes {
			subs.getNode().FindNodeByName(name, out);
		}
	}
}



/*
 * This takes a PSInfo object, detailing the invocation environment for a old
 * and new pipestance. It creates the directory layout for the new pipestance.
 * TODO:
 *  Then it links all of the linkable stages from the old pipestance into the new pipestance.
 */

func DoIt(newinfo *PSInfo, oldinfo *PSInfo, invalidate[]string) {
	SetupSignalHandlers();
	rtnew := NewRuntime("local", "disable", "disable", "2");
	rtold := NewRuntime("local", "disable", "disable", "2");

	if (rtnew == nil) {
		panic("1");
	}

	if (rtold == nil) {
		panic("2");
	}


	newcall, err := ioutil.ReadFile(newinfo.Srcpath);
	DieIf(err);

	psnew, err := rtnew.InvokePipeline(string(newcall),
		newinfo.Srcpath,
		newinfo.Psid,
		newinfo.PipestancePath,
		newinfo.MroPaths,
		newinfo.MroVersion,
		newinfo.Envs,
		[]string{});

	DieIf(err);


	oldcall, err := ioutil.ReadFile(oldinfo.Srcpath);
	DieIf(err);

	psold, err := rtold.ReattachToPipestance(oldinfo.Psid,
		oldinfo.PipestancePath,
		string(oldcall),
		oldinfo.MroPaths,
		oldinfo.MroVersion,
		oldinfo.Envs,
		false,
		true);

	if (err != nil) {
		Println("OMGOMGOMG! AN ERROR: %v", err);
		panic("2");
	}
	

	Println("J1:  %v", psnew.getNode());
	Println("J2:  %v", psold.getNode());

	mapmap := MapTwoPipestances(psnew, psold);

	Println("MMM: %v", mapmap);

	psnew.BlacklistMRTNodes(invalidate);
	Println("JXXXXX: %v", psnew.getNode());

	LinkDirectories(psnew.getNode(), psold.getNode(), mapmap);
}

func JM(x interface{}) string {
	m, err := json.Marshal(x);
	if (err != nil) {
		Println("JSON ERR: %v", err);
	}
	return string(m)
}




package core

import (
	"os"
	"errors"
	"encoding/json"
	"io/ioutil"
)


type PSInfo struct {
	srcpath string;
	psid string;
	pipestancePath string;
	mroPaths []string
	mroVersion string
	envs map[string]string
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


func LinkDirectoriesR(cur * Node, oldRoot * Node, nodemap map[*Node]*Node) {
	oldNode := nodemap[cur];
	if (cur.kind == "stage") {
		if (!cur.blacklistedFromMRT && oldNode != nil) {
			Println("Link %v(%v) to %v(%v)", cur.name, cur.path, oldNode.name, oldNode.path);
			err := os.Symlink(oldNode.path, cur.path);
			if (err != nil) {
				panic(err);
			}
		}
	} else {
		os.Mkdir(cur.path, 0777);
	}

	for _, chld := range cur.subnodes {
		LinkDirectoriesR(chld.getNode(), oldRoot, nodemap);
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

func (self * Pipestance) BlacklistMRTNode(nameToBlacklist string) error {
	var start *Node;
	self.node.FindNodeByName(nameToBlacklist, &start);
	if (start == nil) {
		return errors.New("Your name doesn't exist");
	}
	TaintNode(start);
	return nil;
}


func TaintNode(root * Node) {
	if (root.blacklistedFromMRT == false) {
		Println("Taint: %v", root.name);
		root.blacklistedFromMRT = true;
		/*
		for _, subs := range root.subnodes {
			TaintNode(subs.getNode());
		}
		*/

		for _, subs := range root.postnodes {
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

func DoIt(newinfo *PSInfo, oldinfo *PSInfo) {
	SetupSignalHandlers();
	rtnew := NewRuntime("local", "disable", "disable", "2");
	rtold := NewRuntime("local", "disable", "disable", "2");

	if (rtnew == nil) {
		panic("1");
	}

	if (rtold == nil) {
		panic("2");
	}


	newcall, err := ioutil.ReadFile(newinfo.srcpath);
	DieIf(err);

	psnew, err := rtnew.InvokePipeline(string(newcall),
		newinfo.srcpath,
		newinfo.psid,
		newinfo.pipestancePath,
		newinfo.mroPaths,
		newinfo.mroVersion,
		newinfo.envs,
		[]string{});

	DieIf(err);


	oldcall, err := ioutil.ReadFile(oldinfo.srcpath);
	DieIf(err);

	psold, err := rtold.ReattachToPipestance(oldinfo.psid,
		oldinfo.pipestancePath,
		string(oldcall),
		oldinfo.mroPaths,
		oldinfo.mroVersion,
		oldinfo.envs,
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

	psnew.BlacklistMRTNodes([]string{"COUNT"});
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


//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Martian runtime management for pipeline graph nodes.
//

package core

import (
	"fmt"
	"math"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

//=============================================================================
// Node
//=============================================================================

type Nodable interface {
	getNode() *Node

	// Gets the node's fully-qualified name.
	GetFQName() string

	// Returns the set of nodes which serve as prerequisites to this node,
	// as a mapping from fully-qualified name to node.
	GetPrenodes() map[string]Nodable

	// Returns the set of nodes which are able to run once this node
	// has completed.
	GetPostNodes() map[string]Nodable

	// Gets the mro AST object, if any, which will be executed for this node.
	Callable() syntax.Callable

	// Gets the set of forks of this node which match the given fork ID.
	matchForks(ForkId) []*Fork
}

// Represents a node in the pipeline graph.
type Node struct {
	parent         Nodable
	top            *TopNode
	call           syntax.CallGraphNode
	path           string
	metadata       *Metadata
	resources      *JobResources
	subnodes       map[string]Nodable
	prenodes       map[string]Nodable
	directPrenodes []Nodable
	postnodes      map[string]Nodable
	frontierNodes  *threadSafeNodeMap
	forks          []*Fork
	state          MetadataState
	local          bool
	stagecode      *syntax.SrcParam
	forkRoots      []*syntax.CallStm
	forkIds        ForkIdSet
	resolvedCmd    string
}

// Represents an edge in the pipeline graph.
type EdgeInfo struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Encapsulates information about a node failure.
type NodeErrorInfo struct {
	FQname  string `json:"fqname"`
	Path    string `json:"path"`
	Summary string `json:"summary,omitempty"`
	Log     string `json:"log,omitempty"`
}

type NodeInfo struct {
	Name          string                   `json:"name"`
	Fqname        string                   `json:"fqname"`
	Type          syntax.CallGraphNodeType `json:"type"`
	Path          string                   `json:"path"`
	State         MetadataState            `json:"state"`
	Metadata      *MetadataInfo            `json:"metadata"`
	Forks         []*ForkInfo              `json:"forks"`
	Edges         []EdgeInfo               `json:"edges"`
	StagecodeLang syntax.StageCodeType     `json:"stagecodeLang"`
	StagecodeCmd  string                   `json:"stagecodeCmd"`
	Error         *NodeErrorInfo           `json:"error,omitempty"`
}

func (self *Node) getNode() *Node { return self }

func (self *Node) GetPrenodes() map[string]Nodable {
	return self.prenodes
}

func (self *Node) GetPostNodes() map[string]Nodable {
	return self.postnodes
}

func (self *Node) Callable() syntax.Callable {
	return self.call.Callable()
}

func (self *Node) Types() *syntax.TypeLookup {
	return self.top.types
}

func NewNode(parent Nodable, call syntax.CallGraphNode) *Node {
	self := &Node{
		parent: parent,
		top:    parent.getNode().top,
		call:   call,
	}
	self.top.allNodes[call.GetFqid()] = self
	self.path = path.Join(parent.getNode().path, call.Call().Id)
	self.metadata = NewMetadata(self.call.GetFqid(), self.path)
	if self.call.Call().Modifiers.Preflight || !self.top.rt.Config.NeverLocal {
		self.local = call.Call().Modifiers.Local
	}
	if len(call.GetChildren()) > 0 {
		self.subnodes = make(map[string]Nodable, len(call.GetChildren()))
	} else if s, ok := call.Callable().(*syntax.Stage); ok {
		self.stagecode = s.Src
	}
	self.frontierNodes = parent.getNode().frontierNodes

	self.makeDirectPrenodes()
	self.makePrenodes()

	// Do not set state = getState here, or else nodes will wrongly report
	// complete before the first refreshMetadata call
	return self
}

func (self *Node) makeDirectPrenodes() {
	if parent, ok := self.parent.(*Node); ok {
		allBindings := self.call.Call().Bindings.List
		if mods := self.call.Call().Modifiers; mods != nil {
			if modBindings := mods.Bindings; modBindings != nil {
				allBindings = append(
					allBindings[:len(allBindings):len(allBindings)],
					modBindings.List...)
			}
		}
		var prenodes map[Nodable]struct{}
		for i, binding := range allBindings {
			if binding.Id != "*" {
				prenodes = findDirectRefs(binding.Exp, parent, len(allBindings)-i, prenodes)
			}
		}
		if len(prenodes) == 0 {
			self.directPrenodes = nil
		} else {
			self.directPrenodes = make([]Nodable, 0, len(prenodes))
			for n := range prenodes {
				self.directPrenodes = append(self.directPrenodes, n)
			}
			sort.Slice(self.directPrenodes, func(i, j int) bool {
				return self.directPrenodes[i].GetFQName() < self.directPrenodes[j].GetFQName()
			})
		}
	}
}

func findDirectRefs(exp syntax.Exp, parent *Node, expectedSize int,
	prenodes map[Nodable]struct{}) map[Nodable]struct{} {
	refs := exp.FindRefs()
	for i, ref := range refs {
		if ref.Kind == syntax.KindCall {
			if prenodes == nil {
				prenodes = make(map[Nodable]struct{}, len(refs)-i+expectedSize)
			}
			n := parent.subnodes[ref.Id]
			if n == nil {
				panic(parent.GetFQName() + " has no child " + ref.Id)
			}
			prenodes[n] = struct{}{}
		} else {
			prenodes = parent.addDirectRefNodes(ref, prenodes)
		}
	}
	return prenodes
}

func (self *Node) addDirectRefNodes(ref *syntax.RefExp,
	prenodes map[Nodable]struct{}) map[Nodable]struct{} {
	binding := self.call.Call().Bindings.Table[ref.Id]
	if binding == nil {
		panic(self.GetFQName() + " has no argument " + ref.Id)
	}
	var forks map[*syntax.CallStm]syntax.CollectionIndex
	if fr := self.call.ForkRoots(); len(fr) > 0 {
		forks = make(map[*syntax.CallStm]syntax.CollectionIndex, len(fr))
	}
	exp, err := binding.Exp.BindingPath(ref.OutputId, forks, self.top.types)
	if err != nil {
		panic(err)
	}
	if n := self.parent; n != nil {
		return findDirectRefs(exp, n.getNode(), 0, prenodes)
	}
	return prenodes
}

func (self *Node) makePrenodesForBinding(bind *syntax.ResolvedBinding,
	refs map[Nodable]struct{},
	fileRefs map[Nodable]map[string]syntax.Type) (map[Nodable]struct{}, map[Nodable]map[string]syntax.Type) {
	brefs, err := bind.FindRefs(self.top.types)
	if err != nil {
		// This should never happen if the ast compiled properly
		tid := bind.Type.TypeId()
		panic(fmt.Sprint("finding refs in ", bind.Exp.GoString(),
			" of type ", tid.String(), ":", err))
	}
	if len(brefs) > 0 {
		if refs == nil {
			refs = make(map[Nodable]struct{},
				len(self.call.ResolvedInputs())*len(brefs))
		}
		for _, ref := range brefs {
			rnode := self.top.allNodes[ref.Exp.Id]
			refs[rnode] = struct{}{}
			if ref.Type.IsFile() != syntax.KindIsNotFile {
				if self.top.rt.Config.Debug {
					util.LogInfo("storage",
						"Output %s of %s is a file argument, bound by %s",
						ref.Exp.OutputId,
						rnode.GetFQName(),
						self.GetFQName())
				}
				if fileRefs == nil {
					fileRefs = map[Nodable]map[string]syntax.Type{
						rnode: {
							ref.Exp.OutputId: ref.Type,
						},
					}
				} else if nrefs := fileRefs[rnode]; nrefs == nil {
					fileRefs[rnode] = map[string]syntax.Type{
						ref.Exp.OutputId: ref.Type,
					}
				} else if existing := nrefs[ref.Exp.OutputId]; existing == nil ||
					ref.Type.IsAssignableFrom(existing, self.top.types) != nil {
					nrefs[ref.Exp.OutputId] = ref.Type
				}
			}
		}
	}
	// Make sure we get fork root prenodes as well as actual input prenodes.
	allRefs := bind.Exp.FindRefs()
	if len(allRefs) > 0 {
		if refs == nil {
			refs = make(map[Nodable]struct{}, len(allRefs))
		}
		for _, ref := range allRefs {
			refs[self.top.allNodes[ref.Id]] = struct{}{}
		}
	}
	return refs, fileRefs
}

func (self *Node) makePrenodes() {
	var refs map[Nodable]struct{}
	var fileRefs map[Nodable]map[string]syntax.Type
	for _, bind := range self.call.ResolvedInputs() {
		refs, fileRefs = self.makePrenodesForBinding(bind, refs, fileRefs)
	}
	for _, exp := range self.call.Disabled() {
		brefs := exp.FindRefs()
		if len(brefs) > 0 {
			if refs == nil {
				refs = make(map[Nodable]struct{}, len(brefs))
			}
			for _, ref := range brefs {
				refs[self.top.allNodes[ref.Id]] = struct{}{}
			}
		}
	}
	if len(refs) == 0 {
		self.prenodes = nil
	} else {
		self.prenodes = make(map[string]Nodable)
		for node := range refs {
			self.prenodes[node.GetFQName()] = node
			node.getNode().setPostNode(self)
		}
	}
	if len(fileRefs) > 0 {
		self.attachToFileParents(fileRefs)
	}
}

func (self *Node) makeReturnBindings(directReturn []*syntax.BindStm) {
	prenodes := make(map[string]struct{}, len(directReturn))
	for _, binding := range directReturn {
		refs := binding.Exp.FindRefs()
		for _, ref := range refs {
			if ref.Kind == syntax.KindCall {
				prenodes[ref.Id] = struct{}{}
			}
		}
	}
	if len(prenodes) > 0 {
		if self.directPrenodes == nil {
			self.directPrenodes = make([]Nodable, 0, len(prenodes))
		}
		for id := range prenodes {
			n := self.top.allNodes[self.call.GetFqid()+"."+id]
			if n != nil {
				self.directPrenodes = append(self.directPrenodes, n)
			}
		}
		sort.Slice(self.directPrenodes, func(i, j int) bool {
			return self.directPrenodes[i].GetFQName() < self.directPrenodes[j].GetFQName()
		})
	}
	refs, fileRefs := self.makePrenodesForBinding(
		self.call.ResolvedOutputs(), nil, nil)
	if len(refs) > 0 {
		if self.prenodes == nil {
			self.prenodes = make(map[string]Nodable, len(refs))
		}
		for node := range refs {
			self.prenodes[node.GetFQName()] = node
			node.getNode().setPostNode(self)
		}
	}
	if len(fileRefs) > 0 {
		self.attachToFileParents(fileRefs)
	}
}

func (self *Node) attachToFileParents(fileParents map[Nodable]map[string]syntax.Type) {
	setNode := self
	if self.call.Kind() == syntax.KindPipeline {
		if self.parent == self.top {
			if self.top.rt.Config.Debug {
				util.LogInfo("storage",
					"Top-level pipeline binds files from %d nodes",
					len(fileParents))
			}
			// Don't add to file post-nodes, since this will never count as
			// "done".  However still add to fileArgs since we want to
			// preserve the arg.
			setNode = nil
		} else {
			// Non-top-level pipeline does not force argument retention.
			return
		}
	}
	for prenode, boundArgs := range fileParents {
		for _, fork := range prenode.getNode().forks {
			if setNode != nil {
				if pNodeFiles := fork.filePostNodes; pNodeFiles == nil {
					fork.filePostNodes = map[Nodable]map[string]syntax.Type{
						self: boundArgs,
					}
				} else {
					pNodeFiles[self] = boundArgs
				}
			}
			pArgs := fork.fileArgs
			if pArgs == nil {
				pArgs = make(map[string]map[Nodable]struct{}, len(boundArgs))
				fork.fileArgs = pArgs
			}
			for arg := range boundArgs {
				if nodes := pArgs[arg]; nodes == nil {
					pArgs[arg] = map[Nodable]struct{}{
						setNode: {},
					}
				} else {
					nodes[setNode] = struct{}{}
				}
			}
		}
	}
}

//
// Folder construction
//
func (self *Node) mkdirs() error {
	if err := util.MkdirAll(self.path); err != nil {
		msg := fmt.Sprintf("Could not create root directory for %s: %s",
			self.call.GetFqid(), err.Error())
		util.LogError(err, "runtime", msg)
		self.metadata.WriteErrorString(msg)
		return err
	}
	if err := util.Mkdir(self.top.journalPath); err != nil {
		msg := fmt.Sprintf("Could not create directories for %s: %s",
			self.call.GetFqid(), err.Error())
		util.LogError(err, "runtime", msg)
		self.metadata.WriteErrorString(msg)
		return err
	}
	if err := util.Mkdir(self.top.tmpPath); err != nil {
		msg := fmt.Sprintf("Could not create directories for %s: %s",
			self.call.GetFqid(), err.Error())
		util.LogError(err, "runtime", msg)
		self.metadata.WriteErrorString(msg)
		return err
	}

	var wg sync.WaitGroup
	forks := self.forks
	wg.Add(len(forks))
	for _, fork := range forks {
		go func(f *Fork) {
			f.mkdirs()
			wg.Done()
		}(fork)
	}
	wg.Wait()
	return nil
}

func (self *Node) buildForks() {
	// Build out argument permutations.
	self.forkIds.MakeForkIds(self.call.ForkRoots(), self.top.types)
	if len(self.forkIds.List) == 0 {
		self.forks = []*Fork{NewFork(self, 0, nil, self.call.ResolvedInputs())}
	} else {
		self.forks = make([]*Fork, len(self.forkIds.List), cap(self.forkIds.List))
		for i, id := range self.forkIds.List {
			self.forks[i] = NewFork(self, i, id, self.call.ResolvedInputs())
		}
	}
}

func cloneFork(fork *Fork, id ForkId) *Fork {
	nf := NewFork(fork.node, len(fork.node.forks), id, fork.args)
	// Copy fileArgs
	if len(fork.fileArgs) > 0 {
		nf.fileArgs = make(
			map[string]map[Nodable]struct{},
			len(fork.fileArgs))
		for k, m := range fork.fileArgs {
			if m == nil {
				nf.fileArgs[k] = nil
			} else {
				nm := make(map[Nodable]struct{}, len(m))
				nf.fileArgs[k] = nm
				for k := range m {
					nm[k] = struct{}{}
				}
			}
		}
	}
	if len(fork.filePostNodes) > 0 {
		nf.filePostNodes = make(
			map[Nodable]map[string]syntax.Type,
			len(fork.filePostNodes))
		for k, m := range fork.filePostNodes {
			if m == nil {
				nf.filePostNodes[k] = nil
			} else {
				nm := make(map[string]syntax.Type, len(m))
				for k, t := range m {
					nm[k] = t
				}
				nf.filePostNodes[k] = nm
			}
		}
	}
	return nf
}

func (self *Node) expandForks(must bool) bool {
	any := false
	for i := 0; i < len(self.forks); i++ {
		fork := self.forks[i]
		newForks, err := fork.expand(must)
		for ; err == nil && len(newForks) > 0; newForks, err = fork.expand(must) {
			any = true
			if len(self.forkIds.List) == 0 {
				self.forkIds.List = make([]ForkId, 1, len(newForks)+1)
			}
			self.forkIds.List[i] = fork.forkId
			util.LogInfo("runtime", "Adding %d new forks of %s",
				len(newForks),
				fork.fqname)
			for _, id := range newForks {
				nf := cloneFork(fork, id)
				self.forks = append(self.forks, nf)
				self.forkIds.List = append(self.forkIds.List, id)
			}
		}
		if err != nil {
			if !must {
				return any
			}
			util.PrintError(err, "runtime",
				"Error computing forking for %s\n",
				fork.fqname)
			if err := util.MkdirAll(fork.path); err != nil {
				util.LogError(err, "runtime",
					"Could not create directories for %s", fork.fqname)
			}
			fork.metadata.writeError("resolving forks", err)
			return any
		}
		if len(self.forkIds.List) > 0 {
			self.forkIds.List[i] = fork.forkId
		}
	}
	return any
}

//
// Subnode management
//
func (self *Node) setPrenode(prenode Nodable) {
	for _, subnode := range self.subnodes {
		subnode.getNode().setPrenode(prenode)
	}
	if self.prenodes == nil {
		self.prenodes = make(map[string]Nodable)
	}
	self.prenodes[prenode.GetFQName()] = prenode
	prenode.getNode().setPostNode(self)
}

func (self *Node) setPostNode(postnode *Node) {
	if self.postnodes == nil {
		self.postnodes = map[string]Nodable{
			postnode.call.GetFqid(): postnode,
		}
	} else {
		self.postnodes[postnode.call.GetFqid()] = postnode
	}
}

func (self *Node) addFrontierNode(node Nodable) {
	self.frontierNodes.Add(node.GetFQName(), node)
}

func (self *Node) removeFrontierNode(node Nodable) {
	self.frontierNodes.Remove(node.GetFQName())
}

func (self *Node) getFrontierNodes() []*Node {
	return self.frontierNodes.GetNodes()
}

func (self *Node) allNodes() []*Node {
	// Enumerate and sort the keys in subnodes first.
	// This ensures a stable chirality for the dag UI.
	ids := make([]string, 0, len(self.subnodes))
	for id := range self.subnodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	all := make([]*Node, 1, 1+len(ids))
	all[0] = self

	// Build a list of all subnodes.
	for _, id := range ids {
		subnode := self.subnodes[id]
		all = append(all, subnode.getNode().allNodes()...)
	}
	return all
}

func (self *Node) find(fqname string) *Node {
	sn := self.call.GetFqid()
	// test sn == self.top.fqname + "." + fqname
	// without allocating a new string for it.
	if len(sn) == len(self.top.fqname)+len(fqname)+1 &&
		sn[len(self.top.fqname)] == '.' &&
		sn[:len(self.top.fqname)] == self.top.fqname &&
		sn[len(self.top.fqname)+1:] == fqname {
		return self
	} else if sn == fqname {
		return self
	}
	for _, subnode := range self.subnodes {
		node := subnode.getNode().find(fqname)
		if node != nil {
			return node
		}
	}
	return nil
}

//
// State management
//

func (self *Node) collectMetadatas() []*Metadata {
	metadatas := make([]*Metadata, 1, 1+4*len(self.forks))
	metadatas[0] = self.metadata
	for _, fork := range self.forks {
		metadatas = append(metadatas, fork.collectMetadatas()...)
	}
	return metadatas
}

func (self *Node) loadMetadata() {
	metadatas := self.collectMetadatas()
	for _, metadata := range metadatas {
		// Load metadata file cache
		metadata.loadCache()

		// Reset metadata heartbeat timer
		metadata.resetHeartbeat()
	}
	self.state = self.getState()
	self.addFrontierNode(self)
}

func (self *Node) removeMetadata() {
	for _, fork := range self.forks {
		fork.removeMetadata()
	}
}

func (self *Node) getFork(index string) *Fork {
	i, err := strconv.Atoi(index)
	if err == nil && i >= 0 && i < len(self.forks) {
		return self.forks[i]
	}
	l := len(self.call.GetFqid()) + 5
	for _, f := range self.forks {
		if len(f.fqname) > l && f.fqname[l:] == index {
			return f
		}
	}
	return nil
}

func (self *Node) getState() MetadataState {
	// If any fork is failed, we're failed.
	// If every fork is disabled, we're disabled.
	// Otherwise, if every fork is complete or disabled, we're complete.
	complete := true
	disabled := true
	for _, fork := range self.forks {
		if s := fork.getState(); s == Failed {
			return Failed
		} else if s != Complete && s != DisabledState {
			complete = false
			break
		} else if s != DisabledState {
			disabled = false
		}
	}
	if complete {
		if disabled {
			return DisabledState
		}
		return Complete
	}
	// If any prenode is not complete, we're waiting.
	for _, prenode := range self.prenodes {
		if s := prenode.getNode().getState(); s != Complete && s != DisabledState {
			return Waiting
		}
	}
	// Otherwise we're running.
	return Running
}

func (self *Node) reset() error {
	if self.top.rt.Config.FullStageReset {
		util.PrintInfo("runtime", "(reset)           %s", self.call.GetFqid())

		// Blow away the entire stage node.
		if err := os.RemoveAll(self.path); err != nil {
			util.PrintInfo("runtime",
				`Cannot reset the stage because its folder contents could not be deleted.

Please resolve this error in order to continue running the pipeline:`)
			return err
		}
		// Remove all related files from journal directory.
		if files, err := util.Readdirnames(self.top.journalPath); err == nil {
			base := strings.TrimPrefix(strings.TrimPrefix(self.call.GetFqid(),
				self.top.fqname), ".")
			for _, file := range files {
				if strings.HasPrefix(file, base) {
					os.Remove(path.Join(self.top.journalPath, file))
				}
			}
		}

		// Clear chunks in the forks so they can be rebuilt on split.
		for _, fork := range self.forks {
			fork.reset()
		}

		// Create stage node directories.
		if err := self.mkdirs(); err != nil {
			return err
		}
	} else {
		for _, fork := range self.forks {
			if err := fork.resetPartial(); err != nil {
				return err
			}
		}
	}

	// Refresh the metadata.
	self.loadMetadata()
	return nil
}

func (self *Node) restartLocallyQueuedJobs() error {
	if self.top.rt.Config.FullStageReset {
		// If entire stages got blown away then this isn't needed.
		return nil
	}
	for _, fork := range self.forks {
		if err := fork.restartLocallyQueuedJobs(); err != nil {
			return err
		}
	}
	return nil
}

func (self *Node) restartLocalJobs() error {
	if self.top.rt.Config.FullStageReset {
		// If entire stages got blown away then this isn't needed.
		return nil
	}
	for _, fork := range self.forks {
		if err := fork.restartLocalJobs(); err != nil {
			return err
		}
	}
	return nil
}

func (self *Node) checkHeartbeats() {
	for _, metadata := range self.collectMetadatas() {
		metadata.checkHeartbeat()
	}
}

func (self *Node) kill(message string) {
	for _, fork := range self.forks {
		fork.kill(message)
	}
}

func (self *Node) postProcess() {
	os.RemoveAll(self.top.journalPath)
	os.RemoveAll(self.top.tmpPath)

	var errs syntax.ErrorList
	for _, fork := range self.forks {
		if err := fork.postProcess(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errs.If(); err != nil {
		util.Print("\nCould not move output files:\n%s\n\n", err.Error())
	}
}

func (self *Node) cachePerf() {
	if _, ok := self.vdrKill(); ok {
		// Cache all fork performance info if node can be VDR-ed.
		for _, fork := range self.forks {
			fork.cachePerf()
		}
	}
}

func (self *Node) GetFQName() string {
	if self.call == nil {
		return self.top.fqname
	}
	return self.call.GetFqid()
}

func (self *Node) getFatalError() (string, bool, string, string, MetadataFileName, []string) {
	for _, metadata := range self.collectMetadatas() {
		if state, _ := metadata.getState(); state != Failed {
			continue
		}
		if metadata.exists(Errors) {
			errlog := metadata.readRaw(Errors)
			summary := "<none>"
			if self.stagecode != nil && self.stagecode.Type == syntax.PythonStage {
				errlines := strings.Split(errlog, "\n")
				if len(errlines) >= 2 {
					summary = errlines[len(errlines)-2]
				} else if len(errlines) == 1 {
					summary = errlines[0]
				}
			}
			errpaths := []string{
				metadata.MetadataFilePath(Errors),
				metadata.MetadataFilePath(StdOut),
				metadata.MetadataFilePath(StdErr),
			}
			if self.top.rt.Config.StackVars {
				errpaths = append(errpaths, metadata.MetadataFilePath(Stackvars))
			}
			return metadata.fqname, self.call.Call().Modifiers.Preflight,
				summary, errlog, Errors, errpaths
		}
		if metadata.exists(Assert) {
			assertlog := metadata.readRaw(Assert)
			summary := "<none>"
			assertlines := strings.Split(assertlog, "\n")
			if len(assertlines) >= 1 {
				summary = assertlines[len(assertlines)-1]
			}
			return metadata.fqname, self.call.Call().Modifiers.Preflight,
				summary, assertlog, Assert, []string{
					metadata.MetadataFilePath(Assert),
				}
		}
	}
	return "", false, "", "", "", []string{}
}

// Returns true if there is no error or if the error is one we expect to not
// recur if the pipeline is rerun.
func (self *Node) isErrorTransient() (bool, string) {
	passRegexp, _ := getRetryRegexps()
	for _, metadata := range self.collectMetadatas() {
		if state, _ := metadata.getState(); state != Failed {
			continue
		}
		if metadata.exists(Assert) {
			return false, ""
		}
		if metadata.exists(Errors) {
			errlog := metadata.readRaw(Errors)
			for _, line := range strings.Split(errlog, "\n") {
				for _, re := range passRegexp {
					if re.MatchString(line) {
						return true, errlog
					}
				}
			}
			return false, errlog
		}
	}
	return true, ""
}

func (self *Node) step() bool {
	if self.state == Running {
		for _, fork := range self.forks {
			if self.call.Call().Modifiers.Preflight && self.top.rt.Config.SkipPreflight {
				fork.skip()
			} else {
				fork.step()
			}
		}
	}
	previousState := self.state
	newState := self.getState()
	if newState == Running && newState != previousState {
		if self.expandForks(true) {
			return true
		}
	}
	self.state = newState
	switch self.state {
	case Failed:
		self.addFrontierNode(self)
	case Running:
		if self.state != previousState {
			if err := self.mkdirs(); err != nil {
				util.LogError(err, "runtime",
					"Could not create node directories")
			}
		}
		self.addFrontierNode(self)
	case Complete:
		if vdr := self.top.rt.Config.VdrMode; vdr == VdrRolling || vdr == VdrStrict {
			for _, node := range self.prenodes {
				node.getNode().cachePerf()
			}
			self.cachePerf()
		}
		fallthrough
	case DisabledState:
		for _, node := range self.postnodes {
			self.addFrontierNode(node)
		}
		self.removeFrontierNode(self)
	case ForkWaiting:
		self.removeFrontierNode(self)
	}
	return self.state != previousState
}

// Regular expression to convert a fully qualified name for a chunk into the
// component parts of the pipeline path.  The parts are:
// 1. The fully qualified stage name.
// 2. The fork index.
// 3. The chunk index, if any.
// 4. The job uniquifier, if any.
// 5. The metadata file name.
var jobJournalRe = regexp.MustCompile(`(.*)\.fork([^.]+)(?:\.chnk(\d+))?(?:\.u([a-f0-9]{10}))?\.(.*)$`)

func (self *Node) parseRunFilename(fqname string) (string, string, int, string, string) {
	if match := jobJournalRe.FindStringSubmatch(fqname); len(match) >= 6 {
		chunkIndex := -1
		if match[3] != "" {
			chunkIndex, _ = strconv.Atoi(match[3])
		}
		return match[1], match[2], chunkIndex, match[4], match[5]
	}
	return "", "", -1, "", ""
}

func (self *Node) refreshState(readOnly bool) {
	startTime := time.Now().Add(-self.top.rt.JobManager.queueCheckGrace())
	files, err := util.Readdirnames(self.top.journalPath)
	if err != nil {
		util.LogError(err, "runtime", "Could not read journal directory.")
	}
	updatedForks := make(map[*Fork]struct{})
	for _, file := range files {
		filename := path.Base(file)
		if strings.HasSuffix(filename, ".tmp") {
			continue
		}

		fqname, forkIndex, chunkIndex, uniquifier, state := self.parseRunFilename(filename)
		if fqname == "" {
			util.LogInfo("runtime",
				"WARNING: failed to parse journal file name %s",
				filename)
		} else if node := self.find(fqname); node != nil {
			if fork := node.getFork(forkIndex); fork != nil {
				if chunkIndex >= 0 {
					if chunk := fork.getChunk(chunkIndex); chunk != nil {
						chunk.updateState(MetadataFileName(state), uniquifier)
					} else {
						util.LogInfo("runtime",
							"WARNING: Journal update for unknown chunk %s.fork%s.chnk%d",
							fqname, forkIndex, chunkIndex)
					}
				} else {
					fork.updateState(state, uniquifier)
				}
				updatedForks[fork] = struct{}{}
			} else {
				util.LogInfo("runtime",
					"WARNING: Journal update for unknown fork %s.fork%s",
					fqname, forkIndex)
			}
		} else {
			util.LogInfo("runtime",
				"WARNING: Journal update for unknown node %s (%s)",
				fqname, filename)
		}
		if !readOnly {
			os.Remove(path.Join(self.top.journalPath, file))
		}
	}
	for _, node := range self.getFrontierNodes() {
		for _, meta := range node.collectMetadatas() {
			meta.endRefresh(startTime)
		}
	}
	for fork := range updatedForks {
		fork.printUpdateIfNeeded()
	}
}

//
// Serialization
//
func (self *Node) serializeState() *NodeInfo {
	forks := make([]*ForkInfo, 0, len(self.forks))
	for _, fork := range self.forks {
		forks = append(forks, fork.serializeState())
	}
	edges := make([]EdgeInfo, 0, len(self.directPrenodes))
	for _, prenode := range self.directPrenodes {
		edges = append(edges, EdgeInfo{
			From: prenode.GetFQName(),
			To:   self.call.GetFqid(),
		})
	}
	var err *NodeErrorInfo
	if self.state == Failed {
		fqname, _, summary, log, _, errpaths := self.getFatalError()
		errpath := ""
		if len(errpaths) > 0 {
			errpath = errpaths[0]
		}
		err = &NodeErrorInfo{
			FQname:  fqname,
			Path:    errpath,
			Summary: summary,
			Log:     log,
		}
	}
	info := &NodeInfo{
		Name:     self.call.Call().Id,
		Fqname:   self.call.GetFqid(),
		Type:     self.call.Kind(),
		Path:     self.path,
		State:    self.state,
		Metadata: self.metadata.serializeState(),
		Forks:    forks,
		Edges:    edges,
		Error:    err,
	}
	if src := self.stagecode; src != nil {
		info.StagecodeLang = src.Type
		if len(src.Args) == 0 {
			info.StagecodeCmd = self.resolvedCmd
		} else {
			info.StagecodeCmd = self.resolvedCmd +
				" " + strings.Join(src.Args, " ")
		}
	}
	return info
}

func (self *Node) serializePerf() (*NodePerfInfo, []*VdrEvent) {
	forks := make([]*ForkPerfInfo, 0, len(self.forks))
	var storageEvents []*VdrEvent
	for _, fork := range self.forks {
		forkSer, vdrKill := fork.serializePerf()
		forks = append(forks, forkSer)
		if vdrKill != nil && self.call.Kind() != syntax.KindPipeline {
			storageEvents = append(storageEvents, vdrKill.Events...)
		}
	}
	return &NodePerfInfo{
		Name:   self.call.Call().Id,
		Fqname: self.call.GetFqid(),
		Type:   self.call.Kind(),
		Forks:  forks,
	}, storageEvents
}

//=============================================================================
// Job Runners
//=============================================================================

func (self *Node) getJobReqs(jobDef *JobResources, stageType string) JobResources {
	var res JobResources

	if self.resources != nil {
		res = *self.resources
	}

	// Get values passed from the stage code
	if jobDef != nil {
		if jobDef.Threads != 0 {
			res.Threads = jobDef.Threads
		}
		if jobDef.MemGB != 0 {
			res.MemGB = jobDef.MemGB
		}
		if jobDef.VMemGB != 0 {
			res.VMemGB = jobDef.VMemGB
		}
		if jobDef.Special != "" {
			res.Special = jobDef.Special
		}
	}

	// Override with job manager caps specified from commandline
	self.top.rt.overrides.GetResources(self.GetFQName(), stageType, &res)

	if self.local {
		return self.top.rt.LocalJobManager.GetSystemReqs(&res)
	} else {
		return self.top.rt.JobManager.GetSystemReqs(&res)
	}
}

func (self *Node) getProfileMode(stageType string) ProfileMode {
	return self.top.rt.overrides.GetProfile(self.GetFQName(),
		stageType,
		self.top.rt.Config.ProfileMode)
}

func (self *Node) setJobReqs(jobDef *JobResources, stageType string) JobResources {
	// Get values and possibly modify them
	res := self.getJobReqs(jobDef, stageType)

	// Write modified values back
	if jobDef != nil {
		*jobDef = res
	}

	return res
}

func (self *Node) setSplitJobReqs() JobResources {
	return self.setJobReqs(nil, STAGE_TYPE_SPLIT)
}

func (self *Node) setChunkJobReqs(jobDef *JobResources) JobResources {
	return self.setJobReqs(jobDef, STAGE_TYPE_CHUNK)
}

func (self *Node) setJoinJobReqs(jobDef *JobResources) JobResources {
	return self.setJobReqs(jobDef, STAGE_TYPE_JOIN)
}

func (self *Node) runSplit(fqname string, metadata *Metadata) {
	res := self.setSplitJobReqs()
	self.runJob("split", fqname, STAGE_TYPE_SPLIT, metadata, &res)
}

func (self *Node) runJoin(fqname string, metadata *Metadata, res *JobResources) {
	self.runJob("join", fqname, STAGE_TYPE_JOIN, metadata, res)
}

func (self *Node) runChunk(fqname string, metadata *Metadata, res *JobResources) {
	self.runJob("main", fqname, STAGE_TYPE_CHUNK, metadata, res)
}

func (self *Node) runJob(shellName, fqname, stageType string,
	metadata *Metadata, res *JobResources) {
	// Configure local variable dumping.
	stackVars := disable
	if self.top.rt.Config.StackVars {
		stackVars = "stackvars"
	}

	// Configure memory monitoring.
	monitor := disable
	if self.top.rt.Config.Monitor {
		monitor = "monitor"
	}

	// Construct path to the shell.
	shellCmd := ""
	var argv []string
	runFile := metadata.journalFile()
	version := &self.top.version
	envs := self.top.envs
	if td := metadata.TempDir(); td != "" {
		envs = make(map[string]string, len(self.top.envs)+1)
		for k, v := range self.top.envs {
			envs[k] = v
		}
		envs["TMPDIR"] = td
	}
	switch self.stagecode.Type {
	case syntax.PythonStage:
		if len(self.stagecode.Args) != 0 {
			panic(fmt.Sprintf(
				"Invalid python stage module specification \"%s %s\"",
				self.resolvedCmd, strings.Join(self.stagecode.Args, " ")))
		}
		shellCmd = self.top.rt.mrjob
		argv = []string{
			path.Join(self.top.rt.adaptersPath, "python", "martian_shell.py"),
			self.resolvedCmd,
			shellName,
			metadata.path,
			metadata.curFilesPath,
			runFile,
		}
	case syntax.CompiledStage:
		shellCmd = self.top.rt.mrjob
		argv = make([]string, 1, len(self.stagecode.Args)+4)
		argv[0] = self.resolvedCmd
		argv = append(argv, self.stagecode.Args...)
		argv = append(argv, shellName, metadata.path, metadata.curFilesPath, runFile)
	case syntax.ExecStage:
		shellCmd = self.resolvedCmd
		argv = append(
			self.stagecode.Args[:len(self.stagecode.Args):len(self.stagecode.Args)],
			shellName, metadata.path, metadata.curFilesPath, runFile)
	default:
		panic(fmt.Sprint("Unknown stage code language: ", self.stagecode.Type))
	}

	// Log the job run.
	jobMode := self.top.rt.Config.JobMode
	jobManager := self.top.rt.JobManager
	if self.local {
		jobMode = localMode
		jobManager = self.top.rt.LocalJobManager
	}
	jobModeLabel := strings.Replace(jobMode, ".template", "", -1)
	padding := strings.Repeat(" ", int(math.Max(0, float64(10-len(path.Base(jobModeLabel))))))
	if self.call.Call().Modifiers.Preflight {
		util.LogInfo("runtime", "(run:%s) %s %s.%s",
			path.Base(jobModeLabel), padding, fqname, shellName)
	} else {
		util.PrintInfo("runtime", "(run:%s) %s %s.%s",
			path.Base(jobModeLabel), padding, fqname, shellName)
	}
	profileMode := self.getProfileMode(stageType)
	jobInfo := JobInfo{
		Name:          fqname,
		Type:          jobMode,
		Threads:       res.Threads,
		MemGB:         res.MemGB,
		VMemGB:        res.VMemGB,
		ProfileConfig: self.top.rt.ProfileConfig(profileMode),
		ProfileMode:   profileMode,
		Stackvars:     stackVars,
		Monitor:       monitor,
		Invocation:    self.top.invocation,
		Version:       version,
	}
	if jobInfo.ProfileConfig != nil && jobInfo.ProfileConfig.Adapter != "" {
		jobInfo.ProfileMode = jobInfo.ProfileConfig.Adapter
	}

	if err := func() error {
		util.EnterCriticalSection()
		defer util.ExitCriticalSection()
		if err := metadata.WriteTime(QueuedLocally); err != nil {
			return err
		}
		return metadata.Write(JobInfoFile, &jobInfo)
	}(); err != nil {
		util.PrintError(err, "jobmngr",
			"Could not write jobinfo file, aborting.")
		util.Suicide(false)
	}
	jobManager.execJob(shellCmd, argv, envs, metadata, res, fqname,
		shellName, self.call.Call().Modifiers.Preflight && self.local)
}

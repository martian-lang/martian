//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Margo runtime.
//
package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
)

//=============================================================================
// Metadata
//=============================================================================
type Metadata struct {
	path      string
	contents  map[string]bool
	filesPath string
}

func NewMetadata(p string) *Metadata {
	self := &Metadata{}
	self.path = p
	self.filesPath = path.Join(p, "files")
	return self
}

func (self *Metadata) glob() []string {
	paths, _ := filepath.Glob(path.Join(self.path, "_*"))
	return paths
}

func (self *Metadata) enumerateFiles() ([]string, error) {
	return filepath.Glob(path.Join(self.filesPath, "*"))
}

func (self *Metadata) mkdirs() {
	mkdir(self.path)
	mkdir(self.filesPath)
}

func (self *Metadata) idemMkdirs() {
	idemMkdir(self.path)
	idemMkdir(self.filesPath)
}

func (self *Metadata) getState(name string) (string, bool) {
	if self.exists("errors") {
		return "failed", true
	}
	if self.exists("complete") {
		return name + "complete", true
	}
	if self.exists("log") {
		return name + "running", true
	}
	if self.exists("jobinfo") {
		return name + "queued", true
	}
	return "", false
}

func (self *Metadata) cache() {
	if !self.exists("complete") {
		self.contents = map[string]bool{}
		paths := self.glob()
		for _, p := range paths {
			self.contents[path.Base(p)[1:]] = true
		}
	}
}

func (self *Metadata) makePath(name string) string {
	return path.Join(self.path, "_"+name)
}
func (self *Metadata) exists(name string) bool {
	_, ok := self.contents[name]
	return ok
}
func (self *Metadata) readRaw(name string) string {
	bytes, _ := ioutil.ReadFile(self.makePath(name))
	return string(bytes)
}
func (self *Metadata) read(name string) interface{} {
	var v interface{}
	json.Unmarshal([]byte(self.readRaw(name)), &v)
	return v
}
func (self *Metadata) writeRaw(name string, text string) {
	ioutil.WriteFile(self.makePath(name), []byte(text), 0644)
}
func (self *Metadata) write(name string, object interface{}) {
	bytes, _ := json.MarshalIndent(object, "", "    ")
	self.writeRaw(name, string(bytes))
}
func (self *Metadata) append(name string, text string) {
	f, _ := os.OpenFile(self.makePath(name), os.O_WRONLY|os.O_CREATE, 0644)
	f.Write([]byte(text))
	f.Close()
}
func (self *Metadata) writeTime(name string) {
	self.writeRaw(name, Timestamp())
}
func (self *Metadata) remove(name string) { os.Remove(self.makePath(name)) }

func (self *Metadata) serialize() interface{} {
	names := []string{}
	for content, _ := range self.contents {
		names = append(names, content)
	}
	sort.Strings(names)
	return map[string]interface{}{
		"path":  self.path,
		"names": names,
	}
}

//=============================================================================
// Binding
//=============================================================================
type Binding struct {
	node      *Node
	id        string
	tname     string
	sweep     bool
	waiting   bool
	valexp    string
	mode      string
	boundNode Nodable
	output    string
	value     interface{}
}

func NewBinding(node *Node, bindStm *BindStm) *Binding {
	self := &Binding{}
	self.node = node
	self.id = bindStm.id
	self.tname = bindStm.Tname
	self.sweep = bindStm.sweep
	self.waiting = false
	switch valueExp := bindStm.Exp.(type) {
	case *RefExp:
		if valueExp.Kind == "self" {
			parentBinding := self.node.parent.Node().argbindings[valueExp.Id]
			if parentBinding != nil {
				self.node = parentBinding.node
				self.tname = parentBinding.tname
				self.sweep = parentBinding.sweep
				self.waiting = parentBinding.waiting
				self.mode = parentBinding.mode
				self.boundNode = parentBinding.boundNode
				self.output = parentBinding.output
				self.value = parentBinding.value
			}
			self.id = bindStm.id
			self.valexp = "self." + valueExp.Id
		} else if valueExp.Kind == "call" {
			self.mode = "reference"
			self.boundNode = self.node.parent.Node().subnodes[valueExp.Id]
			self.output = valueExp.outputId
			if valueExp.outputId == "default" {
				self.valexp = valueExp.Id
			} else {
				self.valexp = valueExp.Id + "." + valueExp.outputId
			}
		}
	case *ValExp:
		self.mode = "value"
		self.boundNode = node
		self.value = bindStm.Exp.(*ValExp).Value
		// Unwrap array values.
		if self.value != nil && valueExp.Kind == "array" {
			value := []interface{}{}
			for _, e := range self.value.([]Exp) {
				value = append(value, e.(*ValExp).Value)
			}
			self.value = value
		}
	}
	return self
}

func NewReturnBinding(node *Node, bindStm *BindStm) *Binding {
	self := &Binding{}
	self.node = node
	self.id = bindStm.id
	self.tname = bindStm.Tname
	self.mode = "reference"
	valueExp := bindStm.Exp.(*RefExp)
	self.boundNode = self.node.subnodes[valueExp.Id] // from node, NOT parent; this is diff from Binding
	self.output = valueExp.outputId
	if valueExp.outputId == "default" {
		self.valexp = valueExp.Id
	} else {
		self.valexp = valueExp.Id + "." + valueExp.outputId
	}
	return self
}

func (self *Binding) resolve(argPermute map[string]interface{}) interface{} {
	self.waiting = false
	if self.mode == "value" {
		if argPermute == nil {
			// In this case we want to get the raw value, which might be a sweep array.
			return self.value
		}
		// Replace literal sweep ranges with specific permuted argument values.
		if self.sweep {
			return argPermute[self.id]
		} else {
			return self.value
		}
	}
	if argPermute == nil {
		return nil
	}
	if self.boundNode != nil {
		matchedFork := self.boundNode.Node().matchFork(argPermute)
		outputs, ok := matchedFork.metadata.read("outs").(map[string]interface{})
		if ok {
			output, ok := outputs[self.output]
			if ok {
				return output
			}
		}
	}
	self.waiting = true
	return nil
}

func (self *Binding) serialize(argPermute map[string]interface{}) interface{} {
	var node interface{} = nil
	var matchedFork interface{} = nil
	if self.boundNode != nil {
		node = self.boundNode.Node().name
		f := self.boundNode.Node().matchFork(argPermute)
		if f != nil {
			matchedFork = f.index
		}
	}
	return map[string]interface{}{
		"id":          self.id,
		"type":        self.tname,
		"valexp":      self.valexp,
		"mode":        self.mode,
		"output":      self.output,
		"sweep":       self.sweep,
		"node":        node,
		"matchedFork": matchedFork,
		"value":       self.resolve(argPermute),
		"waiting":     self.waiting,
	}
}

// Helpers
func resolveBindings(bindings map[string]*Binding, argPermute map[string]interface{}) map[string]interface{} {
	resolvedBindings := map[string]interface{}{}
	for id, binding := range bindings {
		resolvedBindings[id] = binding.resolve(argPermute)
	}
	return resolvedBindings
}

func makeOutArgs(outParams *Params, filesPath string) map[string]interface{} {
	args := map[string]interface{}{}
	for id, param := range outParams.table {
		if param.IsFile() {
			args[id] = path.Join(filesPath, param.Id()+"."+param.Tname())
		} else if param.Tname() == "path" {
			args[id] = path.Join(filesPath, param.Id())
		} else {
			args[id] = nil
		}
	}
	return args
}

//=============================================================================
// Chunk
//=============================================================================
type Chunk struct {
	node     *Node
	fork     *Fork
	index    int
	chunkDef map[string]interface{}
	path     string
	fqname   string
	metadata *Metadata
	has_run  bool
}

func NewChunk(nodable Nodable, fork *Fork, index int, chunkDef map[string]interface{}) *Chunk {
	self := &Chunk{}
	self.node = nodable.Node()
	self.fork = fork
	self.index = index
	self.chunkDef = chunkDef
	self.path = path.Join(fork.path, fmt.Sprintf("chnk%d", index))
	self.fqname = fork.fqname + fmt.Sprintf(".chnk%d", index)
	self.metadata = NewMetadata(self.path)
	self.has_run = false
	if !self.node.split {
		// If we're not splitting, just set the sole chunk's filesPath
		// to the filesPath of the parent fork, to save a pseudo-join copy.
		self.metadata.filesPath = self.fork.metadata.filesPath
	}
	// we have to mkdirs here because runtime might have been interrupted after chunk_defs were
	// written but before next step interval caused the actual creation of the chnk folders.
	// in that scenario, upon restart the fork step would try to write _args into chnk folders
	// that don't exist.
	self.mkdirs()
	return self
}

func (self *Chunk) mkdirs() {
	self.metadata.idemMkdirs()
}

func (self *Chunk) getState() string {
	if state, ok := self.metadata.getState(""); ok {
		return state
	} else {
		return "ready"
	}
}

func (self *Chunk) Step() {
	if self.getState() == "ready" {
		resolvedBindings := resolveBindings(self.node.argbindings, self.fork.argPermute)
		for id, value := range self.chunkDef {
			resolvedBindings[id] = value
		}
		self.metadata.write("args", resolvedBindings)
		self.metadata.write("outs", makeOutArgs(self.node.outparams, self.metadata.filesPath))
		if !self.has_run {
			self.has_run = true
			self.node.RunJob("main", self.fqname, self.metadata, self.chunkDef["__threads"], self.chunkDef["__mem_gb"])
		}
	}
}

func (self *Chunk) serialize() interface{} {
	return map[string]interface{}{
		"index":    self.index,
		"chunkDef": self.chunkDef,
		"state":    self.getState(),
		"metadata": self.metadata.serialize(),
	}
}

//=============================================================================
// Fork
//=============================================================================
type Fork struct {
	node           *Node
	index          int
	path           string
	fqname         string
	metadata       *Metadata
	split_metadata *Metadata
	join_metadata  *Metadata
	chunks         []*Chunk
	split_has_run  bool
	join_has_run   bool
	argPermute     map[string]interface{}
}

func NewFork(nodable Nodable, index int, argPermute map[string]interface{}) *Fork {
	self := &Fork{}
	self.node = nodable.Node()
	self.index = index
	self.path = path.Join(self.node.path, fmt.Sprintf("fork%d", index))
	self.fqname = self.node.fqname + fmt.Sprintf(".fork%d", index)
	self.metadata = NewMetadata(self.path)
	self.split_metadata = NewMetadata(path.Join(self.path, "split"))
	self.join_metadata = NewMetadata(path.Join(self.path, "join"))
	self.chunks = []*Chunk{}
	self.argPermute = argPermute
	self.split_has_run = false
	self.join_has_run = false

	// reconstruct chunks using chunk_defs on reattach, do not rely
	// on metadata.exists('chunk_defs') since it may not be cached
	chunkDefIfaces := self.split_metadata.read("chunk_defs")
	if chunkDefs, ok := chunkDefIfaces.([]interface{}); ok {
		for i, chunkDef := range chunkDefs {
			chunk := NewChunk(self.node, self, i, chunkDef.(map[string]interface{}))
			self.chunks = append(self.chunks, chunk)
		}
	}
	return self
}

func (self *Fork) collectMetadatas() []*Metadata {
	metadatas := []*Metadata{self.metadata, self.split_metadata, self.join_metadata}
	for _, chunk := range self.chunks {
		metadatas = append(metadatas, chunk.metadata)
	}
	return metadatas
}

func (self *Fork) mkdirs() {
	self.metadata.mkdirs()
	self.split_metadata.mkdirs()
	self.join_metadata.mkdirs()
}

func (self *Fork) getState() string {
	if self.metadata.exists("complete") {
		return "complete"
	}
	if state, ok := self.join_metadata.getState("join_"); ok {
		return state
	}
	if len(self.chunks) > 0 {
		// If any chunks have failed, we're failed.
		for _, chunk := range self.chunks {
			if chunk.getState() == "failed" {
				return "failed"
			}
		}
		// If every chunk is complete, we're complete.
		every := true
		for _, chunk := range self.chunks {
			if chunk.getState() != "complete" {
				every = false
				break
			}
		}
		if every {
			return "chunks_complete"
		}
		// If every chunk is queued, running, or complete, we're complete.
		every = true
		runningStates := map[string]bool{"queued": true, "running": true, "complete": true}
		for _, chunk := range self.chunks {
			if _, ok := runningStates[chunk.getState()]; !ok {
				every = false
				break
			}
		}
		if every {
			return "chunks_running"
		}
	}
	if state, ok := self.split_metadata.getState("split_"); ok {
		return state
	}
	return "ready"
}

func (self *Fork) Step() {
	if self.node.kind == "stage" {
		state := self.getState()
		if !strings.HasSuffix(state, "_running") && !strings.HasSuffix(state, "_queued") {
			LogInfo("runtime", "(%s) %s", state, self.node.fqname)
		}

		if state == "ready" {
			self.split_metadata.write("args", resolveBindings(self.node.argbindings, self.argPermute))
			if self.node.split {
				if !self.split_has_run {
					self.split_has_run = true
					self.node.RunJob("split", self.fqname, self.split_metadata, nil, nil)
				}
			} else {
				self.split_metadata.write("chunk_defs", []interface{}{map[string]interface{}{}})
				self.split_metadata.writeTime("complete")
			}
		} else if state == "split_complete" {
			chunkDefs := self.split_metadata.read("chunk_defs")
			if len(self.chunks) == 0 {
				for i, chunkDef := range chunkDefs.([]interface{}) {
					chunk := NewChunk(self.node, self, i, chunkDef.(map[string]interface{}))
					self.chunks = append(self.chunks, chunk)
					chunk.mkdirs()
				}
			}
			for _, chunk := range self.chunks {
				chunk.Step()
			}
		} else if state == "chunks_complete" {
			self.join_metadata.write("args", resolveBindings(self.node.argbindings, self.argPermute))
			self.join_metadata.write("chunk_defs", self.split_metadata.read("chunk_defs"))
			if self.node.split {
				chunkOuts := []interface{}{}
				for _, chunk := range self.chunks {
					outs := chunk.metadata.read("outs")
					chunkOuts = append(chunkOuts, outs)
				}
				self.join_metadata.write("chunk_outs", chunkOuts)
				self.join_metadata.write("outs", makeOutArgs(self.node.outparams, self.metadata.filesPath))
				if !self.join_has_run {
					self.join_has_run = true
					self.node.RunJob("join", self.fqname, self.join_metadata, nil, nil)
				}
			} else {
				self.join_metadata.write("outs", self.chunks[0].metadata.read("outs"))
				self.join_metadata.writeTime("complete")
			}
		} else if state == "join_complete" {
			self.metadata.write("outs", self.join_metadata.read("outs"))
			self.metadata.writeTime("complete")
		}

	} else if self.node.kind == "pipeline" {
		self.metadata.write("outs", resolveBindings(self.node.retbindings, self.argPermute))
		self.metadata.writeTime("complete")
	}
}

func (self *Fork) serialize() interface{} {
	argbindings := []interface{}{}
	for _, argbinding := range self.node.argbindingList {
		argbindings = append(argbindings, argbinding.serialize(self.argPermute))
	}
	retbindings := []interface{}{}
	for _, retbinding := range self.node.retbindingList {
		retbindings = append(retbindings, retbinding.serialize(self.argPermute))
	}
	bindings := map[string]interface{}{
		"Argument": argbindings,
		"Return":   retbindings,
	}
	chunks := []interface{}{}
	for _, chunk := range self.chunks {
		chunks = append(chunks, chunk.serialize())
	}
	return map[string]interface{}{
		"index":          self.index,
		"argPermute":     self.argPermute,
		"state":          self.getState(),
		"metadata":       self.metadata.serialize(),
		"split_metadata": self.split_metadata.serialize(),
		"join_metadata":  self.join_metadata.serialize(),
		"chunks":         chunks,
		"bindings":       bindings,
	}
}

//=============================================================================
// Node
//=============================================================================
type Nodable interface {
	Node() *Node
}

type Node struct {
	parent         Nodable
	rt             *Runtime
	kind           string
	name           string
	fqname         string
	path           string
	metadata       *Metadata
	outparams      *Params
	argbindings    map[string]*Binding
	argbindingList []*Binding // for stable ordering
	retbindings    map[string]*Binding
	retbindingList []*Binding // for stable ordering
	sweepbindings  []*Binding
	subnodes       map[string]Nodable
	prenodes       map[string]Nodable
	prenodeList    []Nodable //for stable ordering
	forks          []*Fork
	split          bool
	state          string
	volatile       bool
	stagecodeLang  string
	stagecodePath  string
}

func (self *Node) Node() *Node { return self }

func NewNode(parent Nodable, kind string, callStm *CallStm, callables *Callables) *Node {
	self := &Node{}
	self.parent = parent

	self.rt = parent.Node().rt
	self.kind = kind
	self.name = callStm.Id
	self.fqname = parent.Node().fqname + "." + self.name
	self.path = path.Join(parent.Node().path, self.name)
	self.metadata = NewMetadata(self.path)
	self.volatile = callStm.volatile

	self.outparams = callables.table[self.name].OutParams()
	self.argbindings = map[string]*Binding{}
	self.argbindingList = []*Binding{}
	self.retbindings = map[string]*Binding{}
	self.retbindingList = []*Binding{}
	self.subnodes = map[string]Nodable{}
	self.prenodes = map[string]Nodable{}
	self.prenodeList = []Nodable{}

	for id, bindStm := range callStm.Bindings.table {
		binding := NewBinding(self, bindStm)
		self.argbindings[id] = binding
		self.argbindingList = append(self.argbindingList, binding)
	}
	for _, binding := range self.argbindingList {
		if binding.mode == "reference" && binding.boundNode != nil {
			self.prenodes[binding.boundNode.Node().name] = binding.boundNode
			self.prenodeList = append(self.prenodeList, binding.boundNode)
		}
	}
	// Do not set state = getState here, or else nodes will wrongly report
	// complete before the first refreshMetadata call
	return self
}

func (self *Node) mkdirs(wg *sync.WaitGroup) {
	mkdir(self.path)
	for _, fork := range self.forks {
		wg.Add(1)
		go func(f *Fork) {
			f.mkdirs()
			wg.Done()
		}(fork)
	}
	for _, subnode := range self.subnodes {
		wg.Add(1)
		go func(n Nodable) {
			n.Node().mkdirs(wg)
			wg.Done()
		}(subnode)
	}
}

// State and dataflow management (synchronous)
func (self *Node) GetState() string {
	// If every fork is complete, we're complete.
	complete := true
	for _, fork := range self.forks {
		if fork.getState() != "complete" {
			complete = false
			break
		}
	}
	if complete {
		return "complete"
	}
	// If any fork is failed, we're failed.
	for _, fork := range self.forks {
		if fork.getState() == "failed" {
			return "failed"
		}
	}
	// If any prenode is not complete, we're waiting.
	for _, prenode := range self.prenodes {
		if prenode.Node().GetState() != "complete" {
			return "waiting"
		}
	}
	// Otherwise we're running.
	return "running"
}

// Sweep management
func (self *Node) buildForks(bindings map[string]*Binding) {
	// Use a map to uniquify bindings by id.
	bindingTable := map[string]*Binding{}

	// Add local sweep bindings.
	for _, binding := range bindings {
		if binding.sweep {
			bindingTable[binding.id] = binding
		}
	}
	// Add upstream sweep bindings (from prenodes).
	for _, prenode := range self.prenodes {
		for _, binding := range prenode.Node().sweepbindings {
			bindingTable[binding.id] = binding
		}
	}

	for _, binding := range bindingTable {
		self.sweepbindings = append(self.sweepbindings, binding)
	}

	// Add all unique bindings to self.sweepbindings.
	paramIds := []string{}
	argRanges := []interface{}{}
	for _, binding := range self.sweepbindings {
		//  self.sweepbindings = append(self.sweepbindings, binding)
		paramIds = append(paramIds, binding.id)
		argRanges = append(argRanges, binding.resolve(nil))
	}

	// Build out argument permutations.
	for i, valPermute := range cartesianProduct(argRanges) {
		argPermute := map[string]interface{}{}
		for j, paramId := range paramIds {
			argPermute[paramId] = valPermute.([]interface{})[j]
		}
		self.forks = append(self.forks, NewFork(self, i, argPermute))
	}
}

func (self *Node) matchFork(targetArgPermute map[string]interface{}) *Fork {
	if targetArgPermute == nil {
		return nil
	}
	for _, fork := range self.forks {
		every := true
		for paramId, argValue := range fork.argPermute {
			if targetArgPermute[paramId] != argValue {
				every = false
				break
			}
		}
		if every {
			return fork
		}
	}
	return nil
}

func (self *Node) collectMetadatas() []*Metadata {
	metadatas := []*Metadata{self.metadata}
	for _, fork := range self.forks {
		metadatas = append(metadatas, fork.collectMetadatas()...)
	}
	return metadatas
}

func (self *Node) RefreshMetadata() {
	metadatas := self.collectMetadatas()
	for _, metadata := range metadatas {
		metadata.cache()
	}
	self.state = self.GetState()
}

func (self *Node) RestartFromFailed() {
	// Blow away the entire stage node.
	os.RemoveAll(self.path)

	// Re-create the folders.
	var rewg sync.WaitGroup
	self.mkdirs(&rewg)
	rewg.Wait()

	// Refresh the metadata (clear it all).
	self.RefreshMetadata()
}

func (self *Node) Step() {
	if self.state == "running" {
		for _, fork := range self.forks {
			fork.Step()
		}
	}
}

func (self *Node) AllNodes() []*Node {
	all := []*Node{self}
	for _, subnode := range self.subnodes {
		all = append(all, subnode.Node().AllNodes()...)
	}
	return all
}

func (self *Node) Find(fqname string) *Node {
	if self.fqname == fqname {
		return self
	}
	for _, subnode := range self.subnodes {
		node := subnode.Node().Find(fqname)
		if node != nil {
			return node
		}
	}
	return nil
}

func (self *Node) Serialize() interface{} {
	sweepbindings := []interface{}{}
	for _, sweepbinding := range self.sweepbindings {
		sweepbindings = append(sweepbindings, sweepbinding.serialize(nil))
	}
	forks := []interface{}{}
	for _, fork := range self.forks {
		forks = append(forks, fork.serialize())
	}
	edges := []interface{}{}
	for _, prenode := range self.prenodeList {
		edges = append(edges, map[string]string{
			"from": prenode.Node().name,
			"to":   self.name,
		})
	}
	return map[string]interface{}{
		"name":          self.name,
		"fqname":        self.fqname,
		"type":          self.kind,
		"path":          self.path,
		"state":         self.state,
		"metadata":      self.metadata.serialize(),
		"sweepbindings": sweepbindings,
		"forks":         forks,
		"edges":         edges,
		"stagecodeLang": self.stagecodeLang,
		"stagecodePath": self.stagecodePath,
	}
}

//=============================================================================
// Job Runners
//=============================================================================
func (self *Node) execLocalJob(shellName string, shellCmd string, stagecodePath string,
	libPath string, fqname string, metadata *Metadata, threads interface{},
	memGB interface{}) {

	cmd := exec.Command(shellCmd, stagecodePath, libPath, metadata.path, metadata.filesPath, "profile")
	stdoutFile, _ := os.Create(metadata.makePath("stdout"))
	stderrFile, _ := os.Create(metadata.makePath("stderr"))
	stdoutFile.WriteString("[stdout]\n")
	stderrFile.WriteString("[stderr]\n")
	defer stdoutFile.Close()
	defer stderrFile.Close()

	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	if err := cmd.Start(); err != nil {
		LogError(err, "runtime", "Could not exec local job.")
		return
	}
	metadata.write("jobinfo", map[string]interface{}{"type": "local", "childid": cmd.Process.Pid})
	cmd.Wait()
}

func (self *Node) execSGEJob(shellName string, shellCmd string, stagecodePath string,
	libPath string, fqname string, metadata *Metadata, threads interface{},
	memGB interface{}) {
	qscript := []string{shellCmd, stagecodePath, libPath, metadata.path, metadata.filesPath, "profile"}
	metadata.writeRaw("qscript", strings.Join(qscript, " "))

	cmdline := []string{
		"-N", strings.Join([]string{fqname, shellName}, "."),
		"-V",
	}
	// exec.Command doesn't like it if there are empty members of this
	// arg string array. Problem is empty threads arg string, if it
	// comes before the path to the script, is it gets interpreted
	// as the path to the script and qsub fails.
	if threads != nil {
		cmdline = append(cmdline, []string{
			"-pe",
			"threads",
			fmt.Sprintf("%v", threads),
		}...)
	}
	if memGB != nil {
		cmdline = append(cmdline, []string{
			"-l",
			fmt.Sprintf("h_vmem=%vG", memGB),
		}...)
	}
	cmdline = append(cmdline, []string{
		"-cwd",
		"-o", metadata.makePath("stdout"),
		"-e", metadata.makePath("stderr"),
		metadata.makePath("qscript"),
	}...)

	metadata.write("jobinfo", map[string]string{"type": "sge"})

	cmd := exec.Command("qsub", cmdline...)
	cmd.Dir = metadata.filesPath
	metadata.writeRaw("qsub", strings.Join(cmd.Args, " "))

	stdoutFile, _ := os.Create(metadata.makePath("stdout"))
	stderrFile, _ := os.Create(metadata.makePath("stderr"))
	stdoutFile.WriteString("[stdout]\n")
	stderrFile.WriteString("[stderr]\n")
	defer stdoutFile.Close()
	defer stderrFile.Close()

	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	cmd.Start()
	cmd.Wait()
}

func (self *Node) RunJob(shellName string, fqname string, metadata *Metadata,
	threads interface{}, memGB interface{}) {
	adaptersPath := path.Join(self.rt.adaptersPath, "python")
	libPath := path.Join(self.rt.libPath, "python")
	LogInfo("runtime", "(run-%s) %s.%s", self.rt.jobMode, fqname, shellName)
	metadata.write("jobinfo", map[string]interface{}{"type": nil, "childpid": nil})
	shellCmd := path.Join(adaptersPath, shellName+".py")
	if self.rt.jobMode == "local" {
		self.execLocalJob(shellName, shellCmd, self.stagecodePath, libPath, fqname, metadata, threads, memGB)
	} else if self.rt.jobMode == "sge" {
		self.execSGEJob(shellName, shellCmd, self.stagecodePath, libPath, fqname, metadata, threads, memGB)
	} else {
		panic(fmt.Sprintf("Unknown jobMode: %s", self.rt.jobMode))
	}
}

//=============================================================================
// Stagestance
//=============================================================================
type Stagestance struct {
	node *Node
}

func (self *Stagestance) Node() *Node { return self.node }

func NewStagestance(parent Nodable, callStm *CallStm, callables *Callables) *Stagestance {
	self := &Stagestance{}
	self.node = NewNode(parent, "stage", callStm, callables)
	stage := callables.table[self.node.name].(*Stage)
	self.node.stagecodePath = path.Join(self.node.rt.stagecodePath, stage.src.path)
	self.node.stagecodeLang = "Python"
	self.node.split = len(stage.splitParams.list) > 0
	self.node.buildForks(self.node.argbindings)
	return self
}

//=============================================================================
// Pipestance
//=============================================================================
type Pipestance struct {
	node *Node
}

func (self *Pipestance) Node() *Node { return self.node }

func (self *Pipestance) Pname() string { return self.node.name }
func (self *Pipestance) Psid() string  { return self.node.parent.Node().name }

func NewPipestance(parent Nodable, callStm *CallStm, callables *Callables) *Pipestance {
	self := &Pipestance{}
	self.node = NewNode(parent, "pipeline", callStm, callables)

	// Build subcall tree.
	pipeline := callables.table[self.node.name].(*Pipeline)
	for _, subcallStm := range pipeline.Calls {
		callable := callables.table[subcallStm.Id]
		switch callable.(type) {
		case *Stage:
			self.node.subnodes[subcallStm.Id] = NewStagestance(self.Node(), subcallStm, callables)
		case *Pipeline:
			self.node.subnodes[subcallStm.Id] = NewPipestance(self.Node(), subcallStm, callables)
		}
	}

	// Also depends on stages bound to return values.
	self.node.retbindings = map[string]*Binding{}
	for id, bindStm := range pipeline.ret.bindings.table {
		binding := NewReturnBinding(self.node, bindStm)
		self.node.retbindings[id] = binding
		self.node.retbindingList = append(self.node.retbindingList, binding)
		if binding.mode == "reference" && binding.boundNode != nil {
			self.node.prenodes[binding.boundNode.Node().name] = binding.boundNode
			self.node.prenodeList = append(self.node.prenodeList, binding.boundNode)
		}
	}

	self.node.buildForks(self.node.retbindings)
	return self
}

func (self *Pipestance) GetFQName() string {
	return self.Node().fqname
}

func (self *Pipestance) GetOverallState() string {
	nodes := self.Node().AllNodes()
	for _, node := range nodes {
		if node.state == "failed" {
			return "failed"
		}
	}
	for _, node := range nodes {
		if node.state == "running" {
			return "running"
		}
	}
	every := true
	for _, node := range nodes {
		if node.state != "complete" {
			every = false
			break
		}
	}
	if every {
		return "complete"
	}
	return "waiting"
}

func (self *Pipestance) Immortalize() {
	metadata := NewMetadata(self.Node().parent.Node().path)
	all := []interface{}{}
	for _, node := range self.Node().AllNodes() {
		all = append(all, node.Serialize())
	}
	metadata.write("finalstate", all)
}

func (self *Pipestance) Unimmortalize() {
	metadata := NewMetadata(self.Node().parent.Node().path)
	metadata.remove("finalstate")
}

type VDRKillReport struct {
	Count  uint     `json:"count"`
	Size   uint64   `json:"size"`
	Paths  []string `json:"paths"`
	Errors []string `json:"errors"`
}

func (self *Pipestance) VDRKill() *VDRKillReport {
	killPaths := []string{}

	// Iterate over all nodes.
	for _, node := range self.Node().AllNodes() {
		// Iterate over all forks.
		for _, fork := range node.forks {
			// For volatile nodes, kill fork-level files.
			if node.volatile {
				if paths, err := fork.metadata.enumerateFiles(); err == nil {
					killPaths = append(killPaths, paths...)
				}
				if paths, err := fork.split_metadata.enumerateFiles(); err == nil {
					killPaths = append(killPaths, paths...)
				}
				if paths, err := fork.join_metadata.enumerateFiles(); err == nil {
					killPaths = append(killPaths, paths...)
				}
			}
			// For ALL nodes, if the node splits, kill chunk-level files.
			// Must check for split here, otherwise we'll end up deleting
			// output files of non-volatile nodes because single-chunk nodes
			// get their output redirected to the one chunk's files path.
			if node.split {
				for _, chunk := range fork.chunks {
					if paths, err := chunk.metadata.enumerateFiles(); err == nil {
						killPaths = append(killPaths, paths...)
					}
				}
			}
		}
	}

	// Actually delete the paths.
	killReport := VDRKillReport{}
	for _, p := range killPaths {
		filepath.Walk(p, func(_ string, info os.FileInfo, err error) error {
			if err == nil {
				killReport.Size += uint64(info.Size())
				killReport.Count++
			} else {
				killReport.Errors = append(killReport.Errors, err.Error())
			}
			return nil
		})
		killReport.Paths = append(killReport.Paths, p)
		os.RemoveAll(p)
	}
	metadata := NewMetadata(self.Node().parent.Node().path)
	metadata.write("vdrkill", &killReport)
	return &killReport
}

func (self *Pipestance) GetOuts(forki int) interface{} {
	return self.Node().forks[forki].metadata.read("outs")
}

//=============================================================================
// TopNode
//=============================================================================
type TopNode struct {
	node *Node
}

func (self *TopNode) Node() *Node { return self.node }

func NewTopNode(rt *Runtime, psid string, p string) *TopNode {
	self := &TopNode{}
	self.node = &Node{}
	self.node.path = p
	self.node.rt = rt
	self.node.fqname = "ID." + psid
	self.node.name = psid
	return self
}

//=============================================================================
// Runtime
//=============================================================================
type Runtime struct {
	MroPath       string
	stagecodePath string
	libPath       string
	adaptersPath  string
	globalTable   map[string]*Ast
	srcTable      map[string]string
	typeTable     map[string]string
	CodeVersion   string
	jobMode       string
	/* TODO queue goes here */
}

func NewRuntime(jobMode string, pipelinesPath string) *Runtime {
	self := &Runtime{}
	self.MroPath = path.Join(pipelinesPath, "mro")
	self.stagecodePath = path.Join(pipelinesPath, "stages")
	self.libPath = path.Join(pipelinesPath, "lib")
	self.adaptersPath = RelPath(path.Join("..", "adapters"))
	self.globalTable = map[string]*Ast{}
	self.srcTable = map[string]string{}
	self.typeTable = map[string]string{}
	self.CodeVersion = getGitTag(pipelinesPath)
	self.jobMode = jobMode
	return self
}

func getGitTag(p string) string {
	oldCwd, _ := os.Getwd()
	os.Chdir(p)
	out, err := exec.Command("git", "describe", "--tags", "--dirty", "--always").Output()
	os.Chdir(oldCwd)
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return "noversion"
}

func (self *Runtime) GetPipelineNames() []string {
	names := []string{}
	for name, _ := range self.globalTable {
		names = append(names, name)
	}
	return names
}

// Compile an MRO file in self.mroPath named fname.mro.
func (self *Runtime) Compile(fname string) (*Ast, error) {
	processedSrc, global, err := parseFile(path.Join(self.MroPath, fname))
	if err != nil {
		return nil, err
	}
	for _, pipeline := range global.Pipelines {
		self.globalTable[pipeline.Id] = global
		self.srcTable[pipeline.Id] = processedSrc
	}
	return global, nil
}

// Compile all the MRO files in self.mroPath.
func (self *Runtime) CompileAll() (int, error) {
	paths, err := filepath.Glob(self.MroPath + "/[^_]*.mro")
	if err != nil {
		return 0, err
	}
	for _, p := range paths {
		_, err := self.Compile(path.Base(p))
		if err != nil {
			return 0, err
		}
	}
	return len(paths), nil
}

// Instantiate a pipestance object given a psid, MRO source, and a
// pipestance path. This is the core (private) method called by the
// public InvokeWithSource and Reattach methods.
func (self *Runtime) instantiate(psid string, src string, pipestancePath string) (*Pipestance, error) {
	// Parse the invocation call.
	callGlobal, err := parseCall(src)
	if err != nil {
		return nil, err
	}
	callStm := callGlobal.call

	// Get the global scope that defines the called pipeline.
	global, ok := self.globalTable[callStm.Id]
	if !ok {
		return nil, &MarioError{fmt.Sprintf("PipelineNotFoundError: '%s'", callStm.Id)}
	}

	// Get the actual pipeline definition and check call bindings.
	pipeline := global.callables.table[callStm.Id].(*Pipeline)
	if err := callStm.Bindings.check(global, pipeline, pipeline.InParams()); err != nil {
		return nil, err
	}

	// Instantiate the pipeline.
	pipestance := NewPipestance(NewTopNode(self, psid, pipestancePath), callStm, global.callables)
	return pipestance, nil
}

// Instantiate a stagestance.
func (self *Runtime) InstantiateStage(src string, stagestancePath string) (*Stagestance, error) {
	// Parse the invocation call.
	callGlobal, err := parseCall(src)
	if err != nil {
		return nil, err
	}
	callStm := callGlobal.call

	// Search through all globals for the named stage.
	for _, global := range self.globalTable {
		if stage, ok := global.callables.table[callStm.Id]; ok {
			err := callStm.Bindings.check(global, nil, stage.InParams())
			DieIf(err)

			stagestance := NewStagestance(NewTopNode(self, "", stagestancePath), callStm, global.callables)

			// Create stagestance folder graph concurrently.
			var wg sync.WaitGroup
			stagestance.Node().mkdirs(&wg)
			wg.Wait()

			return stagestance, nil
		}
	}
	return nil, &MarioError{fmt.Sprintf("StageNotFoundError: '%s'", callStm.Id)}
}

// Invokes a new pipestance.
func (self *Runtime) InvokeWithSource(psid string, src string, pipestancePath string) (*Pipestance, error) {
	// Check if pipestance path already exists.
	if _, err := os.Stat(pipestancePath); err == nil {
		return nil, &MarioError{fmt.Sprintf("PipestanceExistsError: '%s'", psid)}
	}

	// Create the pipestance path.
	if err := os.MkdirAll(pipestancePath, 0755); err != nil {
		return nil, err
	}

	// Instantiate the pipestance.
	pipestance, err := self.instantiate(psid, src, pipestancePath)
	if err != nil {
		// If instantiation failed, delete the pipestance folder.
		os.RemoveAll(pipestancePath)
		return nil, err
	}

	// Write top-level metadata files.
	metadata := NewMetadata(pipestancePath)
	metadata.writeRaw("invocation", src)
	metadata.writeRaw("mrosource", self.srcTable[pipestance.Node().name])
	metadata.writeRaw("codeversion", self.CodeVersion)
	metadata.writeTime("timestamp")

	// Create pipestance folder graph concurrently.
	var wg sync.WaitGroup
	pipestance.Node().mkdirs(&wg)
	wg.Wait()

	return pipestance, nil
}

// Reattaches to an existing pipestance.
func (self *Runtime) Reattach(psid string, pipestancePath string) (*Pipestance, error) {
	// Read in the existing _invocation file.
	bytes, err := ioutil.ReadFile(path.Join(pipestancePath, "_invocation"))
	if err != nil {
		return nil, err
	}

	// Instantiate the pipestance.
	return self.instantiate(psid, string(bytes), pipestancePath)
}

func (self *Runtime) GetSerialization(pipestancePath string) (interface{}, bool) {
	metadata := NewMetadata(pipestancePath)
	metadata.cache()
	if metadata.exists("finalstate") {
		return metadata.read("finalstate"), true
	}
	return nil, false
}

func (self *Runtime) buildVal(param Param, val interface{}) string {
	if param.IsFile() {
		return fmt.Sprintf("file(\"%s\")", val)
	}
	if param.Tname() == "path" {
		return fmt.Sprintf("path(\"%s\")", val)
	}
	if param.Tname() == "string" {
		return fmt.Sprintf("\"%s\"", val)
	}
	if param.Tname() == "float" {
		return fmt.Sprintf("%f", val)
	}
	if param.Tname() == "int" {
		if fval, ok := val.(float64); ok {
			return fmt.Sprintf("%d", int(fval))
		}
	}
	if param.Tname() == "bool" {
		return fmt.Sprintf("%t", val)
	}
	return fmt.Sprintf("%v", val)
}

func (self *Runtime) BuildCallSource(pname string, args map[string]interface{}) string {
	lines := []string{}
	for _, param := range self.globalTable[pname].callables.table[pname].InParams().list {
		valstr := ""
		val, ok := args[param.Id()]
		if !ok || val == nil {
			valstr = "null"
		} else if reflect.TypeOf(val).Kind() == reflect.Slice {
			a := []string{}
			slice := reflect.ValueOf(val)
			for i := 0; i < slice.Len(); i++ {
				v := slice.Index(i).Interface()
				a = append(a, self.buildVal(param, v))
			}
			valstr = fmt.Sprintf("[%s]", strings.Join(a, ", "))
		} else {
			valstr = self.buildVal(param, val)
		}
		lines = append(lines, fmt.Sprintf("    %s = %s,", param.Id(), valstr))
	}
	return fmt.Sprintf("call %s(\n%s\n)", pname, strings.Join(lines, "\n"))
}

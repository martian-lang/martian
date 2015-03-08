//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian runtime. This is where the action happens.
//
package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

//=============================================================================
// Metadata
//=============================================================================
type Metadata struct {
	fqname    string
	path      string
	contents  map[string]bool
	filesPath string
	mutex     *sync.Mutex
}

type MetadataInfo struct {
	Path  string   `json:"path"`
	Names []string `json:"names"`
}

func NewMetadata(fqname string, p string) *Metadata {
	self := &Metadata{}
	self.fqname = fqname
	self.path = p
	self.contents = map[string]bool{}
	self.filesPath = path.Join(p, "files")
	self.mutex = &sync.Mutex{}
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

func (self *Metadata) getState(name string) (string, bool) {
	if self.exists("errors") {
		return "failed", true
	}
	if self.exists("assert") {
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

func (self *Metadata) cache(name string) {
	self.mutex.Lock()
	self.contents[name] = true
	self.mutex.Unlock()
}

func (self *Metadata) uncache(name string) {
	self.mutex.Lock()
	delete(self.contents, name)
	self.mutex.Unlock()
}

func (self *Metadata) loadCache() {
	paths := self.glob()
	self.mutex.Lock()
	self.contents = map[string]bool{}
	for _, p := range paths {
		self.contents[path.Base(p)[1:]] = true
	}
	self.mutex.Unlock()
}

func (self *Metadata) makePath(name string) string {
	return path.Join(self.path, "_"+name)
}
func (self *Metadata) exists(name string) bool {
	self.mutex.Lock()
	_, ok := self.contents[name]
	self.mutex.Unlock()
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
	self.cache(name)
}
func (self *Metadata) write(name string, object interface{}) {
	bytes, _ := json.MarshalIndent(object, "", "    ")
	self.writeRaw(name, string(bytes))
}
func (self *Metadata) writeTime(name string) {
	self.writeRaw(name, Timestamp())
}
func (self *Metadata) remove(name string) { os.Remove(self.makePath(name)) }

func (self *Metadata) serializeState() *MetadataInfo {
	names := []string{}
	self.mutex.Lock()
	for content, _ := range self.contents {
		names = append(names, content)
	}
	self.mutex.Unlock()
	sort.Strings(names)
	return &MetadataInfo{
		Path:  self.path,
		Names: names,
	}
}

func (self *Metadata) serializePerf(numThreads int) *PerfInfo {
	if self.exists("complete") && self.exists("jobinfo") {
		data := self.readRaw("jobinfo")

		var jobInfo *JobInfo
		if err := json.Unmarshal([]byte(data), &jobInfo); err == nil {
			fpaths, _ := self.enumerateFiles()
			return reduceJobInfo(jobInfo, fpaths, numThreads)
		}
	}
	return nil
}

//=============================================================================
// Binding
//=============================================================================
type Binding struct {
	node       *Node
	id         string
	tname      string
	sweep      bool
	waiting    bool
	valexp     string
	mode       string
	parentNode Nodable
	boundNode  Nodable
	output     string
	value      interface{}
}

type BindingInfo struct {
	Id          string      `json:"id"`
	Type        string      `json:"type"`
	ValExp      string      `json:"valexp"`
	Mode        string      `json:"mode"`
	Output      string      `json:"output"`
	Sweep       bool        `json:"bool"`
	Node        interface{} `json:"node"`
	MatchedFork interface{} `json:"matchedFork"`
	Value       interface{} `json:"value"`
	Waiting     bool        `json:"waiting"`
}

func newBinding(node *Node, bindStm *BindStm, returnBinding bool) *Binding {
	self := &Binding{}
	self.node = node
	self.id = bindStm.id
	self.tname = bindStm.tname
	self.sweep = bindStm.sweep
	self.waiting = false
	switch valueExp := bindStm.exp.(type) {
	case *RefExp:
		if valueExp.kind == "self" {
			var parentBinding *Binding
			if returnBinding {
				parentBinding = self.node.argbindings[valueExp.id]
			} else {
				parentBinding = self.node.parent.getNode().argbindings[valueExp.id]
			}
			if parentBinding != nil {
				self.node = parentBinding.node
				self.tname = parentBinding.tname
				self.sweep = parentBinding.sweep
				self.waiting = parentBinding.waiting
				self.mode = parentBinding.mode
				self.parentNode = parentBinding.parentNode
				self.boundNode = parentBinding.boundNode
				self.output = parentBinding.output
				self.value = parentBinding.value
			}
			self.id = bindStm.id
			self.valexp = "self." + valueExp.id
		} else if valueExp.kind == "call" {
			if returnBinding {
				self.parentNode = self.node.subnodes[valueExp.id]
				self.boundNode, self.output, self.mode, self.value = self.node.findBoundNode(
					valueExp.id, valueExp.outputId, "reference", nil)
			} else {
				self.parentNode = self.node.parent.getNode().subnodes[valueExp.id]
				self.boundNode, self.output, self.mode, self.value = self.node.parent.getNode().findBoundNode(
					valueExp.id, valueExp.outputId, "reference", nil)
			}
			if valueExp.outputId == "default" {
				self.valexp = valueExp.id
			} else {
				self.valexp = valueExp.id + "." + valueExp.outputId
			}
		}
	case *ValExp:
		self.mode = "value"
		self.parentNode = node
		self.boundNode = node
		self.value = expToInterface(bindStm.exp)
	}
	return self
}

func NewBinding(node *Node, bindStm *BindStm) *Binding {
	return newBinding(node, bindStm, false)
}

func NewReturnBinding(node *Node, bindStm *BindStm) *Binding {
	return newBinding(node, bindStm, true)
}

func expToInterface(exp Exp) interface{} {
	// Convert tree of Exps into a tree of interface{}s.
	valExp, ok := exp.(*ValExp)
	if !ok {
		return nil
	}
	if valExp.kind == "array" {
		varray := []interface{}{}
		for _, exp := range valExp.value.([]Exp) {
			varray = append(varray, expToInterface(exp))
		}
		return varray
	} else if valExp.kind == "map" {
		vmap := map[string]interface{}{}
		for k, exp := range valExp.value.(map[string]Exp) {
			vmap[k] = expToInterface(exp)
		}
		return vmap
	} else {
		return valExp.value
	}
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
		matchedFork := self.boundNode.getNode().matchFork(argPermute)
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

func (self *Binding) serializeState(argPermute map[string]interface{}) *BindingInfo {
	var node interface{} = nil
	var matchedFork interface{} = nil
	if self.boundNode != nil {
		node = self.boundNode.getNode().name
		f := self.boundNode.getNode().matchFork(argPermute)
		if f != nil {
			matchedFork = f.index
		}
	}
	return &BindingInfo{
		Id:          self.id,
		Type:        self.tname,
		ValExp:      self.valexp,
		Mode:        self.mode,
		Output:      self.output,
		Sweep:       self.sweep,
		Node:        node,
		MatchedFork: matchedFork,
		Value:       self.resolve(argPermute),
		Waiting:     self.waiting,
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
		if param.getIsFile() {
			args[id] = path.Join(filesPath, param.getId()+"."+param.getTname())
		} else if param.getTname() == "path" {
			args[id] = path.Join(filesPath, param.getId())
		} else {
			args[id] = nil
		}
	}
	return args
}

func dynamicCast(val interface{}, typename string, arrayDim int) bool {
	ret := true
	if arrayDim > 0 {
		arr, ok := val.([]interface{})
		if !ok {
			return false
		}
		for _, v := range arr {
			ret = ret && dynamicCast(v, typename, arrayDim-1)
		}
	} else {
		switch typename {
		case "path":
			fallthrough
		case "file":
			fallthrough
		case "string":
			_, ret = val.(string)
		case "float":
			_, ret = val.(float64)
		case "int":
			var num float64
			num, ret = val.(float64)
			if ret {
				ret = num == math.Trunc(num)
			}
		case "bool":
			_, ret = val.(bool)
		case "map":
			_, ret = val.(map[string]interface{})
		}
	}
	return ret
}

func VerifyVDRMode(vdrMode string) {
	validModes := []string{"rolling", "post", "disable"}
	for _, validMode := range validModes {
		if validMode == vdrMode {
			return
		}
	}
	LogInfo("runtime", "Invalid VDR mode: %s. Valid VDR modes: %s", vdrMode, strings.Join(validModes, ", "))
	os.Exit(1)
}

func VerifyProfileMode(profileMode string) {
	validModes := []string{"cpu", "mem", "disable"}
	for _, validMode := range validModes {
		if validMode == profileMode {
			return
		}
	}
	LogInfo("runtime", "Invalid profile mode: %s. Valid profile modes: %s", profileMode, strings.Join(validModes, ", "))
	os.Exit(1)
}

//=============================================================================
// Chunk
//=============================================================================
type Chunk struct {
	node       *Node
	fork       *Fork
	index      int
	chunkDef   map[string]interface{}
	path       string
	fqname     string
	metadata   *Metadata
	hasBeenRun bool
}

type ChunkInfo struct {
	Index    int                    `json:"index"`
	ChunkDef map[string]interface{} `json:"chunkDef"`
	State    string                 `json:"state"`
	Metadata *MetadataInfo          `json:"metadata"`
}

func NewChunk(nodable Nodable, fork *Fork, index int, chunkDef map[string]interface{}) *Chunk {
	self := &Chunk{}
	self.node = nodable.getNode()
	self.fork = fork
	self.index = index
	self.chunkDef = chunkDef
	self.path = path.Join(fork.path, fmt.Sprintf("chnk%d", index))
	self.fqname = fork.fqname + fmt.Sprintf(".chnk%d", index)
	self.metadata = NewMetadata(self.fqname, self.path)
	self.hasBeenRun = false
	if !self.node.split {
		// If we're not splitting, just set the sole chunk's filesPath
		// to the filesPath of the parent fork, to save a pseudo-join copy.
		self.metadata.filesPath = self.fork.metadata.filesPath
	}
	return self
}

func (self *Chunk) mkdirs() {
	self.metadata.mkdirs()
}

func (self *Chunk) getState() string {
	if state, ok := self.metadata.getState(""); ok {
		return state
	} else {
		return "ready"
	}
}

func (self *Chunk) updateState(state string) {
	self.metadata.cache(state)
}

func (self *Chunk) step() {
	if self.getState() != "ready" {
		return
	}

	// Belt and suspenders for not double-submitting a job.
	if self.hasBeenRun {
		return
	} else {
		self.hasBeenRun = true
	}

	threads, memGB := self.node.getJobReqs(self.chunkDef)

	// Resolve input argument bindings and merge in the chunk defs.
	resolvedBindings := resolveBindings(self.node.argbindings, self.fork.argPermute)
	for id, value := range self.chunkDef {
		resolvedBindings[id] = value
	}

	// Write out input and ouput args for the chunk.
	self.metadata.write("args", resolvedBindings)
	self.metadata.write("outs", makeOutArgs(self.node.outparams, self.metadata.filesPath))

	// Run the chunk.
	self.node.runChunk(self.fqname, self.metadata, threads, memGB)
}

func (self *Chunk) serializeState() *ChunkInfo {
	return &ChunkInfo{
		Index:    self.index,
		ChunkDef: self.chunkDef,
		State:    self.getState(),
		Metadata: self.metadata.serializeState(),
	}
}

func (self *Chunk) serializePerf() *ChunkPerfInfo {
	numThreads := 1
	if v, ok := self.chunkDef["__threads"].(float64); ok {
		numThreads = int(v)
	}
	stats := self.metadata.serializePerf(numThreads)
	return &ChunkPerfInfo{
		Index:      self.index,
		ChunkStats: stats,
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
	subforks       []*Fork
	chunks         []*Chunk
	split_has_run  bool
	join_has_run   bool
	argPermute     map[string]interface{}
	stageDefs      *StageDefs
}

type ForkInfo struct {
	Index         int                    `json:"index"`
	ArgPermute    map[string]interface{} `json:"argPermute"`
	State         string                 `json:"state"`
	Metadata      *MetadataInfo          `json:"metadata"`
	SplitMetadata *MetadataInfo          `json:"split_metadata"`
	JoinMetadata  *MetadataInfo          `json:"join_metadata"`
	Chunks        []*ChunkInfo           `json:"chunks"`
	Bindings      *ForkBindingsInfo      `json:"bindings"`
}

type ForkBindingsInfo struct {
	Argument []*BindingInfo `json:"Argument"`
	Return   []*BindingInfo `json:"Return"`
}

func NewFork(nodable Nodable, index int, argPermute map[string]interface{}) *Fork {
	self := &Fork{}
	self.node = nodable.getNode()
	self.index = index
	self.path = path.Join(self.node.path, fmt.Sprintf("fork%d", index))
	self.fqname = self.node.fqname + fmt.Sprintf(".fork%d", index)
	self.metadata = NewMetadata(self.fqname, self.path)
	self.split_metadata = NewMetadata(self.fqname+".split", path.Join(self.path, "split"))
	self.join_metadata = NewMetadata(self.fqname+".join", path.Join(self.path, "join"))
	self.argPermute = argPermute
	self.split_has_run = false
	self.join_has_run = false
	self.subforks = []*Fork{}
	self.chunks = []*Chunk{}
	if err := json.Unmarshal([]byte(self.split_metadata.readRaw("stage_defs")), &self.stageDefs); err == nil {
		for i, chunkDef := range self.stageDefs.ChunkDefs {
			chunk := NewChunk(self.node, self, i, chunkDef)
			self.chunks = append(self.chunks, chunk)
		}
	} else {
		// This makes Martian backwards compatible with chunk_defs. Otherwise, Kepler generates an incorrect
		// performance report. Remove this after sufficient time has passed.
		var chunkDefs []map[string]interface{}
		if err := json.Unmarshal([]byte(self.split_metadata.readRaw("chunk_defs")), &chunkDefs); err == nil {
			joinDef := map[string]interface{}{}
			self.stageDefs = &StageDefs{chunkDefs, joinDef}
			for i, chunkDef := range self.stageDefs.ChunkDefs {
				chunk := NewChunk(self.node, self, i, chunkDef)
				self.chunks = append(self.chunks, chunk)
			}
		}
	}
	return self
}

func (self *Fork) reset() {
	self.chunks = []*Chunk{}
	self.split_has_run = false
	self.join_has_run = false
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

	for _, chunk := range self.chunks {
		chunk.mkdirs()
	}
}

func (self *Fork) verifyOutput() (bool, string) {
	outparams := self.node.outparams
	msg := ""
	ret := true
	if len(outparams.list) > 0 {
		outputs := self.metadata.read("outs").(map[string]interface{})
		for _, param := range outparams.table {
			val, ok := outputs[param.getId()]
			if !ok {
				msg += fmt.Sprintf("Fork did not return parameter '%s'\n", param.getId())
				ret = false
				continue
			}
			if val == nil {
				// Allow for null output parameters
				continue
			}
			if !dynamicCast(val, param.getTname(), param.getArrayDim()) {
				msg += fmt.Sprintf("Fork returned %s parameter '%s' with incorrect type\n", param.getTname(), param.getId())
				ret = false
			}
		}
	}
	return ret, msg
}

func (self *Fork) getState() string {
	if state, _ := self.metadata.getState(""); state == "failed" {
		return "failed"
	}
	if state, _ := self.metadata.getState(""); state == "complete" {
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

func (self *Fork) updateState(state string) {
	if strings.HasPrefix(state, "split_") {
		self.split_metadata.cache(strings.TrimPrefix(state, "split_"))
	} else if strings.HasPrefix(state, "join_") {
		self.join_metadata.cache(strings.TrimPrefix(state, "join_"))
	} else {
		self.metadata.cache(state)
	}
}

func (self *Fork) getChunk(index int) *Chunk {
	if index < len(self.chunks) {
		return self.chunks[index]
	}
	return nil
}

type StageDefs struct {
	ChunkDefs []map[string]interface{} `json:"chunks"`
	JoinDef   map[string]interface{}   `json:"join"`
}

func (self *Fork) step() {
	if self.node.kind == "stage" {
		state := self.getState()
		if !strings.HasSuffix(state, "_running") && !strings.HasSuffix(state, "_queued") {
			statePad := strings.Repeat(" ", int(math.Max(0, float64(15-len(state)))))
			msg := fmt.Sprintf("(%s)%s %s", state, statePad, self.node.fqname)
			if self.node.preflight {
				LogInfo("runtime", msg)
			} else {
				PrintInfo("runtime", msg)
			}
		}

		if state == "ready" {
			argBindings := resolveBindings(self.node.argbindings, self.argPermute)
			incpaths := self.node.invocation["incpaths"].([]string)
			invocation, _ := self.node.rt.BuildCallSource(incpaths, self.node.name, argBindings)
			self.metadata.writeRaw("invocation", invocation)
			self.split_metadata.write("args", argBindings)
			if self.node.split {
				if !self.split_has_run {
					self.split_has_run = true
					// Default memory to -1 for no limit.
					self.node.runSplit(self.fqname, self.split_metadata)
				}
			} else {
				// Initialize stage defs with one chunk
				stageDefs := &StageDefs{ChunkDefs: []map[string]interface{}{}}
				stageDefs.ChunkDefs = append(stageDefs.ChunkDefs, map[string]interface{}{})
				self.split_metadata.write("stage_defs", stageDefs)
				self.split_metadata.writeTime("complete")
			}
		} else if state == "split_complete" {
			if err := json.Unmarshal([]byte(self.split_metadata.readRaw("stage_defs")), &self.stageDefs); err != nil {
				self.split_metadata.writeRaw("errors",
					"The split method must return a dictionary {'chunks': [chunk def dicts], 'join': join def dict} but did not.\n")
			} else {
				if len(self.chunks) == 0 {
					for i, chunkDef := range self.stageDefs.ChunkDefs {
						chunk := NewChunk(self.node, self, i, chunkDef)
						self.chunks = append(self.chunks, chunk)
						chunk.mkdirs()
					}
				}
				for _, chunk := range self.chunks {
					chunk.step()
				}
			}
		} else if state == "chunks_complete" {
			threads, memGB := self.node.getJobReqs(self.stageDefs.JoinDef)
			resolvedBindings := resolveBindings(self.node.argbindings, self.argPermute)
			for id, value := range self.stageDefs.JoinDef {
				resolvedBindings[id] = value
			}
			self.join_metadata.write("args", resolvedBindings)
			self.join_metadata.write("chunk_defs", self.stageDefs.ChunkDefs)
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
					self.node.runJoin(self.fqname, self.join_metadata, threads, memGB)
				}
			} else {
				self.join_metadata.write("outs", self.chunks[0].metadata.read("outs"))
				self.join_metadata.writeTime("complete")
			}
		} else if state == "join_complete" {
			self.metadata.write("outs", self.join_metadata.read("outs"))
			if ok, msg := self.verifyOutput(); ok {
				self.metadata.writeTime("complete")
			} else {
				self.metadata.writeRaw("errors", msg)
			}
		}

	} else if self.node.kind == "pipeline" {
		self.metadata.write("outs", resolveBindings(self.node.retbindings, self.argPermute))
		if ok, msg := self.verifyOutput(); ok {
			self.metadata.writeTime("complete")
		} else {
			self.metadata.writeRaw("errors", msg)
		}
	}
}

func (self *Fork) getVdrKillReport() (*VDRKillReport, bool) {
	killReport := &VDRKillReport{}
	ok := false
	if self.metadata.exists("vdrkill") {
		data := self.metadata.readRaw("vdrkill")
		if err := json.Unmarshal([]byte(data), &killReport); err == nil {
			ok = true
		}
	}
	return killReport, ok
}

func (self *Fork) vdrKill() *VDRKillReport {
	killReport := &VDRKillReport{}
	if self.node.rt.vdrMode == "disable" {
		return killReport
	}
	if killReport, ok := self.getVdrKillReport(); ok {
		return killReport
	}

	killPaths := []string{}
	// For volatile nodes, kill fork-level files.
	if self.node.volatile {
		if paths, err := self.metadata.enumerateFiles(); err == nil {
			killPaths = append(killPaths, paths...)
		}
		if paths, err := self.split_metadata.enumerateFiles(); err == nil {
			killPaths = append(killPaths, paths...)
		}
		if paths, err := self.join_metadata.enumerateFiles(); err == nil {
			killPaths = append(killPaths, paths...)
		}
	}
	// If the node splits, kill chunk-level files.
	// Must check for split here, otherwise we'll end up deleting
	// output files of non-volatile nodes because single-chunk nodes
	// get their output redirected to the one chunk's files path.
	if self.node.split {
		for _, chunk := range self.chunks {
			if paths, err := chunk.metadata.enumerateFiles(); err == nil {
				killPaths = append(killPaths, paths...)
			}
		}
	}
	// Actually delete the paths.
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
	self.metadata.write("vdrkill", killReport)
	return killReport
}

func (self *Fork) postProcess(outsPath string) {
	params := self.node.outparams.table

	if len(params) == 0 {
		return
	}

	if len(self.node.forks) > 1 {
		outsPath = path.Join(outsPath, fmt.Sprintf("fork%d", self.index))
		Log("\nOutput (fork%d):\n", self.index)
	} else {
		Log("\nOutput:\n")
	}

	outs := map[string]interface{}{}
	if data := self.metadata.read("outs"); data != nil {
		if v, ok := data.(map[string]interface{}); ok {
			outs = v
		}
	}

	for id, param := range params {
		value, ok := outs[id]
		if ok && value != nil {
			if param.getIsFile() || param.getTname() == "path" {
				if filePath, ok := value.(string); ok {
					if _, err := os.Stat(filePath); err == nil {
						mkdirAll(outsPath)
						newValue := path.Join(outsPath, id)
						if param.getTname() != "path" {
							newValue += "." + param.getTname()
						}
						os.Symlink(filePath, newValue)
						value = newValue
					}
				}
			}
		} else {
			value = "null"
		}
		key := param.getHelp()
		if len(key) == 0 {
			key = param.getId()
		}
		Log("- %s: %v\n", key, value)
	}
	Log("\n")

	if alarms := self.getAlarms(); len(alarms) > 0 {
		if len(self.node.forks) > 1 {
			Log("\nAlarms (fork%d):\n", self.index)
		} else {
			Log("\nAlarms:\n")
		}
		Log(alarms + "\n")
	}
}

func (self *Fork) getAlarms() string {
	alarms := ""
	for _, metadata := range self.collectMetadatas() {
		if !metadata.exists("alarm") {
			continue
		}
		alarms += metadata.readRaw("alarm")
	}
	for _, subfork := range self.subforks {
		alarms += subfork.getAlarms()
	}
	return alarms
}

func (self *Fork) serializeState() *ForkInfo {
	argbindings := []*BindingInfo{}
	for _, argbinding := range self.node.argbindingList {
		argbindings = append(argbindings, argbinding.serializeState(self.argPermute))
	}
	retbindings := []*BindingInfo{}
	for _, retbinding := range self.node.retbindingList {
		retbindings = append(retbindings, retbinding.serializeState(self.argPermute))
	}
	bindings := &ForkBindingsInfo{
		Argument: argbindings,
		Return:   retbindings,
	}
	chunks := []*ChunkInfo{}
	for _, chunk := range self.chunks {
		chunks = append(chunks, chunk.serializeState())
	}
	return &ForkInfo{
		Index:         self.index,
		ArgPermute:    self.argPermute,
		State:         self.getState(),
		Metadata:      self.metadata.serializeState(),
		SplitMetadata: self.split_metadata.serializeState(),
		JoinMetadata:  self.join_metadata.serializeState(),
		Chunks:        chunks,
		Bindings:      bindings,
	}
}

func (self *Fork) getStages() []*StagePerfInfo {
	stages := []*StagePerfInfo{}
	for _, subfork := range self.subforks {
		stages = append(stages, subfork.getStages()...)
	}
	if self.node.kind == "stage" {
		stages = append(stages, &StagePerfInfo{
			Name:   self.node.name,
			Fqname: self.node.fqname,
			Forki:  self.index,
		})
	}
	return stages
}

func (self *Fork) serializePerf() (*ForkPerfInfo, *VDRKillReport) {
	chunks := []*ChunkPerfInfo{}
	stats := []*PerfInfo{}

	for _, chunk := range self.chunks {
		chunkSer := chunk.serializePerf()
		chunks = append(chunks, chunkSer)
		if chunkSer.ChunkStats != nil {
			stats = append(stats, chunkSer.ChunkStats)
		}
	}
	numThreads := 1
	splitStats := self.split_metadata.serializePerf(numThreads)
	joinStats := self.join_metadata.serializePerf(numThreads)
	if splitStats != nil {
		stats = append(stats, splitStats)
	}
	if joinStats != nil {
		stats = append(stats, joinStats)
	}

	killReport, _ := self.getVdrKillReport()
	killReports := []*VDRKillReport{killReport}
	for _, subfork := range self.subforks {
		subforkSer, subforkKillReport := subfork.serializePerf()
		stats = append(stats, subforkSer.ForkStats)
		killReports = append(killReports, subforkKillReport)
	}
	killReport = mergeVDRKillReports(killReports)

	forkStats := &PerfInfo{}
	if len(stats) > 0 {
		forkStats = ComputeStats(stats, killReport)
	}
	return &ForkPerfInfo{
		Stages:     self.getStages(),
		Index:      self.index,
		Chunks:     chunks,
		SplitStats: splitStats,
		JoinStats:  joinStats,
		ForkStats:  forkStats,
	}, killReport
}

//=============================================================================
// Node
//=============================================================================
type Nodable interface {
	getNode() *Node
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
	directPrenodes []Nodable
	postnodes      map[string]Nodable
	frontierNodes  map[string]Nodable
	forks          []*Fork
	split          bool
	state          string
	volatile       bool
	local          bool
	preflight      bool
	stagecodeLang  string
	stagecodeCmd   string
	journalPath    string
	tmpPath        string
	invocation     map[string]interface{}
}

type NodeInfo struct {
	Name          string         `json:"name"`
	Fqname        string         `json:"fqname"`
	Type          string         `json:"type"`
	Path          string         `json:"path"`
	State         string         `json:"state"`
	Metadata      *MetadataInfo  `json:"metadata"`
	SweepBindings []*BindingInfo `json:"sweepbindings"`
	Forks         []*ForkInfo    `json:"forks"`
	Edges         []interface{}  `json:"edges"`
	StagecodeLang string         `json:"stagecodeLang"`
	StagecodeCmd  string         `json:"stagecodeCmd"`
	Error         interface{}    `json:"error"`
}

func (self *Node) getNode() *Node { return self }

func NewNode(parent Nodable, kind string, callStm *CallStm, callables *Callables) *Node {
	self := &Node{}
	self.parent = parent

	self.rt = parent.getNode().rt
	self.kind = kind
	self.name = callStm.id
	self.fqname = parent.getNode().fqname + "." + self.name
	self.path = path.Join(parent.getNode().path, self.name)
	self.journalPath = parent.getNode().journalPath
	self.tmpPath = parent.getNode().tmpPath
	self.invocation = parent.getNode().invocation
	self.metadata = NewMetadata(self.fqname, self.path)
	self.volatile = callStm.modifiers.volatile
	self.local = callStm.modifiers.local
	self.preflight = callStm.modifiers.preflight

	self.outparams = callables.table[self.name].getOutParams()
	self.argbindings = map[string]*Binding{}
	self.argbindingList = []*Binding{}
	self.retbindings = map[string]*Binding{}
	self.retbindingList = []*Binding{}
	self.subnodes = map[string]Nodable{}
	self.prenodes = map[string]Nodable{}
	self.directPrenodes = []Nodable{}
	self.postnodes = map[string]Nodable{}
	self.frontierNodes = parent.getNode().frontierNodes

	for id, bindStm := range callStm.bindings.table {
		binding := NewBinding(self, bindStm)
		self.argbindings[id] = binding
		self.argbindingList = append(self.argbindingList, binding)
	}
	for _, binding := range self.argbindingList {
		if binding.mode == "reference" && binding.boundNode != nil {
			prenode := binding.boundNode
			self.prenodes[prenode.getNode().fqname] = prenode
			self.directPrenodes = append(self.directPrenodes, binding.parentNode)

			prenode.getNode().postnodes[self.fqname] = self
		}
	}
	// Do not set state = getState here, or else nodes will wrongly report
	// complete before the first refreshMetadata call
	return self
}

//
// Folder construction
//
func (self *Node) mkdirs() {
	mkdirAll(self.path)
	mkdir(self.journalPath)
	mkdir(self.tmpPath)

	var wg sync.WaitGroup
	for _, fork := range self.forks {
		wg.Add(1)
		go func(f *Fork) {
			f.mkdirs()
			wg.Done()
		}(fork)
	}
	wg.Wait()
}

//
// Sweep management
//
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
		for _, binding := range prenode.getNode().sweepbindings {
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

	for _, fork := range self.forks {
		for _, subnode := range self.subnodes {
			matchedFork := subnode.getNode().matchFork(fork.argPermute)
			fork.subforks = append(fork.subforks, matchedFork)
		}
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

//
// Subnode management
//
func (self *Node) setPrenode(prenode Nodable) {
	for _, subnode := range self.subnodes {
		subnode.getNode().setPrenode(prenode)
	}
	self.prenodes[prenode.getNode().fqname] = prenode
	prenode.getNode().postnodes[self.fqname] = self
}

func (self *Node) findBoundNode(id string, outputId string, mode string,
	value interface{}) (Nodable, string, string, interface{}) {
	if self.kind == "pipeline" {
		subnode := self.subnodes[id]
		for _, binding := range subnode.getNode().retbindings {
			if binding.id == outputId {
				return binding.boundNode, binding.output, binding.mode, binding.value
			}
		}
		return subnode, outputId, mode, value
	}
	return self, outputId, mode, value
}

func (self *Node) addFrontierNode(node Nodable) {
	self.frontierNodes[node.getNode().fqname] = node
}

func (self *Node) removeFrontierNode(node Nodable) {
	delete(self.frontierNodes, node.getNode().fqname)
}

func (self *Node) getFrontierNodes() []*Node {
	frontierNodes := []*Node{}
	for _, node := range self.frontierNodes {
		frontierNodes = append(frontierNodes, node.getNode())
	}
	return frontierNodes
}

func (self *Node) allNodes() []*Node {
	all := []*Node{self}

	// Enumerate and sort the keys in subnodes first.
	// This ensures a stable chirality for the dag UI.
	ids := []string{}
	for id, _ := range self.subnodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Build a list of all subnodes.
	for _, id := range ids {
		subnode := self.subnodes[id]
		all = append(all, subnode.getNode().allNodes()...)
	}
	return all
}

func (self *Node) find(fqname string) *Node {
	if self.fqname == fqname {
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
	metadatas := []*Metadata{self.metadata}
	for _, fork := range self.forks {
		metadatas = append(metadatas, fork.collectMetadatas()...)
	}
	return metadatas
}

func (self *Node) loadMetadata() {
	metadatas := self.collectMetadatas()
	for _, metadata := range metadatas {
		metadata.loadCache()
	}
	self.state = self.getState()
	self.addFrontierNode(self)
}

func (self *Node) getFork(index int) *Fork {
	if index < len(self.forks) {
		return self.forks[index]
	}
	return nil
}

func (self *Node) getState() string {
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
		if prenode.getNode().getState() != "complete" {
			return "waiting"
		}
	}
	// Otherwise we're running.
	return "running"
}

func (self *Node) reset() error {
	PrintInfo("runtime", "(reset)           %s", self.fqname)

	// Blow away the entire stage node.
	if err := os.RemoveAll(self.path); err != nil {
		PrintInfo("runtime", "mrp cannot reset the stage because its folder contents could not be deleted. Error was:\n\n%s\n\nPlease resolve the error in order to continue running the pipeline.", err.Error())
		return err
	}
	// Remove all related files from journal directory.
	if files, err := filepath.Glob(path.Join(self.journalPath, self.fqname+"*")); err == nil {
		for _, file := range files {
			os.Remove(file)
		}
	}

	// Clear chunks in the forks so they can be rebuilt on split.
	for _, fork := range self.forks {
		fork.reset()
	}

	// Create stage node directories.
	self.mkdirs()

	// Load the metadata.
	self.loadMetadata()

	return nil
}

func (self *Node) resetJobMonitors() {
	for _, metadata := range self.collectMetadatas() {
		state, _ := metadata.getState("")
		if state == "running" || state == "queued" {
			self.rt.JobManager.MonitorJob(metadata)
		}
	}
}

func (self *Node) kill() {
	for _, metadata := range self.collectMetadatas() {
		if state, _ := metadata.getState(""); state == "failed" {
			continue
		}
		metadata.writeRaw("errors", "Job was killed by Martian.")
	}
}

func (self *Node) postProcess() {
	os.RemoveAll(self.journalPath)
	os.RemoveAll(self.tmpPath)

	pipestanceOutsPath := path.Join(self.parent.getNode().path, "outs")
	for _, fork := range self.forks {
		fork.postProcess(pipestanceOutsPath)
	}
}

func (self *Node) getFatalError() (string, string, string, string, []string) {
	for _, metadata := range self.collectMetadatas() {
		if state, _ := metadata.getState(""); state != "failed" {
			continue
		}
		if metadata.exists("errors") {
			errlog := metadata.readRaw("errors")
			summary := "<none>"
			if self.stagecodeLang == "Python" {
				errlines := strings.Split(errlog, "\n")
				if len(errlines) >= 2 {
					summary = errlines[len(errlines)-2]
				}
			}
			errpaths := []string{
				metadata.makePath("errors"),
				metadata.makePath("stdout"),
				metadata.makePath("stderr"),
			}
			if self.rt.enableStackVars {
				errpaths = append(errpaths, metadata.makePath("stackvars"))
			}
			return metadata.fqname, summary, errlog, "errors", errpaths
		}
		if metadata.exists("assert") {
			assertlog := metadata.readRaw("assert")
			summary := "<none>"
			assertlines := strings.Split(assertlog, "\n")
			if len(assertlines) >= 1 {
				summary = assertlines[len(assertlines)-1]
			}
			return metadata.fqname, summary, assertlog, "assert", []string{
				metadata.makePath("assert"),
			}
		}
	}
	return "", "", "", "", []string{}
}

func (self *Node) step() {
	if self.state == "running" {
		for _, fork := range self.forks {
			fork.step()
		}
	}
	previousState := self.state
	self.state = self.getState()
	switch self.state {
	case "failed":
		self.addFrontierNode(self)
	case "running":
		if self.state != previousState {
			self.mkdirs()
		}
		self.addFrontierNode(self)
	case "complete":
		if self.rt.vdrMode == "rolling" {
			for _, node := range self.prenodes {
				node.getNode().vdrKill()
			}
			self.vdrKill()
		}
		for _, node := range self.postnodes {
			self.addFrontierNode(node)
		}
		self.removeFrontierNode(self)
	case "waiting":
		self.removeFrontierNode(self)
	}
}

func (self *Node) parseRunFilename(fqname string) (string, int, int, string) {
	r := regexp.MustCompile("(.*)\\.fork(\\d+)\\.chnk(\\d+)\\.(.*)$")
	if match := r.FindStringSubmatch(fqname); match != nil {
		forkIndex, _ := strconv.Atoi(match[2])
		chunkIndex, _ := strconv.Atoi(match[3])
		return match[1], forkIndex, chunkIndex, match[4]
	}
	r = regexp.MustCompile("(.*)\\.fork(\\d+)\\.(.*)$")
	if match := r.FindStringSubmatch(fqname); match != nil {
		forkIndex, _ := strconv.Atoi(match[2])
		return match[1], forkIndex, -1, match[3]
	}
	return "", -1, -1, ""
}

func (self *Node) refreshState() {
	files, _ := filepath.Glob(path.Join(self.journalPath, "*"))
	for _, file := range files {
		filename := path.Base(file)
		if strings.HasSuffix(filename, ".tmp") {
			continue
		}

		fqname, forkIndex, chunkIndex, state := self.parseRunFilename(filename)
		if node := self.find(fqname); node != nil {
			if fork := node.getFork(forkIndex); fork != nil {
				if chunkIndex >= 0 {
					if chunk := fork.getChunk(chunkIndex); chunk != nil {
						chunk.updateState(state)
					}
				} else {
					fork.updateState(state)
				}
			}
		}
		os.Remove(file)
	}
}

//
// VDR
//
type VDRKillReport struct {
	Count  uint     `json:"count"`
	Size   uint64   `json:"size"`
	Paths  []string `json:"paths"`
	Errors []string `json:"errors"`
}

func mergeVDRKillReports(killReports []*VDRKillReport) *VDRKillReport {
	allKillReport := &VDRKillReport{}
	for _, killReport := range killReports {
		allKillReport.Size += killReport.Size
		allKillReport.Count += killReport.Count
		allKillReport.Errors = append(allKillReport.Errors, killReport.Errors...)
		allKillReport.Paths = append(allKillReport.Paths, killReport.Paths...)
	}
	return allKillReport
}

func (self *Node) vdrKill() *VDRKillReport {
	killReports := []*VDRKillReport{}
	every := true
	for _, node := range self.postnodes {
		if node.getNode().state != "complete" {
			every = false
		}
	}
	if every {
		for _, fork := range self.forks {
			killReports = append(killReports, fork.vdrKill())
		}
	}
	return mergeVDRKillReports(killReports)
}

//
// Serialization
//
func (self *Node) serializeState() *NodeInfo {
	sweepbindings := []*BindingInfo{}
	for _, sweepbinding := range self.sweepbindings {
		sweepbindings = append(sweepbindings, sweepbinding.serializeState(nil))
	}
	forks := []*ForkInfo{}
	for _, fork := range self.forks {
		forks = append(forks, fork.serializeState())
	}
	edges := []interface{}{}
	for _, prenode := range self.directPrenodes {
		edges = append(edges, map[string]string{
			"from": prenode.getNode().fqname,
			"to":   self.fqname,
		})
	}
	var err interface{} = nil
	if self.state == "failed" {
		fqname, summary, log, _, errpaths := self.getFatalError()
		errpath := ""
		if len(errpaths) > 0 {
			errpath = errpaths[0]
		}
		err = map[string]string{
			"fqname":  fqname,
			"path":    errpath,
			"summary": summary,
			"log":     log,
		}
	}
	return &NodeInfo{
		Name:          self.name,
		Fqname:        self.fqname,
		Type:          self.kind,
		Path:          self.path,
		State:         self.state,
		Metadata:      self.metadata.serializeState(),
		SweepBindings: sweepbindings,
		Forks:         forks,
		Edges:         edges,
		StagecodeLang: self.stagecodeLang,
		StagecodeCmd:  self.stagecodeCmd,
		Error:         err,
	}
}

func (self *Node) serializePerf() *NodePerfInfo {
	forks := []*ForkPerfInfo{}
	for _, fork := range self.forks {
		forkSer, _ := fork.serializePerf()
		forks = append(forks, forkSer)
	}
	return &NodePerfInfo{
		Name:   self.name,
		Fqname: self.fqname,
		Type:   self.kind,
		Forks:  forks,
	}
}

//=============================================================================
// Job Runners
//=============================================================================
func (self *Node) getJobReqs(jobDef map[string]interface{}) (int, int) {
	threads := -1
	if v, ok := jobDef["__threads"].(float64); ok {
		threads = int(v)

		// In local mode, cap to the job manager's max cores.
		// It is not sufficient for the job manager to do the capping downstream.
		// We rewrite the chunkDef here to inform the chunk it should use less
		// concurrency.
		if self.rt.jobMode == "local" {
			maxCores := self.rt.JobManager.GetMaxCores()
			if threads > maxCores {
				threads = maxCores
			}
			jobDef["__threads"] = threads
		}
	}

	// Default to -1 to impose no limit (no flag will be passed to SGE).
	// The local mode job manager will convert -1 to 1 downstream.
	memGB := -1
	if v, ok := jobDef["__mem_gb"].(float64); ok {
		memGB = int(v)

		if self.rt.jobMode == "local" {
			maxMemGB := self.rt.JobManager.GetMaxMemGB()
			if memGB > maxMemGB {
				memGB = maxMemGB
			}
			jobDef["__mem_gb"] = memGB
		}
	}
	return threads, memGB
}

func (self *Node) runSplit(fqname string, metadata *Metadata) {
	self.runJob("split", fqname, metadata, -1, -1)
}

func (self *Node) runJoin(fqname string, metadata *Metadata, threads int, memGB int) {
	self.runJob("join", fqname, metadata, threads, memGB)
}

func (self *Node) runChunk(fqname string, metadata *Metadata, threads int, memGB int) {
	self.runJob("main", fqname, metadata, threads, memGB)
}

func (self *Node) runJob(shellName string, fqname string, metadata *Metadata,
	threads int, memGB int) {

	// Configure local variable dumping.
	stackVars := "disable"
	if self.rt.enableStackVars {
		stackVars = "stackvars"
	}

	// Set environment variables
	os.Setenv("TMPDIR", self.tmpPath)
	envs := []string{fmt.Sprintf("TMPDIR=%s", self.tmpPath)}

	// Construct path to the shell.
	shellCmd := ""
	argv := []string{}
	stagecodeParts := strings.Split(self.stagecodeCmd, " ")
	runFile := path.Join(self.journalPath, fqname)
	version := map[string]interface{}{
		"martian":   self.rt.martianVersion,
		"pipelines": self.rt.mroVersion,
	}

	switch self.stagecodeLang {
	case "Python":
		shellCmd = path.Join(self.rt.adaptersPath, "python", shellName+".py")
		argv = append(stagecodeParts, metadata.path, metadata.filesPath, runFile)
	case "Executable":
		shellCmd = stagecodeParts[0]
		argv = append(stagecodeParts[1:], shellName, metadata.path, metadata.filesPath, runFile)
	default:
		panic(fmt.Sprintf("Unknown stage code language: %s", self.stagecodeLang))
	}

	// Log the job run.
	jobMode := self.rt.jobMode
	jobManager := self.rt.JobManager
	if self.local {
		jobMode = "local"
		jobManager = self.rt.LocalJobManager
	}
	padding := strings.Repeat(" ", int(math.Max(0, float64(10-len(jobMode)))))
	msg := fmt.Sprintf("(run:%s) %s %s.%s", jobMode, padding, fqname, shellName)
	if self.preflight {
		LogInfo("runtime", msg)
	} else {
		PrintInfo("runtime", msg)
	}

	EnterCriticalSection()
	metadata.write("jobinfo", map[string]interface{}{
		"name":           fqname,
		"type":           jobMode,
		"profile_mode":   self.rt.profileMode,
		"stackvars_flag": stackVars,
		"invocation":     self.invocation,
		"version":        version,
	})
	jobManager.execJob(shellCmd, argv, envs, metadata, threads, memGB, fqname, shellName)
	ExitCriticalSection()
}

//=============================================================================
// Stagestance
//=============================================================================
type Stagestance struct {
	node *Node
}

func NewStagestance(parent Nodable, callStm *CallStm, callables *Callables) *Stagestance {
	langMap := map[string]string{
		"py":   "Python",
		"exec": "Executable",
	}

	self := &Stagestance{}
	self.node = NewNode(parent, "stage", callStm, callables)
	stage, ok := callables.table[self.node.name].(*Stage)
	if !ok {
		return nil
	}

	stagecodePaths := append([]string{self.node.rt.mroPath}, strings.Split(os.Getenv("PATH"), ":")...)
	stagecodePath, _ := searchPaths(stage.src.path, stagecodePaths)
	self.node.stagecodeCmd = strings.Join(append([]string{stagecodePath}, stage.src.args...), " ")
	if self.node.rt.stest {
		switch stage.src.lang {
		case "py":
			self.node.stagecodeCmd = RelPath(path.Join("..", "adapters", "python", "tester"))
		default:
			panic(fmt.Sprintf("Unsupported stress test language: %s", stage.src.lang))
		}
	}
	self.node.stagecodeLang = langMap[stage.src.lang]
	self.node.split = len(stage.splitParams.list) > 0
	self.node.buildForks(self.node.argbindings)
	return self
}

func (self *Stagestance) getNode() *Node   { return self.node }
func (self *Stagestance) GetState() string { return self.getNode().getState() }
func (self *Stagestance) Step()            { self.getNode().step() }
func (self *Stagestance) RefreshState()    { self.getNode().refreshState() }
func (self *Stagestance) LoadMetadata()    { self.getNode().loadMetadata() }
func (self *Stagestance) PostProcess()     { self.getNode().postProcess() }
func (self *Stagestance) VDRKill() *VDRKillReport {
	return self.getNode().vdrKill()
}
func (self *Stagestance) GetFatalError() (string, string, string, string, []string) {
	return self.getNode().getFatalError()
}

//=============================================================================
// Pipestance
//=============================================================================
type Pipestance struct {
	node *Node
}

func NewPipestance(parent Nodable, callStm *CallStm, callables *Callables) *Pipestance {
	self := &Pipestance{}
	self.node = NewNode(parent, "pipeline", callStm, callables)

	// Build subcall tree.
	pipeline, ok := callables.table[self.node.name].(*Pipeline)
	if !ok {
		return nil
	}
	var preflightNode Nodable = nil
	for _, subcallStm := range pipeline.calls {
		callable := callables.table[subcallStm.id]
		switch callable.(type) {
		case *Stage:
			self.node.subnodes[subcallStm.id] = NewStagestance(self.node, subcallStm, callables)
		case *Pipeline:
			self.node.subnodes[subcallStm.id] = NewPipestance(self.node, subcallStm, callables)
		}
		if self.node.subnodes[subcallStm.id].getNode().preflight {
			preflightNode = self.node.subnodes[subcallStm.id]
		}
	}

	// Also depends on stages bound to return values.
	self.node.retbindings = map[string]*Binding{}
	for id, bindStm := range pipeline.ret.bindings.table {
		binding := NewReturnBinding(self.node, bindStm)
		self.node.retbindings[id] = binding
		self.node.retbindingList = append(self.node.retbindingList, binding)
		if binding.mode == "reference" && binding.boundNode != nil {
			prenode := binding.boundNode
			self.node.prenodes[prenode.getNode().fqname] = prenode
			self.node.directPrenodes = append(self.node.directPrenodes, binding.parentNode)

			prenode.getNode().postnodes[self.node.fqname] = self.node
		}
	}
	// Add preflight dependency if preflight stage exists.
	if preflightNode != nil {
		for _, subnode := range self.node.subnodes {
			if subnode != preflightNode {
				subnode.getNode().setPrenode(preflightNode)
			}
		}
	}

	self.node.buildForks(self.node.retbindings)
	return self
}

func (self *Pipestance) getNode() *Node    { return self.node }
func (self *Pipestance) GetPname() string  { return self.node.name }
func (self *Pipestance) GetPsid() string   { return self.node.parent.getNode().name }
func (self *Pipestance) GetFQName() string { return self.node.fqname }
func (self *Pipestance) RefreshState()     { self.node.refreshState() }

func (self *Pipestance) LoadMetadata() {
	// We used to make this concurrent but ended up with too many
	// goroutines (Pranav's 96-sample run).
	for _, node := range self.node.allNodes() {
		node.loadMetadata()
	}
	for _, node := range self.node.allNodes() {
		node.state = node.getState()
		if node.state == "running" {
			node.mkdirs()
		}
	}
}

func (self *Pipestance) GetState() string {
	nodes := self.node.getFrontierNodes()
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

func (self *Pipestance) Kill() {
	nodes := self.node.getFrontierNodes()
	for _, node := range nodes {
		node.kill()
	}
}

func (self *Pipestance) RestartRunningNodes(jobMode string) error {
	self.LoadMetadata()
	nodes := self.node.getFrontierNodes()
	localNodes := []*Node{}
	remoteNodes := []*Node{}
	for _, node := range nodes {
		if node.state == "running" {
			PrintInfo("runtime", "Found orphaned stage: %s", node.fqname)
			if jobMode == "local" || node.local {
				localNodes = append(localNodes, node)
			} else {
				remoteNodes = append(remoteNodes, node)
			}
		}
	}
	for _, node := range localNodes {
		if err := node.reset(); err != nil {
			return err
		}
	}
	for _, node := range remoteNodes {
		node.resetJobMonitors()
	}
	return nil
}

func (self *Pipestance) GetFatalError() (string, string, string, string, []string) {
	nodes := self.node.getFrontierNodes()
	for _, node := range nodes {
		if node.state == "failed" {
			return node.getFatalError()
		}
	}
	return "", "", "", "", []string{}
}

func (self *Pipestance) StepNodes() {
	for _, node := range self.node.getFrontierNodes() {
		node.step()
	}
}

func (self *Pipestance) Reset() error {
	for _, node := range self.node.allNodes() {
		if node.state == "failed" {
			if err := node.reset(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *Pipestance) Serialize(name string) interface{} {
	ser := []interface{}{}
	for _, node := range self.node.allNodes() {
		switch name {
		case "finalstate":
			ser = append(ser, node.serializeState())
		case "perf":
			ser = append(ser, node.serializePerf())
		default:
			panic(fmt.Sprintf("Unsupported serialization type: %s", name))
		}
	}
	return ser
}

func (self *Pipestance) GetPath() string {
	return self.node.parent.getNode().path
}

func (self *Pipestance) GetInvocation() interface{} {
	return self.node.parent.getNode().invocation
}

func (self *Pipestance) PostProcess() {
	self.node.postProcess()
	metadata := NewMetadata(self.node.parent.getNode().fqname, self.GetPath())
	metadata.writeRaw("timestamp", metadata.readRaw("timestamp")+"\nend: "+Timestamp())
	self.Immortalize()
}

func (self *Pipestance) Immortalize() {
	metadata := NewMetadata(self.node.parent.getNode().fqname, self.GetPath())
	metadata.loadCache()
	if !metadata.exists("finalstate") {
		metadata.write("finalstate", self.Serialize("finalstate"))
	}
	if !metadata.exists("perf") {
		metadata.write("perf", self.Serialize("perf"))
	}
}

func (self *Pipestance) VDRKill() *VDRKillReport {
	killReports := []*VDRKillReport{}
	for _, node := range self.node.allNodes() {
		killReports = append(killReports, node.vdrKill())
	}
	killReport := mergeVDRKillReports(killReports)
	metadata := NewMetadata(self.node.parent.getNode().fqname, self.GetPath())
	metadata.write("vdrkill", killReport)
	return killReport
}

func (self *Pipestance) Lock() error {
	metadata := NewMetadata(self.node.parent.getNode().fqname, self.GetPath())
	metadata.loadCache()
	if metadata.exists("lock") {
		return &PipestanceLockedError{self.node.parent.getNode().name, self.GetPath()}
	}
	RegisterSignalHandler(self)
	metadata.writeTime("lock")
	return nil
}

func (self *Pipestance) unlock() {
	metadata := NewMetadata(self.node.parent.getNode().fqname, self.GetPath())
	metadata.remove("lock")
}

func (self *Pipestance) Unlock() {
	self.unlock()
	UnregisterSignalHandler(self)
}

func (self *Pipestance) handleSignal() {
	self.unlock()
}

//=============================================================================
// TopNode
//=============================================================================
type TopNode struct {
	node *Node
}

func (self *TopNode) getNode() *Node { return self.node }

func NewTopNode(rt *Runtime, psid string, p string, j map[string]interface{}) *TopNode {
	self := &TopNode{}
	self.node = &Node{}
	self.node.frontierNodes = map[string]Nodable{}
	self.node.path = p
	self.node.invocation = j
	self.node.rt = rt
	self.node.journalPath = path.Join(self.node.path, "journal")
	self.node.tmpPath = path.Join(self.node.path, "tmp")
	self.node.fqname = "ID." + psid
	self.node.name = psid
	return self
}

//=============================================================================
// Runtime
//=============================================================================
type Runtime struct {
	mroPath         string
	adaptersPath    string
	martianVersion  string
	mroVersion      string
	callableTable   map[string]Callable
	PipelineNames   []string
	vdrMode         string
	jobMode         string
	profileMode     string
	JobManager      JobManager
	LocalJobManager JobManager
	enableStackVars bool
	stest           bool
}

func NewRuntime(jobMode string, vdrMode string, profileMode string, mroPath string, martianVersion string,
	mroVersion string, enableStackVars bool, debug bool) *Runtime {
	return NewRuntimeWithCores(jobMode, vdrMode, profileMode, mroPath, martianVersion, mroVersion,
		-1, -1, -1, enableStackVars, debug, false)
}

func NewRuntimeWithCores(jobMode string, vdrMode string, profileMode string, mroPath string,
	martianVersion string, mroVersion string, reqCores int, reqMem int, reqMemPerCore int,
	enableStackVars bool, debug bool, stest bool) *Runtime {

	self := &Runtime{}
	self.mroPath = mroPath
	self.adaptersPath = RelPath(path.Join("..", "adapters"))
	self.martianVersion = martianVersion
	self.mroVersion = mroVersion
	self.jobMode = jobMode
	self.vdrMode = vdrMode
	self.profileMode = profileMode
	self.enableStackVars = enableStackVars
	self.callableTable = map[string]Callable{}
	self.PipelineNames = []string{}
	self.stest = stest

	self.LocalJobManager = NewLocalJobManager(reqCores, reqMem, debug)
	if self.jobMode == "local" {
		self.JobManager = self.LocalJobManager
	} else {
		self.JobManager = NewRemoteJobManager(self.jobMode, reqMemPerCore)
	}
	VerifyVDRMode(self.vdrMode)
	VerifyProfileMode(self.profileMode)

	// Parse all MROs in MROPATH and cache pipelines by name.
	fpaths, _ := filepath.Glob(self.mroPath + "/[^_]*.mro")
	for _, fpath := range fpaths {
		if data, err := ioutil.ReadFile(fpath); err == nil {
			if _, _, ast, err := parseSource(string(data), fpath, []string{self.mroPath}, true); err == nil {
				for _, callable := range ast.callables.table {
					self.callableTable[callable.getId()] = callable
					if _, ok := callable.(*Pipeline); ok {
						self.PipelineNames = append(self.PipelineNames, callable.getId())
					}
				}
			}
		}
	}
	return self
}

// Compile an MRO file in cwd or self.mroPath.
func (self *Runtime) Compile(fpath string, checkSrcPath bool) (string, []string, *Ast, error) {
	if data, err := ioutil.ReadFile(fpath); err != nil {
		return "", nil, nil, err
	} else {
		return parseSource(string(data), fpath, []string{self.mroPath}, checkSrcPath)
	}
}

// Compile all the MRO files in self.mroPath.
func (self *Runtime) CompileAll(checkSrcPath bool) (int, error) {
	fpaths, _ := filepath.Glob(self.mroPath + "/[^_]*.mro")
	for _, fpath := range fpaths {
		if _, _, _, err := self.Compile(fpath, checkSrcPath); err != nil {
			return 0, err
		}
	}
	return len(fpaths), nil
}

// Instantiate a pipestance object given a psid, MRO source, and a
// pipestance path. This is the core (private) method called by the
// public InvokeWithSource and Reattach methods.
func (self *Runtime) instantiatePipeline(src string, srcPath string, psid string,
	pipestancePath string, readOnly bool) (string, *Pipestance, error) {
	// Parse the invocation source.
	postsrc, _, ast, err := parseSource(src, srcPath, []string{self.mroPath}, true)
	if err != nil {
		return "", nil, err
	}

	// Check there's a call.
	if ast.call == nil {
		return "", nil, &RuntimeError{"cannot start a pipeline without a call statement"}
	}
	// Make sure it's a pipeline we're calling.
	if pipeline := ast.callables.table[ast.call.id]; pipeline == nil {
		return "", nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared pipeline", ast.call.id)}
	}

	invocationJson, _ := self.BuildCallJSON(src, srcPath)

	// Instantiate the pipeline.
	pipestance := NewPipestance(NewTopNode(self, psid, pipestancePath, invocationJson), ast.call, ast.callables)
	if pipestance == nil {
		return "", nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared pipeline", ast.call.id)}
	}

	// Lock the pipestance if not in read-only mode.
	if !readOnly {
		if err := pipestance.Lock(); err != nil {
			return "", nil, err
		}
	}

	pipestance.getNode().mkdirs()

	return postsrc, pipestance, nil
}

// Invokes a new pipestance.
func (self *Runtime) InvokePipeline(src string, srcPath string, psid string,
	pipestancePath string, tags []string) (*Pipestance, error) {

	// Error if pipestance directory is non-empty, otherwise create.
	if _, err := os.Stat(pipestancePath); err == nil {
		if fileInfos, err := ioutil.ReadDir(pipestancePath); err != nil || len(fileInfos) > 0 {
			return nil, &PipestanceExistsError{psid}
		}
	} else if err := os.MkdirAll(pipestancePath, 0755); err != nil {
		return nil, err
	}

	// Expand env vars in invocation source and instantiate.
	src = os.ExpandEnv(src)
	readOnly := false
	postsrc, pipestance, err := self.instantiatePipeline(src, srcPath, psid, pipestancePath, readOnly)
	if err != nil {
		// If instantiation failed, delete the pipestance folder.
		os.RemoveAll(pipestancePath)
		return nil, err
	}

	// Write top-level metadata files.
	metadata := NewMetadata("ID."+psid, pipestancePath)
	metadata.writeRaw("invocation", src)
	metadata.writeRaw("mrosource", postsrc)
	metadata.write("versions", map[string]string{
		"martian":   GetVersion(),
		"pipelines": GetGitTag(self.mroPath),
	})
	metadata.write("tags", tags)
	metadata.writeRaw("timestamp", "start: "+Timestamp())

	return pipestance, nil
}

// Reattaches to an existing pipestance.
func (self *Runtime) ReattachToPipestance(psid string, pipestancePath string, src string, checkSrc bool,
	readOnly bool) (*Pipestance, error) {
	fname := "_invocation"
	invocationPath := path.Join(pipestancePath, fname)

	// Read in the existing _invocation file.
	data, err := ioutil.ReadFile(invocationPath)
	if err != nil {
		return nil, err
	}

	// Check if _invocation has changed.
	if checkSrc && src != string(data) {
		return nil, &PipestanceInvocationError{psid, invocationPath}
	}

	// Instantiate the pipestance.
	_, pipestance, err := self.instantiatePipeline(string(data), fname, psid, pipestancePath, readOnly)

	// If we're reattaching in local mode, restart any stages that were
	// left in a running state from last mrp run. The actual job would
	// have been killed by the CTRL-C.
	if err == nil {
		PrintInfo("runtime", "Reattaching in %s mode.", self.jobMode)
		err = pipestance.RestartRunningNodes(self.jobMode)
	}

	return pipestance, err
}

// Instantiate a stagestance.
func (self *Runtime) InvokeStage(src string, srcPath string, ssid string,
	stagestancePath string) (*Stagestance, error) {
	// Check if stagestance path already exists.
	if _, err := os.Stat(stagestancePath); err == nil {
		return nil, &RuntimeError{fmt.Sprintf("stagestance '%s' already exists", ssid)}
	} else if err := os.MkdirAll(stagestancePath, 0755); err != nil {
		return nil, err
	}

	// Parse the invocation source.
	src = os.ExpandEnv(src)
	_, _, ast, err := parseSource(src, srcPath, []string{self.mroPath}, true)
	if err != nil {
		return nil, err
	}

	// Check there's a call.
	if ast.call == nil {
		return nil, &RuntimeError{"cannot start a stage without a call statement"}
	}
	// Make sure it's a stage we're calling.
	if _, ok := ast.callables.table[ast.call.id].(*Stage); !ok {
		return nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared stage", ast.call.id)}
	}

	invocationJson, _ := self.BuildCallJSON(src, srcPath)

	// Instantiate stagestance.
	stagestance := NewStagestance(NewTopNode(self, "", stagestancePath, invocationJson), ast.call, ast.callables)
	if stagestance == nil {
		return nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared stage", ast.call.id)}
	}

	stagestance.getNode().mkdirs()

	return stagestance, nil
}

func (self *Runtime) GetSerialization(pipestancePath string, name string) (interface{}, bool) {
	metadata := NewMetadata("", pipestancePath)
	metadata.loadCache()
	if metadata.exists(name) {
		return metadata.read(name), true
	}
	return nil, false
}

/****************************************************************************
 * Used Only for MARSOC
 */
func (self *Runtime) buildVal(param Param, val interface{}) string {
	// MRO value expression syntax is identical to JSON. Just need to make
	// sure floats get printed with decimal points.
	switch {
	case param.getTname() == "float" && val != nil:
		return fmt.Sprintf("%f", val)
	default:
		indent := "    "
		if data, err := json.MarshalIndent(val, "", indent); err == nil {
			// Indent multi-line values (but not first line).
			sublines := strings.Split(string(data), "\n")
			for i, _ := range sublines[1:] {
				sublines[i+1] = indent + sublines[i+1]
			}
			return strings.Join(sublines, "\n")
		}
		return fmt.Sprintf("<ParseError: %v>", val)
	}
}

func (self *Runtime) BuildCallSource(incpaths []string, name string,
	args map[string]interface{}) (string, error) {
	// Make sure pipeline has been imported
	if _, ok := self.callableTable[name]; !ok {
		return "", &RuntimeError{fmt.Sprintf("'%s' is not a declared pipeline or stage", name)}
	}

	// Build @include statements.
	includes := []string{}
	for _, incpath := range incpaths {
		includes = append(includes, fmt.Sprintf("@include \"%s\"", incpath))
	}
	// Loop over the pipeline's in params and print a binding
	// whether the args bag has a value for it not.
	lines := []string{}
	for _, param := range self.callableTable[name].getInParams().list {
		valstr := self.buildVal(param, args[param.getId()])
		lines = append(lines, fmt.Sprintf("    %s = %s,", param.getId(), valstr))
	}
	return fmt.Sprintf("%s\n\ncall %s(\n%s\n)", strings.Join(includes, "\n"),
		name, strings.Join(lines, "\n")), nil
}

func (self *Runtime) BuildCallJSON(src string, srcPath string) (map[string]interface{}, error) {
	_, incpaths, ast, err := parseSource(src, srcPath, []string{self.mroPath}, false)
	if err != nil {
		return nil, err
	}

	if ast.call == nil {
		return nil, &RuntimeError{"cannot jsonify a pipeline without a call statement"}
	}

	args := map[string]interface{}{}
	for _, binding := range ast.call.bindings.list {
		args[binding.id] = expToInterface(binding.exp)
	}
	return map[string]interface{}{
		"call":     ast.call.id,
		"args":     args,
		"incpaths": incpaths,
	}, nil
}

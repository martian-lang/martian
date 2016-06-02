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
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const heartbeatTimeout = 60 // 60 minutes

//=============================================================================
// Metadata
//=============================================================================
type Metadata struct {
	fqname        string
	path          string
	contents      map[string]bool
	filesPath     string
	lastHeartbeat time.Time
	mutex         *sync.Mutex
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

func (self *Metadata) removeAll() {
	os.RemoveAll(self.path)
	os.RemoveAll(self.filesPath)
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

func (self *Metadata) resetHeartbeat() {
	self.lastHeartbeat = time.Time{}
}

func (self *Metadata) checkHeartbeat() {
	if state, _ := self.getState(""); state == "running" {
		if self.lastHeartbeat.IsZero() || self.exists("heartbeat") {
			self.uncache("heartbeat")
			self.lastHeartbeat = time.Now()
		}
		if time.Since(self.lastHeartbeat) > time.Minute*heartbeatTimeout {
			self.writeRaw("errors", fmt.Sprintf(
				"%s: No heartbeat detected for %d minutes. Assuming job has failed. This may be "+
					"due to a user manually terminating the job, or the operating system or cluster "+
					"terminating it due to resource or time limits.",
				Timestamp(), heartbeatTimeout))
		}
	}
}

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
	self.id = bindStm.Id
	self.tname = bindStm.Tname
	self.sweep = bindStm.Sweep
	self.waiting = false
	switch valueExp := bindStm.Exp.(type) {
	case *RefExp:
		if valueExp.Kind == "self" {
			var parentBinding *Binding
			if returnBinding {
				parentBinding = self.node.argbindings[valueExp.Id]
			} else {
				parentBinding = self.node.parent.getNode().argbindings[valueExp.Id]
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
			self.id = bindStm.Id
			self.valexp = "self." + valueExp.Id
		} else if valueExp.Kind == "call" {
			if returnBinding {
				self.parentNode = self.node.subnodes[valueExp.Id]
				self.boundNode, self.output, self.mode, self.value = self.node.findBoundNode(
					valueExp.Id, valueExp.OutputId, "reference", nil)
			} else {
				self.parentNode = self.node.parent.getNode().subnodes[valueExp.Id]
				self.boundNode, self.output, self.mode, self.value = self.node.parent.getNode().findBoundNode(
					valueExp.Id, valueExp.OutputId, "reference", nil)
			}
			if valueExp.OutputId == "default" {
				self.valexp = valueExp.Id
			} else {
				self.valexp = valueExp.Id + "." + valueExp.OutputId
			}
		}
	case *ValExp:
		self.mode = "value"
		self.parentNode = node
		self.boundNode = node
		self.value = expToInterface(bindStm.Exp)
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
	if valExp.Kind == "array" {
		varray := []interface{}{}
		for _, exp := range valExp.Value.([]Exp) {
			varray = append(varray, expToInterface(exp))
		}
		return varray
	} else if valExp.Kind == "map" {
		vmap := map[string]interface{}{}
		// Type assertion fails if map is empty
		valExpMap, ok := valExp.Value.(map[string]Exp)
		if ok {
			for k, exp := range valExpMap {
				vmap[k] = expToInterface(exp)
			}
		}
		return vmap
	} else {
		return valExp.Value
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
	for id, param := range outParams.Table {
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

func ParseFQName(fqname string) (string, string) {
	parts := strings.Split(fqname, ".")
	return parts[2], parts[1]
}

func MakeFQName(pipeline string, psid string) string {
	return fmt.Sprintf("ID.%s.%s", psid, pipeline)
}

func ParseTimestamp(data string) string {
	// Backwards compatible with current and plain timestamp formats
	timestamp := strings.Split(data, "\n")[0]
	prefix := "start:"
	if strings.HasPrefix(timestamp, prefix) {
		timestamp = timestamp[len(prefix):]
		return strings.TrimSpace(timestamp)
	}
	return timestamp
}

func ParseVersions(data string) (string, string, error) {
	var versions map[string]string
	if err := json.Unmarshal([]byte(data), &versions); err != nil {
		return "", "", err
	}
	return versions["martian"], versions["pipelines"], nil
}

func ParseJobMode(data string) (string, string, string) {
	jobmode := "local"
	if m := regexp.MustCompile(".*--jobmode=([^\\s]+).*").FindStringSubmatch(data); len(m) > 0 {
		jobmode = m[1]
	}
	localcores := "max"
	if m := regexp.MustCompile(".*--localcores=([^\\s]+).*").FindStringSubmatch(data); len(m) > 0 {
		localcores = m[1]
	}
	localmem := "max"
	if m := regexp.MustCompile(".*--localmem=([^\\s]+).*").FindStringSubmatch(data); len(m) > 0 {
		localmem = m[1]
	}
	return jobmode, localcores, localmem
}

func VerifyVDRMode(vdrMode string) {
	validModes := []string{"rolling", "post", "disable"}
	for _, validMode := range validModes {
		if validMode == vdrMode {
			return
		}
	}
	PrintInfo("runtime", "Invalid VDR mode: %s. Valid VDR modes: %s", vdrMode, strings.Join(validModes, ", "))
	os.Exit(1)
}

func VerifyProfileMode(profileMode string) {
	validModes := []string{"cpu", "mem", "line", "disable"}
	for _, validMode := range validModes {
		if validMode == profileMode {
			return
		}
	}
	PrintInfo("runtime", "Invalid profile mode: %s. Valid profile modes: %s", profileMode, strings.Join(validModes, ", "))
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

	threads, memGB := self.node.setJobReqs(self.chunkDef)

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
	numThreads, _ := self.node.getJobReqs(self.chunkDef)
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
	parentFork     *Fork
	subforks       []*Fork
	chunks         []*Chunk
	split_has_run  bool
	join_has_run   bool
	argPermute     map[string]interface{}
	stageDefs      *StageDefs
	perfCache      *ForkPerfCache
}

type ForkInfo struct {
	Index         int                    `json:"index"`
	ArgPermute    map[string]interface{} `json:"argPermute"`
	JoinDef       map[string]interface{} `json:"joinDef"`
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

type ForkPerfCache struct {
	perfInfo      *ForkPerfInfo
	vdrKillReport *VDRKillReport
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

	// By default, initialize stage defs with one empty chunk.
	self.stageDefs = &StageDefs{ChunkDefs: []map[string]interface{}{}, JoinDef: map[string]interface{}{}}
	self.stageDefs.ChunkDefs = append(self.stageDefs.ChunkDefs, map[string]interface{}{})

	if err := json.Unmarshal([]byte(self.split_metadata.readRaw("stage_defs")), &self.stageDefs); err == nil {
		for i, chunkDef := range self.stageDefs.ChunkDefs {
			chunk := NewChunk(self.node, self, i, chunkDef)
			self.chunks = append(self.chunks, chunk)
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

func (self *Fork) removeMetadata() {
	metadatas := []*Metadata{self.split_metadata, self.join_metadata}
	for _, chunk := range self.chunks {
		metadatas = append(metadatas, chunk.metadata)
	}
	for _, metadata := range metadatas {
		filePaths, _ := metadata.enumerateFiles()
		if len(filePaths) == 0 {
			metadata.removeAll()
		}
	}
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
	if len(outparams.List) > 0 {
		outputs := self.metadata.read("outs").(map[string]interface{})
		for _, param := range outparams.Table {
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

func (self *Fork) skip() {
	self.metadata.writeTime("complete")
}

func (self *Fork) writeInvocation() {
	if !self.metadata.exists("invocation") {
		argBindings := resolveBindings(self.node.argbindings, self.argPermute)
		sweepBindings := []string{}
		incpaths := self.node.invocation["incpaths"].([]string)
		invocation, _ := self.node.rt.BuildCallSource(incpaths, self.node.name, argBindings, sweepBindings, self.node.mroPaths)
		self.metadata.writeRaw("invocation", invocation)
	}
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
			self.writeInvocation()
			self.split_metadata.write("args", resolveBindings(self.node.argbindings, self.argPermute))
			threads, memGB := self.node.setJobReqs(nil)
			if self.node.split {
				if !self.split_has_run {
					self.split_has_run = true
					self.node.runSplit(self.fqname, self.split_metadata, threads, memGB)
				}
			} else {
				self.split_metadata.write("stage_defs", self.stageDefs)
				self.split_metadata.writeTime("complete")
			}
		} else if state == "split_complete" {
			if err := json.Unmarshal([]byte(self.split_metadata.readRaw("stage_defs")), &self.stageDefs); err != nil || len(self.stageDefs.ChunkDefs) == 0 {
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
			threads, memGB := self.node.setJobReqs(self.stageDefs.JoinDef)
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
		self.writeInvocation()
		self.metadata.write("outs", resolveBindings(self.node.retbindings, self.argPermute))
		if ok, msg := self.verifyOutput(); ok {
			self.metadata.writeTime("complete")
		} else {
			self.metadata.writeRaw("errors", msg)
		}
	}
}

func (self *Fork) cachePerf() {
	perfInfo, vdrKillReport := self.serializePerf()
	self.perfCache = &ForkPerfCache{perfInfo, vdrKillReport}
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
                Println("DELETEDELETEDELETE: %v", p);
	}
	// update timestamp to mark actual kill time
	killReport.Timestamp = Timestamp()
	self.metadata.write("vdrkill", killReport)
	return killReport
}

func (self *Fork) postProcess() {
	pipestancePath := self.node.parent.getNode().path
	outsPath := path.Join(pipestancePath, "outs")
	paramList := self.node.outparams.List

	if len(paramList) == 0 {
		return
	}

	if len(self.node.forks) > 1 {
		outsPath = path.Join(outsPath, fmt.Sprintf("fork%d", self.index))
		Print("\nOutputs (fork%d):\n", self.index)
	} else {
		Print("\nOutputs:\n")
	}

	outs := map[string]interface{}{}
	if data := self.metadata.read("outs"); data != nil {
		if v, ok := data.(map[string]interface{}); ok {
			outs = v
		}
	}

	errorTypes := make(map[string]bool)
	for _, param := range paramList {
		id := param.getId()
		value, ok := outs[id]
		if ok && value != nil {
			if param.getIsFile() || param.getTname() == "path" {
				if filePath, ok := value.(string); ok {
					if _, err := os.Stat(filePath); err == nil {
						if filePath, err := filepath.Rel(outsPath, filePath); err == nil {
							mkdirAll(outsPath)
							newValue := path.Join(outsPath, id)
							if param.getTname() != "path" {
								newValue += "." + param.getTname()
							}
							if err := os.Symlink(filePath, newValue); err != nil {
								errMsg := err.Error()[strings.Index(err.Error(), newValue)+len(newValue)+1:]
								errorTypes[errMsg] = true
								value = "null"
							} else {
								value = newValue
							}
						}
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
		Print("- %s: %v\n", key, value)
	}
	if len(errorTypes) > 0 {
		i := 0
		distinctErrors := make([]string, len(errorTypes))
		for key := range errorTypes {
			distinctErrors[i] = key
			i++
		}
		Print("\nCould not create symlinks to output files: %s", strings.Join(distinctErrors, ", "))
	}
	Print("\n")

	if alarms := self.getAlarms(); len(alarms) > 0 {
		if len(self.node.forks) > 1 {
			Print("Alerts (fork%d):\n", self.index)
		} else {
			Print("Alerts:\n")
		}
		Print(alarms + "\n")
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
		JoinDef:       self.stageDefs.JoinDef,
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
	if self.perfCache != nil {
		// Use cached performance information if it exists.
		return self.perfCache.perfInfo, self.perfCache.vdrKillReport
	}

	chunks := []*ChunkPerfInfo{}
	stats := []*PerfInfo{}
	for _, chunk := range self.chunks {
		chunkSer := chunk.serializePerf()
		chunks = append(chunks, chunkSer)
		if chunkSer.ChunkStats != nil {
			// avoid double-counting of bytes/files if there is no
			// actual split; it will be counted by ComputeStats() below.
			if !self.node.split {
				chunkSer.ChunkStats.OutputBytes = 0
				chunkSer.ChunkStats.OutputFiles = 0
			}
			stats = append(stats, chunkSer.ChunkStats)
		}
	}

	numThreads, _ := self.node.getJobReqs(nil)
	splitStats := self.split_metadata.serializePerf(numThreads)
	if splitStats != nil {
		stats = append(stats, splitStats)
	}

	numThreads, _ = self.node.getJobReqs(self.stageDefs.JoinDef)
	joinStats := self.join_metadata.serializePerf(numThreads)
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
	fpaths, _ := self.metadata.enumerateFiles()

	forkStats := &PerfInfo{}
	if len(stats) > 0 {
		forkStats = ComputeStats(stats, fpaths, killReport)
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
	parent             Nodable
	rt                 *Runtime
	kind               string
	name               string
	fqname             string
	path               string
	metadata           *Metadata
	outparams          *Params
	argbindings        map[string]*Binding
	argbindingList     []*Binding // for stable ordering
	retbindings        map[string]*Binding
	retbindingList     []*Binding // for stable ordering
	sweepbindings      []*Binding
	subnodes           map[string]Nodable
	prenodes           map[string]Nodable
	directPrenodes     []Nodable
	postnodes          map[string]Nodable
	frontierNodes      map[string]Nodable
	forks              []*Fork
	split              bool
	state              string
	volatile           bool
	local              bool
	preflight          bool
	stagecodeLang      string
	stagecodeCmd       string
	journalPath        string
	tmpPath            string
	mroPaths           []string
	mroVersion         string
	envs               map[string]string
	invocation         map[string]interface{}
	blacklistedFromMRT bool // Don't used cached data when MRT'ing
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
	self.name = callStm.Id
	self.fqname = parent.getNode().fqname + "." + self.name
	self.path = path.Join(parent.getNode().path, self.name)
	self.journalPath = parent.getNode().journalPath
	self.tmpPath = parent.getNode().tmpPath
	self.mroPaths = parent.getNode().mroPaths
	self.mroVersion = parent.getNode().mroVersion
	self.envs = parent.getNode().envs
	self.invocation = parent.getNode().invocation
	self.metadata = NewMetadata(self.fqname, self.path)
	self.volatile = callStm.Modifiers.Volatile
	self.local = callStm.Modifiers.Local
	self.preflight = callStm.Modifiers.Preflight

	self.outparams = callables.Table[self.name].getOutParams()
	self.argbindings = map[string]*Binding{}
	self.argbindingList = []*Binding{}
	self.retbindings = map[string]*Binding{}
	self.retbindingList = []*Binding{}
	self.subnodes = map[string]Nodable{}
	self.prenodes = map[string]Nodable{}
	self.directPrenodes = []Nodable{}
	self.postnodes = map[string]Nodable{}
	self.frontierNodes = parent.getNode().frontierNodes

	for id, bindStm := range callStm.Bindings.Table {
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
			if matchedFork := subnode.getNode().matchFork(fork.argPermute); matchedFork != nil {
				matchedFork.parentFork = fork
				fork.subforks = append(fork.subforks, matchedFork)
			}
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
			if !reflect.DeepEqual(targetArgPermute[paramId], argValue) {
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
	// DSTAFF
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

func (self *Node) checkHeartbeats() {
	for _, metadata := range self.collectMetadatas() {
		metadata.checkHeartbeat()
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

	for _, fork := range self.forks {
		fork.postProcess()
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
	return self.fqname
}

func (self *Node) getFatalError() (string, bool, string, string, string, []string) {
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
				} else if len(errlines) == 1 {
					summary = errlines[0]
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
			return metadata.fqname, self.preflight, summary, errlog, "errors", errpaths
		}
		if metadata.exists("assert") {
			assertlog := metadata.readRaw("assert")
			summary := "<none>"
			assertlines := strings.Split(assertlog, "\n")
			if len(assertlines) >= 1 {
				summary = assertlines[len(assertlines)-1]
			}
			return metadata.fqname, self.preflight, summary, assertlog, "assert", []string{
				metadata.makePath("assert"),
			}
		}
	}
	return "", false, "", "", "", []string{}
}

func (self *Node) step() {
	if self.state == "running" {
		for _, fork := range self.forks {
			if self.preflight && self.rt.skipPreflight {
				fork.skip()
			} else {
				fork.step()
			}
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
				node.getNode().cachePerf()
			}
			self.vdrKill()
			self.cachePerf()
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
	Count     uint     `json:"count"`
	Size      uint64   `json:"size"`
	Timestamp string   `json:"timestamp"`
	Paths     []string `json:"paths"`
	Errors    []string `json:"errors"`
}

type VDRByTimestamp []*VDRKillReport

func (self VDRByTimestamp) Len() int {
	return len(self)
}

func (self VDRByTimestamp) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self VDRByTimestamp) Less(i, j int) bool {
	return self[i].Timestamp < self[j].Timestamp
}

func mergeVDRKillReports(killReports []*VDRKillReport) *VDRKillReport {
	allKillReport := &VDRKillReport{}
	for _, killReport := range killReports {
		allKillReport.Size += killReport.Size
		allKillReport.Count += killReport.Count
		allKillReport.Errors = append(allKillReport.Errors, killReport.Errors...)
		allKillReport.Paths = append(allKillReport.Paths, killReport.Paths...)
	}
	sort.Sort(VDRByTimestamp(killReports))
	if len(killReports) > 0 {
		allKillReport.Timestamp = killReports[len(killReports)-1].Timestamp
	}
	return allKillReport
}

/* Is self or any of its ancestors symlinked? */
func (self *Node) vdrCheckSymlink() bool {

	statinfo, err := os.Lstat(self.path)

        /* Yep! Found a symlink */
	if (statinfo.Mode()&os.ModeSymlink) != 0 || err != nil {
                return true;
	}

        /* Nope! Got all the way to the top */
        if (self.parent == nil) {
                return false;
        }

        return self.parent.getNode().vdrCheckSymlink();
}

func (self *Node) vdrKill() (*VDRKillReport, bool) {

        /*
         * Refuse to VDR a node if it, or any of its ancestors are symlinked.
         */
        if self.vdrCheckSymlink() == true{

	        Println("Refuse to VDR across a symlink: %v", self.fqname)
	        return &VDRKillReport{}, true
        }

	killReports := []*VDRKillReport{}
	ok := true
	for _, node := range self.postnodes {
		if node.getNode().state != "complete" {
			ok = false
		}
	}
	if ok {
		for _, fork := range self.forks {
			killReports = append(killReports, fork.vdrKill())
		}
	}
	return mergeVDRKillReports(killReports), ok
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
		fqname, _, summary, log, _, errpaths := self.getFatalError()
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
	memGB := -1
	if jobDef != nil {
		if v, ok := jobDef["__threads"].(float64); ok {
			threads = int(v)
		}
		if v, ok := jobDef["__mem_gb"].(float64); ok {
			memGB = int(v)
		}
	}

	if self.local {
		threads, memGB = self.rt.LocalJobManager.GetSystemReqs(threads, memGB)
	} else {
		threads, memGB = self.rt.JobManager.GetSystemReqs(threads, memGB)
	}

	return threads, memGB
}

func (self *Node) setJobReqs(jobDef map[string]interface{}) (int, int) {
	threads, memGB := self.getJobReqs(jobDef)

	if jobDef != nil {
		jobDef["__threads"] = float64(threads)
		jobDef["__mem_gb"] = float64(memGB)
	}

	return threads, memGB
}

func (self *Node) runSplit(fqname string, metadata *Metadata, threads int, memGB int) {
	self.runJob("split", fqname, metadata, threads, memGB)
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

	// Configure memory monitoring.
	monitor := "disable"
	if self.rt.enableMonitor {
		monitor = "monitor"
	}

	// Construct path to the shell.
	shellCmd := ""
	argv := []string{}
	stagecodeParts := strings.Split(self.stagecodeCmd, " ")
	runFile := path.Join(self.journalPath, fqname)
	version := map[string]interface{}{
		"martian":   self.rt.martianVersion,
		"pipelines": self.mroVersion,
	}
	envs := self.envs

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
		"threads":        threads,
		"memGB":          memGB,
		"profile_mode":   self.rt.profileMode,
		"stackvars_flag": stackVars,
		"monitor_flag":   monitor,
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
	stage, ok := callables.Table[self.node.name].(*Stage)
	if !ok {
		return nil
	}

	stagecodePaths := append(self.node.mroPaths, strings.Split(os.Getenv("PATH"), ":")...)
	stagecodePath, _ := SearchPaths(stage.Src.Path, stagecodePaths)
	self.node.stagecodeCmd = strings.Join(append([]string{stagecodePath}, stage.Src.Args...), " ")
	if self.node.rt.stest {
		switch stage.Src.Lang {
		case "py":
			self.node.stagecodeCmd = RelPath(path.Join("..", "adapters", "python", "tester"))
		default:
			panic(fmt.Sprintf("Unsupported stress test language: %s", stage.Src.Lang))
		}
	}
	self.node.stagecodeLang = langMap[stage.Src.Lang]
	self.node.split = stage.Split
	self.node.buildForks(self.node.argbindings)
	return self
}

func (self *Stagestance) getNode() *Node   { return self.node }
func (self *Stagestance) GetState() string { return self.getNode().getState() }
func (self *Stagestance) Step()            { self.getNode().step() }
func (self *Stagestance) CheckHeartbeats() { self.getNode().checkHeartbeats() }
func (self *Stagestance) RefreshState()    { self.getNode().refreshState() }
func (self *Stagestance) LoadMetadata()    { self.getNode().loadMetadata() }
func (self *Stagestance) PostProcess()     { self.getNode().postProcess() }
func (self *Stagestance) GetFatalError() (string, bool, string, string, string, []string) {
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
	pipeline, ok := callables.Table[self.node.name].(*Pipeline)
	if !ok {
		return nil
	}
	preflightNodes := []Nodable{}
	for _, subcallStm := range pipeline.Calls {
		callable := callables.Table[subcallStm.Id]
		switch callable.(type) {
		case *Stage:
			self.node.subnodes[subcallStm.Id] = NewStagestance(self.node, subcallStm, callables)
		case *Pipeline:
			self.node.subnodes[subcallStm.Id] = NewPipestance(self.node, subcallStm, callables)
		}
		if self.node.subnodes[subcallStm.Id].getNode().preflight {
			preflightNodes = append(preflightNodes, self.node.subnodes[subcallStm.Id])
		}
	}

	// Also depends on stages bound to return values.
	self.node.retbindings = map[string]*Binding{}
	for id, bindStm := range pipeline.Ret.Bindings.Table {
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
	// Add preflight dependencies if preflight stages exist.
	for _, preflightNode := range preflightNodes {
		for _, subnode := range self.node.subnodes {
			if !subnode.getNode().preflight {
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
	for _, node := range self.node.allNodes() {
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
	return nil
}

func (self *Pipestance) CheckHeartbeats() {
	nodes := self.node.getFrontierNodes()
	for _, node := range nodes {
		node.checkHeartbeats()
	}
}

func (self *Pipestance) GetFailedNodes() []*Node {
	failedNodes := []*Node{}

	nodes := self.node.getFrontierNodes()
	for _, node := range nodes {
		if node.state == "failed" {
			failedNodes = append(failedNodes, node)
		}
	}
	return failedNodes
}

func (self *Pipestance) GetFatalError() (string, bool, string, string, string, []string) {
	nodes := self.node.getFrontierNodes()
	for _, node := range nodes {
		if node.state == "failed" {
			return node.getFatalError()
		}
	}
	return "", false, "", "", "", []string{}
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
	if name == "perf" {
		LogInfo("perf", "pipestance serialize")
		if len(ser) > 0 {
			self.ComputeDiskUsage(ser[0].(*NodePerfInfo))
		}
	}
	return ser
}

type StorageEvent struct {
	Timestamp time.Time
	Delta     int64
	Name      string
}

func NewStorageEvent(timestamp time.Time, delta int64, fqname string) *StorageEvent {
	self := &StorageEvent{}
	self.Timestamp = timestamp
	self.Delta = delta
	self.Name = fqname
	return self
}

type StorageEventByTimestamp []*StorageEvent

func (self StorageEventByTimestamp) Len() int {
	return len(self)
}

func (self StorageEventByTimestamp) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self StorageEventByTimestamp) Less(i, j int) bool {
	return self[i].Timestamp.Before(self[j].Timestamp)
}

// this is due to the fact that the VDR bytes/total bytes
// reported at the fork level is the sum of chunk + split
// + join plus any additional files.  The additional
// files that are unique to the fork cannot be resolved
// unless you sub out chunk/split/join and then child
// stages.
type ForkStorageEvent struct {
	Name          string
	ChildNames    []string
	TotalBytes    uint64
	ChunkBytes    uint64
	ForkBytes     uint64
	TotalVDRBytes uint64
	ForkVDRBytes  uint64
	Timestamp     time.Time
	VDRTimestamp  time.Time
}

func NewForkStorageEvent(timestamp time.Time, totalBytes uint64, vdrBytes uint64, fqname string) *ForkStorageEvent {
	self := &ForkStorageEvent{ChildNames: []string{}}
	self.Name = fqname
	self.TotalBytes = totalBytes // sum total of bytes in fork and children
	self.ForkBytes = self.TotalBytes
	self.TotalVDRBytes = vdrBytes          // sum total of VDR bytes in fork and children
	self.ForkVDRBytes = self.TotalVDRBytes // VDR bytes in forkN/files
	self.Timestamp = timestamp
	return self
}

func forkDependentName(fqname string, forkIndex int) string {
	return fmt.Sprintf("%s.fork%d", fqname, forkIndex)
}

func (self *Pipestance) ComputeDiskUsage(nodePerf *NodePerfInfo) *NodePerfInfo {
	storageEvents := []*StorageEvent{}
	forksVisited := make(map[string]*ForkStorageEvent)

	for _, node := range self.node.allNodes() {
		nodePerf := node.serializePerf()
		for forkIdx, fork := range nodePerf.Forks {
			if fork.ForkStats != nil {
				forkEvent := NewForkStorageEvent(fork.ForkStats.Start, fork.ForkStats.TotalBytes,
					fork.ForkStats.VdrBytes, forkDependentName(node.fqname, forkIdx))
				forksVisited[forkDependentName(node.fqname, forkIdx)] = forkEvent
			}
		}
		// remove pipeline counts; they double-count only (do not add own files logic)
		for _, fork := range node.forks {
			if fork.node.kind == "pipeline" {
				if _, ok := forksVisited[fork.fqname]; ok {
					delete(forksVisited, fork.fqname)
				}
			}
		}

		for _, fork := range node.forks {
			forkVDR, _ := fork.getVdrKillReport()
			if forkVDR == nil {
				continue
			}
			vdrTimestamp, _ := time.Parse(TIMEFMT, forkVDR.Timestamp)
			if forkEvent, ok := forksVisited[fork.fqname]; ok {
				forkEvent.VDRTimestamp = vdrTimestamp
			}
		}
	}

	for _, fse := range forksVisited {
		storageEvents = append(
			storageEvents,
			NewStorageEvent(fse.Timestamp, int64(fse.ForkBytes), fmt.Sprintf("%s alloc", fse.Name)))
		if !fse.VDRTimestamp.IsZero() {
			storageEvents = append(
				storageEvents,
				NewStorageEvent(fse.VDRTimestamp, -1*int64(fse.ForkVDRBytes), fmt.Sprintf("%s delete", fse.Name)))
		}
	}

	sort.Sort(StorageEventByTimestamp(storageEvents))
	highMark := int64(0)
	currentMark := int64(0)

	byteStamps := make([]*NodeByteStamp, len(storageEvents))
	for idx, se := range storageEvents {
		currentMark += se.Delta
		byteStamps[idx] = &NodeByteStamp{Timestamp: se.Timestamp, Bytes: currentMark, Description: se.Name}
		if currentMark > highMark {
			highMark = currentMark
		}
	}

	nodePerf.MaxBytes = highMark
	nodePerf.BytesHist = byteStamps
	return nodePerf
}

func (self *Pipestance) ZipMetadata(zipPath string) error {
	if !self.node.rt.enableZip {
		return nil
	}

	nodes := self.node.allNodes()
	metadatas := []*Metadata{}
	filePaths := []string{}
	for _, node := range nodes {
		metadatas = append(metadatas, node.collectMetadatas()...)
	}
	for _, metadata := range metadatas {
		filePaths = append(filePaths, metadata.glob()...)
	}

	EnterCriticalSection()
	defer ExitCriticalSection()

	// Create zip with all metadata.
	if err := CreateZip(zipPath, filePaths); err != nil {
		return err
	}

	// Remove all metadata files.
	for _, filePath := range filePaths {
		os.Remove(filePath)
	}

	// Remove all split, join, chunk metadatas without data files.
	for _, node := range nodes {
		node.removeMetadata()
	}

	return nil
}

func (self *Pipestance) GetPath() string {
	return self.node.parent.getNode().path
}

func (self *Pipestance) GetInvocation() interface{} {
	return self.node.parent.getNode().invocation
}

func (self *Pipestance) VerifyJobMode() error {
	metadata := NewMetadata(self.node.parent.getNode().fqname, self.GetPath())
	metadata.loadCache()
	if metadata.exists("jobmode") {
		jobMode := metadata.readRaw("jobmode")
		if jobMode != self.node.rt.jobMode {
			return &PipestanceJobModeError{self.GetPsid(), jobMode}
		}
	}
	return nil
}

func (self *Pipestance) GetTimestamp() string {
	metadata := NewMetadata(self.node.parent.getNode().fqname, self.GetPath())
	data := metadata.readRaw("timestamp")
	return ParseTimestamp(data)
}

func (self *Pipestance) GetVersions() (string, string, error) {
	metadata := NewMetadata(self.node.parent.getNode().fqname, self.GetPath())
	data := metadata.readRaw("versions")
	return ParseVersions(data)
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
	if !metadata.exists("perf") {
		metadata.write("perf", self.Serialize("perf"))
	}
	if !metadata.exists("finalstate") {
		metadata.write("finalstate", self.Serialize("finalstate"))
	}
	if !metadata.exists("metadata.zip") {
		zipPath := metadata.makePath("metadata.zip")
		if err := self.ZipMetadata(zipPath); err != nil {
			LogError(err, "runtime", "Failed to create metadata zip file %s: %s",
				zipPath, err.Error())
			os.Remove(zipPath)
		}
	}
}

func (self *Pipestance) VDRKill() *VDRKillReport {
	killReports := []*VDRKillReport{}
	for _, node := range self.node.allNodes() {
		killReport, _ := node.vdrKill()
		killReports = append(killReports, killReport)
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

func NewTopNode(rt *Runtime, psid string, p string, mroPaths []string, mroVersion string,
	envs map[string]string, j map[string]interface{}) *TopNode {
	self := &TopNode{}
	self.node = &Node{}
	self.node.frontierNodes = map[string]Nodable{}
	self.node.path = p
	self.node.mroPaths = mroPaths
	self.node.mroVersion = mroVersion
	self.node.invocation = j
	self.node.rt = rt
	self.node.journalPath = path.Join(self.node.path, "journal")
	self.node.tmpPath = path.Join(self.node.path, "tmp")
	self.node.fqname = "ID." + psid
	self.node.name = psid

	// Since we must set other required Martian environment variables,
	// we must make a copy of envs so as not to overwrite envs for
	// other pipestances / stagestances.
	self.node.envs = map[string]string{}
	for key, value := range envs {
		self.node.envs[key] = value
	}
	self.node.envs["TMPDIR"] = self.node.tmpPath

	return self
}

//=============================================================================
// Runtime
//=============================================================================
type Runtime struct {
	adaptersPath    string
	martianVersion  string
	vdrMode         string
	jobMode         string
	profileMode     string
	MroCache        *MroCache
	JobManager      JobManager
	LocalJobManager JobManager
	enableStackVars bool
	enableZip       bool
	skipPreflight   bool
	enableMonitor   bool
	stest           bool
}

func NewRuntime(jobMode string, vdrMode string, profileMode string, martianVersion string) *Runtime {
	return NewRuntimeWithCores(jobMode, vdrMode, profileMode, martianVersion,
		-1, -1, -1, -1, -1, false, false, false, false, false, false)
}

func NewRuntimeWithCores(jobMode string, vdrMode string, profileMode string, martianVersion string,
	reqCores int, reqMem int, reqMemPerCore int, maxJobs int, jobFreqMillis int,
	enableStackVars bool, enableZip bool, skipPreflight bool, enableMonitor bool,
	debug bool, stest bool) *Runtime {

	self := &Runtime{}
	self.adaptersPath = RelPath(path.Join("..", "adapters"))
	self.martianVersion = martianVersion
	self.jobMode = jobMode
	self.vdrMode = vdrMode
	self.profileMode = profileMode
	self.enableStackVars = enableStackVars
	self.enableZip = enableZip
	self.skipPreflight = skipPreflight
	self.enableMonitor = enableMonitor
	self.stest = stest

	self.MroCache = NewMroCache()
	self.LocalJobManager = NewLocalJobManager(reqCores, reqMem, debug)
	if self.jobMode == "local" {
		self.JobManager = self.LocalJobManager
	} else {
		self.JobManager = NewRemoteJobManager(self.jobMode, reqMemPerCore, maxJobs,
			jobFreqMillis, debug)
	}
	VerifyVDRMode(self.vdrMode)
	VerifyProfileMode(self.profileMode)

	return self
}

// Compile an MRO file in cwd or mroPaths.
func (self *Runtime) Compile(fpath string, mroPaths []string, checkSrcPath bool) (string, []string, *Ast, error) {
	if data, err := ioutil.ReadFile(fpath); err != nil {
		return "", nil, nil, err
	} else {
		return parseSource(string(data), fpath, mroPaths, checkSrcPath)
	}
}

// Compile all the MRO files in mroPaths.
func (self *Runtime) CompileAll(mroPaths []string, checkSrcPath bool) (int, error) {
	numFiles := 0
	for _, mroPath := range mroPaths {
		fpaths, _ := filepath.Glob(mroPath + "/[^_]*.mro")
		for _, fpath := range fpaths {
			if _, _, _, err := self.Compile(fpath, mroPaths, checkSrcPath); err != nil {
				return 0, err
			}
		}
		numFiles += len(fpaths)
	}
	return numFiles, nil
}

// Instantiate a pipestance object given a psid, MRO source, and a
// pipestance path. This is the core (private) method called by the
// public InvokeWithSource and Reattach methods.
func (self *Runtime) instantiatePipeline(src string, srcPath string, psid string,
	pipestancePath string, mroPaths []string, mroVersion string,
	envs map[string]string, readOnly bool) (string, *Pipestance, error) {
	// Parse the invocation source.
	postsrc, _, ast, err := parseSource(src, srcPath, mroPaths, !readOnly)
	if err != nil {
		return "", nil, err
	}

	// Check there's a call.
	if ast.Call == nil {
		return "", nil, &RuntimeError{"cannot start a pipeline without a call statement"}
	}
	// Make sure it's a pipeline we're calling.
	if pipeline := ast.Callables.Table[ast.Call.Id]; pipeline == nil {
		return "", nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared pipeline", ast.Call.Id)}
	}

	invocationJson, _ := self.BuildCallJSON(src, srcPath, mroPaths)

	// Instantiate the pipeline.
	pipestance := NewPipestance(NewTopNode(self, psid, pipestancePath, mroPaths, mroVersion, envs, invocationJson),
		ast.Call, ast.Callables)
	if pipestance == nil {
		return "", nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared pipeline", ast.Call.Id)}
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
	pipestancePath string, mroPaths []string, mroVersion string,
	envs map[string]string, tags []string) (*Pipestance, error) {

	// Error if pipestance directory is non-empty, otherwise create.
	if _, err := os.Stat(pipestancePath); err == nil {
		if fileInfos, err := ioutil.ReadDir(pipestancePath); err != nil || len(fileInfos) > 0 {
			return nil, &PipestanceExistsError{psid}
		}
	} else if err := os.MkdirAll(pipestancePath, 0777); err != nil {
		return nil, err
	}

	// Expand env vars in invocation source and instantiate.
	src = os.ExpandEnv(src)
	readOnly := false
	postsrc, pipestance, err := self.instantiatePipeline(src, srcPath, psid, pipestancePath, mroPaths,
		mroVersion, envs, readOnly)
	if err != nil {
		// If instantiation failed, delete the pipestance folder.
		os.RemoveAll(pipestancePath)
		return nil, err
	}

	// Write top-level metadata files.
	metadata := NewMetadata("ID."+psid, pipestancePath)
	metadata.writeRaw("invocation", src)
	metadata.writeRaw("mrosource", postsrc)
	metadata.writeRaw("jobmode", self.jobMode)
	metadata.write("versions", map[string]string{
		"martian":   self.martianVersion,
		"pipelines": mroVersion,
	})
	metadata.write("tags", tags)
	metadata.writeRaw("timestamp", "start: "+Timestamp())

	return pipestance, nil
}

func (self *Runtime) ReattachToPipestance(psid string, pipestancePath string, src string, mroPaths []string,
	mroVersion string, envs map[string]string, checkSrc bool, readOnly bool) (*Pipestance, error) {
	return self.reattachToPipestance(psid, pipestancePath, src, mroPaths, mroVersion, envs, checkSrc,
		readOnly, "invocation")
}

func (self *Runtime) ReattachToPipestanceWithMroSrc(psid string, pipestancePath string, src string, mroPaths []string,
	mroVersion string, envs map[string]string, checkSrc bool, readOnly bool) (*Pipestance, error) {
	return self.reattachToPipestance(psid, pipestancePath, src, mroPaths, mroVersion, envs, checkSrc,
		readOnly, "mrosource")
}

// Reattaches to an existing pipestance.
func (self *Runtime) reattachToPipestance(psid string, pipestancePath string, src string, mroPaths []string,
	mroVersion string, envs map[string]string, checkSrc bool, readOnly bool,
	srcType string) (*Pipestance, error) {
	fname := "_" + srcType
	invocationPath := path.Join(pipestancePath, fname)
	metadataPath := path.Join(pipestancePath, "_metadata.zip")

	// Read in the existing _invocation file.
	data, err := ioutil.ReadFile(invocationPath)
	if err != nil {
		return nil, &PipestancePathError{pipestancePath}
	}

	// Check if _invocation has changed.
	if checkSrc && src != string(data) {
		return nil, &PipestanceInvocationError{psid, invocationPath}
	}

	// Instantiate the pipestance.
	_, pipestance, err := self.instantiatePipeline(string(data), invocationPath, psid, pipestancePath, mroPaths,
		mroVersion, envs, readOnly)
	if err != nil {
		return nil, err
	}

	// If _jobmode exists, make sure we reattach to pipestance in the same job mode.
	if !readOnly {
		if err := pipestance.VerifyJobMode(); err != nil {
			pipestance.Unlock()
			return nil, err
		}
	}

	// If _metadata exists, unzip it so the pipestance can reads its metadata.
	if _, err := os.Stat(metadataPath); err == nil {
		if err := Unzip(metadataPath); err != nil {
			pipestance.Unlock()
			return nil, err
		}
		os.Remove(metadataPath)
	}

	// If we're reattaching in local mode, restart any stages that were
	// left in a running state from last mrp run. The actual job would
	// have been killed by the CTRL-C.
	if !readOnly {
		PrintInfo("runtime", "Reattaching in %s mode.", self.jobMode)
		if err = pipestance.RestartRunningNodes(self.jobMode); err != nil {
			pipestance.Unlock()
			return nil, err
		}
	}

	return pipestance, nil
}

// Instantiate a stagestance.
func (self *Runtime) InvokeStage(src string, srcPath string, ssid string,
	stagestancePath string, mroPaths []string, mroVersion string,
	envs map[string]string) (*Stagestance, error) {
	// Check if stagestance path already exists.
	if _, err := os.Stat(stagestancePath); err == nil {
		return nil, &RuntimeError{fmt.Sprintf("stagestance '%s' already exists", ssid)}
	} else if err := os.MkdirAll(stagestancePath, 0777); err != nil {
		return nil, err
	}

	// Parse the invocation source.
	src = os.ExpandEnv(src)
	_, _, ast, err := parseSource(src, srcPath, mroPaths, true)
	if err != nil {
		return nil, err
	}

	// Check there's a call.
	if ast.Call == nil {
		return nil, &RuntimeError{"cannot start a stage without a call statement"}
	}
	// Make sure it's a stage we're calling.
	if _, ok := ast.Callables.Table[ast.Call.Id].(*Stage); !ok {
		return nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared stage", ast.Call.Id)}
	}

	invocationJson, _ := self.BuildCallJSON(src, srcPath, mroPaths)

	// Instantiate stagestance.
	stagestance := NewStagestance(NewTopNode(self, "", stagestancePath, mroPaths, mroVersion, envs, invocationJson),
		ast.Call, ast.Callables)
	if stagestance == nil {
		return nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared stage", ast.Call.Id)}
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

func (self *Runtime) GetMetadata(pipestancePath string, metadataPath string) (string, error) {
	metadata := NewMetadata("", pipestancePath)
	metadata.loadCache()
	if metadata.exists("metadata.zip") {
		relPath, _ := filepath.Rel(pipestancePath, metadataPath)

		// Relative paths outside the pipestance directory will be ignored.
		if !strings.Contains(relPath, "..") {
			if data, err := ReadZip(metadata.makePath("metadata.zip"), relPath); err == nil {
				return data, nil
			}
		}
	}
	data, err := ioutil.ReadFile(metadataPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type MroCache struct {
	callableTable map[string]map[string]Callable
	pipelines     map[string]bool
}

func NewMroCache() *MroCache {
	self := &MroCache{}
	self.callableTable = map[string]map[string]Callable{}
	self.pipelines = map[string]bool{}

	return self
}

func (self *MroCache) CacheMros(mroPaths []string) {
	for _, mroPath := range mroPaths {
		self.callableTable[mroPath] = map[string]Callable{}
		fpaths, _ := filepath.Glob(mroPath + "/[^_]*.mro")
		for _, fpath := range fpaths {
			if data, err := ioutil.ReadFile(fpath); err == nil {
				if _, _, ast, err := parseSource(string(data), fpath, mroPaths, true); err == nil {
					for _, callable := range ast.Callables.Table {
						self.callableTable[mroPath][callable.getId()] = callable
						if _, ok := callable.(*Pipeline); ok {
							self.pipelines[callable.getId()] = true
						}
					}
				}
			}
		}
	}
}

func (self *MroCache) GetPipelines() []string {
	pipelines := []string{}
	for pipeline, _ := range self.pipelines {
		pipelines = append(pipelines, pipeline)
	}
	return pipelines
}

func (self *MroCache) GetCallable(mroPaths []string, name string) (Callable, error) {
	for _, mroPath := range mroPaths {
		// Make sure MROs from mroPath have been loaded.
		if _, ok := self.callableTable[mroPath]; !ok {
			return nil, &RuntimeError{fmt.Sprintf("MROs from mro path '%s' have not been loaded", mroPath)}
		}

		// Make sure pipeline has been loaded
		if callable, ok := self.callableTable[mroPath][name]; ok {
			return callable, nil
		}
	}
	return nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared pipeline or stage", name)}
}

func buildVal(param Param, val interface{}) string {
	// MRO value expression syntax is identical to JSON. Just need to make
	// sure floats get printed with decimal points.
	switch {
	case param.getTname() == "float" && param.getArrayDim() == 0 && val != nil:
		switch val.(type) {
		case json.Number:
			if num, err := val.(json.Number).Float64(); err == nil {
				return fmt.Sprintf("%f", num)
			}
		default:
			return fmt.Sprintf("%f", val)
		}
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
	}
	return fmt.Sprintf("<ParseError: %v>", val)
}

func (self *Runtime) BuildCallSource(incpaths []string, name string, args map[string]interface{},
	sweepargs []string, mroPaths []string) (string, error) {
	callable, err := self.MroCache.GetCallable(mroPaths, name)
	if err != nil {
		return "", err
	}

	// Build @include statements.
	includes := []string{}
	for _, incpath := range incpaths {
		includes = append(includes, fmt.Sprintf("@include \"%s\"", incpath))
	}
	// Loop over the pipeline's in params and print a binding
	// whether the args bag has a value for it not.
	lines := []string{}
	for _, param := range callable.getInParams().List {
		valstr := buildVal(param, args[param.getId()])

		for _, id := range sweepargs {
			if id == param.getId() {
				valstr = fmt.Sprintf("sweep(%s)", strings.Trim(valstr, "[]"))
				break
			}
		}

		lines = append(lines, fmt.Sprintf("    %s = %s,", param.getId(), valstr))
	}
	return fmt.Sprintf("%s\n\ncall %s(\n%s\n)", strings.Join(includes, "\n"),
		name, strings.Join(lines, "\n")), nil
}

func (self *Runtime) BuildCallJSON(src string, srcPath string, mroPaths []string) (map[string]interface{}, error) {
	_, incpaths, ast, err := parseSource(src, srcPath, mroPaths, false)
	if err != nil {
		return nil, err
	}

	if ast.Call == nil {
		return nil, &RuntimeError{"cannot jsonify a pipeline without a call statement"}
	}

	args := map[string]interface{}{}
	sweepargs := []string{}
	for _, binding := range ast.Call.Bindings.List {
		args[binding.Id] = expToInterface(binding.Exp)
		if binding.Sweep {
			sweepargs = append(sweepargs, binding.Id)
		}
	}
	return map[string]interface{}{
		"call":      ast.Call.Id,
		"args":      args,
		"sweepargs": sweepargs,
		"incpaths":  incpaths,
	}, nil
}

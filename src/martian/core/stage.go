//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian runtime stage management.
//

package core

import (
	"encoding/json"
	"fmt"
	"martian/syntax"
	"martian/util"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func makeOutArgs(outParams *syntax.Params, filesPath string) map[string]interface{} {
	args := map[string]interface{}{}
	for id, param := range outParams.Table {
		if param.IsFile() {
			args[id] = path.Join(filesPath, param.GetId()+"."+param.GetTname())
		} else if param.GetTname() == "path" {
			args[id] = path.Join(filesPath, param.GetId())
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

type StageDefs struct {
	ChunkDefs []map[string]interface{} `json:"chunks"`
	JoinDef   map[string]interface{}   `json:"join"`
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
	State    MetadataState          `json:"state"`
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
	self.metadata = NewMetadataWithJournalPath(self.fqname, self.path, self.node.journalPath)
	self.hasBeenRun = false
	if !self.node.split {
		// If we're not splitting, just set the sole chunk's filesPath
		// to the filesPath of the parent fork, to save a pseudo-join copy.
		self.metadata.filesPath = self.fork.metadata.filesPath
	}
	return self
}

func (self *Chunk) mkdirs() error {
	return self.metadata.mkdirs()
}

func (self *Chunk) getState() MetadataState {
	if state, ok := self.metadata.getState(); ok {
		return state
	} else {
		return Ready
	}
}

func (self *Chunk) updateState(state MetadataFileName) {
	self.metadata.cache(state)
	if state == ProgressFile {
		self.fork.lastPrint = time.Now()
		if msg, err := self.metadata.readRawSafe(state); err == nil {
			util.PrintInfo("runtime", "(progress)        %s: %s", self.fqname, msg)
		} else {
			util.LogError(err, "progres", "Error reading progress file for %s", self.fqname)
		}
	}
}

func (self *Chunk) step() {
	if self.getState() != Ready {
		return
	}

	// Belt and suspenders for not double-submitting a job.
	if self.hasBeenRun {
		return
	} else {
		self.hasBeenRun = true
	}

	threads, memGB, special := self.node.setChunkJobReqs(self.chunkDef)

	// Resolve input argument bindings and merge in the chunk defs.
	resolvedBindings := resolveBindings(self.node.argbindings, self.fork.argPermute)
	for id, value := range self.chunkDef {
		resolvedBindings[id] = value
	}

	// Write out input and ouput args for the chunk.
	self.metadata.Write(ArgsFile, resolvedBindings)
	self.metadata.Write(OutsFile, makeOutArgs(self.node.outparams, self.metadata.filesPath))

	// Run the chunk.
	self.fork.lastPrint = time.Now()
	self.node.runChunk(self.fqname, self.metadata, threads, memGB, special)
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
	numThreads, _, _ := self.node.getJobReqs(self.chunkDef, STAGE_TYPE_CHUNK)
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
	lastPrint      time.Time
}

type ForkInfo struct {
	Index         int                    `json:"index"`
	ArgPermute    map[string]interface{} `json:"argPermute"`
	JoinDef       map[string]interface{} `json:"joinDef"`
	State         MetadataState          `json:"state"`
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
	self.lastPrint = time.Now()

	// By default, initialize stage defs with one empty chunk.
	self.stageDefs = &StageDefs{ChunkDefs: []map[string]interface{}{}, JoinDef: map[string]interface{}{}}
	self.stageDefs.ChunkDefs = append(self.stageDefs.ChunkDefs, map[string]interface{}{})

	if err := json.Unmarshal([]byte(self.split_metadata.readRaw(StageDefsFile)), &self.stageDefs); err == nil {
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
	self.split_metadata.notRunningSince = time.Time{}
	self.split_metadata.lastRefresh = time.Time{}
	self.join_metadata.notRunningSince = time.Time{}
	self.join_metadata.lastRefresh = time.Time{}
}

func (self *Fork) resetPartial() error {
	self.lastPrint = time.Now()
	if err := self.split_metadata.checkedReset(); err != nil {
		return err
	}
	if err := self.join_metadata.checkedReset(); err != nil {
		return err
	}
	for _, chunk := range self.chunks {
		if err := chunk.metadata.checkedReset(); err != nil {
			return err
		}
	}
	return nil
}

func (self *Fork) restartLocallyQueuedJobs() error {
	self.lastPrint = time.Now()
	if err := self.split_metadata.restartQueuedLocal(); err != nil {
		return err
	}
	if err := self.join_metadata.restartQueuedLocal(); err != nil {
		return err
	}
	for _, chunk := range self.chunks {
		if err := chunk.metadata.restartQueuedLocal(); err != nil {
			return err
		}
	}
	return nil
}

func (self *Fork) restartLocalJobs() error {
	self.lastPrint = time.Now()
	if err := self.split_metadata.restartLocal(); err != nil {
		return err
	}
	if err := self.join_metadata.restartLocal(); err != nil {
		return err
	}
	for _, chunk := range self.chunks {
		if err := chunk.metadata.restartLocal(); err != nil {
			return err
		}
	}
	return nil
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
		outputs, ok := self.metadata.read(OutsFile).(map[string]interface{})
		if !ok {
			msg += fmt.Sprintf("Fork outs were not a map\n")
		}
		for _, param := range outparams.Table {
			val, ok := outputs[param.GetId()]
			if !ok {
				msg += fmt.Sprintf("Fork did not return parameter '%s'\n", param.GetId())
				ret = false
				continue
			}
			if val == nil {
				// Allow for null output parameters
				continue
			}
			if !dynamicCast(val, param.GetTname(), param.GetArrayDim()) {
				msg += fmt.Sprintf("Fork returned %s parameter '%s' with incorrect type\n", param.GetTname(), param.GetId())
				ret = false
			}
		}
	}
	return ret, msg
}

func (self *Fork) getState() MetadataState {
	if state, _ := self.metadata.getState(); state == Failed || state == Complete {
		return state
	}
	if state, ok := self.join_metadata.getState(); ok {
		if state == Failed {
			return state
		} else {
			return state.Prefixed(JoinPrefix)
		}
	}
	if len(self.chunks) > 0 {
		// If any chunks have failed, we're failed.
		for _, chunk := range self.chunks {
			if chunk.getState() == Failed {
				return Failed
			}
		}
		// If every chunk is complete, we're complete.
		every := true
		for _, chunk := range self.chunks {
			if chunk.getState() != Complete {
				every = false
				break
			}
		}
		if every {
			return Complete.Prefixed(ChunksPrefix)
		}
		// If every chunk is queued, running, or complete, we're complete.
		every = true
		runningStates := map[MetadataState]bool{Queued: true, Running: true, Complete: true}
		for _, chunk := range self.chunks {
			if _, ok := runningStates[chunk.getState()]; !ok {
				every = false
				break
			}
		}
		if every {
			return Running.Prefixed(ChunksPrefix)
		}
	}
	if state, ok := self.split_metadata.getState(); ok {
		if state == Failed {
			return state
		} else {
			return state.Prefixed(SplitPrefix)
		}
	}
	return Ready
}

func (self *Fork) updateState(state string) {
	if state == string(ProgressFile) {
		self.lastPrint = time.Now()
		if msg, err := self.metadata.readRawSafe(MetadataFileName(state)); err == nil {
			util.PrintInfo("runtime", "(progress)        %s: %s", self.fqname, msg)
		} else {
			util.LogError(err, "progres", "Error reading progress file for %s", self.fqname)
		}
	}
	if strings.HasPrefix(state, SplitPrefix) {
		self.split_metadata.cache(MetadataFileName(strings.TrimPrefix(state, SplitPrefix)))
	} else if strings.HasPrefix(state, JoinPrefix) {
		self.join_metadata.cache(MetadataFileName(strings.TrimPrefix(state, JoinPrefix)))
	} else {
		self.metadata.cache(MetadataFileName(state))
	}
}

func (self *Fork) getChunk(index int) *Chunk {
	if index < len(self.chunks) {
		return self.chunks[index]
	}
	return nil
}

func (self *Fork) skip() {
	self.metadata.WriteTime(CompleteFile)
}

func (self *Fork) writeInvocation() {
	if !self.metadata.exists(InvocationFile) {
		argBindings := resolveBindings(self.node.argbindings, self.argPermute)
		sweepBindings := []string{}
		incpaths := self.node.invocation.IncludePaths
		invocation, _ := self.node.rt.BuildCallSource(incpaths, self.node.name, argBindings, sweepBindings, self.node.mroPaths)
		self.metadata.WriteRaw(InvocationFile, invocation)
	}
}

func (self *Fork) step() {
	if self.node.kind == "stage" {
		state := self.getState()
		if !state.IsRunning() && !state.IsQueued() {
			statePad := strings.Repeat(" ", int(math.Max(0, float64(15-len(state)))))
			// Only log fork num if we've got more than one fork
			fqname := self.node.fqname
			if len(self.node.forks) > 1 {
				fqname = self.fqname
			}
			self.lastPrint = time.Now()
			msg := fmt.Sprintf("(%s)%s %s", state, statePad, fqname)
			if self.node.preflight {
				util.LogInfo("runtime", msg)
			} else {
				util.PrintInfo("runtime", msg)
			}
		}

		if state == Ready {
			self.writeInvocation()
			self.split_metadata.Write(ArgsFile, resolveBindings(self.node.argbindings, self.argPermute))
			threads, memGB, special := self.node.setSplitJobReqs()
			if self.node.split {
				if !self.split_has_run {
					self.split_has_run = true
					self.lastPrint = time.Now()
					self.node.runSplit(self.fqname, self.split_metadata, threads, memGB, special)
				}
			} else {
				self.split_metadata.Write(StageDefsFile, self.stageDefs)
				self.split_metadata.WriteTime(CompleteFile)
			}
		} else if state == Complete.Prefixed(SplitPrefix) {
			// MARTIAN-395 We have observed a possible race condition where
			// split_complete could be detected but _stage_defs is not
			// written yet or is corrupted. Check that stage_defs exists
			// before attempting to read and unmarshal it.
			if self.split_metadata.exists(StageDefsFile) {
				if err := json.Unmarshal([]byte(self.split_metadata.readRaw(StageDefsFile)), &self.stageDefs); err != nil || len(self.stageDefs.ChunkDefs) == 0 {
					errstring := "none"
					if err != nil {
						errstring = err.Error()
					}
					self.split_metadata.WriteRaw(Errors,
						fmt.Sprintf("The split method did not return a dictionary {'chunks': [{}], 'join': {}}.\nError: %s\nChunk count: %d", errstring, len(self.stageDefs.ChunkDefs)))
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
			}
		} else if state == Complete.Prefixed(ChunksPrefix) {
			threads, memGB, special := self.node.setJoinJobReqs(self.stageDefs.JoinDef)
			resolvedBindings := resolveBindings(self.node.argbindings, self.argPermute)
			for id, value := range self.stageDefs.JoinDef {
				resolvedBindings[id] = value
			}
			self.join_metadata.Write(ArgsFile, resolvedBindings)
			self.join_metadata.Write(ChunkDefsFile, self.stageDefs.ChunkDefs)
			if self.node.split {
				chunkOuts := []interface{}{}
				for _, chunk := range self.chunks {
					outs := chunk.metadata.read(OutsFile)
					chunkOuts = append(chunkOuts, outs)
				}
				self.join_metadata.Write(ChunkOutsFile, chunkOuts)
				self.join_metadata.Write(OutsFile, makeOutArgs(self.node.outparams, self.metadata.filesPath))
				if !self.join_has_run {
					self.join_has_run = true
					self.lastPrint = time.Now()
					self.node.runJoin(self.fqname, self.join_metadata, threads, memGB, special)
				}
			} else {
				self.join_metadata.Write(OutsFile, self.chunks[0].metadata.read(OutsFile))
				self.join_metadata.WriteTime(CompleteFile)
			}
		} else if state == Complete.Prefixed(JoinPrefix) {
			self.metadata.Write(OutsFile, self.join_metadata.read(OutsFile))
			if ok, msg := self.verifyOutput(); ok {
				self.metadata.WriteTime(CompleteFile)
			} else {
				self.metadata.WriteRaw(Errors, msg)
			}
		}

	} else if self.node.kind == "pipeline" {
		self.writeInvocation()
		self.metadata.Write(OutsFile, resolveBindings(self.node.retbindings, self.argPermute))
		if ok, msg := self.verifyOutput(); ok {
			self.metadata.WriteTime(CompleteFile)
		} else {
			self.metadata.WriteRaw(Errors, msg)
		}
	}
}

func (self *Fork) printUpdateIfNeeded() {
	if time.Since(self.lastPrint) > forkPrintInterval {
		if state := self.getState(); state.IsRunning() {
			if state.HasPrefix(ChunksPrefix) && len(self.chunks) > 1 {
				doneCount := 0
				for _, chunk := range self.chunks {
					if chunk.getState() == Complete {
						doneCount++
					}
				}
				self.lastPrint = time.Now()
				util.PrintInfo("runtime",
					"(update)          %s chunks running (%d/%d completed)",
					self.fqname, doneCount, len(self.chunks))
			} else {
				self.lastPrint = time.Now()
				util.PrintInfo("runtime",
					"(update)          %s %v", self.fqname, state)
			}
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
	if self.metadata.exists(VdrKill) {
		data := self.metadata.readRaw(VdrKill)
		if err := json.Unmarshal([]byte(data), &killReport); err == nil {
			ok = true
		}
	}
	return killReport, ok
}

func (self *Fork) postProcess() {
	// Handle formal output parameters
	pipestancePath := self.node.parent.getNode().path
	outsPath := path.Join(pipestancePath, "outs")

	// Handle multi-fork sweeps
	if len(self.node.forks) > 1 {
		outsPath = path.Join(outsPath, fmt.Sprintf("fork%d", self.index))
		util.Print("\nOutputs (fork%d):\n", self.index)
	} else {
		util.Print("\nOutputs:\n")
	}

	// Create the fork-specific outs/ folder
	util.MkdirAll(outsPath)

	// Get fork's output parameter values
	outs := map[string]interface{}{}
	if data := self.metadata.read(OutsFile); data != nil {
		if v, ok := data.(map[string]interface{}); ok {
			outs = v
		}
	}

	// Error message accumulator
	errors := []error{}

	// Calculate longest key name for alignment
	paramList := self.node.outparams.List
	keyWidth := 0
	for _, param := range paramList {
		// Print out the param help and value
		key := param.GetHelp()
		if len(key) == 0 {
			key = param.GetId()
		}
		if len(key) > keyWidth {
			keyWidth = len(key)
		}
	}

	// Iterate through output parameters
	for _, param := range paramList {
		// Pull the param value from the fork _outs
		// If value not available, report null
		id := param.GetId()
		value, ok := outs[id]
		if !ok || value == nil {
			value = "null"
		}

		// Handle file and path params
		for {
			if !param.IsFile() && param.GetTname() != "path" {
				break
			}
			// Make sure value is a string
			filePath, ok := value.(string)
			if !ok {
				break
			}

			// If file doesn't exist (e.g. stage just didn't create it)
			// then report null
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				value = "null"
				break
			}

			// Generate the outs path for this param
			outPath := ""
			if len(param.GetOutName()) > 0 {
				// If MRO explicitly specifies an out name
				// override, just use that verbatim.
				outPath = path.Join(outsPath, param.GetOutName())
			} else {
				// Otherwise, just use the parameter name, and
				// append the type unless it is a path.
				outPath = path.Join(outsPath, id)
				if param.GetTname() != "path" {
					outPath += "." + param.GetTname()
				}
			}

			// Only continue if path to be copied is inside the pipestance
			if absFilePath, err := filepath.Abs(filePath); err == nil {
				if absPipestancePath, err := filepath.Abs(pipestancePath); err == nil {
					if !strings.Contains(absFilePath, absPipestancePath) {
						// But we still want a symlink
						if err := os.Symlink(absFilePath, outPath); err != nil {
							errors = append(errors, err)
						}
						break
					}
				}
			}

			// If this param has already been moved to outs/, we're done
			if _, err := os.Stat(outPath); err == nil {
				break
			}

			// If source file exists, move it to outs/
			if err := os.Rename(filePath, outPath); err != nil {
				errors = append(errors, err)
				break
			}

			// Generate the relative path from files/ to outs/
			relPath, err := filepath.Rel(filepath.Dir(filePath), outPath)
			if err != nil {
				errors = append(errors, err)
				break
			}

			// Symlink it back to the original files/ folder
			if err := os.Symlink(relPath, filePath); err != nil {
				errors = append(errors, err)
				break
			}

			value = outPath
			break
		}

		// Print out the param help and value
		key := param.GetHelp()
		if len(key) == 0 {
			key = param.GetId()
		}
		keyPad := strings.Repeat(" ", keyWidth-len(key))
		util.Print("- %s:%s %v\n", key, keyPad, value)
	}

	// Print errors
	if len(errors) > 0 {
		util.Print("\nCould not move output files:\n")
		for _, err := range errors {
			util.Print("%s\n", err.Error())
		}
	}
	util.Print("\n")

	// Print alerts
	if alarms := self.getAlarms(); len(alarms) > 0 {
		self.lastPrint = time.Now()
		if len(self.node.forks) > 1 {
			util.Print("Alerts (fork%d):\n", self.index)
		} else {
			util.Print("Alerts:\n")
		}
		util.Print(alarms + "\n")
	}
}

func (self *Fork) getAlarms() string {
	alarms := ""
	for _, metadata := range self.collectMetadatas() {
		if !metadata.exists(AlarmFile) {
			continue
		}
		alarms += metadata.readRaw(AlarmFile)
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

	numThreads, _, _ := self.node.getJobReqs(nil, STAGE_TYPE_SPLIT)
	splitStats := self.split_metadata.serializePerf(numThreads)
	if splitStats != nil {
		stats = append(stats, splitStats)
	}

	numThreads, _, _ = self.node.getJobReqs(self.stageDefs.JoinDef, STAGE_TYPE_JOIN)
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

//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Martian runtime stage management.
//

package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"runtime/trace"
	"strings"
	"sync"
	"time"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

// For convenience, a string which implements json.Marshaler.
type jsonString string

func (s jsonString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

// makeOutArg generates the value with which to pre-populate an entry in the
// _outs json file.
//
// If brokenFileArrays is true, then for backwards compatibility it will return
// a file name for arrays of file types.
//
// An empty array is returned for arrays.  An empty map is returned for maps.
// A map with recursively-populated fields is returned for struct types.
// Nil is returned for any other type.
func makeOutArg(field *syntax.StructMember, filesPath string) json.Marshaler {
	if field.Tname.ArrayDim > 0 {
		return marshallerArray{}
	} else if field.Tname.MapDim > 0 {
		return MarshalerMap{}
	} else if field.IsFile() == syntax.KindIsFile {
		if fn := field.GetOutFilename(); fn != "" {
			return jsonString(path.Join(filesPath, fn))
		}
	}
	return nil
}

func makeOutArgs(outParams *syntax.OutParams, filesPath string, nullAll bool) MarshalerMap {
	args := make(MarshalerMap, len(outParams.List))
	for _, param := range outParams.List {
		if nullAll {
			args[param.Id] = nil
		} else {
			args[param.Id] = makeOutArg(&param.StructMember, filesPath)
		}
	}
	return args
}

// Escape hatch for this feature in case of weird nfs servers which don't
// work for whatever reason.
var disableUniquification = (os.Getenv("MRO_UNIQUIFIED_DIRECTORIES") == disable)

//=============================================================================
// Chunk
//=============================================================================

// Represents the state of a stage chunk (the "main" method).
type Chunk struct {
	fork       *Fork
	chunkDef   *ChunkDef
	metadata   *Metadata
	fqname     string
	index      int
	hasBeenRun bool
}

// Exportable information about a Chunk object.
type ChunkInfo struct {
	ChunkDef *ChunkDef     `json:"chunkDef"`
	Metadata *MetadataInfo `json:"metadata"`
	State    MetadataState `json:"state"`
	Index    int           `json:"index"`
}

func NewChunk(fork *Fork, index int,
	chunkDef *ChunkDef, chunkIndexWidth int) *Chunk {
	self := &Chunk{
		fork:     fork,
		index:    index,
		chunkDef: chunkDef,
	}
	chnkNum := fmt.Sprintf("chnk%0*d",
		chunkIndexWidth, index)
	chunkPath := path.Join(fork.path, chnkNum)
	self.fqname = fork.fqname + "." + chnkNum

	journalName := strings.TrimPrefix(strings.TrimPrefix(self.fqname, self.fork.node.top.fqname), ".")
	self.metadata = newMetadataWithJournalPath(self.fqname, journalName,
		chunkPath, self.fork.node.top.journalPath)
	self.metadata.discoverUniquify()
	// HACK: Sometimes we need to load older pipestances with newer martian
	// versions.  Because of this, we may sometimes encounter chunks which
	// used the older, non-padded chunk ID.  On the brighter side, these
	// were all created pre-uniquification so there's nothing to worry about
	// with symlinks.
	if self.metadata.uniquifier == "" {
		legacyPath := path.Join(fork.path, fmt.Sprintf("chnk%d", index))
		if legacyPath != chunkPath {
			if info, err := os.Stat(legacyPath); err == nil && info != nil {
				if info.IsDir() {
					self.metadata = newMetadataWithJournalPath(
						self.fqname, journalName,
						legacyPath, self.fork.node.top.journalPath)
				}
			}
		}
	}
	self.hasBeenRun = false
	if !self.fork.Split() {
		// If we're not splitting, just set the sole chunk's filesPath
		// to the filesPath of the parent fork, to save a pseudo-join copy.
		self.metadata.finalFilePath = self.fork.metadata.finalFilePath
		if disableUniquification {
			self.metadata.curFilesPath = self.metadata.finalFilePath
		}
	}
	return self
}

func (self *Chunk) verifyDef() {
	level := syntax.GetEnforcementLevel()
	if level <= syntax.EnforceDisable {
		return
	}
	inParams := self.Stage().ChunkIns
	if inParams == nil {
		return
	}
	if self.chunkDef.Args == nil {
		self.metadata.WriteErrorString("Chunk def args were nil.")
		return
	}
	err, alarms := self.chunkDef.Args.ValidateInputs(self.fork.node.top.types, inParams)
	if err != nil {
		if level >= syntax.EnforceError {
			self.metadata.WriteErrorString(err.Error() + alarms)
			return
		}
		if alarms == "" {
			alarms = err.Error() + "\n"
		} else {
			alarms = err.Error() + "\n" + alarms
		}
	}
	if alarms != "" {
		switch syntax.GetEnforcementLevel() {
		case syntax.EnforceError:
			self.metadata.WriteErrorString(alarms)
		case syntax.EnforceAlarm:
			err := self.metadata.AppendAlarm(alarms)
			if err == nil {
				return
			}
			// Error writing alarm, so log it at least.
			fallthrough
		case syntax.EnforceLog:
			util.PrintInfo("runtime",
				"(outputs)         %s: WARNING: invalid chunk definition\n%s",
				self.fork.fqname, strings.TrimSpace(alarms))
		}
	}
}

func (self *Chunk) verifyOutput(output LazyArgumentMap) bool {
	level := syntax.GetEnforcementLevel()
	if level <= syntax.EnforceDisable {
		return true
	}
	if len(self.fork.OutParams().List) == 0 &&
		(self.Stage().ChunkOuts == nil ||
			len(self.Stage().ChunkOuts.List) == 0) {
		return true
	}
	if output == nil {
		self.metadata.WriteErrorString("Output not found.")
	} else {
		outParams := self.Stage().ChunkOuts
		err, alarms := output.ValidateOutputs(self.fork.node.top.types,
			outParams, self.fork.OutParams())
		if err != nil {
			if level >= syntax.EnforceError {
				self.metadata.WriteErrorString(err.Error() + alarms)
				return false
			}
			if alarms == "" {
				alarms = err.Error() + "\n"
			} else {
				alarms = err.Error() + "\n" + alarms
			}
		}
		if alarms != "" {
			switch syntax.GetEnforcementLevel() {
			case syntax.EnforceError:
				self.metadata.WriteErrorString(alarms)
				return false
			case syntax.EnforceAlarm:
				err := self.metadata.AppendAlarm(alarms)
				if err == nil {
					return true
				}
				fallthrough
			case syntax.EnforceLog:
				util.PrintInfo("runtime",
					"(outputs)         %s: WARNING: invalid chunk definition\n%s",
					self.fork.fqname, strings.TrimSpace(alarms))
			}
		}
	}
	return true
}

func (self *Chunk) mkdirs() error {
	if state := self.getState(); !disableUniquification &&
		state != Complete {
		return self.metadata.uniquify()
	} else {
		return self.metadata.mkdirs()
	}
}

func (self *Chunk) getState() MetadataState {
	if state, ok := self.metadata.getState(); ok {
		return state
	} else {
		return Ready
	}
}

func (self *Chunk) updateState(state MetadataFileName, uniquifier string) {
	beginState, _ := self.metadata.getState()
	self.metadata.cache(state, uniquifier)
	if state == ProgressFile {
		self.fork.lastPrint = time.Now()
		if msg, err := self.metadata.readRawSafe(state); err == nil {
			util.PrintInfo("runtime",
				"(progress)        %s: %s",
				self.fqname, msg)
		} else {
			util.LogError(err, "progres",
				"Error reading progress file for %s",
				self.fqname)
		}
	}
	if beginState == Running || beginState == Queued {
		if st, _ := self.metadata.getState(); st != Running && st != Queued {
			self.fork.node.top.rt.JobManager.endJob(self.metadata)
		}
	}
}

func (self *Chunk) step(bindings MarshalerMap) {
	if self.getState() != Ready {
		return
	}

	// Belt and suspenders for not double-submitting a job.
	if self.hasBeenRun {
		return
	} else {
		self.hasBeenRun = true
	}

	if self.chunkDef.Resources == nil {
		self.chunkDef.Resources = &JobResources{}
	}
	res := self.fork.node.setChunkJobReqs(self.chunkDef.Resources)

	// Resolve input argument bindings and merge in the chunk defs.
	resolvedBindings := self.chunkDef.Merge(bindings)

	// Write out input and output args for the chunk.
	self.metadata.Write(ArgsFile, resolvedBindings)
	outs := makeOutArgs(self.fork.OutParams(), self.metadata.curFilesPath, false)
	if self.fork.Split() {
		for k, v := range makeOutArgs(self.Stage().ChunkOuts,
			self.metadata.curFilesPath, false) {
			outs[k] = v
		}
	}
	self.metadata.Write(OutsFile, outs)

	// Run the chunk.
	self.fork.lastPrint = time.Now()
	self.fork.node.runChunk(self.fqname, self.metadata, &res)
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
	res := self.fork.node.getJobReqs(self.chunkDef.Resources, STAGE_TYPE_CHUNK)
	stats := self.metadata.serializePerf(res.Threads)
	return &ChunkPerfInfo{
		Index:      self.index,
		ChunkStats: stats,
	}
}

// Get the stage definition for this chunk.  Panics if this is not a stage fork.
func (self *Chunk) Stage() *syntax.Stage {
	return self.fork.node.call.Callable().(*syntax.Stage)
}

//=============================================================================
// Fork
//=============================================================================

// Represents a fork of a stage or pipeline.  When sweaping over multiple
// possible values for an input parameter, there will be more than one fork for
// a given pipeline or stage.
type Fork struct {
	node           *Node
	forkId         ForkId
	id             string
	path           string
	fqname         string
	metadata       *Metadata
	split_metadata *Metadata
	join_metadata  *Metadata
	chunks         []*Chunk
	stageDefs      *StageDefs
	perfCache      *ForkPerfCache
	lastPrint      time.Time

	// Caches the set of strict-mode VDR-able files and the
	// arguments which are keeping them alive.
	fileParamMap map[string]*vdrFileCache

	// Mapping from argument name to set of nodes which depend on the
	// argument, for arguments which may contain any file names.  This
	// includes user-defined file types, strings, maps, or arrays of any
	// of those.  Nothing else (int, float, bool) can contain file names.
	fileArgs map[string]map[Nodable]struct{}

	// Mapping from post-node to set of file-type args it depends on.
	filePostNodes map[Nodable]map[string]syntax.Type

	metadatasCache []*Metadata // cache for collectMetadata

	storageLock   sync.Mutex
	index         int
	split_has_run bool
	join_has_run  bool
}

// Exportable information from a Fork object.
type ForkInfo struct {
	ArgPermute    map[string]interface{} `json:"argPermute"`
	JoinDef       *JobResources          `json:"joinDef"`
	State         MetadataState          `json:"state"`
	Metadata      *MetadataInfo          `json:"metadata"`
	SplitMetadata *MetadataInfo          `json:"split_metadata"`
	JoinMetadata  *MetadataInfo          `json:"join_metadata"`
	Bindings      *ForkBindingsInfo      `json:"bindings"`
	Chunks        []*ChunkInfo           `json:"chunks"`
	Index         int                    `json:"index"`
}

type ForkBindingsInfo struct {
	Argument []BindingInfo `json:"Argument"`
	Return   []BindingInfo `json:"Return"`
}

type ForkPerfCache struct {
	perfInfo      *ForkPerfInfo
	vdrKillReport *VDRKillReport
}

func NewFork(nodable Nodable, index int, id ForkId) *Fork {
	self := &Fork{
		node:  nodable.getNode(),
		index: index,
		// By default, initialize stage defs with one empty chunk.
		stageDefs: &StageDefs{ChunkDefs: []*ChunkDef{new(ChunkDef)}},
	}
	self.updateId(id)
	self.lastPrint = time.Now()

	return self
}

// The constraints on the fork part of the metadata journal file name are
// a bit more than those on the fork ID - they can't use a slash to
// separate nested fork components, and they can't contain a '.' character
// as that would break the journal filename parsing scheme.
var encodeJournalName = strings.NewReplacer(".", "%2E", "/", "%2F")

func (self *Fork) updateId(id ForkId) {
	self.forkId = id
	if idx, err := id.ForkIdString(); err != nil {
		panic(err)
	} else {
		self.id = idx
	}
	oldPath := self.path
	self.path = path.Join(self.node.path, self.id)
	self.fqname = self.node.call.GetFqid() + "." + encodeJournalName.Replace(self.id)
	self.metadata = NewMetadata(self.fqname, self.path)
	self.split_metadata = NewMetadata(self.fqname+".split",
		path.Join(self.path, "split"))
	self.split_metadata.journalPath = path.Join(self.node.top.journalPath,
		strings.TrimPrefix(strings.TrimPrefix(
			self.fqname, self.node.top.fqname), "."))
	self.join_metadata = NewMetadata(self.fqname+".join",
		path.Join(self.path, "join"))
	self.join_metadata.journalPath = self.split_metadata.journalPath
	if self.Split() {
		self.split_metadata.discoverUniquify()
		self.join_metadata.finalFilePath = self.metadata.finalFilePath
		self.join_metadata.discoverUniquify()
	}
	// If we updated the path, we should load stage defs and create chunks.
	if self.path != oldPath {
		if err := self.split_metadata.ReadInto(StageDefsFile, &self.stageDefs); err == nil {
			width := util.WidthForInt(len(self.stageDefs.ChunkDefs))
			self.chunks = make([]*Chunk, 0, len(self.stageDefs.ChunkDefs))
			for i, chunkDef := range self.stageDefs.ChunkDefs {
				chunk := NewChunk(self, i, chunkDef, width)
				self.chunks = append(self.chunks, chunk)
			}
		}
	}
}

func (self *Fork) Split() bool {
	if stage, ok := self.node.call.Callable().(*syntax.Stage); ok {
		return stage.Split
	} else {
		return false
	}
}

func (self *Fork) ChunkOutParams() *syntax.OutParams {
	if stage, ok := self.node.call.Callable().(*syntax.Stage); ok {
		return stage.ChunkOuts
	} else {
		return nil
	}
}

// Get the fork's output parameter list.
func (self *Fork) OutParams() *syntax.OutParams {
	return self.node.call.Callable().GetOutParams()
}

func (self *Fork) kill(message string) {
	if state, _ := self.split_metadata.getState(); state == Queued || state == Running {
		self.split_metadata.WriteErrorString(message)
	}
	if state, _ := self.join_metadata.getState(); state == Queued || state == Running {
		self.join_metadata.WriteErrorString(message)
	}
	for _, chunk := range self.chunks {
		if state := chunk.getState(); state == Queued || state == Running {
			chunk.metadata.WriteErrorString(message)
		}
	}
}

func (self *Fork) reset() {
	for _, chunk := range self.chunks {
		self.node.top.rt.JobManager.endJob(chunk.metadata)
	}
	self.chunks = nil
	self.metadatasCache = nil
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

func (metadata *Metadata) reattachJob(j JobManager) bool {
	if metadata.exists(QueuedLocally) {
		return true
	}
	st, _ := metadata.getState()
	if st == Running || st == Queued {
		j.reattach(metadata)
	}
	return false
}

// Re-acquires the max jobs semaphore for jobs which were queued or running.
// Does _not_ do so for jobs which are queued locally, as we need to make sure
// we acquire the semaphore for all of the jobs which weren't queued locally
// before we can do those.
func (self *Fork) reattachJobs() error {
	anyQueuedLocally := self.split_metadata.reattachJob(self.node.top.rt.JobManager)
	anyQueuedLocally = self.join_metadata.reattachJob(self.node.top.rt.JobManager) || anyQueuedLocally
	for _, chunk := range self.chunks {
		anyQueuedLocally = chunk.metadata.reattachJob(self.node.top.rt.JobManager) || anyQueuedLocally
	}
	if anyQueuedLocally {
		return self.restartLocallyQueuedJobs()
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
	metadatas := self.metadatasCache
	if metadatas == nil {
		metadatas = make([]*Metadata, 3, 3+len(self.chunks))
		metadatas[0], metadatas[1], metadatas[2] =
			self.metadata, self.split_metadata, self.join_metadata
		for _, chunk := range self.chunks {
			metadatas = append(metadatas, chunk.metadata)
		}
		self.metadatasCache = metadatas
	}
	return metadatas
}

func (self *Fork) removeMetadata() {
	rem := func(metadata *Metadata) {
		filePaths, _ := metadata.enumerateFiles()
		if len(filePaths) == 0 {
			metadata.removeAll()
		}
	}
	rem(self.split_metadata)
	rem(self.join_metadata)
	for _, chunk := range self.chunks {
		rem(chunk.metadata)
	}
}

func (self *Fork) mkdirs() {
	// Use MkdirAll here in case of nested forking.
	if err := self.metadata.mkForkDirs(); err != nil {
		self.metadata.writeError("Could not create directories for ", err)
	}
	if state, ok := self.split_metadata.getState(); !disableUniquification &&
		self.Split() &&
		(!ok || (state != Complete && state != DisabledState)) {
		self.split_metadata.uniquify()
	} else {
		self.split_metadata.mkdirs()
	}
	if state, ok := self.join_metadata.getState(); !disableUniquification &&
		self.Split() &&
		(!ok || (state != Complete && state != DisabledState)) {
		self.join_metadata.uniquify()
	} else {
		self.join_metadata.mkdirs()
	}

	for _, chunk := range self.chunks {
		chunk.mkdirs()
	}
}

var escapedPathSep = func() []byte {
	s, err := json.Marshal(string([]rune{os.PathSeparator}))
	if err != nil {
		panic(err)
	}
	return bytes.Trim(s, "\"")
}()

// Get strings appearing in data as deserialized from json, e.g. recursively
// searching map[string]interface{}, []interface{} and string.  Ignores bool
// and json.Number/float64.  Ignores strings which do not contain a path
// separator character, since there is no way for downstream stages to figure
// out the location of this stage's file outputs without an absolute path, or
// at least one relative to something the downstream stage can know about.
func getMaybeFileNames(value json.Marshaler) []string {
	switch value := value.(type) {
	case marshallerArray:
		var result []string
		for _, v := range value {
			if r := getMaybeFileNames(v); len(r) > 0 {
				if len(result) == 0 {
					result = r
				} else {
					result = append(result, r...)
				}
			}
		}
		return result
	case LazyArgumentMap:
		var result []string
		for k, v := range value {
			if len(k) > 0 && k[0] == os.PathSeparator {
				result = append(result, k)
			}
			if r := getMaybeFileNames(v); len(r) > 0 {
				if len(result) == 0 {
					result = r
				} else {
					result = append(result, r...)
				}
			}
		}
		return result
	case json.RawMessage:
		value = json.RawMessage(bytes.TrimSpace(value))
		if len(value) == 0 || bytes.Equal(value, nullBytes) {
			return nil
		}
		if !bytes.Contains(value, escapedPathSep) {
			return nil
		}
		if value[0] == '[' {
			var arr []json.RawMessage
			if json.Unmarshal(value, &arr) == nil {
				var result []string
				for _, v := range arr {
					if r := getMaybeFileNames(v); len(r) > 0 {
						if len(result) == 0 {
							result = r
						} else {
							result = append(result, r...)
						}
					}
				}
				return result
			}
		} else if value[0] == '{' {
			var m LazyArgumentMap
			if json.Unmarshal(value, &m) == nil {
				return getMaybeFileNames(m)
			}
		} else if value[0] == '"' {
			var s string
			if json.Unmarshal(value, &s) == nil &&
				len(s) > 0 && path.IsAbs(s) {
				return []string{s}
			}
		}
		return nil
	default:
		return nil
	}
}

// Once the job has completed, look for arguments which might contain file
// names and remove any which didn't actually have any.
func (self *Fork) removeEmptyFileArgs(outs LazyArgumentMap) {
	self.storageLock.Lock()
	defer self.storageLock.Unlock()
	if len(self.fileArgs) == 0 {
		return
	} else {
		for arg := range self.fileArgs {
			if val := outs.jsonPath(arg); len(getMaybeFileNames(val)) == 0 {
				self.removeFileArg(arg)
			}
		}
	}
}

func (self *Fork) verifyOutput(outs LazyArgumentMap) (bool, string) {
	outparams := self.OutParams()
	if len(outparams.List) > 0 {
		if err, alarms := outs.ValidateOutputs(self.node.top.types, outparams); err != nil {
			return false, err.Error() + alarms
		} else if alarms != "" {
			switch syntax.GetEnforcementLevel() {
			case syntax.EnforceError:
				return false, alarms
			case syntax.EnforceAlarm:
				return true, alarms
			case syntax.EnforceLog:
				util.PrintInfo("runtime",
					"(outputs)         %s: WARNING: invalid output\n%s",
					self.fqname, alarms)
				fallthrough
			default:
				return true, ""
			}
		}
	}
	return true, ""
}

func (self *Fork) verifyPipelineOutput(outs json.Marshaler, t syntax.Type) (bool, string) {
	switch t := t.(type) {
	case *syntax.TypedMapType:
		if outs, ok := outs.(MarshalerMap); ok {
			for _, v := range outs {
				if ok, msg := self.verifyPipelineOutput(v, t.Elem); !ok {
					return ok, msg
				}
			}
		}
	case *syntax.ArrayType:
		if outs, ok := outs.(marshallerArray); ok {
			for _, v := range outs {
				if ok, msg := self.verifyPipelineOutput(v, t.Elem); !ok {
					return ok, msg
				}
			}
		}
	case *syntax.StructType:
		if outs, ok := outs.(MarshalerMap); ok {
			if len(t.Members) > 0 {
				if err, alarms := outs.ValidatePipelineOutputs(
					self.node.top.types, t); err != nil {
					return false, err.Error() + alarms
				} else if alarms != "" {
					switch syntax.GetEnforcementLevel() {
					case syntax.EnforceError:
						return false, alarms
					case syntax.EnforceAlarm:
						return true, alarms
					case syntax.EnforceLog:
						util.PrintInfo("runtime",
							"(outputs)         %s: WARNING: invalid output\n%s",
							self.fqname, alarms)
						fallthrough
					default:
						return true, ""
					}
				}
			}
		}
	default:
		util.PrintInfo("runtime", "Invalid output type %T for pipeline", t)
	}
	return true, ""
}

func (self *Fork) getState() MetadataState {
	if state, _ := self.metadata.getState(); state == Failed ||
		state == Complete ||
		state == DisabledState {
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
		// If any chunks have failed, the state is failed.
		// If all have completed, the state is chunks_complete.
		// If all are queued, running, or complete, the state is chunks_running.
		complete := true
		running := true
		for _, chunk := range self.chunks {
			switch chunk.getState() {
			case Failed:
				return Failed
			case Complete:
			case Queued, Running:
				complete = false
			default:
				complete = false
				running = false
			}
		}
		if complete {
			return Complete.Prefixed(ChunksPrefix)
		}
		if running {
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

func (self *Fork) disabled() (bool, error) {
	top := self.node.top
	var errs syntax.ErrorList
	for _, p := range self.forkId {
		if r := p.Range; r != nil && r.Length() == 0 {
			return true, nil
		}
	}
	for _, bind := range self.node.call.Disabled() {
		if ready, res, err := top.resolve(bind,
			top.types.Get(syntax.TypeId{
				Tname: syntax.KindBool,
			}), self.forkId, top.rt.FreeMemBytes()/2); err != nil {
			errs = append(errs, err)
		} else if ready {
			if res != nil {
				switch d := res.(type) {
				case *syntax.BoolExp:
					if d.Value {
						return true, nil
					}
				case *syntax.NullExp:
					errs = append(errs, fmt.Errorf(
						"disabled is bound to a null value, which the compiler should not allow"))
				case json.RawMessage:
					var v bool
					if err := json.Unmarshal(d, &v); err == nil && v {
						return true, nil
					} else if err != nil {
						if bytes.Equal(d, nullBytes) {
							errs = append(errs, fmt.Errorf(
								"disabled is bound to a null value, which is not permitted"))
						} else {
							errs = append(errs, err)
						}
					}
				}
			} else {
				errs = append(errs, fmt.Errorf(
					"disabled is bound to a null value, which the compiler should not allow"))
			}
		}
	}
	return false, errs.If()
}

func (self *Fork) writeDisable() {
	if err := util.MkdirAll(self.path); err != nil {
		util.LogError(err, "runtime",
			"Could not create directories for %s", self.fqname)
	}
	self.metadata.Write(OutsFile, makeOutArgs(
		self.OutParams(), self.metadata.curFilesPath, true))
	self.skip()
	self.printState(DisabledState)
}

func (self *Fork) updateState(state, uniquifier string) {
	if state == string(ProgressFile) {
		self.lastPrint = time.Now()
		if msg, err := self.metadata.readRawSafe(MetadataFileName(state)); err == nil {
			util.PrintInfo("runtime",
				"(progress)        %s: %s",
				self.fqname, msg)
		} else {
			util.LogError(err, "progres",
				"Error reading progress file for %s",
				self.fqname)
		}
	}
	if strings.HasPrefix(state, SplitPrefix) {
		self.split_metadata.cache(
			MetadataFileName(strings.TrimPrefix(state, SplitPrefix)),
			uniquifier)
		if st, _ := self.split_metadata.getState(); st != Running && st != Queued {
			self.node.top.rt.JobManager.endJob(self.split_metadata)
		}
	} else if strings.HasPrefix(state, JoinPrefix) {
		self.join_metadata.cache(
			MetadataFileName(strings.TrimPrefix(state, JoinPrefix)),
			uniquifier)
		if st, _ := self.join_metadata.getState(); st != Running && st != Queued {
			self.node.top.rt.JobManager.endJob(self.join_metadata)
		}
	} else {
		self.metadata.cache(MetadataFileName(state), uniquifier)
	}
}

func (self *Fork) getChunk(index int) *Chunk {
	if index < len(self.chunks) {
		return self.chunks[index]
	}
	return nil
}

func (self *Fork) skip() {
	self.metadata.WriteTime(DisabledFile)
}

func (self *Fork) writeInvocation() {
	if !self.metadata.exists(InvocationFile) {
		splitArgs, argBindings, err := self.node.resolveInputs(self.forkId, true)
		if err != nil {
			switch syntax.GetEnforcementLevel() {
			case syntax.EnforceError:
				self.metadata.WriteErrorString(err.Error())
			case syntax.EnforceAlarm:
				aerr := self.metadata.AppendAlarm(err.Error())
				if aerr != nil {
					util.PrintError(aerr, "runtime",
						"(inputs )          %s: Error writing alarms",
						self.fqname)
					util.PrintError(err, "runtime",
						"(inputs )          %s: WARNING: invalid args",
						self.fqname)
				}
			case syntax.EnforceLog:
				util.PrintError(err, "runtime",
					"(inputs )          %s: WARNING: invalid args",
					self.fqname)
			}
		}
		invocation, _ := BuildCallSource(
			self.node.call.Call().Id,
			argBindings, splitArgs,
			self.node.call.Callable(),
			self.node.top.types,
			self.node.top.mroPaths)
		if err := self.metadata.WriteRaw(InvocationFile, invocation); err != nil {
			util.LogError(err, "runtime",
				"%s: Error writing invocation file.",
				self.fqname)
		}
	}
}

func (self *Fork) printState(state MetadataState) {
	statePad := strings.Repeat(" ", int(math.Max(0, float64(15-len(state)))))
	// Only log fork num if we've got more than one fork
	fqname := self.node.GetFQName()
	if len(self.node.forks) > 1 {
		fqname = self.fqname
	}
	self.lastPrint = time.Now()
	if self.node.call.Call().Modifiers.Preflight || state == DisabledState {
		util.LogInfo("runtime", "(%s)%s %s", state, statePad, fqname)
	} else {
		util.PrintInfo("runtime", "(%s)%s %s", state, statePad, fqname)
	}
}

func (self *Fork) doSplit(getBindings func() MarshalerMap) MetadataState {
	if disabled, err := self.disabled(); disabled {
		self.writeDisable()
		return DisabledState
	} else if err != nil {
		self.metadata.writeError("Could not evaluate disabled state", err)
		return Failed
	}
	self.writeInvocation()
	if err := self.split_metadata.Write(ArgsFile, getBindings()); err != nil {
		util.LogError(err, "runtime",
			"%s: Error writing args file.",
			self.fqname)
	}
	if self.Split() {
		if !self.split_has_run {
			self.split_has_run = true
			self.lastPrint = time.Now()
			self.node.runSplit(self.fqname, self.split_metadata)
		}
	} else {
		_ = self.split_metadata.Write(StageDefsFile, self.stageDefs)
		if err := self.split_metadata.WriteTime(CompleteFile); err != nil {
			util.LogError(err, "runtime",
				"%s: Error writing split completion stub file.",
				self.fqname)
		}
		return Complete.Prefixed(SplitPrefix)
	}
	return Ready
}

func (self *Fork) doChunks(state MetadataState, getBindings func() MarshalerMap) MetadataState {
	self.node.top.rt.JobManager.endJob(self.split_metadata)
	if self.isVolatile() {
		lockAquired := make(chan struct{}, 1)
		go func() {
			self.storageLock.Lock()
			defer self.storageLock.Unlock()
			lockAquired <- struct{}{}
			self.cleanSplitTemp(nil)
		}()
		<-lockAquired
	}
	// MARTIAN-395 We have observed a possible race condition where
	// split_complete could be detected but _stage_defs is not
	// written yet or is corrupted. Check that stage_defs exists
	// before attempting to read and unmarshal it.
	if !self.split_metadata.exists(StageDefsFile) {
		// We might have missed the journal update.  Check if
		// the file exists in the directory.
		self.split_metadata.poll()
	}
	if self.split_metadata.exists(StageDefsFile) {
		if err := self.split_metadata.ReadInto(StageDefsFile, &self.stageDefs); err != nil {
			errstring := err.Error()
			self.split_metadata.WriteErrorString(fmt.Sprintf(
				`The split method did not return a dictionary {"chunks": [{}], "join": {}}.
Error: %s
Chunk count: %d`,
				errstring, len(self.stageDefs.ChunkDefs)))
		} else if len(self.stageDefs.ChunkDefs) == 0 {
			// Skip the chunk phase.
			state = Complete.Prefixed(ChunksPrefix)
		} else {
			if len(self.chunks) == 0 {
				self.chunks = make([]*Chunk, 0, len(self.stageDefs.ChunkDefs))
				width := util.WidthForInt(len(self.stageDefs.ChunkDefs))
				for i, chunkDef := range self.stageDefs.ChunkDefs {
					chunk := NewChunk(self, i, chunkDef, width)
					self.chunks = append(self.chunks, chunk)
					if err := chunk.mkdirs(); err != nil {
						util.LogError(err, "runtime",
							"%s: Error making chunk directory.",
							self.fqname)
					}
					chunk.verifyDef()
				}
				self.metadatasCache = nil
			}
			if len(self.chunks) > 0 {
				bindings := getBindings()
				for _, chunk := range self.chunks {
					chunk.step(bindings)
				}
			}
		}
	} else {
		// If the stage "succeeded" without writing _stage_defs,
		// don't wait forever for the file to show up. Normal
		// heartbeat checks only happen when "running" but this
		// replicates much of the logic of metadata.checkHeartbeat()
		if self.split_metadata.lastHeartbeat.IsZero() ||
			self.split_metadata.exists(Heartbeat) {
			self.split_metadata.uncache(Heartbeat)
			self.split_metadata.lastHeartbeat = time.Now()
		}
		if time.Since(self.split_metadata.lastHeartbeat) >
			time.Minute*heartbeatTimeout {
			// Pretend we do see it, so it will try to read next time
			// around.  If it succeeds, that means we missed a journal
			// update.  If it doesn't, the split will be errored out.
			self.split_metadata.cache(StageDefsFile, self.split_metadata.uniquifier)
		}
	}
	return state
}

func (self *Fork) doJoin(state MetadataState, getBindings func() MarshalerMap) MetadataState {
	go self.partialVdrKill()
	if self.stageDefs.JoinDef == nil {
		self.stageDefs.JoinDef = &JobResources{}
	}
	res := self.node.setJoinJobReqs(self.stageDefs.JoinDef)
	args, err := getBindings().ToLazyArgumentMap()
	if err != nil {
		panic(err)
	}
	resolvedBindings := ChunkDef{
		Resources: self.stageDefs.JoinDef,
		Args:      args,
	}
	if err := self.join_metadata.Write(ArgsFile, &resolvedBindings); err != nil {
		util.LogError(err, "runtime", "%s: Error writing join args file.",
			self.fqname)
	}
	chunkDefs := make([]ChunkDef, len(self.stageDefs.ChunkDefs))
	for i := range chunkDefs {
		// Strip the resources from the chunk defs given to the join.  This is
		// because the resources will not be accurite if the pipestance was
		// restarted, and it isn't worth trying to recompute just in case.
		chunkDefs[i].Args = self.stageDefs.ChunkDefs[i].Args
	}
	if err := self.join_metadata.Write(ChunkDefsFile, chunkDefs); err != nil {
		util.LogError(err, "runtime", "%s: Error writing chunk defs file.",
			self.fqname)
	}
	if self.Split() {
		ok := true
		if len(self.chunks) > 0 {
			if co := self.ChunkOutParams(); len(self.OutParams().List) > 0 ||
				(co != nil && len(co.List) > 0) {
				chunkOuts := make([]LazyArgumentMap, 0, len(self.chunks))
				readSize := self.node.top.rt.FreeMemBytes() / int64(2*len(self.chunks))
				for _, chunk := range self.chunks {
					if outs, err := chunk.metadata.read(OutsFile, readSize); err != nil {
						chunk.metadata.WriteErrorString(err.Error())
						ok = false
					} else {
						chunkOuts = append(chunkOuts, outs)
						ok = chunk.verifyOutput(outs) && ok
					}
				}
				if err := self.join_metadata.Write(ChunkOutsFile, chunkOuts); err != nil {
					util.LogError(err, "runtime",
						"%s: Error writing chunk outs file.",
						self.fqname)
				}
			} else {
				// Write a list of empty outs.
				var buf bytes.Buffer
				buf.Grow(1 + 3*len(self.chunks))
				buf.WriteByte('[')
				for i := range self.chunks {
					if i != 0 {
						buf.WriteByte(',')
					}
					buf.WriteString("{}")
				}
				buf.WriteByte(']')
				if err := self.join_metadata.WriteRawBytes(ChunkOutsFile,
					buf.Bytes()); err != nil {
					util.LogError(err, "runtime",
						"%s: Error writing chunk outs file.",
						self.fqname)
				}
			}
			if !ok {
				return Failed
			}
		} else {
			if err := self.join_metadata.WriteRaw(ChunkOutsFile, "[]"); err != nil {
				util.LogError(err, "runtime",
					"%s: Error writing chunk outs file.",
					self.fqname)
			}
		}
		self.join_metadata.Write(
			OutsFile,
			makeOutArgs(self.OutParams(),
				self.join_metadata.curFilesPath, false))
		if !self.join_has_run {
			self.join_has_run = true
			self.lastPrint = time.Now()
			self.node.runJoin(self.fqname, self.join_metadata, &res)
		}
	} else {
		if b, err := self.chunks[0].metadata.readRawBytes(OutsFile); err == nil {
			self.join_metadata.WriteRawBytes(OutsFile, b)
		} else {
			util.LogError(err, "runtime", "Could not read stage outs file.")
		}
		self.join_metadata.WriteTime(CompleteFile)
		state = Complete.Prefixed(JoinPrefix)
	}
	return state
}

func (self *Fork) doComplete() {
	self.node.top.rt.JobManager.endJob(self.join_metadata)
	var joinOut LazyArgumentMap
	if len(self.OutParams().List) > 0 {
		var err error
		joinOut, err = self.join_metadata.read(OutsFile, self.node.top.rt.FreeMemBytes()/3)
		if err != nil {
			self.join_metadata.WriteErrorString(err.Error())
			return
		} else if joinOut == nil {
			self.metadata.WriteRaw(OutsFile, "{}")
		} else {
			self.metadata.Write(OutsFile, joinOut)
		}
	} else {
		self.metadata.WriteRaw(OutsFile, "{}")
	}
	if self.node.top.rt.Config.VdrMode == VdrPost {
		// Still clean up tmp, but run before we've declared
		// the stage maybe complete.
		func() {
			self.storageLock.Lock()
			defer self.storageLock.Unlock()
			self.cacheParamFileMap(joinOut)
		}()
		self.partialVdrKill()
	}
	if ok, msg := self.verifyOutput(joinOut); ok {
		if msg != "" {
			err := self.metadata.AppendAlarm("Incorrect _outs: " + msg)
			if err != nil {
				util.LogError(err, "runtime", "Error writing alarm")
			}
		}
		self.metadata.WriteTime(CompleteFile)
		// Print alerts
		var alarms strings.Builder
		self.getAlarms(&alarms)
		if alarms.Len() > 0 {
			self.lastPrint = time.Now()
			if len(self.node.forks) > 1 {
				util.Print("Alerts for %s:\n%s\n",
					self.fqname, alarms.String())
			} else {
				util.Print("Alerts for %s:\n%s\n",
					self.node.GetFQName(), alarms.String())
			}
		}
	} else {
		self.metadata.WriteErrorString(msg)
	}
	self.removeEmptyFileArgs(joinOut)
	if self.node.top.rt.Config.VdrMode != VdrPost {
		go func() {
			func() {
				self.storageLock.Lock()
				defer self.storageLock.Unlock()
				self.cacheParamFileMap(joinOut)
			}()
			self.partialVdrKill()
		}()
	}
}

func (self *Fork) stepPipeline() {
	if disabled, err := self.disabled(); disabled {
		self.writeDisable()
		return
	} else if err != nil {
		// Pipelines only sort-of fork.  Their final outputs may already have
		// been resolved statically, in which case it may not make sense to
		// talk about the entire pipeline being disabled.
		var forkErr *forkResolutionError
		if !errors.As(err, &forkErr) {
			self.metadata.writeError(
				"Could not evaluate pipeline disabled state",
				err)
			return
		}
	}
	self.writeInvocation()
	if outs, t, err := self.node.resolvePipelineOutputs(self.forkId); err != nil {
		util.PrintError(err, "runtime",
			"(%s) Error resolving output argument bindings.",
			self.fqname)
		self.metadata.WriteErrorString(err.Error())
	} else {
		self.metadata.Write(OutsFile, outs)
		if len(self.OutParams().List) > 0 {
			if ok, msg := self.verifyPipelineOutput(outs, t); ok {
				if msg != "" {
					err := self.metadata.AppendAlarm("Incorrect _outs: " + msg)
					if err != nil {
						util.LogError(err, "runtime", "Error writing alarm")
					}
				}
				self.metadata.WriteTime(CompleteFile)
			} else {
				self.metadata.WriteErrorString(msg)
			}
		} else {
			self.metadata.WriteTime(CompleteFile)
		}
	}
}

func (self *Fork) step() {
	if self.node.call.Kind() == syntax.KindStage {
		self.stepStage()
	} else if self.node.call.Kind() == syntax.KindPipeline {
		self.stepPipeline()
	}
}

func (self *Fork) stepStage() {
	state := self.getState()
	if !state.IsRunning() && !state.IsQueued() && state != DisabledState {
		self.printState(state)
	}

	// Lazy-evaluate bindings, only once per step.
	var bindings MarshalerMap
	getBindings := func() MarshalerMap {
		if bindings == nil {
			var err error
			_, bindings, err = self.node.resolveInputs(self.forkId, false)
			if err != nil {
				util.PrintError(err, "runtime",
					"Error resolving input argument bindings for %s",
					self.fqname)
				switch syntax.GetEnforcementLevel() {
				case syntax.EnforceError:
					if err := util.MkdirAll(self.path); err != nil {
						util.LogError(err, "runtime",
							"Could not create directories for %s", self.fqname)
					}
					if self.index != 0 {
						self.metadata.writeError(
							"Error resolving input argument bindings for "+
								self.forkId.GoString(),
							err)
					} else {
						self.metadata.writeError(
							"Error resolving input argument bindings",
							err)
					}
					state = Failed
				case syntax.EnforceAlarm:
					if err := util.MkdirAll(self.path); err != nil {
						util.LogError(err, "runtime",
							"Could not create directories for %s", self.fqname)
					}
					err := self.metadata.AppendAlarm(fmt.Sprintln(
						"Error resolving input argument bindings:",
						err))
					if err != nil {
						util.LogError(err, "runtime",
							"(inputs )         %s: Could not write alarms.",
							self.fqname)
					}
				}
			} else if bindings == nil {
				util.LogInfo("runtime",
					"Failed to resolve bindings for %s",
					self.fqname)
			}
		}
		return bindings
	}
	if state == DisabledState {
		return
	}
	if state == Ready {
		state = self.doSplit(getBindings)
		if state == DisabledState {
			return
		}
	}
	if state == Complete.Prefixed(SplitPrefix) {
		state = self.doChunks(state, getBindings)
	}
	if state == Complete.Prefixed(ChunksPrefix) {
		state = self.doJoin(state, getBindings)
	}
	if state == Complete.Prefixed(JoinPrefix) {
		self.doComplete()
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

func (self *Fork) cachePerf(ctx context.Context) {
	perfInfo, vdrKillReport := self.serializePerf(ctx)
	self.perfCache = &ForkPerfCache{perfInfo, vdrKillReport}
}

func (self *Fork) getVdrKillReport() (*VDRKillReport, bool) {
	var killReport VDRKillReport
	ok := false
	if self.metadata.exists(VdrKill) {
		ok = (self.metadata.ReadInto(VdrKill, &killReport) == nil)
	}
	return &killReport, ok
}

func (self *Fork) getPartialKillReport() *PartialVdrKillReport {
	if self.metadata.exists(PartialVdr) {
		var killReport PartialVdrKillReport
		if self.metadata.ReadInto(PartialVdr, &killReport) == nil {
			return &killReport
		}
	}
	return nil
}

func (self *Fork) deletePartialKill() {
	if self.metadata.exists(PartialVdr) {
		self.metadata.remove(PartialVdr)
	}
}

func (self *Fork) writePartialKill(killReport *PartialVdrKillReport) {
	self.metadata.Write(PartialVdr, killReport)
}

func (self *Fork) printAlarms() {
	// Print alerts
	var alarms strings.Builder
	self.getAlarms(&alarms)
	if alarms.Len() > 0 {
		self.lastPrint = time.Now()
		if len(self.node.forks) > 1 {
			util.Print("Alerts (%s):\n", self.id)
		} else {
			util.Print("Alerts:\n")
		}
		util.Print("%s\n", alarms.String())
	}
}

func (self *Fork) getAlarms(alarms *strings.Builder) {
	for _, metadata := range self.collectMetadatas() {
		if !metadata.exists(AlarmFile) {
			continue
		}
		if b, err := metadata.readRawBytes(AlarmFile); err == nil {
			alarms.Write(b)
		}
	}
	for _, subnode := range self.node.subnodes {
		for _, subfork := range subnode.matchForks(self.forkId) {
			subfork.getAlarms(alarms)
		}
	}
}

func (self *Fork) serializeState(ctx context.Context) *ForkInfo {
	defer trace.StartRegion(ctx, "Fork_serializeState").End()
	argbindings := self.node.inputBindingInfo(self.forkId)
	outputs := self.node.outputBindingInfo(self.forkId)
	bindings := &ForkBindingsInfo{
		Argument: argbindings,
		Return:   outputs,
	}
	chunks := make([]*ChunkInfo, 0, len(self.chunks))
	for _, chunk := range self.chunks {
		if ctx.Err() != nil {
			return nil
		}
		chunks = append(chunks, chunk.serializeState())
	}
	return &ForkInfo{
		Index:         self.index,
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
	stages := make([]*StagePerfInfo, 0, len(self.node.subnodes)+1)
	for _, node := range self.node.subnodes {
		for _, subfork := range node.matchForks(self.forkId) {
			stages = append(stages, subfork.getStages()...)
		}
	}
	if self.node.call.Kind() == syntax.KindStage {
		stages = append(stages, &StagePerfInfo{
			Name:   self.node.call.Call().Id,
			Fqname: self.node.GetFQName(),
			Forki:  self.index,
		})
	}
	return stages
}

func (self *Fork) serializePerf(ctx context.Context) (*ForkPerfInfo, *VDRKillReport) {
	defer trace.StartRegion(ctx, "Fork_serializePerf").End()
	if self.perfCache != nil {
		// Use cached performance information if it exists.
		return self.perfCache.perfInfo, self.perfCache.vdrKillReport
	}

	chunks := make([]*ChunkPerfInfo, 0, len(self.chunks))
	stats := make([]*PerfInfo, 0, len(self.chunks)+len(self.node.subnodes)+2)
	for _, chunk := range self.chunks {
		if ctx.Err() != nil {
			return nil, nil
		}
		chunkSer := chunk.serializePerf()
		chunks = append(chunks, chunkSer)
		if chunkSer.ChunkStats != nil {
			// avoid double-counting of bytes/files if there is no
			// actual split; it will be counted by ComputeStats() below.
			if !self.Split() {
				chunkSer.ChunkStats.OutputBytes = 0
				chunkSer.ChunkStats.OutputFiles = 0
			}
			stats = append(stats, chunkSer.ChunkStats)
		}
	}

	numThreads := self.node.getJobReqs(nil, STAGE_TYPE_SPLIT).Threads
	splitStats := self.split_metadata.serializePerf(numThreads)
	if splitStats != nil {
		stats = append(stats, splitStats)
	}

	numThreads = self.node.getJobReqs(self.stageDefs.JoinDef, STAGE_TYPE_JOIN).Threads
	joinStats := self.join_metadata.serializePerf(numThreads)
	if joinStats != nil {
		stats = append(stats, joinStats)
	}

	killReports := make([]*VDRKillReport, 1, len(self.node.subnodes)+1)
	killReports[0], _ = self.getVdrKillReport()
	for _, node := range self.node.subnodes {
		if ctx.Err() != nil {
			return nil, nil
		}
		for _, subfork := range node.matchForks(self.forkId) {
			if ctx.Err() != nil {
				return nil, nil
			}
			subforkSer, subforkKillReport := subfork.serializePerf(ctx)
			if ctx.Err() != nil {
				// If the context has expired, subforkSer will be nil.
				return nil, nil
			}
			stats = append(stats, subforkSer.ForkStats)
			if subforkKillReport != nil {
				killReports = append(killReports, subforkKillReport)
			}
		}
	}
	if ctx.Err() != nil {
		return nil, nil
	}
	killReport := mergeVDRKillReports(killReports)
	fpaths, _ := self.metadata.enumerateFiles()

	var forkStats *PerfInfo
	if len(stats) > 0 {
		forkStats = ComputeStats(stats, fpaths, killReport)
	} else {
		forkStats = new(PerfInfo)
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

// Marks a possible file out argument as not actually containing any files.
// For example, a map output which does not actually contain any strings.
// This may result in the removal of some file post-nodes, which may allow for
// earlier VDR.
func (self *Fork) removeFileArg(arg string) {
	if nodes, ok := self.fileArgs[arg]; !ok {
		return
	} else {
		delete(self.fileArgs, arg)
		for node := range nodes {
			if args, ok := self.filePostNodes[node]; ok {
				delete(args, arg)
				if len(args) == 0 {
					delete(self.filePostNodes, node)
				}
			}
		}
	}
}

// Removes file post-nodes which are complete from the set of nodes on which
// VDR for this stage is still waiting, and also removes arguments from
// fileArgs for which no post-node is still waiting.  This is how it is
// determined when VDR is safe to run.
func (self *Fork) removeFilePostNodes(nodes []Nodable) {
	for _, node := range nodes {
		if args, ok := self.filePostNodes[node]; ok {
			delete(self.filePostNodes, node)
			for arg := range args {
				if remaining, ok := self.fileArgs[arg]; ok {
					delete(remaining, node)
					if len(remaining) == 0 {
						delete(self.fileArgs, arg)
					}
				}
			}
		}
	}
}

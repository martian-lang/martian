//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian runtime pipestance management.
//

package core

import (
	"fmt"
	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

//=============================================================================
// Stagestance
//=============================================================================

// Similar to a pipestance, except for a single stage.  Intended for use
// during testing and development of pipelines, e.g. with `mrs`.
type Stagestance struct {
	node *Node
}

func NewStagestance(parent Nodable, callStm *syntax.CallStm, callables *syntax.Callables) (*Stagestance, error) {
	self := &Stagestance{}
	self.node = NewNode(parent, "stage", callStm, callables)
	stage, ok := callables.Table[callStm.DecId].(*syntax.Stage)
	if !ok {
		return nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared stage", callStm.DecId)}
	}

	stagecodePaths := append(self.node.mroPaths, strings.Split(os.Getenv("PATH"), ":")...)
	stagecodePath, _ := util.SearchPaths(stage.Src.Path, stagecodePaths)
	self.node.stagecodeCmd = strings.Join(append([]string{stagecodePath}, stage.Src.Args...), " ")
	var err error
	if self.node.stagecodeLang, err = stage.Src.Lang.Parse(); err != nil {
		return self, fmt.Errorf("Unsupported language in stage %s: %v", callStm.DecId, stage.Src.Lang)
	}
	if self.node.rt.Config.StressTest {
		switch self.node.stagecodeLang {
		case syntax.PythonStage:
			self.node.stagecodeCmd = util.RelPath(path.Join("..", "adapters", "python", "tester"))
		default:
			return self, fmt.Errorf("Unsupported stress test language: %v", stage.Src.Lang)
		}
	}
	if stage.Resources != nil {
		self.node.resources = &JobResources{
			Threads: stage.Resources.Threads,
			MemGB:   stage.Resources.MemGB,
			Special: stage.Resources.Special,
		}
	}
	self.node.buildForks(self.node.argbindingList)
	return self, nil
}

func (self *Stagestance) getNode() *Node    { return self.node }
func (self *Stagestance) GetFQName() string { return self.node.fqname }

func (self *Stagestance) GetPrenodes() map[string]Nodable {
	return self.node.GetPrenodes()
}

func (self *Stagestance) GetPostNodes() map[string]Nodable {
	return self.node.GetPostNodes()
}

func (self *Stagestance) Callable() syntax.Callable {
	return self.node.Callable()
}

func (self *Stagestance) GetState() MetadataState { return self.getNode().getState() }

func (self *Stagestance) Step() bool {
	if err := self.node.rt.JobManager.refreshResources(
		self.node.rt.Config.JobMode == "local"); err != nil {
		util.LogError(err, "runtime",
			"Error refreshing resources: %s", err.Error())
	}
	return self.getNode().step()
}

func (self *Stagestance) CheckHeartbeats() { self.getNode().checkHeartbeats() }
func (self *Stagestance) RefreshState()    { self.getNode().refreshState(false) }
func (self *Stagestance) LoadMetadata()    { self.getNode().loadMetadata() }
func (self *Stagestance) PostProcess()     { self.getNode().postProcess() }
func (self *Stagestance) GetFatalError() (string, bool, string, string, MetadataFileName, []string) {
	return self.getNode().getFatalError()
}

//=============================================================================
// Pipestance
//=============================================================================

// Encapsulates information about an instance of a running (or failed, or
// completed) pipeline.
type Pipestance struct {
	node     *Node
	metadata *Metadata
	uuid     string

	// Cache for self.node.allNodes()
	allNodesCache    []*Node
	queueCheckLock   sync.Mutex
	queueCheckActive bool
	lastQueueCheck   time.Time
}

/* Run a script whenever a pipestance finishes */
func (self *Pipestance) OnFinishHook() {
	exec_path := self.getNode().rt.Config.OnFinishHandler
	if exec_path != "" {
		util.Println("\nRunning onfinish handler...")

		// Build command line arguments:
		// $1 = path to piestance
		// $2 = {complete|failed}
		// $3 = pipestance ID
		// $4 = path to error file (if there was an error)
		args := []string{exec_path, self.GetPath(), string(self.GetState()), self.getNode().name}
		if self.GetState() == Failed {
			_, _, _, _, _, err_paths := self.GetFatalError()
			if len(err_paths) > 0 {
				err_path, _ := filepath.Rel(filepath.Dir(self.GetPath()), err_paths[0])
				args = append(args, err_path)
			}
		}

		/* Set up attributes for exec */
		var pa os.ProcAttr
		pa.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}

		/* Find the real path to the script */
		real_path, err := exec.LookPath(exec_path)
		if err != nil {
			util.LogInfo("finishr", "Could not find %v: %v", exec_path, err)
			return
		}

		/* Run it */
		p, err := os.StartProcess(real_path, args, &pa)
		if err != nil {
			util.LogInfo("finishr", "Could not run %v: %v", real_path, err)
			return
		}

		/* Wait for it to finish */
		res, err := p.Wait()
		if err != nil {
			util.LogInfo("finishr", "Error running %v: %v", real_path, err)
		}
		if !res.Success() {
			util.LogInfo("finishr", "%v exited with non-zero status.", real_path)
		}
	}
}

func NewPipestance(parent Nodable, callStm *syntax.CallStm, callables *syntax.Callables) (*Pipestance, error) {
	self := &Pipestance{}
	self.node = NewNode(parent, "pipeline", callStm, callables)
	self.metadata = NewMetadata(self.node.parent.getNode().fqname, self.GetPath())

	// Build subcall tree.
	pipeline, ok := callables.Table[callStm.DecId].(*syntax.Pipeline)
	if !ok {
		return nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared pipeline", callStm.DecId)}
	}
	preflightNodes := []Nodable{}
	for _, subcallStm := range pipeline.Calls {
		callable := callables.Table[subcallStm.DecId]
		switch callable.(type) {
		case *syntax.Stage:
			if s, err := NewStagestance(self.node, subcallStm, callables); err != nil {
				return nil, err
			} else {
				self.node.subnodes[subcallStm.Id] = s
			}
		case *syntax.Pipeline:
			if p, err := NewPipestance(self.node, subcallStm, callables); err != nil {
				return nil, err
			} else {
				self.node.subnodes[subcallStm.Id] = p
			}
		default:
			return nil, fmt.Errorf("Unsupported callable type %v", callable)
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
	}
	self.node.attachBindings(self.node.retbindingList)

	// Add preflight dependencies if preflight stages exist.
	for _, preflightNode := range preflightNodes {
		for _, subnode := range self.node.subnodes {
			if !subnode.getNode().preflight {
				subnode.getNode().setPrenode(preflightNode)
			}
		}
	}

	self.node.buildForks(self.node.retbindingList)
	return self, nil
}

func (self *Pipestance) getNode() *Node    { return self.node }
func (self *Pipestance) GetPname() string  { return self.node.name }
func (self *Pipestance) GetPsid() string   { return self.node.parent.getNode().name }
func (self *Pipestance) GetFQName() string { return self.node.fqname }
func (self *Pipestance) RefreshState()     { self.node.refreshState(self.readOnly()) }
func (self *Pipestance) readOnly() bool    { return !self.metadata.exists(Lock) }

func (self *Pipestance) GetPrenodes() map[string]Nodable {
	return self.node.GetPrenodes()
}

func (self *Pipestance) GetPostNodes() map[string]Nodable {
	return self.node.GetPostNodes()
}

func (self *Pipestance) Callable() syntax.Callable {
	return self.node.Callable()
}

func (self *Pipestance) allNodes() []*Node {
	if self.allNodesCache == nil {
		self.allNodesCache = self.node.allNodes()
	}
	return self.allNodesCache
}

func (self *Pipestance) LoadMetadata() {
	// We used to make this concurrent but ended up with too many
	// goroutines (Pranav's 96-sample run).
	for _, node := range self.allNodes() {
		node.loadMetadata()
	}
	for _, node := range self.allNodes() {
		node.state = node.getState()
		if node.state == Running && !self.readOnly() {
			node.mkdirs()
		}
	}
}

func (self *Pipestance) GetState() MetadataState {
	nodes := self.node.getFrontierNodes()
	for _, node := range nodes {
		if node.state == Failed {
			return Failed
		}
	}
	for _, node := range nodes {
		if node.state == Running {
			return Running
		}
	}
	every := true
	for _, node := range self.allNodes() {
		if node.state != DisabledState {
			every = false
			break
		}
	}
	if every {
		return DisabledState
	}
	every = true
	for _, node := range self.allNodes() {
		if node.state != Complete && node.state != DisabledState {
			every = false
			break
		}
	}
	if every {
		return Complete
	}
	return ForkWaiting
}

func (self *Pipestance) Kill() {
	self.KillWithMessage("Job was killed by Martian.")
}

func (self *Pipestance) KillWithMessage(message string) {
	if self.readOnly() {
		return
	}
	nodes := self.node.getFrontierNodes()
	for _, node := range nodes {
		node.kill(message)
	}
}

func (self *Pipestance) RestartRunningNodes(jobMode string) error {
	if self.readOnly() {
		return &RuntimeError{"Pipestance is in read only mode."}
	}
	self.LoadMetadata()
	nodes := self.node.getFrontierNodes()
	localNodes := []*Node{}
	for _, node := range nodes {
		if node.state == Running {
			util.PrintInfo("runtime", "Found orphaned stage: %s", node.fqname)
			if jobMode == "local" || node.local {
				localNodes = append(localNodes, node)
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

// Resets local nodes which are queued or are running with a PID that is not
// a running job.  If |jobMode| is "local" then all nodes are treated as local.
// This is nessessary for when e.g. mrp is restarted in local mode after ctrl-C
// kills it and all of its child processes.
func (self *Pipestance) RestartLocalJobs(jobMode string) error {
	if self.readOnly() {
		return &RuntimeError{"Pipestance is in read only mode."}
	}
	for _, node := range self.node.getFrontierNodes() {
		if node.state == Running {
			if err := node.restartLocallyQueuedJobs(); err != nil {
				return err
			}
		}
		if node.state == Running && (jobMode == "local" || node.local) {
			util.PrintInfo("runtime", "Found orphaned local stage: %s", node.fqname)
			if err := node.restartLocalJobs(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *Pipestance) CheckHeartbeats() {
	if self.readOnly() {
		return
	}
	self.queryQueue()

	nodes := self.node.getFrontierNodes()
	for _, node := range nodes {
		node.checkHeartbeats()
	}
}

// Check that the queued jobs are actually queued.
func (self *Pipestance) queryQueue() {
	if self.node == nil || self.node.rt == nil ||
		self.node.rt.JobManager == nil ||
		!self.node.rt.JobManager.hasQueueCheck() {
		return
	}
	QUEUE_CHECK_LIMIT := 5 * time.Minute
	self.queueCheckLock.Lock()
	if self.queueCheckActive || time.Since(self.lastQueueCheck) < QUEUE_CHECK_LIMIT {
		self.queueCheckLock.Unlock()
		return
	} else {
		self.queueCheckActive = true
		self.queueCheckLock.Unlock()
	}
	// Get the jobids which need to be queried, and the metadatas which need to
	// be poked if they're not in the queue.
	needsQuery := make(map[string]*Metadata)
	{
		metas := make(map[*Metadata]bool) // avoid double-reading any metadatas
		nodes := self.node.getFrontierNodes()
		for _, node := range nodes {
			for _, m := range node.collectMetadatas() {
				if !metas[m] {
					if st, ok := m.getState(); ok &&
						(st == Queued || st == Running) &&
						m.exists(JobId) {
						metas[m] = true
						id := m.readRaw(JobId)
						if id != "" {
							needsQuery[id] = m
						}
					}
				}
			}
		}
	}
	if len(needsQuery) == 0 {
		self.queueCheckLock.Lock()
		self.queueCheckActive = false
		self.queueCheckLock.Unlock()
		return
	}
	jobsIn := make([]string, 0, len(needsQuery))
	for id := range needsQuery {
		jobsIn = append(jobsIn, id)
	}
	go func() {
		queued, raw := self.node.rt.JobManager.checkQueue(jobsIn)
		for _, id := range queued {
			delete(needsQuery, id)
		}
		if len(needsQuery) > 0 && raw != "" {
			util.LogInfo("runtime",
				"Some jobs thought to be queued were unknown to the job manager.  Raw output:\n%s\n",
				raw)
		}
		if !self.readOnly() {
			for id, m := range needsQuery {
				if m != nil {
					m.failNotRunning(id)
				}
			}
		}
		self.queueCheckLock.Lock()
		self.queueCheckActive = false
		self.lastQueueCheck = time.Now()
		self.queueCheckLock.Unlock()
	}()
}

func (self *Pipestance) GetFailedNodes() []*Node {
	failedNodes := []*Node{}

	nodes := self.node.getFrontierNodes()
	for _, node := range nodes {
		if node.state == Failed {
			failedNodes = append(failedNodes, node)
		}
	}
	return failedNodes
}

func (self *Pipestance) GetFatalError() (string, bool, string, string, MetadataFileName, []string) {
	nodes := self.node.getFrontierNodes()
	for _, node := range nodes {
		if node.state == Failed {
			return node.getFatalError()
		}
	}
	return "", false, "", "", "", []string{}
}

// Returns true if there is no error or if the error is one we expect to not
// recur if the pipeline is rerun, and the log message from the first error
// found, if any.
func (self *Pipestance) IsErrorTransient() (bool, string) {
	nodes := self.node.getFrontierNodes()
	firstLog := ""
	for _, node := range nodes {
		if transient, log := node.isErrorTransient(); !transient {
			return false, log
		} else if firstLog == "" {
			firstLog = log
		}
	}
	return true, firstLog
}

// Process state updates for nodes.  Returns true if there was a change in
// state which would make it productive to call StepNodes again immediately.
func (self *Pipestance) StepNodes() bool {
	if self.readOnly() {
		return false
	}
	if err := CheckMinimalSpace(self.node.path); err != nil {
		if _, ok := err.(*DiskSpaceError); ok {
			util.PrintError(err, "runtime",
				"Pipestance directory out of disk space.")
			self.KillWithMessage(err.Error())
			return false
		}
	}
	if err := self.node.rt.LocalJobManager.refreshResources(
		self.node.rt.Config.JobMode == "local"); err != nil {
		util.LogError(err, "runtime",
			"Error refreshing local resources: %s", err.Error())
	}
	if self.node.rt.LocalJobManager != self.node.rt.JobManager {
		if err := self.node.rt.JobManager.refreshResources(false); err != nil {
			util.LogError(err, "runtime",
				"Error refreshing cluster resources: %s", err.Error())
		}
	}
	hadProgress := false
	for _, node := range self.node.getFrontierNodes() {
		hadProgress = node.step() || hadProgress
	}
	for _, node := range self.allNodes() {
		for _, m := range node.collectMetadatas() {
			m.clearReadCache()
		}
	}
	return hadProgress
}

func (self *Pipestance) Reset() error {
	if self.readOnly() {
		return &RuntimeError{"Pipestance is in read only mode."}
	}
	for _, node := range self.allNodes() {
		if node.state == Failed {
			if err := node.reset(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *Pipestance) SerializeState() []*NodeInfo {
	nodes := self.allNodes()
	ser := make([]*NodeInfo, 0, len(nodes))
	for _, node := range nodes {
		ser = append(ser, node.serializeState())
	}
	return ser
}

func (self *Pipestance) SerializePerf() []*NodePerfInfo {
	nodes := self.allNodes()
	ser := make([]*NodePerfInfo, 0, len(nodes))
	for _, node := range nodes {
		ser = append(ser, node.serializePerf())
	}
	util.LogInfo("perform", "Serializing pipestance performance data.")
	if len(ser) > 0 {
		overallPerf := ser[0]
		self.ComputeDiskUsage(overallPerf)
		overallPerf.HighMem = &self.node.rt.LocalJobManager.(*LocalJobManager).highMem
	}
	return ser
}

func (self *Pipestance) Serialize(name MetadataFileName) interface{} {
	switch name {
	case FinalState:
		return self.SerializeState()
	case Perf:
		return self.SerializePerf()
	default:
		panic(fmt.Sprintf("Unsupported serialization type: %v", name))
	}
}

func forkDependentName(fqname string, forkIndex int) string {
	return fmt.Sprintf("%s.fork%d", fqname, forkIndex)
}

func (self *Pipestance) ComputeDiskUsage(nodePerf *NodePerfInfo) *NodePerfInfo {
	forksVisited := make(map[string]*ForkStorageEvent)

	for _, node := range self.allNodes() {
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
			vdrTimestamp, _ := time.Parse(util.TIMEFMT, forkVDR.Timestamp)
			if forkEvent, ok := forksVisited[fork.fqname]; ok {
				forkEvent.VDRTimestamp = vdrTimestamp
			}
		}
	}

	storageEvents := make([]*StorageEvent, 0, len(forksVisited)*2)
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
	if !self.node.rt.Config.Zip {
		return nil
	}

	nodes := self.allNodes()
	metadatas := []*Metadata{}
	for _, node := range nodes {
		metadatas = append(metadatas, node.collectMetadatas()...)
	}
	filePaths := make([]string, 0, 7*len(metadatas))
	removePaths := make([]string, 0, len(metadatas))
	for _, metadata := range metadatas {
		files := metadata.glob()
		filePaths = append(filePaths, files...)
		removePaths = append(removePaths, files...)
		filePaths = append(filePaths, metadata.symlinks()...)
	}

	util.EnterCriticalSection()
	defer util.ExitCriticalSection()

	// Create zip with all metadata.
	if err := util.CreateZip(zipPath, filePaths); err != nil {
		util.LogError(err, "runtime", "Failed to zip metadata")
		return err
	}

	// Remove all metadata files.
	for _, filePath := range removePaths {
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
	self.metadata.loadCache()
	if self.metadata.exists(JobModeFile) {
		jobMode := self.metadata.readRaw(JobModeFile)
		if jobMode != self.node.rt.Config.JobMode {
			return &PipestanceJobModeError{self.GetPsid(), jobMode}
		}
	}
	return nil
}

func (self *Pipestance) GetTimestamp() string {
	data := self.metadata.readRaw(TimestampFile)
	return ParseTimestamp(data)
}

func (self *Pipestance) GetVersions() (string, string, error) {
	data := self.metadata.readRaw(VersionsFile)
	return ParseVersions(data)
}

func (self *Pipestance) PostProcess() {
	self.node.postProcess()
	self.metadata.WriteRaw(TimestampFile, self.metadata.readRaw(TimestampFile)+"\nend: "+util.Timestamp())
	self.Immortalize(false)
}

// Generate the final state file for the pipestance and zip the content up
// for posterity.
//
// Unless force is true, this is only permitted for locked pipestances.
func (self *Pipestance) Immortalize(force bool) error {
	if !force && self.readOnly() {
		return &RuntimeError{"Pipestance is in read only mode."}
	}
	self.metadata.loadCache()
	if !self.metadata.exists(Perf) {
		self.metadata.Write(Perf, self.SerializePerf())
	}
	if !self.metadata.exists(FinalState) {
		self.metadata.Write(FinalState, self.SerializeState())
	}
	if !self.metadata.exists(MetadataZip) {
		zipPath := self.metadata.MetadataFilePath(MetadataZip)
		if err := self.ZipMetadata(zipPath); err != nil {
			util.LogError(err, "runtime", "Failed to create metadata zip file %s: %s",
				zipPath, err.Error())
			os.Remove(zipPath)
			return err
		}
	}
	return nil
}

func (self *Pipestance) RecordUiPort(url string) error {
	return self.metadata.WriteRaw(UiPort, url)
}

func (self *Pipestance) ClearUiPort() error {
	return self.metadata.remove(UiPort)
}

func (self *Pipestance) GetUuid() (string, error) {
	if self.uuid != "" {
		return self.uuid, nil
	} else {
		return self.metadata.readRawSafe(UuidFile)
	}
}

func (self *Pipestance) SetUuid(uuid string) error {
	if err := self.metadata.WriteRaw(UuidFile, uuid); err == nil {
		self.uuid = uuid
		return nil
	} else {
		return err
	}
}

func (self *Pipestance) Lock() error {
	self.metadata.loadCache()
	if self.metadata.exists(Lock) {
		return &PipestanceLockedError{self.node.parent.getNode().name, self.GetPath()}
	}
	util.RegisterSignalHandler(self)
	self.metadata.WriteTime(Lock)
	return nil
}

func (self *Pipestance) unlock() {
	self.metadata.remove(Lock)
}

func (self *Pipestance) Unlock() {
	self.unlock()
	util.UnregisterSignalHandler(self)
}

func (self *Pipestance) HandleSignal(sig os.Signal) {
	self.unlock()
}

// Map of nodes protected by a lock.
type threadSafeNodeMap struct {
	nodes map[string]Nodable
	lock  sync.Mutex
}

func (self *threadSafeNodeMap) Add(key string, value Nodable) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.nodes[key] = value
}

func (self *threadSafeNodeMap) Remove(key string) {
	self.lock.Lock()
	defer self.lock.Unlock()
	delete(self.nodes, key)
}

func (self *threadSafeNodeMap) GetNodes() []*Node {
	self.lock.Lock()
	defer self.lock.Unlock()
	nodes := make([]*Node, 0, len(self.nodes))
	for _, node := range self.nodes {
		nodes = append(nodes, node.getNode())
	}
	return nodes
}

//=============================================================================
// TopNode
//=============================================================================

// The top-level node for a pipestance.
type TopNode struct {
	node *Node
}

func (self *TopNode) getNode() *Node { return self.node }

func (self *TopNode) GetFQName() string {
	return self.node.fqname
}

func (self *TopNode) GetPrenodes() map[string]Nodable {
	return make(map[string]Nodable)
}

func (self *TopNode) GetPostNodes() map[string]Nodable {
	return make(map[string]Nodable)
}

func (self *TopNode) Callable() syntax.Callable {
	return nil
}

func NewTopNode(rt *Runtime, psid string, p string, mroPaths []string, mroVersion string,
	envs map[string]string, j *InvocationData) *TopNode {
	self := &TopNode{}
	self.node = &Node{}
	self.node.frontierNodes = &threadSafeNodeMap{nodes: make(map[string]Nodable)}
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
// Factory
//=============================================================================

// Encapsulates the information needed to instantiate a pipestance, either by
// creating one or reattaching to an existing one.
type PipestanceFactory interface {
	ReattachToPipestance() (*Pipestance, error)
	InvokePipeline() (*Pipestance, error)
}

type runtimePipeFactory struct {
	rt             *Runtime
	invocationSrc  string
	invocationPath string
	psid           string
	mroPaths       []string
	pipestancePath string
	mroVersion     string
	envs           map[string]string
	checkSrc       bool
	readOnly       bool
	tags           []string
}

func NewRuntimePipestanceFactory(rt *Runtime,
	invocationSrc string,
	invocationPath string,
	psid string,
	mroPaths []string,
	pipestancePath string,
	mroVersion string,
	envs map[string]string,
	checkSrc bool,
	readOnly bool,
	tags []string) PipestanceFactory {
	return runtimePipeFactory{rt,
		invocationSrc, invocationPath, psid, mroPaths, pipestancePath, mroVersion,
		envs, checkSrc, readOnly, tags}
}

func (self runtimePipeFactory) ReattachToPipestance() (*Pipestance, error) {
	return self.rt.ReattachToPipestance(self.psid, self.pipestancePath,
		self.invocationSrc, self.mroPaths, self.mroVersion, self.envs,
		self.checkSrc, self.readOnly)
}

func (self runtimePipeFactory) InvokePipeline() (*Pipestance, error) {
	return self.rt.InvokePipeline(self.invocationSrc, self.invocationPath, self.psid,
		self.pipestancePath, self.mroPaths, self.mroVersion, self.envs, self.tags)
}

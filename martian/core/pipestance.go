//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Martian runtime pipestance management.
//

package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime/trace"
	"sync"
	"syscall"
	"time"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

//=============================================================================
// Stagestance
//=============================================================================

// Similar to a pipestance, except for a single stage.  Intended for use
// during testing and development of pipelines, e.g. with `mrs`.
type Stagestance struct {
	node *Node
}

func NewStagestance(parent Nodable, call *syntax.CallGraphStage, srcPaths []string) (*Stagestance, error) {
	self := &Stagestance{}
	self.node = NewNode(parent, call)
	stage := call.Callable().(*syntax.Stage)

	stagecodePath, err := stage.Src.FindPath(srcPaths)
	if err != nil && len(srcPaths) > 0 {
		util.PrintError(err, "runtime", "WARNING: stage code not found")
	}
	self.node.resolvedCmd = stagecodePath
	if self.node.top.rt.Config.StressTest {
		switch self.node.stagecode.Type {
		case syntax.PythonStage:
			self.node.stagecode.Path = util.RelPath(path.Join("..", "adapters", "python", "tester"))
		default:
			return self, fmt.Errorf("Unsupported stress test language: %v", stage.Src.Lang)
		}
	}
	if stage.Resources != nil {
		self.node.resources = &JobResources{
			Threads: float64(stage.Resources.Threads),
			MemGB:   float64(stage.Resources.MemGB),
			VMemGB:  float64(stage.Resources.VMemGB),
			Special: stage.Resources.Special,
		}
	}

	if splits := call.Forks; len(splits) > 0 {
		exps := make([]*syntax.CallStm, len(splits))
		for i, s := range splits {
			exps[i] = s.Call()
		}
		self.node.forkRoots = exps
	}
	return self, self.buildForks()
}

func (self *Stagestance) buildForks() error {
	self.node.buildForks()

	return setupRetains(
		self.node.call.Callable().(*syntax.Stage).Retain,
		self.node.forks)
}

func setupRetains(retain *syntax.RetainParams, forks []*Fork) error {
	if retain != nil {
		for _, param := range retain.Params {
			for _, fork := range forks {
				if fork.fileArgs == nil {
					fork.fileArgs = make(
						map[string]map[Nodable]struct{},
						len(retain.Params))
				}
				if arg := fork.fileArgs[param.Id]; arg == nil {
					fork.fileArgs[param.Id] = map[Nodable]struct{}{
						nil: {},
					}
				} else {
					arg[nil] = struct{}{}
				}
			}
		}
	}
	return nil
}

func (self *Stagestance) getNode() *Node    { return self.node }
func (self *Stagestance) GetFQName() string { return self.node.GetFQName() }

func (self *Stagestance) GetPrenodes() map[string]Nodable {
	return self.node.GetPrenodes()
}

func (self *Stagestance) GetPostNodes() map[string]Nodable {
	return self.node.GetPostNodes()
}

func (self *Stagestance) matchForks(id ForkId) []*Fork {
	return self.node.matchForks(id)
}

func (self *Stagestance) Callable() syntax.Callable {
	return self.node.Callable()
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
	lastQueueCheck   time.Time
	queueCheckLock   sync.Mutex
	queueCheckActive bool
}

// Run a script whenever a pipestance finishes.
func (self *Pipestance) OnFinishHook(outerCtx context.Context) {
	if exec_path := self.getNode().top.rt.Config.OnFinishHandler; exec_path != "" {
		ctx, task := trace.NewTask(outerCtx, "onfinish")
		defer task.End()
		util.Println("\nRunning onfinish handler...")

		// Build command line arguments:
		// $1 = path to piestance
		// $2 = {complete|failed}
		// $3 = pipestance ID
		// $4 = path to error file (if there was an error)
		args := []string{self.GetPath(), string(self.GetState(ctx)), self.node.top.GetPsid()}
		if self.GetState(ctx) == Failed {
			_, _, _, _, _, err_paths := self.GetFatalError()
			if len(err_paths) > 0 {
				err_path, _ := filepath.Rel(filepath.Dir(self.GetPath()), err_paths[0])
				args = append(args, err_path)
			}
		}

		/* Find the real path to the script */
		real_path, err := exec.LookPath(exec_path)
		if err != nil {
			util.LogInfo("finishr", "Could not find %v: %v", exec_path, err)
			return
		}

		ectx, cancel := context.WithTimeout(ctx, time.Hour)
		defer cancel()

		cmd := exec.CommandContext(ectx, real_path, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = util.Pdeathsig(
			new(syscall.SysProcAttr),
			syscall.SIGINT)

		if err := cmd.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok &&
				ee.ProcessState != nil && ee.ProcessState.Sys() != nil {
				if ws, ok := ee.ProcessState.Sys().(*syscall.WaitStatus); ok && ws.Signaled() {
					util.LogError(err, "finishr", "%s died with signal %v",
						real_path, ws.Signal())
				} else {
					util.LogError(err, "finishr", "%v failed", real_path)
				}
			} else {
				util.LogInfo("finishr", "Error running %v: %v",
					real_path, err)
			}
		}
	}
}

func NewPipestance(parent Nodable, call *syntax.CallGraphPipeline, srcPaths []string) (*Pipestance, error) {
	self := &Pipestance{}
	self.node = NewNode(parent, call)
	self.metadata = NewMetadata(self.node.parent.GetFQName(), self.GetPath())

	// Build subcall tree.
	pipeline := call.Callable().(*syntax.Pipeline)
	preflightNodes := []Nodable{}
	for _, subcall := range call.Children {
		switch subcall := subcall.(type) {
		case *syntax.CallGraphStage:
			if s, err := NewStagestance(self.node, subcall, srcPaths); err != nil {
				return nil, err
			} else {
				self.node.subnodes[subcall.Call().Id] = s

				// check if the stage is a preflight.  Preflights have no
				// outputs, cannot take a dependency on another stage in the
				// pipeline, and are a dependency for all other stages in the
				// pipeline.
				//
				// Only stages can be preflight.
				if subcall.Call().Modifiers.Preflight {
					preflightNodes = append(preflightNodes, s)
				}
			}
		case *syntax.CallGraphPipeline:
			if p, err := NewPipestance(self.node, subcall, srcPaths); err != nil {
				return nil, err
			} else {
				self.node.subnodes[subcall.Call().Id] = p
			}
		default:
			return nil, fmt.Errorf("Unsupported callable type %v", subcall)
		}
	}

	// Also depends on stages bound to return values.
	if r := pipeline.Ret; r != nil && r.Bindings != nil && len(r.Bindings.List) > 0 {
		self.node.makeReturnBindings(r.Bindings.List)
	}

	// Add preflight dependencies if preflight stages exist.
	for _, preflightNode := range preflightNodes {
		for _, subnode := range self.node.subnodes {
			if !subnode.getNode().call.Call().Modifiers.Preflight {
				subnode.getNode().setPrenode(preflightNode)
			}
		}
	}
	if splits := call.Forks; len(splits) > 0 {
		exps := make([]*syntax.CallStm, len(splits))
		for i, s := range splits {
			exps[i] = s.Call()
		}
		self.node.forkRoots = exps
	}
	return self, self.buildForks()
}

func (self *Pipestance) buildForks() error {
	self.node.buildForks()
	if rs := self.node.call.Retained(); len(rs) > 0 {
		for _, r := range rs {
			node := self.node.top.allNodes[r.Id]
			if node == nil {
				return fmt.Errorf("Retaining unknown node %s", r.Id)
			}
			for _, fork := range node.forks {
				if fork.fileArgs == nil {
					fork.fileArgs = make(
						map[string]map[Nodable]struct{},
						len(rs))
				}
				if arg := fork.fileArgs[r.OutputId]; arg == nil {
					fork.fileArgs[r.OutputId] = map[Nodable]struct{}{
						nil: {},
					}
				} else {
					arg[nil] = struct{}{}
				}
			}
		}
	}

	return nil
}

func (self *Pipestance) getNode() *Node    { return self.node }
func (self *Pipestance) GetPname() string  { return self.node.call.Call().Id }
func (self *Pipestance) GetPsid() string   { return self.node.top.GetPsid() }
func (self *Pipestance) GetFQName() string { return self.node.GetFQName() }
func (self *Pipestance) RefreshState(ctx context.Context) {
	r := trace.StartRegion(ctx, "refresh")
	defer r.End()
	self.node.refreshState(self.readOnly())
}
func (self *Pipestance) readOnly() bool { return !self.metadata.exists(Lock) }

func (self *Pipestance) GetPrenodes() map[string]Nodable {
	return self.node.GetPrenodes()
}

func (self *Pipestance) GetPostNodes() map[string]Nodable {
	return self.node.GetPostNodes()
}

func (self *Pipestance) matchForks(id ForkId) []*Fork {
	return self.node.matchForks(id)
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

func (self *Pipestance) LoadMetadata(ctx context.Context) {
	// We used to make this concurrent but ended up with too many
	// goroutines (Pranav's 96-sample run).
	r := trace.StartRegion(ctx, "LoadMetadata")
	defer r.End()
	for _, node := range self.allNodes() {
		node.loadMetadata()
	}
	for _, node := range self.allNodes() {
		node.state = node.getState()
		if node.state == Running && !self.readOnly() {
			if err := node.mkdirs(); err != nil {
				util.LogError(err, "runtime",
					"Error creating pipestance directories.")
			}
		}
	}
}

func (self *Pipestance) GetState(ctx context.Context) MetadataState {
	r := trace.StartRegion(ctx, "pipestance.GetState")
	defer r.End()
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

// RestoreForks attempts to compute dynamic forks for nodes in a pipestance.
//
// Normally, dynamic forks are computed when a stage or pipeline transitions to
// the running state.  However, if the pipestance was restarted then some of
// those transitions may have already hapend so the forks need to be computed
// at the moment of reattachment instead.
func (self *Pipestance) RestoreForks(ctx context.Context) {
	defer trace.StartRegion(ctx, "restoreForks").End()
	for _, node := range self.allNodes() {
		node.expandForks(false)
		for _, fork := range node.forks {
			// Force a cache refresh.
			fork.metadatasCache = nil
		}
	}
}

// Restart jobs for running nodes, during an auto-restart event.
//
// During auto-restart, running jobs are generally allowed to continue running.
// However, because the graph of stage objects is rebuilt, the MaxJobsSemaphore
// on the remote job manager must be rebuilt, and we need to reset any jobs
// which were waiting on that semaphore.
func (self *Pipestance) RestartRunningNodes(jobMode string, outerCtx context.Context) error {
	ctx, task := trace.NewTask(outerCtx, "restartNodes")
	defer task.End()
	if self.readOnly() {
		return &RuntimeError{"Pipestance is in read only mode."}
	}
	self.LoadMetadata(ctx)
	nodes := self.node.getFrontierNodes()
	var errs syntax.ErrorList
	for _, node := range nodes {
		if node.state == Running {
			util.PrintInfo("runtime", "Found orphaned stage: %s", node.GetFQName())
			if jobMode == localMode || node.local {
				if err := node.reset(); err != nil {
					errs = append(errs, err)
				}
			}
		}
		if jobMode != localMode &&
			!node.local &&
			(node.state == Running ||
				node.state == Failed && !node.top.rt.Config.FullStageReset) {
			if node.top.rt.Config.Debug {
				util.PrintInfo("runtime",
					"Found failed cluster-mode node: %s",
					node.GetFQName())
			}
			if err := node.reattachJobs(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs.If()
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
		if node.state == Running && (jobMode == localMode || node.local) {
			util.PrintInfo("runtime", "Found orphaned local stage: %s", node.GetFQName())
			if err := node.restartLocalJobs(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *Pipestance) CheckHeartbeats(ctx context.Context) {
	r := trace.StartRegion(ctx, "heartbeat")
	defer r.End()
	if self.readOnly() {
		return
	}
	self.queryQueue(ctx)

	nodes := self.node.getFrontierNodes()
	for _, node := range nodes {
		node.checkHeartbeats()
	}
}

// Check that the queued jobs are actually queued.
func (self *Pipestance) queryQueue(outerCtx context.Context) {
	prepDone := false
	ctx, task := trace.NewTask(outerCtx, "queryQueue")
	defer func() {
		if !prepDone {
			task.End()
		}
	}()
	if self.node == nil || self.node.top == nil || self.node.top.rt == nil ||
		self.node.top.rt.JobManager == nil ||
		!self.node.top.rt.JobManager.hasQueueCheck() {
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
	prepDone = true
	go func(ctx context.Context, task *trace.Task) {
		defer task.End()
		queued, raw := self.node.top.rt.JobManager.checkQueue(jobsIn, ctx)
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
	}(ctx, task)
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
func (self *Pipestance) StepNodes(ctx context.Context) bool {
	r := trace.StartRegion(ctx, "StepNodes")
	defer r.End()
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
	if err := self.node.top.rt.LocalJobManager.refreshResources(
		self.node.top.rt.Config.JobMode == localMode); err != nil {
		util.LogError(err, "runtime",
			"Error refreshing local resources: %s", err.Error())
	}
	if self.node.top.rt.LocalJobManager != self.node.top.rt.JobManager {
		if err := self.node.top.rt.JobManager.refreshResources(false); err != nil {
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
		perf, _ := node.serializePerf()
		ser = append(ser, perf)
	}
	util.LogInfo("perform", "Serializing pipestance performance data.")
	if len(ser) > 0 {
		overallPerf := ser[0]
		self.ComputeDiskUsage(overallPerf)
		overallPerf.HighMem = &self.node.top.rt.LocalJobManager.highMem
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

func (self *Pipestance) ComputeDiskUsage(nodePerf *NodePerfInfo) *NodePerfInfo {
	nodes := self.allNodes()
	allStorageEvents := make(StorageEventByTimestamp, 0, len(nodes)*2)
	for _, node := range nodes {
		_, storageEvents := node.serializePerf()
		for _, ev := range storageEvents {
			if ev.DeltaBytes != 0 {
				allStorageEvents = append(allStorageEvents,
					NewStorageEvent(ev.Timestamp, ev.DeltaBytes, func(name string, ev *VdrEvent) string {
						if ev.DeltaBytes > 0 {
							return fmt.Sprintf("%s alloc", name)
						} else {
							return fmt.Sprintf("%s delete", name)
						}
					}(node.GetFQName(), ev)))
			}
		}
	}

	allStorageEvents = allStorageEvents.Collapse()

	var highMark, currentMark int64

	byteStamps := make([]*NodeByteStamp, len(allStorageEvents))
	for idx, se := range allStorageEvents {
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
	if !self.node.top.rt.Config.Zip {
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
		files, _ := metadata.glob()
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
	return self.node.parent.getNode().top.invocation
}

func (self *Pipestance) VerifyJobMode() error {
	self.metadata.loadCache()
	if self.metadata.exists(JobModeFile) {
		jobMode := self.metadata.readRaw(JobModeFile)
		if jobMode != self.node.top.rt.Config.JobMode {
			return &PipestanceJobModeError{self.GetPsid(), jobMode}
		}
	}
	return nil
}

func (self *Pipestance) GetTimestamp() string {
	data := self.metadata.readRaw(TimestampFile)
	return parseTimestamp(data)
}

func (self *Pipestance) GetVersions() (string, string, error) {
	data := self.metadata.readRaw(VersionsFile)
	return ParseVersions(data)
}

func (self *Pipestance) PostProcess() {
	self.node.postProcess()
	start, _ := self.metadata.readRawBytes(TimestampFile)
	start = append(start, "\nend: "...)
	if err := self.metadata.WriteRawBytes(TimestampFile, append(start, util.Timestamp()...)); err != nil {
		util.LogError(err, "runtime",
			"Error writing completion timestamp.")
	}
	if err := self.Immortalize(false); err != nil {
		util.LogError(err, "runtime",
			"Error finalizing pipestance state.")
	}
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
	var errs syntax.ErrorList
	if !self.metadata.exists(Perf) {
		if err := self.metadata.Write(Perf, self.SerializePerf()); err != nil {
			errs = append(errs, err)
		}
	}
	if !self.metadata.exists(FinalState) {
		if err := self.metadata.Write(FinalState, self.SerializeState()); err != nil {
			errs = append(errs, err)
		}
	}
	if !self.metadata.exists(MetadataZip) {
		zipPath := self.metadata.MetadataFilePath(MetadataZip)
		if err := self.ZipMetadata(zipPath); err != nil {
			util.LogError(err, "runtime", "Failed to create metadata zip file %s: %s",
				zipPath, err.Error())
			os.Remove(zipPath)
			errs = append(errs, err)
		}
	}
	return errs.If()
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
		uuid, err := self.metadata.readRawSafe(UuidFile)
		self.uuid = uuid
		if self.node.top.envs != nil && self.node.top.envs["MRO_UUID"] == "" {
			self.node.top.envs["MRO_UUID"] = uuid
		}
		return uuid, err
	}
}

func (self *Pipestance) SetUuid(uuid string) error {
	if self.node.top.envs != nil {
		self.node.top.envs["MRO_UUID"] = uuid
	}
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
		return &PipestanceLockedError{self.node.top.GetPsid(), self.GetPath()}
	}
	util.RegisterSignalHandler(self)
	if err := self.metadata.WriteTime(Lock); err != nil {
		util.LogError(err, "runtime", "Error writing pipestance lock file.")
	}
	return nil
}

func (self *Pipestance) unlock() {
	if err := self.metadata.remove(Lock); err != nil {
		util.LogError(err, "runtime", "Error removing pipestance lock file.")
	}
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
	fqname      string
	rt          *Runtime
	types       *syntax.TypeLookup
	journalPath string
	tmpPath     string
	mroPaths    []string
	envs        map[string]string
	invocation  *InvocationData
	version     VersionInfo
	allNodes    map[string]*Node
	node        Node
}

func (self *TopNode) getNode() *Node { return &self.node }

func (self *TopNode) GetFQName() string {
	return self.fqname
}

func (self *TopNode) GetPsid() string {
	return self.fqname[3:]
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

func (self *TopNode) Types() *syntax.TypeLookup {
	return self.types
}

func (self *TopNode) matchForks(id ForkId) []*Fork {
	return self.node.matchForks(id)
}

func NewTopNode(rt *Runtime, fqname string, p string,
	mroPaths []string, mroVersion string,
	envs map[string]string, j *InvocationData,
	types *syntax.TypeLookup) *TopNode {
	self := &TopNode{
		fqname:     fqname,
		rt:         rt,
		types:      types,
		invocation: j,
		mroPaths:   mroPaths,
		version: VersionInfo{
			Pipelines: mroVersion,
			Martian:   rt.Config.MartianVersion,
		},
		node: Node{
			frontierNodes: &threadSafeNodeMap{nodes: make(map[string]Nodable)},
			path:          p,
		},
		journalPath: path.Join(p, "journal"),
		tmpPath:     path.Join(p, "tmp"),
		envs:        make(map[string]string, len(envs)+1),
		allNodes:    make(map[string]*Node),
	}
	self.node.top = self

	for key, value := range envs {
		self.envs[key] = value
	}
	self.envs["TMPDIR"] = self.tmpPath

	return self
}

//=============================================================================
// Factory
//=============================================================================

// Encapsulates the information needed to instantiate a pipestance, either by
// creating one or reattaching to an existing one.
type PipestanceFactory interface {
	ReattachToPipestance(ctx context.Context) (*Pipestance, error)
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
	tags           []string
	checkSrc       bool
	readOnly       bool
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
	return runtimePipeFactory{
		rt:             rt,
		invocationSrc:  invocationSrc,
		invocationPath: invocationPath,
		psid:           psid,
		mroPaths:       mroPaths,
		pipestancePath: pipestancePath,
		mroVersion:     mroVersion,
		envs:           envs,
		tags:           tags,
		checkSrc:       checkSrc,
		readOnly:       readOnly,
	}
}

func (self runtimePipeFactory) ReattachToPipestance(ctx context.Context) (*Pipestance, error) {
	attachMethod := self.rt.ReattachToPipestance
	if self.checkSrc && path.Base(self.invocationPath) == MroSourceFile.FileName() {
		attachMethod = self.rt.ReattachToPipestanceWithMroSrc
	}
	return attachMethod(
		self.psid, self.pipestancePath,
		self.invocationSrc, self.invocationPath,
		self.mroPaths, self.mroVersion, self.envs,
		self.checkSrc, self.readOnly, ctx)
}

func (self runtimePipeFactory) InvokePipeline() (*Pipestance, error) {
	return self.rt.InvokePipeline(self.invocationSrc, self.invocationPath, self.psid,
		self.pipestancePath, self.mroPaths, self.mroVersion, self.envs, self.tags)
}

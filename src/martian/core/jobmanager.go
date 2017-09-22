// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

// Martian job managers for local and remote (SGE, LSF, etc) modes.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/gosigar"
	"io/ioutil"
	"martian/util"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const maxRetries = 5
const retryExitCode = 513

func max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

//
// Job managers
//
type JobManager interface {
	execJob(string, []string, map[string]string, *Metadata, int, int, string, string, string, bool)

	// Given a list of candidate job IDs, returns a list of jobIds which may be
	// still queued or running, as well as the stderr output of the queue check.
	// If this job manager doesn't know how to check the queue or the query
	// fails, it simply returns the list it was given.
	checkQueue([]string) ([]string, string)
	// Returns true if checkQueue does something useful.
	hasQueueCheck() bool
	// Returns the amount of time to wait, after a job is found to be unknown
	// to the job manager, before declaring the job dead.  This is to protect
	// against races between NFS caching in the directories Martian watches and
	// whatever the queue manager uses to syncronize state.
	queueCheckGrace() time.Duration
	refreshLocalResources(localMode bool) error
	GetSystemReqs(int, int) (int, int)
	GetMaxCores() int
	GetMaxMemGB() int
	GetSettings() *JobManagerSettings
}

type LocalJobManager struct {
	maxCores    int
	maxMemGB    int
	jobSettings *JobManagerSettings
	coreSem     *ResourceSemaphore
	memMBSem    *ResourceSemaphore
	lastMemDiff int64
	queue       []*exec.Cmd
	debug       bool
	limitLoad   bool
	highMem     ObservedMemory
}

func NewLocalJobManager(userMaxCores int, userMaxMemGB int,
	debug bool, limitLoadavg bool) *LocalJobManager {
	self := &LocalJobManager{
		debug:     debug,
		limitLoad: limitLoadavg,
	}
	self.jobSettings = verifyJobManager("local", -1).jobSettings

	// Set Max number of cores usable at one time.
	if userMaxCores > 0 {
		// If user specified --localcores, use that value for Max usable cores.
		self.maxCores = userMaxCores
		util.LogInfo("jobmngr", "Using %d core%s, per --localcores option.",
			self.maxCores, util.Pluralize(self.maxCores))
	} else {
		// Otherwise, set Max usable cores to total number of cores reported
		// by the system.
		self.maxCores = runtime.NumCPU()
		util.LogInfo("jobmngr", "Using %d logical core%s available on system.",
			self.maxCores, util.Pluralize(self.maxCores))
	}

	// Set Max GB of memory usable at one time.
	if userMaxMemGB > 0 {
		// If user specified --localmem, use that value for Max usable GB.
		self.maxMemGB = userMaxMemGB
		util.LogInfo("jobmngr", "Using %d GB, per --localmem option.", self.maxMemGB)
	} else {
		// Otherwise, set Max usable GB to MAXMEM_FRACTION * GB of total
		// memory reported by the system.
		MAXMEM_FRACTION := 0.9
		sysMem := sigar.Mem{}
		sysMem.Get()
		sysMemGB := int(float64(sysMem.Total) * MAXMEM_FRACTION / 1073741824)
		// Set floor to 1GB.
		if sysMemGB < 1 {
			sysMemGB = 1
		}
		self.maxMemGB = sysMemGB
		util.LogInfo("jobmngr", "Using %d GB, %d%% of system memory.", self.maxMemGB,
			int(MAXMEM_FRACTION*100))
	}

	self.coreSem = NewResourceSemaphore(int64(self.maxCores), "threads")
	self.memMBSem = NewResourceSemaphore(int64(self.maxMemGB)*1024, "MB of memory")
	self.queue = []*exec.Cmd{}
	util.RegisterSignalHandler(self)
	return self
}

func (self *LocalJobManager) GetSettings() *JobManagerSettings {
	return self.jobSettings
}

func (self *LocalJobManager) refreshLocalResources(localMode bool) error {
	sysMem := sigar.Mem{}
	if err := sysMem.Get(); err != nil {
		return err
	}
	usedMem, err := GetProcessTreeMemory(os.Getpid(), false)
	if err != nil {
		util.LogError(err, "jobmngr", "Error getting process tree memory usage.")
	}
	self.highMem.IncreaseTo(usedMem)
	memDiff := self.memMBSem.UpdateFreeUsed(
		(int64(sysMem.ActualFree)+(1024*1024-1))/(1024*1024),
		(usedMem.Rss+(1024*1024-1))/(1024*1024))
	if memDiff < -int64(self.maxMemGB)*1024/8 &&
		memDiff/128 < self.lastMemDiff &&
		(localMode || sysMem.ActualFree < 2*1024*1024*1024) {
		util.LogInfo("jobmngr", "%.1fGB less memory than expected was free", float64(-memDiff)/1024)
		if usedMem.Rss > self.memMBSem.Reserved()*1024*1024 {
			util.LogInfo("jobmngr",
				"MRP and its child processes are using %.1fGB of rss.  %.1fGB are reserved.",
				float64(usedMem.Rss)/(1024*1024*1024), self.memMBSem.Reserved()/1024)
		}
	}
	self.lastMemDiff = memDiff / 128
	if self.limitLoad {
		load := sigar.LoadAverage{}
		if err := load.Get(); err != nil {
			return err
		}
		if diff := self.coreSem.UpdateActual(int64(
			float64(runtime.NumCPU()) - load.One + 0.9)); diff < -int64(self.maxCores)/4 &&
			localMode {
			util.LogInfo("jobmngr", "%d fewer core%s than expected were free.", -diff, util.Pluralize(int(-diff)))
		}
	}
	return nil
}

func (self *LocalJobManager) HandleSignal(sig os.Signal) {
	if self.highMem.Rss > 0 {
		if ser, err := json.MarshalIndent(self.highMem, "", "  "); err == nil {
			util.LogInfo("jobmngr", "Highest memory usage observed: %s", string(ser))
		}
	}
}

func (self *LocalJobManager) GetSystemReqs(threads int, memGB int) (int, int) {
	// Sanity check and cap to self.maxCores.
	if threads < 1 {
		threads = self.jobSettings.ThreadsPerJob
	}
	if threads > self.maxCores {
		if self.debug {
			util.LogInfo("jobmngr", "Need %d core%s but settling for %d.", threads,
				util.Pluralize(threads), self.maxCores)
		}
		threads = self.maxCores
	}

	// Sanity check and cap to self.maxMemGB.
	if memGB == 0 {
		memGB = self.jobSettings.MemGBPerJob
	}
	if memGB < 0 {
		avail := int(self.memMBSem.CurrentSize() / 1024)
		if avail < 1 || avail < -memGB {
			memGB = -memGB
		} else {
			if self.debug {
				util.LogInfo("jobmngr", "Adaptive request for at least %d GB being given %d.",
					-memGB, avail)
			}
			memGB = avail
		}
	}
	// TODO: Stop allowing stages to ask for more than the max.  Require
	// stages which can adapt to the available memory to ask for a negative
	// amount as a sentinel.
	if memGB > self.maxMemGB {
		if self.debug {
			util.LogInfo("jobmngr", "Need %d GB but settling for %d.", memGB,
				self.maxMemGB)
		}
		util.LogInfo(
			"jobmngr", "Job asked for %d GB but is being given %d.  This behavior is deprecated - jobs which can adapt their memory usage should ask for -%d.",
			memGB, self.maxMemGB, memGB)
		memGB = self.maxMemGB
	}

	return threads, memGB
}

func (self *LocalJobManager) checkQueue(ids []string) ([]string, string) {
	return ids, ""
}

func (self *LocalJobManager) hasQueueCheck() bool {
	return false
}

func (self *LocalJobManager) queueCheckGrace() time.Duration {
	return 0
}

func (self *LocalJobManager) Enqueue(shellCmd string, argv []string,
	envs map[string]string, metadata *Metadata, threads int, memGB int,
	fqname string, retries int, waitTime int, localpreflight bool) {

	time.Sleep(time.Second * time.Duration(waitTime))
	go func() {
		// Exec the shell directly.
		cmd := exec.Command(shellCmd, argv...)
		cmd.Dir = metadata.curFilesPath
		if self.maxCores < runtime.NumCPU() {
			// If, and only if, the user specified a core limit less than the
			// detected core count, make sure jobs actually don't use more
			// threads than they're supposed to.
			cmd.Env = util.MergeEnv(threadEnvs(self, threads, envs))
		} else {
			// In this case it's ok if we oversubscribe a bit since we're
			// (probably) not sharing the machine.
			cmd.Env = util.MergeEnv(envs)
		}

		stdoutPath := metadata.MetadataFilePath("stdout")
		stderrPath := metadata.MetadataFilePath("stderr")

		threads, memGB = self.GetSystemReqs(threads, memGB)

		// Acquire cores.
		if self.debug {
			util.LogInfo("jobmngr", "Waiting for %d core%s", threads, util.Pluralize(threads))
		}
		if err := self.coreSem.Acquire(int64(threads)); err != nil {
			util.LogError(err, "jobmngr",
				"%s requested %d threads, but the job manager was only configured to use %d.",
				metadata.fqname, threads, self.maxCores)
			metadata.WriteRaw("errors", err.Error())
			return
		}
		if self.debug {
			util.LogInfo("jobmngr", "Acquired %d core%s (%d/%d in use)", threads,
				util.Pluralize(threads), self.coreSem.InUse(), self.maxCores)
		}

		// Acquire memory.
		if self.debug {
			util.LogInfo("jobmngr", "Waiting for %d GB", memGB)
		}
		if err := self.memMBSem.Acquire(int64(memGB) * 1024); err != nil {
			util.LogError(err, "jobmngr",
				"%s requested %d GB of memory, but the job manager was only configured to use %d.",
				metadata.fqname, memGB, self.maxMemGB)
			metadata.WriteRaw("errors", err.Error())
			return
		}
		if self.debug {
			util.LogInfo("jobmngr", "Acquired %d GB (%.1f/%d in use)", memGB,
				float64(self.memMBSem.InUse())/1024, self.maxMemGB)
		}
		if self.debug {
			util.LogInfo("jobmngr", "%d goroutines", runtime.NumGoroutine())
		}

		// Set up _stdout and _stderr for the job.
		if stdoutFile, err := os.Create(stdoutPath); err == nil {
			stdoutFile.WriteString("[stdout]\n")
			// If local preflight stage, let stdout go to the console
			if localpreflight {
				cmd.Stdout = os.Stdout
			} else {
				cmd.Stdout = stdoutFile
			}
			defer stdoutFile.Close()
		}
		cmd.SysProcAttr = Pdeathsig(&syscall.SysProcAttr{}, syscall.SIGTERM)
		if stderrFile, err := os.Create(stderrPath); err == nil {
			stderrFile.WriteString("[stderr]\n")
			cmd.Stderr = stderrFile
			defer stderrFile.Close()
		}

		// Run the command and wait for completion.
		err := func(metadata *Metadata, cmd *exec.Cmd) error {
			util.EnterCriticalSection()
			defer util.ExitCriticalSection()
			err := cmd.Start()
			if err == nil {
				metadata.remove("queued_locally")
			}
			return err
		}(metadata, cmd)
		if err == nil {
			err = cmd.Wait()
		}

		// CentOS < 5.5 workaround
		if err != nil {
			exitCodeString := fmt.Sprintf("errno %d", retryExitCode)
			if strings.Contains(err.Error(), exitCodeString) {
				retries += 1
				if waitTime == 0 {
					waitTime = 2
				} else {
					waitTime *= 2
				}
			} else {
				retries = maxRetries + 1
			}
			if retries > maxRetries {
				metadata.WriteRaw("errors", err.Error())
			} else {
				util.LogInfo("jobmngr", "Job failed: %s. Retrying job %s in %d seconds", err.Error(), fqname, waitTime)
				self.Enqueue(shellCmd, argv, envs, metadata, threads, memGB, fqname, retries,
					waitTime, localpreflight)
			}
		}

		// Release cores.
		self.coreSem.Release(int64(threads))
		if self.debug {
			util.LogInfo("jobmngr", "Released %d core%s (%d/%d in use)", threads,
				util.Pluralize(threads), self.coreSem.InUse(), self.maxCores)
		}
		// Release memory.
		self.memMBSem.Release(int64(memGB) * 1024)
		if self.debug {
			util.LogInfo("jobmngr", "Released %d GB (%.1f/%d in use)", memGB,
				float64(self.memMBSem.InUse())/1024, self.maxMemGB)
		}
	}()
}

func (self *LocalJobManager) GetMaxCores() int {
	return self.maxCores
}

func (self *LocalJobManager) GetMaxMemGB() int {
	return self.maxMemGB
}

func (self *LocalJobManager) execJob(shellCmd string, argv []string,
	envs map[string]string, metadata *Metadata, threads int, memGB int,
	special string, fqname string, shellName string, preflight bool) {
	self.Enqueue(shellCmd, argv, envs, metadata, threads, memGB, fqname, 0, 0, preflight)
}

type RemoteJobManager struct {
	jobMode              string
	jobResourcesMappings map[string]string
	config               jobManagerConfig
	memGBPerCore         int
	maxJobs              int
	jobFreqMillis        int
	jobSem               *ResourceSemaphore
	limiter              *time.Ticker
	debug                bool
}

func NewRemoteJobManager(jobMode string, memGBPerCore int, maxJobs int, jobFreqMillis int,
	jobResources string, debug bool) *RemoteJobManager {
	self := &RemoteJobManager{}
	self.jobMode = jobMode
	self.memGBPerCore = memGBPerCore
	self.maxJobs = maxJobs
	self.jobFreqMillis = jobFreqMillis
	self.debug = debug
	self.config = verifyJobManager(jobMode, memGBPerCore)

	// Parse jobresources mappings
	self.jobResourcesMappings = map[string]string{}
	for _, mapping := range strings.Split(jobResources, ";") {
		if len(mapping) > 0 {
			parts := strings.Split(mapping, ":")
			if len(parts) == 2 {
				self.jobResourcesMappings[parts[0]] = parts[1]
				util.LogInfo("jobmngr", "Mapping %s to %s", parts[0], parts[1])
			} else {
				util.LogInfo("jobmngr", "Could not parse mapping: %s", mapping)
			}
		}
	}

	if self.maxJobs > 0 {
		self.jobSem = NewResourceSemaphore(int64(self.maxJobs), "jobs")
	}
	if self.jobFreqMillis > 0 {
		self.limiter = time.NewTicker(time.Millisecond * time.Duration(self.jobFreqMillis))
	} else {
		// dummy limiter to keep struct OK
		self.limiter = time.NewTicker(time.Millisecond * 1)
	}
	return self
}

func (self *RemoteJobManager) refreshLocalResources(localMode bool) error {
	// Remote job manager doesn't manage resources.
	return nil
}

func (self *RemoteJobManager) GetMaxCores() int {
	return 0
}

func (self *RemoteJobManager) GetMaxMemGB() int {
	return 0
}

func (self *RemoteJobManager) GetSettings() *JobManagerSettings {
	return self.config.jobSettings
}

func (self *RemoteJobManager) GetSystemReqs(threads int, memGB int) (int, int) {
	// Sanity check the thread count.
	if threads < 1 {
		threads = self.config.jobSettings.ThreadsPerJob
	}

	// Sanity check memory requirements.
	if memGB < 0 {
		// Negative request is a sentinel value requesting as much as possible.
		// For remote jobs, at least for now, give reserve the minimum usable.
		memGB = -memGB
	}
	if memGB < 1 {
		memGB = self.config.jobSettings.MemGBPerJob
	}

	// Compute threads needed based on memory requirements.
	if self.memGBPerCore > 0 {
		threads = max(threads, (memGB+self.memGBPerCore-1)/self.memGBPerCore)
	}

	// If threading is disabled, use only 1 thread.
	if !self.config.threadingEnabled {
		threads = 1
	}

	return threads, memGB
}

func (self *RemoteJobManager) execJob(shellCmd string, argv []string,
	envs map[string]string, metadata *Metadata, threads int, memGB int,
	special string, fqname string, shellName string, localpreflight bool) {

	// no limit, send the job
	if self.maxJobs <= 0 {
		self.sendJob(shellCmd, argv, envs, metadata, threads, memGB, special, fqname, shellName)
		return
	}

	// grab job when ready, block until job state changes to a finalized state
	go func() {
		if self.debug {
			util.LogInfo("jobmngr", "Waiting for job: %s", fqname)
		}
		// if we want to try to put a more precise cap on cluster execution load,
		// might be preferable to request num threads here instead of a slot per job
		if err := self.jobSem.Acquire(1); err != nil {
			panic(err)
		}
		if self.debug {
			util.LogInfo("jobmngr", "Job sent: %s", fqname)
		}
		self.sendJob(shellCmd, argv, envs, metadata, threads, memGB, special, fqname, shellName)
		for {
			if state, _ := metadata.getState(); state == Complete || state == Failed {
				self.jobSem.Release(1)
				if self.debug {
					util.LogInfo("jobmngr", "Job finished: %s (%s)", fqname, state)
				}
				break
			}
			time.Sleep(time.Second * 1)
		}
	}()
}

// Set environment variables which control thread count.  Do not override
// envs from above.
func threadEnvs(self JobManager, threads int, envs map[string]string) map[string]string {
	thr := strconv.Itoa(threads)
	newEnvs := make(map[string]string)
	for _, env := range self.GetSettings().ThreadEnvs {
		newEnvs[env] = thr
	}
	for key, value := range envs {
		newEnvs[key] = value
	}
	return newEnvs
}

func (self *RemoteJobManager) sendJob(shellCmd string, argv []string, envs map[string]string,
	metadata *Metadata, threads int, memGB int, special string, fqname string, shellName string) {

	if self.jobFreqMillis > 0 {
		<-(self.limiter.C)
		if self.debug {
			util.LogInfo("jobmngr", "Job rate-limit released: %s", fqname)
		}
	}
	threads, memGB = self.GetSystemReqs(threads, memGB)

	// figure out per-thread memory requirements for the template.  If
	// mempercore is specified, use that as what we send.
	memGBPerThread := memGB
	if self.memGBPerCore > 0 {
		memGBPerThread = self.memGBPerCore
	} else {
		// ceil to make sure that we're not starving a job
		memGBPerThread = memGB / threads
		if memGB%threads > 0 {
			memGBPerThread += 1
		}
	}

	mappedJobResourcesOpt := ""
	// If a __special is specified for this stage, and the runtime was called
	// with MRO_JOBRESOURCES defining a mapping from __special to a complex value
	// expression, then populate the resources option into the template. Otherwise,
	// leave it blank to revert to default behavior.
	if len(special) > 0 {
		if resources, ok := self.jobResourcesMappings[special]; ok {
			mappedJobResourcesOpt = strings.Replace(
				self.config.jobResourcesOpt,
				"__RESOURCES__", resources, 1)
		}
	}

	argv = append(
		util.FormatEnv(threadEnvs(self, threads, envs)),
		append([]string{shellCmd},
			argv...)...,
	)
	params := map[string]string{
		"JOB_NAME":          fqname + "." + shellName,
		"THREADS":           fmt.Sprintf("%d", threads),
		"STDOUT":            metadata.MetadataFilePath("stdout"),
		"STDERR":            metadata.MetadataFilePath("stderr"),
		"JOB_WORKDIR":       metadata.curFilesPath,
		"CMD":               strings.Join(argv, " "),
		"MEM_GB":            fmt.Sprintf("%d", memGB),
		"MEM_MB":            fmt.Sprintf("%d", memGB*1024),
		"MEM_KB":            fmt.Sprintf("%d", memGB*1024*1024),
		"MEM_B":             fmt.Sprintf("%d", memGB*1024*1024*1024),
		"MEM_GB_PER_THREAD": fmt.Sprintf("%d", memGBPerThread),
		"MEM_MB_PER_THREAD": fmt.Sprintf("%d", memGBPerThread*1024),
		"MEM_KB_PER_THREAD": fmt.Sprintf("%d", memGBPerThread*1024*1024),
		"MEM_B_PER_THREAD":  fmt.Sprintf("%d", memGBPerThread*1024*1024*1024),
		"ACCOUNT":           os.Getenv("MRO_ACCOUNT"),
		"RESOURCES":         mappedJobResourcesOpt,
	}

	// Replace template annotations with actual values
	args := []string{}
	template := self.config.jobTemplate
	for key, val := range params {
		if len(val) > 0 {
			args = append(args, fmt.Sprintf("__MRO_%s__", key), val)
		} else {
			// Remove line containing parameter from template
			for _, line := range strings.Split(template, "\n") {
				if strings.Contains(line, fmt.Sprintf("__MRO_%s__", key)) {
					template = strings.Replace(template, line, "", 1)
				}
			}
		}
	}
	r := strings.NewReplacer(args...)
	jobscript := r.Replace(template)
	metadata.WriteRaw("jobscript", jobscript)

	cmd := exec.Command(self.config.jobCmd, self.config.jobCmdArgs...)
	cmd.Dir = metadata.curFilesPath
	cmd.Stdin = strings.NewReader(jobscript)

	util.EnterCriticalSection()
	defer util.ExitCriticalSection()
	metadata.remove("queued_locally")
	if output, err := cmd.CombinedOutput(); err != nil {
		metadata.WriteRaw("errors", "jobcmd error ("+err.Error()+"):\n"+string(output))
	} else {
		trimmed := strings.TrimSpace(string(output))
		// jobids should not have spaces in them.  This is the most general way to
		// check that a string is actually a jobid.
		if trimmed != "" && !strings.ContainsAny(trimmed, " \t\n\r") {
			metadata.WriteRaw("jobid", strings.TrimSpace(string(output)))
			metadata.cache("jobid", metadata.uniquifier)
		}
	}
}

func (self *RemoteJobManager) checkQueue(ids []string) ([]string, string) {
	if self.config.queueQueryCmd == "" {
		return ids, ""
	}
	jobPath := util.RelPath(path.Join("..", "jobmanagers"))
	cmd := exec.Command(path.Join(jobPath, self.config.queueQueryCmd))
	cmd.Dir = jobPath
	cmd.Stdin = strings.NewReader(strings.Join(ids, "\n"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return ids, stderr.String()
	}
	return strings.Split(string(output), "\n"), stderr.String()
}

func (self *RemoteJobManager) hasQueueCheck() bool {
	return self.config.queueQueryCmd != ""
}

func (self *RemoteJobManager) queueCheckGrace() time.Duration {
	return self.config.queueQueryGrace
}

//
// Helper functions for job manager file parsing
//

type JobModeEnv struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type JobModeJson struct {
	Cmd             string        `json:"cmd"`
	Args            []string      `json:"args,omitempty"`
	QueueQuery      string        `json:"queue_query,omitempty"`
	QueueQueryGrace int           `json:"queue_query_grace_secs,omitempty"`
	ResourcesOpt    string        `json:"resopt"`
	JobEnvs         []*JobModeEnv `json:"envs"`
}

type JobManagerSettings struct {
	ThreadsPerJob int      `json:"threads_per_job"`
	MemGBPerJob   int      `json:"memGB_per_job"`
	ThreadEnvs    []string `json:"thread_envs"`
}

type JobManagerJson struct {
	JobSettings *JobManagerSettings     `json:"settings"`
	JobModes    map[string]*JobModeJson `json:"jobmodes"`
}

type jobManagerConfig struct {
	jobSettings      *JobManagerSettings
	jobCmd           string
	jobCmdArgs       []string
	queueQueryCmd    string
	queueQueryGrace  time.Duration
	jobResourcesOpt  string
	jobTemplate      string
	threadingEnabled bool
}

func verifyJobManager(jobMode string, memGBPerCore int) jobManagerConfig {
	jobPath := util.RelPath(path.Join("..", "jobmanagers"))

	// Check for existence of job manager JSON file
	jobJsonFile := path.Join(jobPath, "config.json")
	if _, err := os.Stat(jobJsonFile); os.IsNotExist(err) {
		util.PrintInfo("jobmngr", "Job manager config file %s does not exist.", jobJsonFile)
		os.Exit(1)
	}
	util.LogInfo("jobmngr", "Job config = %s", jobJsonFile)
	bytes, _ := ioutil.ReadFile(jobJsonFile)

	// Parse job manager JSON file
	var jobJson *JobManagerJson
	if err := json.Unmarshal(bytes, &jobJson); err != nil {
		util.PrintInfo("jobmngr", "Job manager config file %s does not contain valid JSON.", jobJsonFile)
		os.Exit(1)
	}

	// Validate settings fields
	jobSettings := jobJson.JobSettings
	if jobSettings == nil {
		util.PrintInfo("jobmngr", "Job manager config file %s should contain 'settings' field.", jobJsonFile)
		os.Exit(1)
	}
	if jobSettings.ThreadsPerJob <= 0 {
		util.PrintInfo("jobmngr", "Job manager config %s contains invalid default threads per job.", jobJsonFile)
		os.Exit(1)
	}
	if jobSettings.MemGBPerJob <= 0 {
		util.PrintInfo("jobmngr", "Job manager config %s contains invalid default memory (GB) per job.", jobJsonFile)
		os.Exit(1)
	}

	if jobMode == "local" {
		// Local job mode only needs to verify settings parameters
		return jobManagerConfig{jobSettings: jobSettings}
	}

	var jobTemplateFile string
	var jobErrorMsg string

	jobModeJson, ok := jobJson.JobModes[jobMode]
	if ok {
		jobTemplateFile = path.Join(jobPath, jobMode+".template")
		exampleJobTemplateFile := jobTemplateFile + ".example"
		jobErrorMsg = fmt.Sprintf("Job manager template file %s does not exist.\n\nTo set up a job manager template, please follow instructions in %s.",
			jobTemplateFile, exampleJobTemplateFile)
	} else {
		jobTemplateFile = jobMode
		jobMode = strings.Replace(path.Base(jobTemplateFile), ".template", "", 1)

		jobModeJson, ok = jobJson.JobModes[jobMode]
		if !strings.HasSuffix(jobTemplateFile, ".template") || !ok {
			util.PrintInfo("jobmngr", "Job manager template file %s must be named <name_of_job_manager>.template.", jobTemplateFile)
			os.Exit(1)
		}
		jobErrorMsg = fmt.Sprintf("Job manager template file %s does not exist.", jobTemplateFile)
	}

	jobCmd := jobModeJson.Cmd
	if len(jobModeJson.Args) == 0 {
		util.LogInfo("jobmngr", "Job submit command = %s", jobCmd)
	} else {
		util.LogInfo("jobmngr", "Job submit comand = %s %s", jobCmd, strings.Join(jobModeJson.Args, " "))
	}

	jobResourcesOpt := jobModeJson.ResourcesOpt
	util.LogInfo("jobmngr", "Job submit resources option = %s", jobResourcesOpt)

	// Check for existence of job manager template file
	if _, err := os.Stat(jobTemplateFile); os.IsNotExist(err) {
		util.PrintInfo("jobmngr", jobErrorMsg)
		os.Exit(1)
	}
	util.LogInfo("jobmngr", "Job template = %s", jobTemplateFile)
	bytes, _ = ioutil.ReadFile(jobTemplateFile)
	jobTemplate := string(bytes)

	// Check if template includes threading.
	jobThreadingEnabled := false
	if strings.Contains(jobTemplate, "__MRO_THREADS__") {
		jobThreadingEnabled = true
	}

	// Check if memory reservations or mempercore are enabled
	if !strings.Contains(jobTemplate, "__MRO_MEM_GB") && !strings.Contains(jobTemplate, "__MRO_MEM_MB") && memGBPerCore <= 0 {
		util.Println("\nCLUSTER MODE WARNING:\n   Memory reservations are not enabled in your job template.\n   To avoid memory over-subscription, we highly recommend that you enable\n   memory reservations on your cluster, or use the --mempercore option.\nPlease consult the documentation for details.\n")
	}

	// Verify job command exists
	incPaths := strings.Split(os.Getenv("PATH"), ":")
	if _, found := util.SearchPaths(jobCmd, incPaths); !found {
		util.Println("Job command '%s' not found in (%s)",
			jobCmd, strings.Join(incPaths, ", "))
		os.Exit(1)
	}

	// Verify environment variables
	envs := [][]string{}
	for _, entry := range jobModeJson.JobEnvs {
		envs = append(envs, []string{entry.Name, entry.Description})
	}
	util.EnvRequire(envs, true)

	var queueGrace time.Duration
	if jobModeJson.QueueQuery != "" {
		queueGrace = time.Duration(jobModeJson.QueueQueryGrace) * time.Second
		// Default to 1 hour.
		if queueGrace == 0 {
			queueGrace = time.Hour
		}
	}

	return jobManagerConfig{
		jobSettings,
		jobCmd,
		jobModeJson.Args,
		jobModeJson.QueueQuery,
		queueGrace,
		jobResourcesOpt,
		jobTemplate,
		jobThreadingEnabled,
	}
}

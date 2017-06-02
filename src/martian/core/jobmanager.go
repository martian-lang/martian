//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian job managers for local and remote (SGE, LSF, etc) modes.
//
package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/gosigar"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxRetries = 5
const retryExitCode = 513

func GetCPUInfo() (int, int, int, int) {
	// From Linux /proc/cpuinfo, count sockets, physical cores, and logical cores.
	// runtime.numCPU() does not provide this
	//
	fname := "/proc/cpuinfo"

	f, err := os.Open(fname)
	if err != nil {
		return 0, 0, 0, 0
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	sockets := -1
	physicalCoresPerSocket := -1
	logicalCores := -1

	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}

		fields := strings.Split(string(line), ":")
		if len(fields) < 2 {
			continue
		}

		k := strings.TrimSpace(fields[0])
		v := strings.TrimSpace(fields[1])

		switch k {
		case "physical id":
			i, err := strconv.ParseInt(v, 10, 32)
			if err == nil && int(i) > sockets {
				sockets = int(i)
			}
		case "core id":
			i, err := strconv.ParseInt(v, 10, 32)
			if err == nil && int(i) > physicalCoresPerSocket {
				physicalCoresPerSocket = int(i)
			}
		case "processor":
			i, err := strconv.ParseInt(v, 10, 32)
			if err == nil && int(i) > logicalCores {
				logicalCores = int(i)
			}
		}
	}

	if sockets > -1 && physicalCoresPerSocket > -1 && logicalCores > -1 {
		sockets += 1
		physicalCoresPerSocket += 1
		physicalCores := sockets * physicalCoresPerSocket
		logicalCores += 1
		return sockets, physicalCoresPerSocket, physicalCores, logicalCores
	} else {
		return 0, 0, 0, 0
	}
}

//
// Semaphore implementation
//
type Semaphore struct {
	counter chan bool
	pmutex  *sync.Mutex
	vmutex  *sync.Mutex
}

func NewSemaphore(capacity int) *Semaphore {
	return &Semaphore{
		make(chan bool, capacity),
		&sync.Mutex{},
		&sync.Mutex{},
	}
}

func (self *Semaphore) P(n int) {
	self.pmutex.Lock()
	for i := 0; i < n; i++ {
		self.counter <- true
	}
	self.pmutex.Unlock()
}

func (self *Semaphore) V(n int) {
	self.vmutex.Lock()
	for i := 0; i < n; i++ {
		<-self.counter
	}
	self.vmutex.Unlock()
}

func (self *Semaphore) len() int {
	return len(self.counter)
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
	GetSystemReqs(int, int) (int, int)
	GetMaxCores() int
	GetMaxMemGB() int
}

type LocalJobManager struct {
	maxCores    int
	maxMemGB    int
	jobSettings *JobManagerSettings
	coreSem     *Semaphore
	memGBSem    *Semaphore
	queue       []*exec.Cmd
	debug       bool
}

func NewLocalJobManager(userMaxCores int, userMaxMemGB int, debug bool) *LocalJobManager {
	self := &LocalJobManager{}
	self.debug = debug
	self.jobSettings = verifyJobManager("local", -1).jobSettings

	// Set Max number of cores usable at one time.
	if userMaxCores > 0 {
		// If user specified --localcores, use that value for Max usable cores.
		self.maxCores = userMaxCores
		LogInfo("jobmngr", "Using %d core%s, per --localcores option.",
			self.maxCores, Pluralize(self.maxCores))
	} else {
		// Otherwise, set Max usable cores to total number of cores reported
		// by the system.
		self.maxCores = runtime.NumCPU()
		LogInfo("jobmngr", "Using %d logical core%s available on system.",
			self.maxCores, Pluralize(self.maxCores))
	}

	// Set Max GB of memory usable at one time.
	if userMaxMemGB > 0 {
		// If user specified --localmem, use that value for Max usable GB.
		self.maxMemGB = userMaxMemGB
		LogInfo("jobmngr", "Using %d GB, per --localmem option.", self.maxMemGB)
	} else {
		// Otherwise, set Max usable GB to MAXMEM_FRACTION * GB of total
		// memory reported by the system.
		MAXMEM_FRACTION := 1.0
		sysMem := sigar.Mem{}
		sysMem.Get()
		sysMemGB := int(float64(sysMem.Total) * MAXMEM_FRACTION / 1073741824)
		// Set floor to 1GB.
		if sysMemGB < 1 {
			sysMemGB = 1
		}
		self.maxMemGB = sysMemGB
		LogInfo("jobmngr", "Using %d GB, %d%% of system memory.", self.maxMemGB,
			int(MAXMEM_FRACTION*100))
	}

	self.coreSem = NewSemaphore(self.maxCores)
	self.memGBSem = NewSemaphore(self.maxMemGB)
	self.queue = []*exec.Cmd{}
	return self
}

func (self *LocalJobManager) GetSystemReqs(threads int, memGB int) (int, int) {
	// Sanity check and cap to self.maxCores.
	if threads < 1 {
		threads = self.jobSettings.ThreadsPerJob
	}
	if threads > self.maxCores {
		if self.debug {
			LogInfo("jobmngr", "Need %d core%s but settling for %d.", threads,
				Pluralize(threads), self.maxCores)
		}
		threads = self.maxCores
	}

	// Sanity check and cap to self.maxMemGB.
	if memGB < 1 {
		memGB = self.jobSettings.MemGBPerJob
	}
	if memGB > self.maxMemGB {
		if self.debug {
			LogInfo("jobmngr", "Need %d GB but settling for %d.", memGB,
				self.maxMemGB)
		}
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
		cmd.Dir = metadata.filesPath
		cmd.Env = MergeEnv(envs)

		stdoutPath := metadata.makePath("stdout")
		stderrPath := metadata.makePath("stderr")

		threads, memGB = self.GetSystemReqs(threads, memGB)

		// Acquire cores.
		if self.debug {
			LogInfo("jobmngr", "Waiting for %d core%s", threads, Pluralize(threads))
		}
		self.coreSem.P(threads)
		if self.debug {
			LogInfo("jobmngr", "Acquired %d core%s (%d/%d in use)", threads,
				Pluralize(threads), self.coreSem.len(), self.maxCores)
		}

		// Acquire memory.
		if self.debug {
			LogInfo("jobmngr", "Waiting for %d GB", memGB)
		}
		self.memGBSem.P(memGB)
		if self.debug {
			LogInfo("jobmngr", "Acquired %d GB (%d/%d in use)", memGB,
				self.memGBSem.len(), self.maxMemGB)
		}
		if self.debug {
			LogInfo("jobmngr", "%d goroutines", runtime.NumGoroutine())
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
		if stderrFile, err := os.Create(stderrPath); err == nil {
			stderrFile.WriteString("[stderr]\n")
			cmd.Stderr = stderrFile
			defer stderrFile.Close()
		}

		// Run the command and wait for completion.
		var err error
		if err = cmd.Start(); err == nil {
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
				metadata.writeRaw("errors", err.Error())
			} else {
				LogInfo("jobmngr", "Job failed: %s. Retrying job %s in %d seconds", err.Error(), fqname, waitTime)
				self.Enqueue(shellCmd, argv, envs, metadata, threads, memGB, fqname, retries,
					waitTime, localpreflight)
			}
		}

		// Release cores.
		self.coreSem.V(threads)
		if self.debug {
			LogInfo("jobmngr", "Released %d core%s (%d/%d in use)", threads,
				Pluralize(threads), self.coreSem.len(), self.maxCores)
		}
		// Release memory.
		self.memGBSem.V(memGB)
		if self.debug {
			LogInfo("jobmngr", "Released %d GB (%d/%d in use)", memGB,
				self.memGBSem.len(), self.maxMemGB)
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
	jobSem               *Semaphore
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
				LogInfo("jobmngr", "Mapping %s to %s", parts[0], parts[1])
			} else {
				LogInfo("jobmngr", "Could not parse mapping: %s", mapping)
			}
		}
	}

	if self.maxJobs > 0 {
		self.jobSem = NewSemaphore(self.maxJobs)
	} else {
		// dummy value to keep struct OK
		self.jobSem = NewSemaphore(0)
	}
	if self.jobFreqMillis > 0 {
		self.limiter = time.NewTicker(time.Millisecond * time.Duration(self.jobFreqMillis))
	} else {
		// dummy limiter to keep struct OK
		self.limiter = time.NewTicker(time.Millisecond * 1)
	}
	return self
}

func (self *RemoteJobManager) GetMaxCores() int {
	return 0
}

func (self *RemoteJobManager) GetMaxMemGB() int {
	return 0
}

func (self *RemoteJobManager) GetSystemReqs(threads int, memGB int) (int, int) {
	// Sanity check the thread count.
	if threads < 1 {
		threads = self.config.jobSettings.ThreadsPerJob
	}

	// Sanity check memory requirements.
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
			LogInfo("jobmngr", "Waiting for job: %s", fqname)
		}
		// if we want to try to put a more precise cap on cluster execution load,
		// might be preferable to request num threads here instead of a slot per job
		self.jobSem.P(1)
		if self.debug {
			LogInfo("jobmngr", "Job sent: %s", fqname)
		}
		self.sendJob(shellCmd, argv, envs, metadata, threads, memGB, special, fqname, shellName)
		for {
			if state, _ := metadata.getState(""); state == "complete" || state == "failed" {
				self.jobSem.V(1)
				if self.debug {
					LogInfo("jobmngr", "Job finished: %s (%s)", fqname, state)
				}
				break
			}
			time.Sleep(time.Second * 1)
		}
	}()
}

func (self *RemoteJobManager) sendJob(shellCmd string, argv []string, envs map[string]string,
	metadata *Metadata, threads int, memGB int, special string, fqname string, shellName string) {

	if self.jobFreqMillis > 0 {
		<-(self.limiter.C)
		if self.debug {
			LogInfo("jobmngr", "Job rate-limit released: %s", fqname)
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

	argv = append([]string{shellCmd}, argv...)
	argv = append(FormatEnv(envs), argv...)
	params := map[string]string{
		"JOB_NAME":          fqname + "." + shellName,
		"THREADS":           fmt.Sprintf("%d", threads),
		"STDOUT":            metadata.makePath("stdout"),
		"STDERR":            metadata.makePath("stderr"),
		"CMD":               strings.Join(argv, " "),
		"MEM_GB":            fmt.Sprintf("%d", memGB),
		"MEM_MB":            fmt.Sprintf("%d", memGB*1024),
		"MEM_KB":            fmt.Sprintf("%d", memGB*1024*1024),
		"MEM_B":             fmt.Sprintf("%d", memGB*1024*1024*1024),
		"MEM_GB_PER_THREAD": fmt.Sprintf("%d", memGBPerThread),
		"MEM_MB_PER_THREAD": fmt.Sprintf("%d", memGBPerThread*1024),
		"MEM_KB_PER_THREAD": fmt.Sprintf("%d", memGBPerThread*1024*1024),
		"MEM_B_PER_THREAD":  fmt.Sprintf("%d", memGBPerThread*1024*1024*1024),
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
	metadata.writeRaw("jobscript", jobscript)

	cmd := exec.Command(self.config.jobCmd, self.config.jobCmdArgs...)
	cmd.Dir = metadata.filesPath
	cmd.Stdin = strings.NewReader(jobscript)
	if output, err := cmd.CombinedOutput(); err != nil {
		metadata.writeRaw("errors", "jobcmd error ("+err.Error()+"):\n"+string(output))
	} else {
		trimmed := strings.TrimSpace(string(output))
		// jobids should not have spaces in them.  This is the most general way to
		// check that a string is actually a jobid.
		if trimmed != "" && !strings.ContainsAny(trimmed, " \t\n\r") {
			metadata.writeRaw("jobid", strings.TrimSpace(string(output)))
			metadata.cache("jobid")
		}
	}
}

func (self *RemoteJobManager) checkQueue(ids []string) ([]string, string) {
	if self.config.queueQueryCmd == "" {
		return ids, ""
	}
	jobPath := RelPath(path.Join("..", "jobmanagers"))
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
	ThreadsPerJob int `json:"threads_per_job"`
	MemGBPerJob   int `json:"memGB_per_job"`
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
	jobPath := RelPath(path.Join("..", "jobmanagers"))

	// Check for existence of job manager JSON file
	jobJsonFile := path.Join(jobPath, "config.json")
	if _, err := os.Stat(jobJsonFile); os.IsNotExist(err) {
		PrintInfo("jobmngr", "Job manager config file %s does not exist.", jobJsonFile)
		os.Exit(1)
	}
	LogInfo("jobmngr", "Job config = %s", jobJsonFile)
	bytes, _ := ioutil.ReadFile(jobJsonFile)

	// Parse job manager JSON file
	var jobJson *JobManagerJson
	if err := json.Unmarshal(bytes, &jobJson); err != nil {
		PrintInfo("jobmngr", "Job manager config file %s does not contain valid JSON.", jobJsonFile)
		os.Exit(1)
	}

	// Validate settings fields
	jobSettings := jobJson.JobSettings
	if jobSettings == nil {
		PrintInfo("jobmngr", "Job manager config file %s should contain 'settings' field.", jobJsonFile)
		os.Exit(1)
	}
	if jobSettings.ThreadsPerJob <= 0 {
		PrintInfo("jobmngr", "Job manager config %s contains invalid default threads per job.", jobJsonFile)
		os.Exit(1)
	}
	if jobSettings.MemGBPerJob <= 0 {
		PrintInfo("jobmngr", "Job manager config %s contains invalid default memory (GB) per job.", jobJsonFile)
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
			PrintInfo("jobmngr", "Job manager template file %s must be named <name_of_job_manager>.template.", jobTemplateFile)
			os.Exit(1)
		}
		jobErrorMsg = fmt.Sprintf("Job manager template file %s does not exist.", jobTemplateFile)
	}

	jobCmd := jobModeJson.Cmd
	if len(jobModeJson.Args) == 0 {
		LogInfo("jobmngr", "Job submit command = %s", jobCmd)
	} else {
		LogInfo("jobmngr", "Job submit comand = %s %s", jobCmd, strings.Join(jobModeJson.Args, " "))
	}

	jobResourcesOpt := jobModeJson.ResourcesOpt
	LogInfo("jobmngr", "Job submit resources option = %s", jobResourcesOpt)

	// Check for existence of job manager template file
	if _, err := os.Stat(jobTemplateFile); os.IsNotExist(err) {
		PrintInfo("jobmngr", jobErrorMsg)
		os.Exit(1)
	}
	LogInfo("jobmngr", "Job template = %s", jobTemplateFile)
	bytes, _ = ioutil.ReadFile(jobTemplateFile)
	jobTemplate := string(bytes)

	// Check if template includes threading.
	jobThreadingEnabled := false
	if strings.Contains(jobTemplate, "__MRO_THREADS__") {
		jobThreadingEnabled = true
	}

	// Check if memory reservations or mempercore are enabled
	if !strings.Contains(jobTemplate, "__MRO_MEM_GB") && !strings.Contains(jobTemplate, "__MRO_MEM_MB") && memGBPerCore <= 0 {
		Println("\nCLUSTER MODE WARNING:\n   Memory reservations are not enabled in your job template.\n   To avoid memory over-subscription, we highly recommend that you enable\n   memory reservations on your cluster, or use the --mempercore option.\nPlease consult the documentation for details.\n")
	}

	// Verify job command exists
	incPaths := strings.Split(os.Getenv("PATH"), ":")
	if _, found := SearchPaths(jobCmd, incPaths); !found {
		Println("Job command '%s' not found in (%s)",
			jobCmd, strings.Join(incPaths, ", "))
		os.Exit(1)
	}

	// Verify environment variables
	envs := [][]string{}
	for _, entry := range jobModeJson.JobEnvs {
		envs = append(envs, []string{entry.Name, entry.Description})
	}
	EnvRequire(envs, true)

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

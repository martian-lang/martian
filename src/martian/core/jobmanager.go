//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian job managers for local and remote (SGE, LSF, etc) modes.
//
package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry/gosigar"
)

const heartbeatTimeout = 60 // 60 minutes
const maxRetries = 5
const retryExitCode = 513

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
	execJob(string, []string, map[string]string, *Metadata, int, int, string, string)
	GetSystemReqs(int, int) (int, int)
	GetMaxCores() int
	GetMaxMemGB() int
	MonitorJob(*Metadata)
	UnmonitorJob(*Metadata)
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
	_, _, self.jobSettings, _, _, _ = verifyJobManager("local", -1)

	// Set Max number of cores usable at one time.
	if userMaxCores > 0 {
		// If user specified --Maxcores, use that value for Max usable cores.
		self.maxCores = userMaxCores
		LogInfo("jobmngr", "Using %d core%s, per --maxcores option.",
			self.maxCores, Pluralize(self.maxCores))
	} else {
		// Otherwise, set Max usable cores to total number of cores reported
		// by the system.
		self.maxCores = runtime.NumCPU()
		LogInfo("jobmngr", "Using %d core%s available on system.",
			self.maxCores, Pluralize(self.maxCores))
	}

	// Set Max GB of memory usable at one time.
	if userMaxMemGB > 0 {
		// If user specified --Maxmem, use that value for Max usable GB.
		self.maxMemGB = userMaxMemGB
		LogInfo("jobmngr", "Using %d GB, per --maxmem option.", self.maxMemGB)
	} else {
		// Otherwise, set Max usable GB to MAXMEM_FRACTION * GB of total
		// memory reported by the system.
		MAXMEM_FRACTION := 0.75
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

func (self *LocalJobManager) Enqueue(shellCmd string, argv []string, envs map[string]string, metadata *Metadata,
	threads int, memGB int, fqname string, retries int, waitTime int) {

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
			LogInfo("jobmngr", "Acquiring %d core%s (%d/%d in use)", threads,
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
			cmd.Stdout = stdoutFile
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
				self.Enqueue(shellCmd, argv, envs, metadata, threads, memGB, fqname, retries, waitTime)
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

func (self *LocalJobManager) MonitorJob(metadata *Metadata) {
}

func (self *LocalJobManager) UnmonitorJob(metadata *Metadata) {
}

func (self *LocalJobManager) execJob(shellCmd string, argv []string, envs map[string]string,
	metadata *Metadata, threads int, memGB int, fqname string, shellName string) {
	self.Enqueue(shellCmd, argv, envs, metadata, threads, memGB, fqname, 0, 0)
}

type JobMonitor struct {
	metadata      *Metadata
	lastHeartbeat time.Time
	running       bool
}

type RemoteJobManager struct {
	jobMode          string
	jobTemplate      string
	jobCmd           string
	jobSettings      *JobManagerSettings
	threadingEnabled bool
	memGBPerCore     int
	monitorList      []*JobMonitor
	monitorListMutex *sync.Mutex
}

func NewRemoteJobManager(jobMode string, memGBPerCore int) *RemoteJobManager {
	self := &RemoteJobManager{}
	self.jobMode = jobMode
	self.monitorList = []*JobMonitor{}
	self.monitorListMutex = &sync.Mutex{}
	self.memGBPerCore = memGBPerCore
	_, _, self.jobSettings, self.jobCmd, self.jobTemplate, self.threadingEnabled = verifyJobManager(jobMode, memGBPerCore)
	self.processMonitorList()
	return self
}

func (self *RemoteJobManager) GetMaxCores() int {
	return 0
}

func (self *RemoteJobManager) GetMaxMemGB() int {
	return 0
}

func (self *RemoteJobManager) MonitorJob(metadata *Metadata) {
	self.monitorListMutex.Lock()
	self.monitorList = append(self.monitorList, &JobMonitor{metadata, time.Now(), false})
	self.monitorListMutex.Unlock()
}

func (self *RemoteJobManager) UnmonitorJob(metadata *Metadata) {
	self.monitorListMutex.Lock()
	for i, monitor := range self.monitorList {
		if monitor.metadata == metadata {
			self.monitorList = append(self.monitorList[:i], self.monitorList[i+1:]...)
			break
		}
	}
	self.monitorListMutex.Unlock()
}

func (self *RemoteJobManager) GetSystemReqs(threads int, memGB int) (int, int) {
	// Sanity check the thread count.
	if threads < 1 {
		threads = self.jobSettings.ThreadsPerJob
	}

	// Sanity check memory requirements.
	if memGB < 1 {
		memGB = self.jobSettings.MemGBPerJob
	}

	// Compute threads needed based on memory requirements.
	if self.memGBPerCore > 0 {
		threads = max(threads, (memGB+self.memGBPerCore-1)/self.memGBPerCore)
	}

	// If threading is disabled, use only 1 thread.
	if !self.threadingEnabled {
		threads = 1
	}

	return threads, memGB
}

func (self *RemoteJobManager) execJob(shellCmd string, argv []string, envs map[string]string,
	metadata *Metadata, threads int, memGB int, fqname string, shellName string) {

	threads, memGB = self.GetSystemReqs(threads, memGB)

	argv = append([]string{shellCmd}, argv...)
	argv = append(FormatEnv(envs), argv...)
	params := map[string]string{
		"JOB_NAME": fqname + "." + shellName,
		"THREADS":  fmt.Sprintf("%d", threads),
		"STDOUT":   metadata.makePath("stdout"),
		"STDERR":   metadata.makePath("stderr"),
		"CMD":      strings.Join(argv, " "),
		"MEM_GB":   fmt.Sprintf("%d", memGB),
		"MEM_MB":   fmt.Sprintf("%d", memGB*1024),
	}

	// Replace template annotations with actual values
	args := []string{}
	template := self.jobTemplate
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

	cmd := exec.Command(self.jobCmd)
	cmd.Dir = metadata.filesPath
	cmd.Stdin = strings.NewReader(jobscript)
	if _, err := cmd.CombinedOutput(); err != nil {
		metadata.writeRaw("errors", "jobcmd error:\n"+err.Error())
	} else {
		self.MonitorJob(metadata)
	}
}

func (self *RemoteJobManager) copyAndClearMonitorList() []*JobMonitor {
	self.monitorListMutex.Lock()
	monitorList := make([]*JobMonitor, len(self.monitorList))
	copy(monitorList, self.monitorList)
	self.monitorList = []*JobMonitor{}
	self.monitorListMutex.Unlock()
	return monitorList
}

func (self *RemoteJobManager) processMonitorList() {
	go func() {
		for {
			monitorList := self.copyAndClearMonitorList()
			newMonitorList := []*JobMonitor{}
			for _, monitor := range monitorList {
				state, _ := monitor.metadata.getState("")
				if state == "complete" || state == "failed" {
					continue
				}
				if state == "running" {
					if !monitor.running || monitor.metadata.exists("heartbeat") {
						monitor.metadata.uncache("heartbeat")
						monitor.lastHeartbeat = time.Now()
						monitor.running = true
					}
					if time.Since(monitor.lastHeartbeat) > time.Minute*heartbeatTimeout {
						monitor.metadata.writeRaw("errors", fmt.Sprintf("Heartbeat not detected for %d minutes. Assuming job has failed", heartbeatTimeout))
					}
				}
				newMonitorList = append(newMonitorList, monitor)
			}
			self.monitorListMutex.Lock()
			self.monitorList = append(self.monitorList, newMonitorList...)
			self.monitorListMutex.Unlock()
			time.Sleep(time.Minute * time.Duration(5))
		}
	}()
}

//
// Helper functions for job manager file parsing
//

type JobManagerEnv struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type JobManagerSettings struct {
	ThreadsPerJob int `json:"threads_per_job"`
	MemGBPerJob   int `json:"memGB_per_job"`
}

type JobManagerJson struct {
	JobSettings *JobManagerSettings         `json:"settings"`
	JobCmds     map[string]string           `json:"jobcmd"`
	JobEnvs     map[string][]*JobManagerEnv `json:"env"`
}

func verifyJobManager(jobMode string, memGBPerCore int) (string, *JobManagerJson, *JobManagerSettings, string, string, bool) {
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
		return jobJsonFile, jobJson, jobSettings, "", "", false
	}

	// Check if job mode is supported by default
	jobTemplateFile := jobMode
	jobErrorMsg := ""
	jobCmd, ok := jobJson.JobCmds[jobMode]
	if ok {
		jobTemplateFile = path.Join(jobPath, fmt.Sprintf("%s.template", jobCmd))
		exampleJobTemplateFile := jobTemplateFile + ".example"
		jobErrorMsg = fmt.Sprintf("Job manager template file %s does not exist.\n\nTo set up a job manager template, please follow instructions in %s.",
			jobTemplateFile, exampleJobTemplateFile)
	} else {
		if !strings.HasSuffix(jobTemplateFile, ".template") {
			PrintInfo("jobmngr", "Job manager template file %s must be named <name_of_job_submit_cmd>.template.", jobTemplateFile)
			os.Exit(1)
		}
		jobCmd = strings.Replace(path.Base(jobTemplateFile), ".template", "", 1)
		jobErrorMsg = fmt.Sprintf("Job manager template file %s does not exist.", jobTemplateFile)
	}
	LogInfo("jobmngr", "Job submit command = %s", jobCmd)

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
	if !strings.Contains(jobTemplate, "__MRO_MEM_GB__") && !strings.Contains(jobTemplate, "__MRO_MEM_MB__") && memGBPerCore <= 0 {
		Println("\nCLUSTER MODE WARNING:\n   Memory reservations are not enabled in your job template.\n   To avoid memory over-subscription, we highly recommend that you enable\n   memory reservations on your cluster, or use the --mempercore option.\nPlease consult the documentation for details.\n")
	}

	// Verify job command exists
	incPaths := strings.Split(os.Getenv("PATH"), ":")
	if _, found := searchPaths(jobCmd, incPaths); !found {
		Println("Job command '%s' not found in (%s)",
			jobCmd, strings.Join(incPaths, ", "))
		os.Exit(1)
	}

	// Verify environment variables
	entries, ok := jobJson.JobEnvs[jobCmd]
	if ok {
		envs := [][]string{}
		for _, entry := range entries {
			envs = append(envs, []string{entry.Name, entry.Description})
		}
		EnvRequire(envs, true)
	}

	return jobJsonFile, jobJson, jobSettings, jobCmd, jobTemplate, jobThreadingEnabled
}

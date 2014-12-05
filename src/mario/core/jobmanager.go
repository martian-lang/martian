//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Mario job managers for local and remote (SGE, LSF, etc) modes.
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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry/gosigar"
)

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
	execJob(string, []string, *Metadata, int, int, string, string)
	GetMaxCores() int
	GetMaxMemGB() int
}

type LocalJobManager struct {
	maxCores int
	maxMemGB int
	coreSem  *Semaphore
	memGBSem *Semaphore
	queue    []*exec.Cmd
	debug    bool
}

func NewLocalJobManager(userMaxCores int, userMaxMemGB int, debug bool) *LocalJobManager {
	self := &LocalJobManager{}
	self.debug = debug

	// Set Max number of cores usable at one time.
	if userMaxCores > 0 {
		// If user specified --Maxcores, use that value for Max usable cores.
		self.maxCores = userMaxCores
		LogInfo("jobmgr", "Using %d core%s, per --Maxcores option.",
			self.maxCores, Pluralize(self.maxCores))
	} else {
		// Otherwise, set Max usable cores to total number of cores reported
		// by the system.
		self.maxCores = runtime.NumCPU()
		LogInfo("jobmgr", "Using %d core%s available on system.",
			self.maxCores, Pluralize(self.maxCores))
	}

	// Set Max GB of memory usable at one time.
	if userMaxMemGB > 0 {
		// If user specified --Maxmem, use that value for Max usable GB.
		self.maxMemGB = userMaxMemGB
		LogInfo("jobmgr", "Using %d GB, per --Maxmem option.", self.maxMemGB)
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
		LogInfo("jobmgr", "Using %d GB, %d%% of system memory.", self.maxMemGB,
			int(MAXMEM_FRACTION*100))
	}

	self.coreSem = NewSemaphore(self.maxCores)
	self.memGBSem = NewSemaphore(self.maxMemGB)
	self.queue = []*exec.Cmd{}
	return self
}

func (self *LocalJobManager) Enqueue(cmd *exec.Cmd, threads int, memGB int,
	stdoutPath string, stderrPath string, errorsPath string) {

	go func() {
		// Sanity check and cap to self.maxCores.
		if threads < 1 {
			threads = 1
		}
		if threads > self.maxCores {
			if self.debug {
				LogInfo("jobmgr", "Need %d core%s but settling for %d.", threads,
					Pluralize(threads), self.maxCores)
			}
			threads = self.maxCores
		}

		// Sanity check and cap to self.maxMemGB.
		if memGB < 1 {
			memGB = 1
		}
		if memGB > self.maxMemGB {
			if self.debug {
				LogInfo("jobmgr", "Need %d GB but settling for %d.", memGB,
					self.maxMemGB)
			}
			memGB = self.maxMemGB
		}

		// Acquire cores.
		if self.debug {
			LogInfo("jobmgr", "Waiting for %d core%s", threads, Pluralize(threads))
		}
		self.coreSem.P(threads)
		if self.debug {
			LogInfo("jobmgr", "Acquiring %d core%s (%d/%d in use)", threads,
				Pluralize(threads), self.coreSem.len(), self.maxCores)
		}

		// Acquire memory.
		if self.debug {
			LogInfo("jobmgr", "Waiting for %d GB", memGB)
		}
		self.memGBSem.P(memGB)
		if self.debug {
			LogInfo("jobmgr", "Acquired %d GB (%d/%d in use)", memGB,
				self.memGBSem.len(), self.maxMemGB)
		}
		if self.debug {
			LogInfo("jobmgr", "%d goroutines", runtime.NumGoroutine())
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
		if err := cmd.Start(); err != nil {
			ioutil.WriteFile(errorsPath, []byte(err.Error()), 0644)
		} else {
			if err := cmd.Wait(); err != nil {
				ioutil.WriteFile(errorsPath, []byte(err.Error()), 0644)
			}
		}

		// Release cores.
		self.coreSem.V(threads)
		if self.debug {
			LogInfo("jobmgr", "Released %d core%s (%d/%d in use)", threads,
				Pluralize(threads), self.coreSem.len(), self.maxCores)
		}
		// Release memory.
		self.memGBSem.V(memGB)
		if self.debug {
			LogInfo("jobmgr", "Released %d GB (%d/%d in use)", memGB,
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

func (self *LocalJobManager) execJob(shellCmd string, argv []string, metadata *Metadata,
	threads int, memGB int, fqname string, shellName string) {

	// Exec the shell directly.
	cmd := exec.Command(shellCmd, argv...)

	// Connect child to _stdout and _stderr metadata files.
	stdoutPath := metadata.makePath("stdout")
	stderrPath := metadata.makePath("stderr")
	errorsPath := metadata.makePath("errors")

	// Enqueue the command to the local job manager.
	self.Enqueue(cmd, threads, memGB, stdoutPath, stderrPath, errorsPath)
}

type JobMonitor struct {
	jobId    int
	metadata *Metadata
}

type RemoteJobManager struct {
	jobMode          string
	jobTemplate      string
	jobCmd           string
	monitorCmd       string
	monitorList      []*JobMonitor
	monitorListMutex *sync.Mutex
}

func NewRemoteJobManager(jobMode string) *RemoteJobManager {
	self := &RemoteJobManager{}
	self.jobMode = jobMode
	self.monitorList = []*JobMonitor{}
	self.monitorListMutex = &sync.Mutex{}
	_, _, self.jobCmd, self.monitorCmd, self.jobTemplate = verifyJobManagerFiles(jobMode)
	if len(self.monitorCmd) > 0 {
		self.processMonitorList()
	}
	return self
}

func (self *RemoteJobManager) GetMaxCores() int {
	return 0
}

func (self *RemoteJobManager) GetMaxMemGB() int {
	return 0
}

func (self *RemoteJobManager) execJob(shellCmd string, argv []string, metadata *Metadata,
	threads int, memGB int, fqname string, shellName string) {

	// Sanity check the thread count.
	if threads < 1 {
		threads = 1
	}

	argv = append([]string{shellCmd}, argv...)
	params := map[string]string{
		"JOB_NAME": fqname + "." + shellName,
		"THREADS":  fmt.Sprintf("%d", threads),
		"STDOUT":   metadata.makePath("stdout"),
		"STDERR":   metadata.makePath("stderr"),
		"CMD":      strings.Join(argv, " "),
		"MEM_GB":   "",
	}

	// Only append memory cap if value is sane.
	if memGB > 0 {
		params["MEM_GB"] = fmt.Sprintf("%d", memGB)
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
	metadata.writeRaw("jobscript", r.Replace(template))

	cmd := exec.Command(self.jobCmd, metadata.makePath("jobscript"))
	cmd.Dir = metadata.filesPath
	if output, err := cmd.CombinedOutput(); err != nil {
		metadata.writeRaw("errors", "jobcmd error:\n"+err.Error())
	} else {
		// Get job ID from output and write to metadata file
		outputList := strings.Split(string(output), " ")
		jobId := 0
		jobIdFound := false
		for _, word := range outputList {
			if id, err := strconv.Atoi(word); err == nil {
				jobIdFound = true
				jobId = id
				break
			}
		}
		if jobIdFound {
			metadata.writeRaw("jobid", fmt.Sprintf("%d", jobId))
			if len(self.monitorCmd) > 0 {
				self.monitorListMutex.Lock()
				self.monitorList = append(self.monitorList, &JobMonitor{jobId, metadata})
				self.monitorListMutex.Unlock()
			}
		}
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
			for _, monitor := range monitorList {
				monitorCmd := fmt.Sprintf("%s %d", self.monitorCmd, monitor.jobId)
				monitorCmdParts := strings.Split(monitorCmd, " ")
				cmd := exec.Command(monitorCmdParts[0], monitorCmdParts[1:]...)
				if err := cmd.Run(); err != nil {
					monitor.metadata.writeRaw("errors", "job has been killed:\n"+err.Error())
				} else {
					monitorList = append(monitorList, monitor)
				}
			}
			self.monitorListMutex.Lock()
			self.monitorList = append(self.monitorList, monitorList...)
			self.monitorListMutex.Unlock()
			time.Sleep(time.Minute)
		}
	}()
}

//
// Helper functions for job manager file parsing
//

func verifyJobManagerFiles(jobMode string) (string, map[string]interface{}, string, string, string) {
	jobPath := RelPath(path.Join("..", "jobmanagers"))

	// Check for existence of job manager JSON file
	jobJsonFile := path.Join(jobPath, "config.json")
	if _, err := os.Stat(jobJsonFile); os.IsNotExist(err) {
		LogError(err, "jobmgr", "JobManagerJsonPathError: job config file %s does not exist", jobJsonFile)
		os.Exit(1)
	}
	LogInfo("jobmgr", "Job config: %s", jobJsonFile)
	bytes, _ := ioutil.ReadFile(jobJsonFile)

	// Parse job manager JSON file
	var jobJson map[string]interface{}
	if err := json.Unmarshal(bytes, &jobJson); err != nil {
		LogError(err, "jobmgr", "JobManagerJsonError: job config file %s does not contain valid JSON", jobJsonFile)
		os.Exit(1)
	}

	// Check if job mode is supported by default
	jobCmdsJson, ok := jobJson["jobcmd"]
	if !ok {
		LogInfo("jobmgr", "JobManagerJsonError: job config file %s does not contain 'jobcmd' field", jobJsonFile)
		os.Exit(1)
	}
	jobCmds, ok := jobCmdsJson.(map[string]interface{})
	if !ok {
		LogInfo("jobmgr", "JobManagerJsonError: job config file %s has non-map 'jobcmd' field", jobJsonFile)
		os.Exit(1)
	}
	jobTemplateFile := jobMode
	jobCmd := ""
	val, ok := jobCmds[jobMode]
	if ok {
		if jobCmd, ok = val.(string); !ok {
			LogInfo("jobmgr", "JobManagerJsonError: job config file %s has non-string 'jobcmd[%s]' field", jobJsonFile, jobMode)
			os.Exit(1)
		}
		jobTemplateFile = path.Join(jobPath, fmt.Sprintf("%s.template", jobCmd))
	} else {
		if !strings.HasSuffix(jobTemplateFile, ".template") {
			LogInfo("jobmgr", "JobTemplateFilenameError: job template file %s must have name <jobcmd>.template", jobTemplateFile)
			os.Exit(1)
		}
		jobCmd = strings.Replace(path.Base(jobTemplateFile), ".template", "", 1)
	}
	LogInfo("jobmgr", "Job command: %s", jobCmd)

	// Check for existence of monitor command
	monitorCmdsJson, ok := jobJson["monitorcmd"]
	if !ok {
		LogInfo("jobmgr", "JobManagerJsonError: job config file %s does not contain 'monitorcmd' field", jobJsonFile)
		os.Exit(1)
	}
	monitorCmds, ok := monitorCmdsJson.(map[string]interface{})
	if !ok {
		LogInfo("jobmgr", "JobManagerJsonError: job config file %s has non-map 'monitorcmd' field", jobJsonFile)
		os.Exit(1)
	}
	monitorCmd, ok := monitorCmds[jobCmd].(string)
	if ok {
		monitorCmdParts := strings.Split(monitorCmd, " ")
		LogInfo("jobmgr", "Job monitor command: %s", monitorCmdParts[0])
	}

	// Check for existence of job manager template file
	if _, err := os.Stat(jobTemplateFile); os.IsNotExist(err) {
		LogError(err, "jobmgr", "JobTemplatePathError: job template file %s does not exist", jobTemplateFile)
		os.Exit(1)
	}
	LogInfo("jobmgr", "Job template: %s", jobTemplateFile)
	bytes, _ = ioutil.ReadFile(jobTemplateFile)
	jobTemplate := string(bytes)
	return jobJsonFile, jobJson, jobCmd, monitorCmd, jobTemplate
}

func verifyJobManagerEnv(jobJsonFile string, jobJson map[string]interface{}, jobCmd string, monitorCmd string) {
	// Verify job and monitor commands exist
	incPaths := strings.Split(os.Getenv("PATH"), ":")
	if _, found := searchPaths(jobCmd, incPaths); !found {
		LogInfo("jobmgr", "JobCommandPathError: searched (%s) but job command %s not found '%s'",
			strings.Join(incPaths, ", "), jobCmd)
		os.Exit(1)
	}
	if len(monitorCmd) > 0 {
		monitorCmdPath := strings.Split(monitorCmd, " ")[0]
		if _, found := searchPaths(monitorCmdPath, incPaths); !found {
			LogInfo("jobmgr", "JobMonitorCommandPathError: searched (%s) but job monitor command not found '%s'",
				strings.Join(incPaths, ", "), monitorCmdPath)
			os.Exit(1)
		}
	}

	// Verify environment variables
	val, ok := jobJson["env"]
	if !ok {
		LogInfo("jobmgr", "JobManagerJsonError: job config file %s does not have 'env' field", jobJsonFile)
		os.Exit(1)
	}
	valMap, ok := val.(map[string]interface{})
	if !ok {
		LogInfo("jobmgr", "JobManagerJsonError: job config file %s has non-map 'env' field", jobJsonFile)
		os.Exit(1)
	}
	entries, ok := valMap[jobCmd].([]interface{})
	if !ok {
		// No environment variable check required for job manager
		return
	}

	fields := []string{"name", "description"}
	envs := [][]string{}
	for _, entry := range entries {
		envMap, ok := entry.(map[string]interface{})
		if !ok {
			LogInfo("jobmgr", "JobManagerJsonError: job config file %s has non-map entry in 'env[%s]'", jobJsonFile, jobCmd)
			os.Exit(1)
		}
		env := []string{}
		for _, field := range fields {
			if _, ok := envMap[field]; !ok {
				LogInfo("jobmgr", "JobManagerJsonError: job config file %s does not contain 'env[%s][%s]' field", jobJsonFile, jobCmd, field)
				os.Exit(1)
			}
			if _, ok := envMap[field].(string); !ok {
				LogInfo("jobmgr", "JobManagerJsonError: job config file %s has non-string 'env[%s][%s]' field", jobJsonFile, jobCmd, field)
				os.Exit(1)
			}
			env = append(env, envMap[field].(string))
		}
		envs = append(envs, env)
	}
	EnvRequire(envs, true)
}

func VerifyJobManager(jobMode string) {
	if jobMode == "local" {
		return
	}
	jobJsonFile, jobJson, jobCmd, monitorCmd, _ := verifyJobManagerFiles(jobMode)
	verifyJobManagerEnv(jobJsonFile, jobJson, jobCmd, monitorCmd)
}

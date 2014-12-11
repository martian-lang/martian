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
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry/gosigar"
)

const jobSubmitDelay = 10 // 10 minutes
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

func (self *LocalJobManager) Enqueue(shellCmd string, argv []string, metadata *Metadata, threads int,
	memGB int, fqname string, retries int, waitTime int) {

	time.Sleep(time.Second * time.Duration(waitTime))
	go func() {
		// Exec the shell directly.
		cmd := exec.Command(shellCmd, argv...)

		stdoutPath := metadata.makePath("stdout")
		stderrPath := metadata.makePath("stderr")
		errorsPath := metadata.makePath("errors")

		// Sanity check and cap to self.maxCores.
		if threads < 1 {
			threads = 1
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
			memGB = 1
		}
		if memGB > self.maxMemGB {
			if self.debug {
				LogInfo("jobmngr", "Need %d GB but settling for %d.", memGB,
					self.maxMemGB)
			}
			memGB = self.maxMemGB
		}

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
			if exitCode, ok := getExitCode(err); ok && exitCode == retryExitCode {
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
				ioutil.WriteFile(errorsPath, []byte(err.Error()), 0644)
			} else {
				LogInfo("jobmgr", "Job failed: %s. Retrying job %s in %d seconds", err.Error(), fqname, waitTime)
				self.Enqueue(shellCmd, argv, metadata, threads, memGB, fqname, retries, waitTime)
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

func (self *LocalJobManager) execJob(shellCmd string, argv []string, metadata *Metadata,
	threads int, memGB int, fqname string, shellName string) {
	self.Enqueue(shellCmd, argv, metadata, threads, memGB, fqname, 0, 0)
}

type JobMonitor struct {
	jobId      int
	metadata   *Metadata
	submitTime time.Time
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
		r := regexp.MustCompile("[0-9]+")
		if jobIdString := r.FindString(string(output)); len(jobIdString) > 0 {
			jobId, _ := strconv.Atoi(jobIdString)
			metadata.writeRaw("jobid", jobIdString)
			if len(self.monitorCmd) > 0 {
				self.monitorListMutex.Lock()
				self.monitorList = append(self.monitorList, &JobMonitor{jobId, metadata, time.Now()})
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
				if time.Since(monitor.submitTime) < time.Minute*jobSubmitDelay {
					monitorList = append(monitorList, monitor)
					continue
				}
				monitorCmd := fmt.Sprintf("%s %d", self.monitorCmd, monitor.jobId)
				monitorCmdParts := strings.Split(monitorCmd, " ")
				cmd := exec.Command(monitorCmdParts[0], monitorCmdParts[1:]...)
				if err := cmd.Run(); err != nil {
					monitor.metadata.cache()
					if monitor.metadata.exists("complete") {
						// Job has completed successfully
						continue
					}
					if !monitor.metadata.exists("errors") {
						// Job was killed by cluster resource manager
						monitor.metadata.writeRaw("errors", fmt.Sprintf("Job was killed by %s.",
							self.jobMode))
					}
				} else {
					monitorList = append(monitorList, monitor)
				}
			}
			self.monitorListMutex.Lock()
			self.monitorList = append(self.monitorList, monitorList...)
			self.monitorListMutex.Unlock()
			time.Sleep(time.Minute * time.Duration(5))
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
		LogError(err, "jobmngr", "Job manager config file %s does not exist.", jobJsonFile)
		os.Exit(1)
	}
	LogInfo("jobmngr", "Job config: %s", jobJsonFile)
	bytes, _ := ioutil.ReadFile(jobJsonFile)

	// Parse job manager JSON file
	var jobJson map[string]interface{}
	if err := json.Unmarshal(bytes, &jobJson); err != nil {
		LogError(err, "jobmngr", "Job manager config file %s does not contain valid JSON.", jobJsonFile)
		os.Exit(1)
	}

	// Check if job mode is supported by default
	jobCmdsJson, ok := jobJson["jobcmd"]
	if !ok {
		LogInfo("jobmngr", "Job manager config file %s does not contain 'jobcmd' field.", jobJsonFile)
		os.Exit(1)
	}
	jobCmds, ok := jobCmdsJson.(map[string]interface{})
	if !ok {
		LogInfo("jobmngr", "Job manager config file %s has non-map 'jobcmd' field.", jobJsonFile)
		os.Exit(1)
	}
	jobTemplateFile := jobMode
	jobCmd := ""
	val, ok := jobCmds[jobMode]
	if ok {
		if jobCmd, ok = val.(string); !ok {
			LogInfo("jobmngr", "Job manager config file %s has non-string 'jobcmd[%s]' field.", jobJsonFile, jobMode)
			os.Exit(1)
		}
		jobTemplateFile = path.Join(jobPath, fmt.Sprintf("%s.template", jobCmd))
	} else {
		if !strings.HasSuffix(jobTemplateFile, ".template") {
			LogInfo("jobmngr", "Job manager template file %s must be named <name_of_job_submit_cmd>.template.", jobTemplateFile)
			os.Exit(1)
		}
		jobCmd = strings.Replace(path.Base(jobTemplateFile), ".template", "", 1)
	}
	LogInfo("jobmngr", "Job submit command: %s.", jobCmd)

	// Check for existence of monitor command
	monitorCmdsJson, ok := jobJson["monitorcmd"]
	if !ok {
		LogInfo("jobmngr", "Job manager config file %s does not contain 'monitorcmd' field.", jobJsonFile)
		os.Exit(1)
	}
	monitorCmds, ok := monitorCmdsJson.(map[string]interface{})
	if !ok {
		LogInfo("jobmngr", "Job manager config file %s has non-map 'monitorcmd' field.", jobJsonFile)
		os.Exit(1)
	}
	monitorCmd, ok := monitorCmds[jobCmd].(string)
	if ok {
		monitorCmdParts := strings.Split(monitorCmd, " ")
		LogInfo("jobmngr", "Job monitor command: %s.", monitorCmdParts[0])
	}

	// Check for existence of job manager template file
	if _, err := os.Stat(jobTemplateFile); os.IsNotExist(err) {
		LogError(err, "jobmngr", "Job manager template file %s does not exist.", jobTemplateFile)
		os.Exit(1)
	}
	LogInfo("jobmngr", "Job template: %s.", jobTemplateFile)
	bytes, _ = ioutil.ReadFile(jobTemplateFile)
	jobTemplate := string(bytes)
	return jobJsonFile, jobJson, jobCmd, monitorCmd, jobTemplate
}

func verifyJobManagerEnv(jobJsonFile string, jobJson map[string]interface{}, jobCmd string, monitorCmd string) {
	// Verify job and monitor commands exist
	incPaths := strings.Split(os.Getenv("PATH"), ":")
	if _, found := searchPaths(jobCmd, incPaths); !found {
		LogInfo("jobmngr", "Searched (%s) but job command '%s' not found.",
			strings.Join(incPaths, ", "), jobCmd)
		os.Exit(1)
	}
	if len(monitorCmd) > 0 {
		monitorCmdPath := strings.Split(monitorCmd, " ")[0]
		if _, found := searchPaths(monitorCmdPath, incPaths); !found {
			LogInfo("jobmngr", "JobMonitorCommandPathError: searched (%s) but job monitor command '%s' not found.",
				strings.Join(incPaths, ", "), monitorCmdPath)
			os.Exit(1)
		}
	}

	// Verify environment variables
	val, ok := jobJson["env"]
	if !ok {
		LogInfo("jobmngr", "Job manager config file %s does not have 'env' field.", jobJsonFile)
		os.Exit(1)
	}
	valMap, ok := val.(map[string]interface{})
	if !ok {
		LogInfo("jobmngr", "Job manager config file %s has non-map 'env' field.", jobJsonFile)
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
			LogInfo("jobmngr", "Job manager config file %s has non-map entry in 'env[%s]'.", jobJsonFile, jobCmd)
			os.Exit(1)
		}
		env := []string{}
		for _, field := range fields {
			if _, ok := envMap[field]; !ok {
				LogInfo("jobmngr", "Job manager config file %s does not contain 'env[%s][%s]' field.", jobJsonFile, jobCmd, field)
				os.Exit(1)
			}
			if _, ok := envMap[field].(string); !ok {
				LogInfo("jobmngr", "Job manager config file %s has non-string 'env[%s][%s]' field.", jobJsonFile, jobCmd, field)
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

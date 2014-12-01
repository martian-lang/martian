//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Mario schedulers for local and remote (SGE, LSF, etc) modes.
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
// Schedulers
//
type Scheduler interface {
	execJob(string, string, *Metadata, int, int, string, string, string)
	GetMaxCores() int
	GetMaxMemGB() int
}

type LocalScheduler struct {
	maxCores int
	maxMemGB int
	coreSem  *Semaphore
	memGBSem *Semaphore
	queue    []*exec.Cmd
	debug    bool
}

func NewLocalScheduler(userMaxCores int, userMaxMemGB int, debug bool) *LocalScheduler {
	self := &LocalScheduler{}
	self.debug = debug

	// Set Max number of cores usable at one time.
	if userMaxCores > 0 {
		// If user specified --Maxcores, use that value for Max usable cores.
		self.maxCores = userMaxCores
		LogInfo("schedlr", "Using %d core%s, per --Maxcores option.",
			self.maxCores, Pluralize(self.maxCores))
	} else {
		// Otherwise, set Max usable cores to total number of cores reported
		// by the system.
		self.maxCores = runtime.NumCPU()
		LogInfo("schedlr", "Using %d core%s available on system.",
			self.maxCores, Pluralize(self.maxCores))
	}

	// Set Max GB of memory usable at one time.
	if userMaxMemGB > 0 {
		// If user specified --Maxmem, use that value for Max usable GB.
		self.maxMemGB = userMaxMemGB
		LogInfo("schedlr", "Using %d GB, per --Maxmem option.", self.maxMemGB)
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
		LogInfo("schedlr", "Using %d GB, %d%% of system memory.", self.maxMemGB,
			int(MAXMEM_FRACTION*100))
	}

	self.coreSem = NewSemaphore(self.maxCores)
	self.memGBSem = NewSemaphore(self.maxMemGB)
	self.queue = []*exec.Cmd{}
	return self
}

func (self *LocalScheduler) Enqueue(cmd *exec.Cmd, threads int, memGB int,
	stdoutPath string, stderrPath string, errorsPath string) {

	go func() {
		// Sanity check and cap to self.maxCores.
		if threads < 1 {
			threads = 1
		}
		if threads > self.maxCores {
			if self.debug {
				LogInfo("schedlr", "Need %d core%s but settling for %d.", threads,
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
				LogInfo("schedlr", "Need %d GB but settling for %d.", memGB,
					self.maxMemGB)
			}
			memGB = self.maxMemGB
		}

		// Acquire cores.
		if self.debug {
			LogInfo("schedlr", "Waiting for %d core%s", threads, Pluralize(threads))
		}
		self.coreSem.P(threads)
		if self.debug {
			LogInfo("schedlr", "Acquiring %d core%s (%d/%d in use)", threads,
				Pluralize(threads), self.coreSem.len(), self.maxCores)
		}

		// Acquire memory.
		if self.debug {
			LogInfo("schedlr", "Waiting for %d GB", memGB)
		}
		self.memGBSem.P(memGB)
		if self.debug {
			LogInfo("schedlr", "Acquired %d GB (%d/%d in use)", memGB,
				self.memGBSem.len(), self.maxMemGB)
		}
		if self.debug {
			LogInfo("schedlr", "%d goroutines", runtime.NumGoroutine())
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
			LogInfo("schedlr", "Released %d core%s (%d/%d in use)", threads,
				Pluralize(threads), self.coreSem.len(), self.maxCores)
		}
		// Release memory.
		self.memGBSem.V(memGB)
		if self.debug {
			LogInfo("schedlr", "Released %d GB (%d/%d in use)", memGB,
				self.memGBSem.len(), self.maxMemGB)
		}
	}()
}

func (self *LocalScheduler) GetMaxCores() int {
	return self.maxCores
}

func (self *LocalScheduler) GetMaxMemGB() int {
	return self.maxMemGB
}

func (self *LocalScheduler) execJob(shellCmd string, stagecodePath string, metadata *Metadata,
	threads int, memGB int, profile string, fqname string, shellName string) {

	// Exec the shell directly.
	argv := []string{stagecodePath, metadata.path, metadata.filesPath, profile}
	cmd := exec.Command(shellCmd, argv...)

	// Connect child to _stdout and _stderr metadata files.
	stdoutPath := metadata.makePath("stdout")
	stderrPath := metadata.makePath("stderr")
	errorsPath := metadata.makePath("errors")

	// Enqueue the command to the local scheduler.
	self.Enqueue(cmd, threads, memGB, stdoutPath, stderrPath, errorsPath)
}

type RemoteScheduler struct {
	jobMode           string
	schedulerTemplate string
	schedulerCmd      string
}

func NewRemoteScheduler(jobMode string) *RemoteScheduler {
	self := &RemoteScheduler{}
	self.jobMode = jobMode
	_, self.schedulerCmd, self.schedulerTemplate = verifySchedulerFiles(jobMode)
	return self
}

func (self *RemoteScheduler) GetMaxCores() int {
	return 0
}

func (self *RemoteScheduler) GetMaxMemGB() int {
	return 0
}

func (self *RemoteScheduler) execJob(shellCmd string, stagecodePath string, metadata *Metadata,
	threads int, memGB int, profile string, fqname string, shellName string) {

	// Sanity check the thread count.
	if threads < 1 {
		threads = 1
	}

	argv := []string{shellCmd, stagecodePath, metadata.path, metadata.filesPath, profile}
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
	template := self.schedulerTemplate
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
	metadata.writeRaw("schedscript", r.Replace(template))

	cmd := exec.Command(self.schedulerCmd, metadata.makePath("schedscript"))
	cmd.Dir = metadata.filesPath
	out := ""
	if data, err := cmd.CombinedOutput(); err == nil {
		out = string(data)
	} else {
		out = err.Error()
		metadata.writeRaw("errors", "schedcmd error:\n"+out)
	}
	status := strings.Join(cmd.Args, " ") + "\n\n" + out
	metadata.write("jobinfo", map[string]string{"type": self.jobMode, "status": status})
}

//
// Helper functions for scheduler file parsing
//

func verifySchedulerFiles(jobMode string) (map[string]interface{}, string, string) {
	schedulerPath := RelPath(path.Join("..", "schedulers"))

	// Check for existence of scheduler JSON file
	schedulerJsonFile := path.Join(schedulerPath, "config.json")
	if _, err := os.Stat(schedulerJsonFile); os.IsNotExist(err) {
		LogError(err, "scheduler", "Scheduler config file %s does not exist", schedulerJsonFile)
		os.Exit(1)
	}
	LogInfo("scheduler", "Scheduler config: %s", schedulerJsonFile)
	bytes, _ := ioutil.ReadFile(schedulerJsonFile)

	// Parse scheduler JSON file
	var schedulerJson map[string]interface{}
	if err := json.Unmarshal(bytes, &schedulerJson); err != nil {
		LogError(err, "scheduler", "Scheduler config file %s does not contain valid JSON", schedulerJsonFile)
	}

	// Check if job mode is supported by default
	schedulerCmds := schedulerJson["schedcmd"].(map[string]interface{})
	schedulerTemplateFile := ""
	schedulerCmd := ""
	val, ok := schedulerCmds[jobMode]
	if ok {
		schedulerCmd = val.(string)
		schedulerTemplateFile = path.Join(schedulerPath, fmt.Sprintf("%s.template", schedulerCmd))
	} else {
		schedulerTemplateFile = strings.Replace(jobMode, "~", os.Getenv("HOME"), -1)
		if !strings.HasSuffix(schedulerTemplateFile, ".template") {
			LogInfo("Scheduler template file %s must have name <schedcmd>.template", schedulerTemplateFile)
			os.Exit(1)
		}
		schedulerCmd = strings.Replace(path.Base(schedulerTemplateFile), ".template", "", 1)
	}
	LogInfo("scheduler", "Scheduler command: %s", schedulerCmd)

	// Check for existence of scheduler template file
	if _, err := os.Stat(schedulerTemplateFile); os.IsNotExist(err) {
		LogError(err, "scheduler", "Scheduler template file %s does not exist", schedulerTemplateFile)
		os.Exit(1)
	}
	LogInfo("scheduler", "Scheduler template: %s", schedulerTemplateFile)
	bytes, _ = ioutil.ReadFile(schedulerTemplateFile)
	schedulerTemplate := string(bytes)
	return schedulerJson, schedulerCmd, schedulerTemplate
}

func verifySchedulerEnv(schedulerJson map[string]interface{}, schedulerCmd string) {
	val := schedulerJson["env"].(map[string]interface{})
	entries, ok := val[schedulerCmd].([]interface{})
	if !ok {
		// No environment variable check required for scheduler
		return
	}

	envs := [][]string{}
	for _, entry := range entries {
		env := entry.(map[string]interface{})
		envs = append(envs, []string{env["name"].(string), env["description"].(string)})
	}
	EnvRequire(envs, true)
}

func VerifyScheduler(jobMode string) {
	if jobMode == "local" {
		return
	}
	schedulerJson, schedulerCmd, _ := verifySchedulerFiles(jobMode)
	verifySchedulerEnv(schedulerJson, schedulerCmd)
}

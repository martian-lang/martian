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

	var schedulerJson map[string]interface{}
	schedulerJson, self.schedulerTemplate = verifySchedulerFiles(jobMode)
	self.schedulerCmd = verifySchedulerCmd(schedulerJson)
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
		"job_name": fqname + "." + shellName,
		"threads":  fmt.Sprintf("%d", threads),
		"stdout":   metadata.makePath("stdout"),
		"stderr":   metadata.makePath("stderr"),
		"cmd":      strings.Join(argv, " "),
		"mem_gb":   "",
	}

	// Only append memory cap if value is sane.
	if memGB > 0 {
		params["mem_gb"] = fmt.Sprintf("%d", memGB)
	}

	// Replace template annotations with actual values
	args := []string{}
	template := self.schedulerTemplate
	for key, val := range params {
		if len(val) > 0 {
			args = append(args, fmt.Sprintf("<%s>", key), val)
		} else {
			// Remove line containing parameter from template
			for _, line := range strings.Split(template, "\n") {
				if strings.Contains(line, fmt.Sprintf("<%s>", key)) {
					template = strings.Replace(template, line, "", 1)
				}
			}
		}
	}
	r := strings.NewReplacer(args...)
	metadata.writeRaw("exec", r.Replace(template))
	metadata.write("jobinfo", map[string]string{"type": self.jobMode})

	// Exec scheduler command synchronously and write result out to _schedcmd.
	cmd := exec.Command(self.schedulerCmd, metadata.makePath("exec"))
	cmd.Dir = metadata.filesPath
	out := ""
	if data, err := cmd.CombinedOutput(); err == nil {
		out = string(data)
	} else {
		out = err.Error()
		metadata.writeRaw("errors", "schedcmd error:\n"+out)
	}
	metadata.writeRaw("schedcmd", strings.Join(cmd.Args, " ")+"\n\n"+out)
}

//
// Helper functions for scheduler file parsing
//

func verifySchedulerFiles(jobMode string) (map[string]interface{}, string) {
	schedulerPath := RelPath(path.Join("..", "schedulers", jobMode))

	// Check for existence of scheduler JSON file
	schedulerJsonFile := path.Join(schedulerPath, fmt.Sprintf("%s.json", jobMode))
	if _, err := os.Stat(schedulerJsonFile); os.IsNotExist(err) {
		LogError(err, "scheduler", "Scheduler JSON file %s does not exist", schedulerJsonFile)
		os.Exit(1)
	}
	LogInfo("scheduler", "Scheduler JSON: %s", schedulerJsonFile)
	bytes, _ := ioutil.ReadFile(schedulerJsonFile)

	// Parse scheduler JSON file
	var schedulerJson map[string]interface{}
	if err := json.Unmarshal(bytes, &schedulerJson); err != nil {
		LogError(err, "scheduler", "Scheduler json file %s does not contain valid JSON", schedulerJsonFile)
	}

	// Check for existence of scheduler template file
	schedulerTemplateFile := path.Join(schedulerPath, fmt.Sprintf("%s.template", jobMode))
	if _, err := os.Stat(schedulerTemplateFile); os.IsNotExist(err) {
		LogError(err, "scheduler", fmt.Sprintf("Scheduler template file %s does not exist", schedulerTemplateFile))
		os.Exit(1)
	}
	LogInfo("scheduler", "Scheduler template: %s", schedulerTemplateFile)
	bytes, _ = ioutil.ReadFile(schedulerTemplateFile)
	schedulerTemplate := string(bytes)
	return schedulerJson, schedulerTemplate
}

func verifySchedulerCmd(schedulerJson map[string]interface{}) string {
	val, ok := schedulerJson["schedcmd"]
	if !ok {
		LogInfo("scheduler", "Scheduler JSON does not contain 'schedcmd' field")
		os.Exit(1)
	}
	if _, ok = val.(string); !ok {
		LogInfo("scheduler", "Scheduler JSON 'schedcmd' field has a non-string value")
		os.Exit(1)
	}
	return val.(string)
}

func verifySchedulerEnv(schedulerJson map[string]interface{}) {
	val, ok := schedulerJson["env"]
	if !ok {
		LogInfo("scheduler", "Scheduler JSON does not contain 'env' field")
		os.Exit(1)
	}
	if _, ok = val.([]interface{}); !ok {
		LogInfo("scheduler", "Scheduler JSON 'env' field is not an array")
		os.Exit(1)
	}

	envs := [][]string{}
	for _, entry := range val.([]interface{}) {
		var envMap map[string]interface{}
		envMap, ok = entry.(map[string]interface{})
		if !ok {
			LogInfo("scheduler", "Scheduler JSON 'env' field has a non-dictionary entry")
			os.Exit(1)
		}
		env := []string{}
		fields := []string{"name", "description"}
		for _, field := range fields {
			if _, ok = envMap[field]; !ok {
				LogInfo("scheduler", "Scheduler JSON 'env' field has a dictionary entry without '%s' field", field)
				os.Exit(1)
			}
			env = append(env, envMap[field].(string))
		}
		envs = append(envs, env)
	}
	EnvRequire(envs, true)
}

func VerifyScheduler(jobMode string) {
	if jobMode == "local" {
		return
	}
	schedulerJson, _ := verifySchedulerFiles(jobMode)
	verifySchedulerCmd(schedulerJson)
	verifySchedulerEnv(schedulerJson)
}

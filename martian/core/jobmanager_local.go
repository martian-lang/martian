// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/trace"
	"strings"
	"syscall"
	"time"

	sigar "github.com/cloudfoundry/gosigar"
	"github.com/martian-lang/martian/martian/util"
)

const maxRetries = 5
const retryExitCode = 513

// The following constants are used to estimate process (thread) counts.
// In local mode, these are used to estimate the number of processes spawned,
// to avoid bumping into the process ulimit.  This is all very rough approximation.

// The approximate number of physical threads assumed used by the user at
// startup, including those used by mrp.  The actual value is used as well in
// real time, but this amount is reserved in the semaphore regardless.
const startingThreadCount = 45

// The base number of threads assumed per job.  This includes the threads used
// by the mrjob process and whatever threads the spawned process uses for
// runtime management.  Very approximate since it depends on many details of
// the stage code langauge and implementation.  The number of threads reserved
// by the job will be added to this number.
const procsPerJob = 15

type LocalJobManager struct {
	maxCores    int
	maxMemGB    int
	jobSettings *JobManagerSettings
	coreSem     *ResourceSemaphore
	memMBSem    *ResourceSemaphore
	procsSem    *ResourceSemaphore
	lastMemDiff int64
	queue       []*exec.Cmd
	debug       bool
	limitLoad   bool
	highMem     ObservedMemory
}

func NewLocalJobManager(userMaxCores int, userMaxMemGB int,
	debug bool, limitLoadavg bool, clusterMode bool,
	config *JobManagerJson) *LocalJobManager {
	self := &LocalJobManager{
		debug:     debug,
		limitLoad: limitLoadavg,
	}
	self.jobSettings = verifyJobManager("local", config, -1).jobSettings

	// Set Max number of cores usable at one time.
	if userMaxCores > 0 {
		// If user specified --localcores, use that value for Max usable cores.
		self.maxCores = userMaxCores
		util.LogInfo("jobmngr", "Using %d core%s, per --localcores option.",
			self.maxCores, util.Pluralize(self.maxCores))
	} else {
		if clusterMode {
			self.maxCores = self.jobSettings.ThreadsPerJob
		} else {
			// Otherwise, set Max usable cores to total number of cores reported
			// by the system.
			self.maxCores = runtime.NumCPU()
			util.LogInfo("jobmngr", "Using %d logical core%s available on system.",
				self.maxCores, util.Pluralize(self.maxCores))
		}
	}

	sysMem := sigar.Mem{}
	sysMem.Get()
	// Set Max GB of memory usable at one time.
	if userMaxMemGB > 0 {
		// If user specified --localmem, use that value for Max usable GB.
		self.maxMemGB = userMaxMemGB
		util.LogInfo("jobmngr", "Using %d GB, per --localmem option.", self.maxMemGB)
	} else {
		// Otherwise, set Max usable GB to MAXMEM_FRACTION * GB of total
		// memory reported by the system.
		MAXMEM_FRACTION := 0.9
		if clusterMode {
			sysMemGB := int((sysMem.ActualFree + (1024*1024 - 1)) / (1024 * 1024 * 1024))
			if self.jobSettings.MemGBPerJob < sysMemGB {
				sysMemGB = self.jobSettings.MemGBPerJob
			}
			if sysMemGB < 1 {
				sysMemGB = 1
			}
			self.maxMemGB = sysMemGB
			if sysMemGB < self.jobSettings.MemGBPerJob {
				util.PrintInfo("jobmngr",
					"WARNING: Using %d GB for local jobs.  Recommended free memory is %d GB",
					self.maxMemGB, self.jobSettings.MemGBPerJob)
			} else {
				util.LogInfo("jobmngr", "Using %d GB for local jobs.", self.maxMemGB)
			}
		} else {
			sysMemGB := int(float64(sysMem.Total) * MAXMEM_FRACTION / 1073741824)
			// Set floor to 1GB.
			if sysMemGB < 1 {
				sysMemGB = 1
			}
			self.maxMemGB = sysMemGB
			util.LogInfo("jobmngr", "Using %d GB, %d%% of system memory.", self.maxMemGB,
				int(MAXMEM_FRACTION*100))
		}
	}

	if uint64(self.maxMemGB*1024) > (sysMem.ActualFree+(1024*1024-1))/(1024*1024) {
		util.PrintInfo("jobmngr",
			"WARNING: configured to use %dGB of local memory, but only %.1fGB is currently available.",
			self.maxMemGB, float64(sysMem.ActualFree+(1024*1024-1))/(1024*1024*1024))
	}

	self.coreSem = NewResourceSemaphore(int64(self.maxCores), "threads")
	self.memMBSem = NewResourceSemaphore(int64(self.maxMemGB)*1024, "MB of memory")
	if rlim, err := GetMaxProcs(); err != nil {
		util.LogError(err, "jobmngr",
			"WARNING: Could not get process rlimit.")
	} else if int64(rlim.Max) > startingThreadCount &&
		int64(rlim.Cur) > startingThreadCount {
		self.procsSem = NewResourceSemaphore(int64(rlim.Max), "processes")
		self.procsSem.Acquire(startingThreadCount)

		if userProcs, err := GetUserProcessCount(); err != nil {
			self.procsSem.UpdateSize(int64(rlim.Cur))
		} else {
			self.procsSem.UpdateFreeUsed(
				int64(rlim.Cur)-int64(userProcs),
				startingThreadCount)
		}
		if self.procsSem.Available()/(procsPerJob+1) < int64(self.maxCores) {
			if rlim.Max > rlim.Cur {
				util.PrintInfo("jobmngr",
					"WARNING: The current process count limit %d is low. To increase parallelism, set ulimit -u %d.",
					rlim.Cur, rlim.Max)
			} else {
				util.PrintInfo("jobmngr",
					"WARNING: The current process count limit %d is low. Contact your system administrator to increase it.",
					rlim.Cur)
			}
		}
	}
	self.queue = []*exec.Cmd{}
	util.RegisterSignalHandler(self)
	return self
}

func (self *LocalJobManager) GetSettings() *JobManagerSettings {
	return self.jobSettings
}

func (self *LocalJobManager) refreshResources(localMode bool) error {
	sysMem := sigar.Mem{}
	if err := sysMem.Get(); err != nil {
		return err
	}
	usedMem, err := GetProcessTreeMemory(os.Getpid(), false, nil)
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
				float64(usedMem.Rss)/(1024*1024*1024), float64(self.memMBSem.Reserved())/1024)
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
	if self.procsSem != nil {
		if rlim, err := GetMaxProcs(); err != nil {
			return err
		} else if userProcs, err := GetUserProcessCount(); err != nil {
			return err
		} else {
			self.procsSem.UpdateFreeUsed(
				int64(rlim.Cur)-int64(userProcs),
				int64(usedMem.Procs)+startingThreadCount)
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
	if threads == 0 {
		threads = self.jobSettings.ThreadsPerJob
	} else if threads < 0 {
		threads = self.maxCores
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
			"jobmngr",
			"Job asked for %d GB but is being given %d.  This behavior is deprecated - jobs which can adapt their memory usage should ask for -%d.",
			memGB, self.maxMemGB, memGB)
		memGB = self.maxMemGB
	}

	return threads, memGB
}

func (self *LocalJobManager) checkQueue(ids []string, _ context.Context) ([]string, string) {
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
		r := trace.StartRegion(context.Background(), "queueLocal")
		defer r.End()
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
			metadata.WriteRaw(Errors, err.Error())
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
			self.coreSem.Release(int64(threads))
			metadata.WriteRaw(Errors, err.Error())
			return
		}
		if self.debug {
			util.LogInfo("jobmngr", "Acquired %d GB (%.1f/%d in use)", memGB,
				float64(self.memMBSem.InUse())/1024, self.maxMemGB)
		}
		if self.debug {
			util.LogInfo("jobmngr", "%d goroutines", runtime.NumGoroutine())
		}

		procEstimate := int64(procsPerJob + threads)
		if self.procsSem != nil {
			// Acquire processes
			if self.debug {
				util.LogInfo("jobmngr", "Waiting for %d processes", memGB)
			}
			if err := self.procsSem.Acquire(procEstimate); err != nil {
				util.LogError(err, "jobmngr",
					"%s estimated to require %d processes, but the process ulimit is %d.",
					metadata.fqname, procEstimate, self.procsSem.CurrentSize())
				self.coreSem.Release(int64(threads))
				self.memMBSem.Release(int64(memGB) * 1024)
				metadata.WriteRaw(Errors, err.Error())
				return
			}
			if self.debug {
				util.LogInfo("jobmngr", "Acquired %d processes (%d/%d in use)",
					procEstimate, self.procsSem.InUse(), self.procsSem.CurrentSize())
			}
			if self.debug {
				util.LogInfo("jobmngr", "%d goroutines", runtime.NumGoroutine())
			}
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
		cmd.SysProcAttr = util.Pdeathsig(&syscall.SysProcAttr{}, syscall.SIGTERM)
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
				if _, err2 := metadata.readRawSafe(Errors); os.IsNotExist(err2) {
					// Only write _errors if the job didn't write one before
					// failing.  Because this is local mode, we don't need to
					// worry about nfs data races.
					metadata.WriteRaw(Errors, err.Error())
				}
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
		if self.procsSem != nil {
			// Release processes.
			self.procsSem.Release(procEstimate)
			if self.debug {
				util.LogInfo("jobmngr", "Released %d processes (%d/%d in use)",
					procEstimate, self.procsSem.InUse(), self.procsSem.CurrentSize())
			}
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

func (self *LocalJobManager) endJob(*Metadata) {}

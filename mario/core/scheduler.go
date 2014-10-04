//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario scheduler for local mode.
//
package core

import (
	"github.com/cloudfoundry/gosigar"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
)

type semaphore chan bool

func (s semaphore) P(n int) {
	for i := 0; i < n; i++ {
		s <- true
	}
}

func (s semaphore) V(n int) {
	for i := 0; i < n; i++ {
		<-s
	}
}

type Scheduler struct {
	maxCores int
	maxMemGB int
	coreSem  semaphore
	memGBSem semaphore
	queue    []*exec.Cmd
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func NewScheduler(userMaxCores int, userMaxMemGB int) *Scheduler {
	self := &Scheduler{}

	// Set max number of cores usable at one time.
	if userMaxCores > 0 {
		// If user specified --maxcores, use that value for max usable cores.
		self.maxCores = userMaxCores
		LogInfo("schedlr", "Using %d core%s, per --maxcores option.",
			self.maxCores, pluralize(self.maxCores))
	} else {
		// Otherwise, set max usable cores to total number of cores reported
		// by the system.
		self.maxCores = runtime.NumCPU()
		LogInfo("schedlr", "Using %d core%s available on system.",
			self.maxCores, pluralize(self.maxCores))
	}

	// Set max GB of memory usable at one time.
	if userMaxMemGB > 0 {
		// If user specified --maxmem, use that value for max usable GB.
		self.maxMemGB = userMaxMemGB
		LogInfo("schedlr", "Using %d GB, per --maxmem option.", self.maxMemGB)
	} else {
		// Otherwise, set max usable GB to MAXMEM_FRACTION * GB of total
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

	self.coreSem = make(semaphore, self.maxCores)
	self.memGBSem = make(semaphore, self.maxMemGB)
	self.queue = []*exec.Cmd{}
	return self
}

func (self *Scheduler) Enqueue(cmd *exec.Cmd, threads int, memGB int,
	stdoutFile *os.File, stderrFile *os.File, errorsPath string) {

	go func() {
		log := false // convenience flag for toggling debug logging

		defer stdoutFile.Close()
		defer stderrFile.Close()

		// Sanity check and cap to self.maxCores.
		if threads < 1 {
			threads = 1
		}
		if threads > self.maxCores {
			if log {
				LogInfo("schedlr", "Need %d core%s but settling for %d.", threads,
					pluralize(threads), self.maxCores)
			}
			threads = self.maxCores
		}

		// Sanity check and cap to self.maxMemGB.
		if memGB < 1 {
			memGB = 1
		}
		if memGB > self.maxMemGB {
			if log {
				LogInfo("schedlr", "Need %d GB but settling for %d.", memGB,
					self.maxMemGB)
			}
			memGB = self.maxMemGB
		}

		// Acquire cores.
		if log {
			LogInfo("schedlr", "Waiting for %d core%s.", threads, pluralize(threads))
		}
		self.coreSem.P(threads)
		if log {
			LogInfo("schedlr", "Acquiring %d core%s (%d/%d in use).", threads,
				pluralize(threads), len(self.coreSem), self.maxCores)
		}

		// Acquire memory.
		if log {
			LogInfo("schedlr", "Waiting for %d GB.", memGB)
		}
		self.memGBSem.P(memGB)
		if log {
			LogInfo("schedlr", "Acquiring %d GB (%d/%d in use).", memGB,
				len(self.memGBSem), self.maxMemGB)
		}

		if err := cmd.Start(); err != nil {
			ioutil.WriteFile(errorsPath, []byte(err.Error()), 0600)
		} else {
			if err := cmd.Wait(); err != nil {
				ioutil.WriteFile(errorsPath, []byte(err.Error()), 0600)
			}
		}

		self.coreSem.V(threads)
		if log {
			LogInfo("schedlr", "Releasing %d core%s (%d/%d in use).", threads,
				pluralize(threads), len(self.coreSem), self.maxCores)
		}
		self.memGBSem.V(memGB)
		if log {
			LogInfo("schedlr", "Releasing %d GB (%d/%d in use).", memGB,
				len(self.memGBSem), self.maxMemGB)
		}
	}()
}

func (self *Scheduler) getMaxCores() int {
	return self.maxCores
}

func (self *Scheduler) getMaxMemGB() int {
	return self.maxMemGB
}

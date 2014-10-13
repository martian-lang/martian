//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario scheduler for local mode.
//
package core

import (
	"fmt"
	"github.com/cloudfoundry/gosigar"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
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
	debug    bool
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func NewScheduler(userMaxCores int, userMaxMemGB int, debug bool) *Scheduler {
	self := &Scheduler{}
	self.debug = debug

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

func countOpenFiles() int {
	out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("lsof -p %v", os.Getpid())).Output()
	if err != nil {
		return -1
	}
	lines := strings.Split(string(out), "\n")
	return len(lines) - 1
}

func (self *Scheduler) Enqueue(cmd *exec.Cmd, threads int, memGB int,
	stdoutPath string, stderrPath string, errorsPath string) {

	go func() {
		// Sanity check and cap to self.maxCores.
		if threads < 1 {
			threads = 1
		}
		if threads > self.maxCores {
			if self.debug {
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
			if self.debug {
				LogInfo("schedlr", "Need %d GB but settling for %d.", memGB,
					self.maxMemGB)
			}
			memGB = self.maxMemGB
		}

		// Acquire cores.
		if self.debug {
			LogInfo("schedlr", "Waiting for %d core%s.", threads, pluralize(threads))
		}
		self.coreSem.P(threads)
		if self.debug {
			LogInfo("schedlr", "Acquiring %d core%s (%d/%d in use).", threads,
				pluralize(threads), len(self.coreSem), self.maxCores)
		}

		// Acquire memory.
		if self.debug {
			LogInfo("schedlr", "Waiting for %d GB.", memGB)
		}
		self.memGBSem.P(memGB)
		if self.debug {
			LogInfo("schedlr", "Acquiring %d GB (%d/%d in use).", memGB,
				len(self.memGBSem), self.maxMemGB)
		}

		// Set up _stdout and _stderr for the job.
		if self.debug {
			fmt.Printf("%d open files\n", countOpenFiles())
		}
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
		if self.debug {
			fmt.Printf("%d open files\n", countOpenFiles())
			defer fmt.Printf("%d open files\n", countOpenFiles())
		}

		// Run the command and wait for completion.
		if err := cmd.Start(); err != nil {
			ioutil.WriteFile(errorsPath, []byte(err.Error()), 0600)
		} else {
			if err := cmd.Wait(); err != nil {
				ioutil.WriteFile(errorsPath, []byte(err.Error()), 0600)
			}
		}

		// Release cores.
		self.coreSem.V(threads)
		if self.debug {
			LogInfo("schedlr", "Releasing %d core%s (%d/%d in use).", threads,
				pluralize(threads), len(self.coreSem), self.maxCores)
		}
		// Release memory.
		self.memGBSem.V(memGB)
		if self.debug {
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

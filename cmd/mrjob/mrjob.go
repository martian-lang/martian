//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

// Command mrjob manages process lifetimes for Martian stage code.
//
// Also collects various performance statistics.
package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/shlex"
	"github.com/martian-lang/martian/martian/core"
	"github.com/martian-lang/martian/martian/util"
)

const HeartbeatInterval = time.Minute * 2
const MemorySampleInterval = time.Second * 5

// Sample memory usage with much higher frequency when monitoring.
const MonitorMemorySampleInterval = time.Second * 1

type runner struct {
	job         *exec.Cmd
	log         *os.File
	errorReader *os.File
	highMem     core.ObservedMemory
	ioStats     *core.IoStatsBuilder
	metadata    *core.Metadata
	runType     string
	jobInfo     *core.JobInfo
	monitoring  bool
	start       time.Time
	isDone      chan struct{}
	perfDone    <-chan struct{}
}

func main() {
	util.SetupSignalHandlers()
	if len(os.Args) < 6 {
		panic("Insufficient arguments.\n" +
			"Expected: mrjob <exe> [exe args...] <split|main|join> " +
			"<metadata_path> <files_path> <journal_prefix>")
	}
	args := os.Args[len(os.Args)-4:]
	runType := args[0]
	metadataPath := args[1]
	filesPath := args[2]
	fqname := path.Base(args[3])
	journalPath := path.Dir(args[3])

	if os.Getenv("MRO_SELF_PROFILE") != "" {
		startCpuProfile(metadataPath)
	}

	run := runner{
		ioStats:  core.NewIoStatsBuilder(),
		metadata: core.NewMetadataRunWithJournalPath(fqname, metadataPath, filesPath, journalPath, runType),
		runType:  runType,
		start:    time.Now(),
	}
	util.RegisterSignalHandler(&run)
	if log, err := os.OpenFile(run.metadata.MetadataFilePath(core.LogFile),
		os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
		run.Fail(err, "Can't open log file.")
	} else {
		run.log = log
		util.LogTeeWriter(log)
		defer run.log.Close()
	}
	run.metadata.UpdateJournal(core.StdOut)
	run.metadata.UpdateJournal(core.StdErr)

	run.Init()
	if err := run.StartJob(os.Args[1:]); err != nil {
		run.Fail(err, "Error starting job.")
	}
	run.isDone = make(chan struct{})
	run.WaitLoop()
}

// If we're running a CPU self-profile, this is the handle to it.
var selfProfile *os.File

func startCpuProfile(metadataPath string) {
	// This isn't going through the  usual metadata API because we want
	// the profile to include construction of that.
	f, err := os.Create(path.Join(metadataPath, "_selfProfile.pprof"))
	if err != nil {
		util.PrintError(err, "profile", "Error recording CPU profile")
		return
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		util.PrintError(err, "profile", "Error recording CPU profile")
		return
	}
	selfProfile = f
}

func (self *runner) Init() {
	// In case the job template was wrong, set the working directory now.
	if err := os.Chdir(self.metadata.FilesPath()); err != nil {
		self.Fail(err, "Could not change to the correct working directory")
	}
	self.writeJobinfo()
	util.LogInfo("time", "__start__")
	if jErr := self.metadata.UpdateJournal(core.LogFile); jErr != nil {
		util.PrintError(jErr, "monitor",
			"Could not update log journal file.  Continuing, hoping for the best.")
	}
	core.CheckMaxVmem(
		uint64(self.jobInfo.VMemGB*1024*1024) * 1024)
	self.setRlimit()
	if cgLim, cgSoftLim, _ := util.GetCgroupMemoryLimit(); cgLim > 0 {
		if cgLim < int64(math.Ceil(self.jobInfo.MemGB*(1024*1024*1024))) {
			util.LogInfo("monitor",
				"WARNING: cgroup memory limit of %d bytes is less than the requested %g GB",
				cgLim, self.jobInfo.MemGB)
		} else {
			util.LogInfo("monitor",
				"cgroup memory limit of %d bytes detected", cgLim)
		}
		if cgSoftLim != 0 && cgSoftLim < int64(math.Ceil(self.jobInfo.MemGB*1024*1024*1024)) {
			util.LogInfo("monitor",
				"WARNING: cgroup soft memory limit of %d bytes is less than the requested %g GB",
				cgSoftLim, self.jobInfo.MemGB)
		}
	}
	setSubreaper()
}

func getClusterEnv() map[string]string {
	re := regexp.MustCompile("^(?:EGO|SGE|LS[BF]|PBS|SLURM|JOB)_[^O]")
	captures := make(map[string]string)
	for _, env := range os.Environ() {
		sep := strings.Index(env, "=")
		if sep > 0 && re.MatchString(env[:sep]) {
			captures[env[:sep]] = env[sep+1:]
		}
	}
	if len(captures) == 0 {
		return nil
	} else {
		return captures
	}
}

func (self *runner) writeJobinfo() {
	jobInfo := new(core.JobInfo)
	if err := self.metadata.ReadInto(core.JobInfoFile, jobInfo); err != nil {
		self.Fail(err, "Error reading jobInfo.")
	} else {
		self.jobInfo = jobInfo
		self.monitoring = jobInfo.Monitor == "monitor"
	}
	self.jobInfo.Cwd = self.metadata.FilesPath()
	self.jobInfo.Host, _ = os.Hostname()
	self.jobInfo.Pid = os.Getpid()
	self.jobInfo.ClusterEnv = getClusterEnv()
	if err := self.metadata.WriteAtomic(core.JobInfoFile, self.jobInfo); err != nil {
		self.Fail(err, "Could not write updated jobInfo.")
	}
}

func (self *runner) setRlimit() {
	if err := core.MaximizeMaxFiles(); err != nil {
		util.PrintError(err, "monitor", "Error setting the file rlimit.")
		return
	}
}

func (self *runner) done() {
	util.LogInfo("time", "__end__")
	// refresh jobInfo if possible, but if we can't that's ok.
	self.metadata.ReadInto(core.JobInfoFile, self.jobInfo)
	if self.jobInfo != nil {
		end := time.Now()
		self.jobInfo.WallClockInfo = &core.WallClockInfo{
			Start:    core.WallClockTime(self.start),
			End:      core.WallClockTime(end),
			Duration: end.Sub(self.start).Seconds(),
		}
		if waitChildren() {
			if !reportChildren() {
				// waitChildren detected that there were remaining child
				// processes, but reportChildren wasn't able to report them for
				// whatever reason.
				util.LogInfo("monitor",
					"Orphaned child processes detected, which did not terminate.")
			}
		}
		self.jobInfo.RusageInfo = core.GetRusage()
		if !self.highMem.IsZero() {
			self.jobInfo.MemoryUsage = &self.highMem
		}
		self.jobInfo.IoStats = &self.ioStats.IoStats
		if err := self.metadata.WriteAtomic(core.JobInfoFile, self.jobInfo); err != nil {
			util.PrintError(err, "monitor", "Could not write final jobInfo.")
		} else {
			self.metadata.UpdateJournal(core.JobInfoFile)
		}
	}
}

func (self *runner) Fail(err error, message string) {
	self.done()
	errStr := err.Error()
	target := core.Errors
	if _, ok := err.(*stageReturnedError); !ok {
		errStr = fmt.Sprintf("%s\n\n%s\n", message, err.Error())
		fmt.Fprint(os.Stderr, errStr)
	} else {
		if strings.HasPrefix(errStr, "ASSERT:") {
			errStr = errStr[len("ASSERT:"):]
			target = core.Assert
		}
	}
	if writeError := self.metadata.WriteRaw(target, errStr); writeError != nil {
		util.PrintError(writeError, "monitor", "Could not write errors file.")
	}
	if jErr := self.metadata.UpdateJournal(target); jErr != nil {
		util.PrintError(jErr, "monitor", "Could not update %v journal file.", target)
	}
	self.waitForPerf()
	os.Exit(0)
}

// Wait for up to 15 seconds after the stage code terminates for perf record to
// terminate (if applicable).  Otherwise some cluster managers might kill perf
// as soon as the head process for the job (mrjob, in this case) terminates.
func (self *runner) waitForPerf() {
	if c := self.perfDone; c != nil {
		select {
		case <-c:
		case <-time.After(15 * time.Second):
		}
	}
	if selfProfile != nil {
		pprof.StopCPUProfile()
		if err := selfProfile.Close(); err != nil {
			util.PrintError(err, "profile", "Error closing cpu profile")
		}
	}
}

func totalCpu(ru *core.RusageInfo) float64 {
	if ru == nil {
		return 0
	}
	var total float64
	if ru.Self != nil {
		total += ru.Self.UserTime + ru.Self.SystemTime
	}
	if ru.Children != nil {
		total += ru.Children.UserTime + ru.Children.SystemTime
	}
	return total
}

func (self *runner) Complete() {
	self.done()
	target := core.CompleteFile
	if self.monitoring && self.jobInfo.RusageInfo != nil {
		if t := time.Since(self.start); t > time.Minute*15 {
			if threads := totalCpu(self.jobInfo.RusageInfo) /
				t.Seconds(); threads > 1.5*self.jobInfo.Threads {
				target = core.Errors
				if writeError := self.metadata.WriteRaw(target, fmt.Sprintf(
					"Stage exceeded its threads quota (using %.1f, allowed %g)",
					threads,
					self.jobInfo.Threads)); writeError != nil {
					util.PrintError(writeError, "monitor",
						"Could not write errors file.")
				}
			}
		}
		if target != core.Errors {
			kb := int(math.Ceil(self.jobInfo.MemGB * 1024 * 1024))
			if self.jobInfo.RusageInfo.Children.MaxRss > kb {
				target = core.Errors
				if writeError := self.metadata.WriteRaw(target, fmt.Sprintf(
					"Stage exceeded its memory quota (using %.1f, allowed %g)",
					float64(self.jobInfo.RusageInfo.Children.MaxRss)/(1024*1024),
					self.jobInfo.MemGB)); writeError != nil {
					util.PrintError(writeError, "monitor",
						"Could not write errors file.")
				}
			}
		}
	}
	if target == core.CompleteFile {
		if writeError := self.metadata.WriteTime(core.CompleteFile); writeError != nil {
			util.PrintError(writeError, "monitor", "Could not write complete file.")
		}
	}
	self.sync()
	if jErr := self.metadata.UpdateJournal(target); jErr != nil {
		util.PrintError(jErr, "monitor", "Could not update %v journal file.", target)
	}
	self.waitForPerf()
	os.Exit(0)
}

func (self *runner) sync() {
	if self.runType == "split" {
		syncFile(self.metadata.MetadataFilePath(core.StageDefsFile))
	} else {
		syncFile(self.metadata.MetadataFilePath(core.OutsFile))
	}
	syncFile(path.Dir(self.metadata.FilePath("nil")))
	syncFile(path.Dir(self.metadata.MetadataFilePath(core.CompleteFile)))
}

func (self *runner) makeErrorPipe() (*os.File, error) {
	var err error
	var writer *os.File
	self.errorReader, writer, err = os.Pipe()
	return writer, err
}

func (self *runner) StartJob(args []string) error {
	cmd := exec.Command(args[0], args[1:]...)
	if writer, err := self.makeErrorPipe(); err != nil {
		return err
	} else {
		cmd.ExtraFiles = []*os.File{self.log, writer}
		defer writer.Close()
	}
	// We really don't want the child outliving the parent.
	cmd.SysProcAttr = util.Pdeathsig(&syscall.SysProcAttr{}, syscall.SIGKILL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if pc := self.jobInfo.ProfileConfig; pc != nil && len(pc.Env) > 0 {
		cmd.Env = pc.MakeEnv(
			self.metadata.MetadataFilePath(core.PerfData),
			self.metadata.MetadataFilePath(core.ProfileOut))
	}
	if err := func(cmd *exec.Cmd) error {
		if self.monitoring && self.jobInfo.VMemGB > 0 {
			// Exclude mrjob's vmem usage from the rlimit.
			mem, _ := core.GetProcessTreeMemory(self.jobInfo.Pid, true, nil)
			amount := int64(self.jobInfo.VMemGB)*1024*1024*1024 - mem.Vmem
			if amount < mem.Vmem+1024*1024 {
				amount = mem.Vmem + 1024*1024
			}
			if oldAmount, err := core.SetVMemRLimit(uint64(amount)); err != nil {
				util.LogError(err, "monitor",
					"Could not set VM rlimit.")
			} else {
				// After launching the subprocess, restore the vmem
				// limit for this process.  Otherwise the go runtime can run
				//  into various kinds of trouble.
				defer func(amt uint64) {
					if _, err := core.SetVMemRLimit(amt); err != nil {
						util.LogError(err, "monitor",
							"Could not restore VM rlimit.")
					}
				}(oldAmount)
			}
		}
		if err := func() error {
			util.EnterCriticalSection()
			defer util.ExitCriticalSection()
			self.job = cmd
			return self.job.Start()
		}(); err != nil {
			self.errorReader.Close()
			return err
		}
		return nil
	}(cmd); err != nil {
		return err
	}
	if err := self.startProfile(); err != nil {
		util.PrintError(err, "monitor", "Could not start profiling.")
	}
	return nil
}

func (self *runner) startProfile() error {
	var cmd *exec.Cmd
	var journaledFiles []core.MetadataFileName
	if perfArgs := os.Getenv("MRO_PERF_ARGS"); perfArgs != "" &&
		self.jobInfo.ProfileMode == core.PerfRecordProfile {
		// For backwards compatibility, ignore the custom config.
		journaledFiles = []core.MetadataFileName{core.PerfData}
		if args, err := shlex.Split(perfArgs); err != nil {
			util.PrintError(err, "profile", "Error parsing perf args")
			return nil
		} else {
			baseArgs := []string{
				"record",
				"-p", strconv.Itoa(self.job.Process.Pid),
				"-o", self.metadata.MetadataFilePath(journaledFiles[0]),
			}
			cmd = exec.Command("perf", append(baseArgs, args...)...)
		}
	} else if pc := self.jobInfo.ProfileConfig; pc == nil || pc.Command == "" {
		return nil
	} else {
		cmd = exec.Command(pc.Command, pc.ExpandedArgs(
			self.metadata.MetadataFilePath(core.PerfData),
			self.metadata.MetadataFilePath(core.ProfileOut),
			self.job.Process.Pid)...)
	}
	cmd.SysProcAttr = util.Pdeathsig(&syscall.SysProcAttr{}, syscall.SIGINT)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	} else {
		perfDone := make(chan struct{})
		self.perfDone = perfDone
		go func(cmd *exec.Cmd, c chan<- struct{}) {
			cmd.Wait()
			close(c)
		}(cmd, perfDone)
		for _, file := range journaledFiles {
			self.metadata.UpdateJournal(file)
		}
		return nil
	}
}

func (self *runner) HandleSignal(sig os.Signal) {
	util.PrintInfo("monitor", "Caught signal %v", sig)
	cmd := self.job
	if cmd != nil {
		proc := cmd.Process
		if proc != nil {
			proc.Kill()
		}
	}
	if c := self.isDone; c != nil {
		t := time.NewTimer(time.Second * 5)
		// Wait up to 5 seconds for the child process to terminate.
		select {
		case <-t.C:
		case <-c:
		}
		t.Stop()
	}
	self.done()
	if err := self.metadata.WriteRaw(core.Errors, fmt.Sprintf("Caught signal %v", sig)); err != nil {
		util.PrintError(err, "monitor", "Could not write errors file.")
	}
	if err := self.metadata.UpdateJournal(core.Errors); err != nil {
		util.PrintError(err, "monitor", "Could not update error journal file.")
	}
}

// Reads at most n bytes from the reader, returning when either n bytes are read
// or the end of the reader is reached.  Errors are ignored.
func readBytes(n int, reader *os.File) []byte {
	if n <= 0 {
		panic("Cannot read non-positive number of bytes!")
	}
	result := make([]byte, n)
	cursor := 0
	for {
		lastRead, err := reader.Read(result[cursor:])
		if lastRead > 0 {
			cursor += lastRead
		}
		if err != nil || lastRead <= 0 || cursor >= n {
			return result[:cursor]
		}
	}
}

// This error contains a string written by the stage code.  It is already
// formatted, so when this is seen, any additional message is ignored.
type stageReturnedError struct {
	message string
}

func (self *stageReturnedError) Error() string {
	return self.message
}

// Returns true if sig is a signal which we expect is not due to a
// bug in the stage code.
func externalSignal(sig syscall.Signal) bool {
	for _, handled := range util.HANDLED_SIGNALS {
		if sig == handled {
			return true
		}
	}
	// SIGKLL isn't in the handled set because it can't be handled, but
	// should be treated equivalently to SIGTERM for these purposes.
	if sig == syscall.SIGKILL {
		return true
	}
	return false
}

// Convert an exec.ExitError to a stageReturnedError if the failure was due to
// one of the signals that we choose to handle.  This allows restart logic to
// work correctly.
func sigToErr(err error) error {
	if err == nil {
		return err
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if state, ok := exitErr.Sys().(*syscall.WaitStatus); ok &&
			state.Signaled() && externalSignal(state.Signal()) {
			return &stageReturnedError{
				message: fmt.Sprintf(
					"stage code received signal: %v", state.Signal()),
			}
		}
	}
	return err
}

// Wait for the process to complete or, if monitoring is enabled, for it to
// exceed its memory quota.
func (self *runner) WaitLoop() {
	wait := make(chan error, 1)
	go func(wait chan<- error) {
		defer func(wait chan<- error) {
			close(wait)
		}(wait)
		errorBytes := readBytes(8100, self.errorReader)
		if len(errorBytes) > 0 {
			// If the job has finished, we want to wait on it so it isn't
			// a zombie while we do our cleanup, and also so that its rusage
			// gets reported.  However, if it doesn't die we don't want to
			// block our own exit.  Under most circumstances the process will
			// have already terminated by the time we get here.
			go func() {
				self.job.Wait()
				close(self.isDone)
			}()
			wait <- &stageReturnedError{message: string(errorBytes)}
		} else {
			close(self.isDone)
			wait <- sigToErr(self.job.Wait())
		}
	}(wait)
	err := func(wait <-chan error) error {
		defer self.errorReader.Close()
		// Make sure we record at least one memory high-water mark, even
		// for short stages.
		self.getChildMemGB()
		lastHeartbeat := time.Now()
		// Do the first memory sample after just 500ms, to capture information
		// about very short stages.
		timer := time.NewTimer(time.Millisecond * 500)
		defer timer.Stop()
		for {
			select {
			case err := <-wait:
				return err
			case <-timer.C:
				// Minimize parent process impact on memory stats, and
				// prevent mrjob from using too many resources for polling.
				runtime.GC()
				if err := self.monitor(&lastHeartbeat); err != nil {
					return err
				}
			}
			// Don't start a new timer going if we're already done.
			select {
			case err := <-wait:
				return err
			default:
			}
			if self.monitoring {
				timer.Reset(MonitorMemorySampleInterval)
			} else {
				timer.Reset(MemorySampleInterval)
			}
		}
	}(wait)
	{
		// Wait up to 5 seconds for the job to finish, to ensure we get rusage.
		select {
		case <-time.After(time.Second * 5):
			self.job.Process.Signal(syscall.SIGKILL)
		case <-self.isDone:
		}
	}
	util.EnterCriticalSection()
	defer util.ExitCriticalSection()
	if err != nil {
		self.Fail(err, "Job failed in stage code")
	} else {
		self.Complete()
	}
}

func (self *runner) getChildMemGB() (rss, vmem float64) {
	proc := self.job.Process
	if proc == nil {
		return 0, 0
	}
	io := make(map[int]*core.IoAmount)
	mem, err := core.GetProcessTreeMemory(proc.Pid, true, io)
	if selfMem, err := core.GetRunningMemory(self.jobInfo.Pid); err == nil {
		// Do this rather than just calling core.GetProcessTreeMemory,
		// above, because we don't want to include the profiling child
		// process (if any).
		mem.Add(selfMem)
	}
	mem.IncreaseRusage(core.GetRusage())
	self.highMem.IncreaseTo(mem)
	if err != nil {
		util.LogError(err, "monitor",
			"Error updating job statistics. Final statistics may not be accurite.")
	} else {
		self.ioStats.Update(io, time.Now())
	}
	return float64(mem.Rss) / (1024 * 1024 * 1024),
		float64(mem.Vmem) / (1024 * 1024 * 1024)
}

func (self *runner) logProcessTree() {
	tree, _ := core.GetProcessTreeMemoryList(os.Getpid())
	if len(tree) > 0 {
		util.LogInfo("monitor", "Process tree:\n%s",
			tree.Format("       "))
	}
}

func (self *runner) monitor(lastHeartbeat *time.Time) error {
	if rss, vmem := self.getChildMemGB(); rss > float64(self.jobInfo.MemGB) {
		self.logProcessTree()
		if self.monitoring {
			if proc := self.job.Process; proc != nil {
				tree, _ := core.GetProcessTreeMemoryList(proc.Pid)
				if len(tree) > 0 {
					util.LogInfo("monitor", "Process tree:\n%s",
						tree.Format("       "))
				}
			}
			self.job.Process.Kill()
			return fmt.Errorf(
				"Stage exceeded its memory quota (using %.1f, allowed %gG)",
				rss, self.jobInfo.MemGB)
		} else {
			util.LogInfo("monitor",
				"Stage exceeded its memory quota (using %.1f, allowed %gG)",
				rss, self.jobInfo.MemGB)
		}
	} else if self.jobInfo.VMemGB > 0 && vmem > float64(self.jobInfo.VMemGB) {
		self.logProcessTree()
		if self.monitoring {
			self.job.Process.Kill()
			return fmt.Errorf(
				"Stage exceeded its address space quota (using %.1f, allowed %gG)",
				vmem, self.jobInfo.VMemGB)
		} else {
			util.LogInfo("monitor",
				"Stage exceeded its address space quota (using %.1f, allowed %gG)",
				vmem, self.jobInfo.VMemGB)
		}
	}
	if time.Since(*lastHeartbeat) > HeartbeatInterval {
		if err := self.metadata.UpdateJournal(core.Heartbeat); err != nil {
			util.PrintError(err, "monitor", "Could not write heartbeat.")
		} else {
			*lastHeartbeat = time.Now()
		}
		if _, err := os.Stat(self.metadata.MetadataFilePath(core.LogFile)); os.IsNotExist(err) {
			self.job.Process.Kill()
			return fmt.Errorf("Stage log file has been deleted.  Aborting run.\n" +
				"  This is usually the result of `mrp` thinking the stage failed\n" +
				"  and deleting the stage directory in order to retry.")
		}
	}
	return nil
}

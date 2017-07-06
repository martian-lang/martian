//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian runtime. This is where the action happens.
//
package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"
)

const heartbeatTimeout = 60 // 60 minutes

//=============================================================================
// Metadata
//=============================================================================
type Metadata struct {
	fqname        string
	path          string
	contents      map[string]bool
	readCache     map[string]interface{}
	filesPath     string
	journalPath   string
	lastRefresh   time.Time
	lastHeartbeat time.Time
	mutex         *sync.Mutex

	// If non-zero the job was not found last time the job manager was queried,
	// the chunk will be failed out if the state seems like it's still running
	// after the job manager's grace period has elapsed.
	notRunningSince time.Time
}

type MetadataInfo struct {
	Path  string   `json:"path"`
	Names []string `json:"names"`
}

func NewMetadata(fqname string, p string) *Metadata {
	self := &Metadata{}
	self.fqname = fqname
	self.path = p
	self.contents = map[string]bool{}
	self.readCache = make(map[string]interface{})
	self.filesPath = path.Join(p, "files")
	self.journalPath = ""
	self.mutex = &sync.Mutex{}
	return self
}

func NewMetadataWithJournalPath(fqname string, p string, journalPath string) *Metadata {
	self := NewMetadata(fqname, p)
	self.journalPath = journalPath
	return self
}

func (self *Metadata) glob() []string {
	paths, _ := filepath.Glob(path.Join(self.path, "_*"))
	return paths
}

func (self *Metadata) enumerateFiles() ([]string, error) {
	return filepath.Glob(path.Join(self.filesPath, "*"))
}

func (self *Metadata) mkdirs() error {
	if err := mkdir(self.path); err != nil {
		msg := fmt.Sprintf("Could not create directories for %s: %s", self.fqname, err.Error())
		LogError(err, "runtime", msg)
		self.writeRaw("errors", msg)
		return err
	}
	if err := mkdir(self.filesPath); err != nil {
		msg := fmt.Sprintf("Could not create directories for %s: %s", self.fqname, err.Error())
		LogError(err, "runtime", msg)
		self.writeRaw("errors", msg)
		return err
	}
	return nil
}

func (self *Metadata) removeAll() error {
	self.mutex.Lock()
	if len(self.contents) > 0 {
		self.contents = map[string]bool{}
	}
	if len(self.readCache) > 0 {
		self.readCache = make(map[string]interface{})
	}
	self.notRunningSince = time.Time{}
	self.lastRefresh = time.Time{}
	self.mutex.Unlock()
	if err := os.RemoveAll(self.path); err != nil {
		return err
	}
	return os.RemoveAll(self.filesPath)
}

// Must be called within a lock.
func (self *Metadata) _getStateNoLock(name string) (string, bool) {
	if self._existsNoLock("errors") {
		return "failed", true
	}
	if self._existsNoLock("assert") {
		return "failed", true
	}
	if self._existsNoLock("complete") {
		if self._existsNoLock("jobid") {
			self._removeNoLock("jobid")
		}
		return name + "complete", true
	}
	if self._existsNoLock("log") {
		return name + "running", true
	}
	if self._existsNoLock("jobinfo") {
		return name + "queued", true
	}
	return "", false
}

func (self *Metadata) getState(name string) (string, bool) {
	self.mutex.Lock()
	state, ok := self._getStateNoLock(name)
	self.mutex.Unlock()
	return state, ok
}

func (self *Metadata) _cacheNoLock(name string) {
	self.contents[name] = true
	// cache is usually called on write or update
	delete(self.readCache, name)
}

func (self *Metadata) cache(name string) {
	self.mutex.Lock()
	self._cacheNoLock(name)
	self.mutex.Unlock()
}

func (self *Metadata) _uncacheNoLock(name string) {
	delete(self.contents, name)
	delete(self.readCache, name)
}

func (self *Metadata) uncache(name string) {
	self.mutex.Lock()
	self._uncacheNoLock(name)
	self.mutex.Unlock()
}

func (self *Metadata) loadCache() {
	paths := self.glob()
	self.mutex.Lock()
	if len(self.contents) > 0 {
		self.contents = map[string]bool{}
	}
	if len(self.readCache) > 0 {
		self.readCache = make(map[string]interface{})
	}
	for _, p := range paths {
		self.contents[path.Base(p)[1:]] = true
	}
	self.notRunningSince = time.Time{}
	self.lastRefresh = time.Time{}
	self.mutex.Unlock()
}

func (self *Metadata) makePath(name string) string {
	return path.Join(self.path, "_"+name)
}

func (self *Metadata) _existsNoLock(name string) bool {
	_, ok := self.contents[name]
	return ok
}

func (self *Metadata) exists(name string) bool {
	self.mutex.Lock()
	ok := self._existsNoLock(name)
	self.mutex.Unlock()
	return ok
}

func (self *Metadata) readRawSafe(name string) (string, error) {
	bytes, err := ioutil.ReadFile(self.makePath(name))
	return string(bytes), err
}

func (self *Metadata) readRaw(name string) string {
	s, _ := self.readRawSafe(name)
	return s
}

func (self *Metadata) readFromCache(name string) (interface{}, bool) {
	self.mutex.Lock()
	i, ok := self.readCache[name]
	self.mutex.Unlock()
	return i, ok
}

func (self *Metadata) saveToCache(name string, value interface{}) {
	self.mutex.Lock()
	self.readCache[name] = value
	self.mutex.Unlock()
}

func (self *Metadata) read(name string) interface{} {
	v, ok := self.readFromCache(name)
	if ok {
		return v
	}
	str, err := self.readRawSafe(name)
	json.Unmarshal([]byte(str), &v)
	if err != nil {
		self.saveToCache(name, v)
	}
	return v
}
func (self *Metadata) _writeRawNoLock(name string, text string) error {
	err := ioutil.WriteFile(self.makePath(name), []byte(text), 0644)
	self._cacheNoLock(name)
	if err != nil {
		msg := fmt.Sprintf("Could not write %s for %s: %s", name, self.fqname, err.Error())
		LogError(err, "runtime", msg)
		if name != "errors" {
			self._writeRawNoLock("errors", msg)
		}
	}
	return err
}
func (self *Metadata) writeRaw(name string, text string) error {
	err := ioutil.WriteFile(self.makePath(name), []byte(text), 0644)
	self.cache(name)
	if err != nil {
		msg := fmt.Sprintf("Could not write %s for %s: %s", name, self.fqname, err.Error())
		LogError(err, "runtime", msg)
		if name != "errors" {
			self.writeRaw("errors", msg)
		}
	}
	return err
}
func (self *Metadata) write(name string, object interface{}) error {
	bytes, _ := json.MarshalIndent(object, "", "    ")
	return self.writeRaw(name, string(bytes))
}
func (self *Metadata) writeTime(name string) error {
	return self.writeRaw(name, Timestamp())
}
func (self *Metadata) remove(name string) {
	self.uncache(name)
	os.Remove(self.makePath(name))
}
func (self *Metadata) _removeNoLock(name string) {
	self._uncacheNoLock(name)
	os.Remove(self.makePath(name))
}

func (self *Metadata) clearReadCache() {
	self.mutex.Lock()
	if len(self.readCache) > 0 {
		self.readCache = make(map[string]interface{})
	}
	self.mutex.Unlock()
}

func (self *Metadata) resetHeartbeat() {
	self.lastHeartbeat = time.Time{}
}

// After a metadata refresh scan has completed, this is called.  If
// notRuningSince was before the given time, which should be the start of the
// refresh cycle minus the configured queue query grace period, then the
// pipestance should be marked failed.
func (self *Metadata) endRefresh(lastRefresh time.Time) {
	self.mutex.Lock()
	self.lastRefresh = lastRefresh
	if !self.notRunningSince.IsZero() && self.notRunningSince.Before(lastRefresh) {
		notRunningSince := self.notRunningSince
		self.notRunningSince = time.Time{}
		if state, _ := self._getStateNoLock(""); state == "running" || state == "queued" {
			// The job is not running but the metadata thinks it still is.
			// The check for metadata updates was completed since the time that
			// the queue query completed.  This job has failed.  Write an error.
			self._writeRawNoLock("errors", fmt.Sprintf(
				"According to the job manager, the job for %s was not queued "+
					"or running, since at least %s.",
				self.fqname, notRunningSince.Format(TIMEFMT)))
		}
	}
	self.mutex.Unlock()
}

// Mark a job as possibly failed if it is not running.
//
// In case the metadata was reset between when the query began and when it
// ended, the job is marked as failed only if the jobid matches what was
// queried and the job has not already completed.  The actual error is not
// written until the next time the pipestance run loop has a chance to refresh
// the metadata, as it's possible the job completed between the last check for
// metadata updates and when the query completed.
func (self *Metadata) failNotRunning(jobid string) {
	if !self.exists("jobid") {
		return
	}
	if st, _ := self.getState(""); st != "running" && st != "queued" {
		return
	}
	self.mutex.Lock()
	if !self.notRunningSince.IsZero() {
		self.mutex.Unlock()
		return
	}
	if self.readRaw("jobid") != jobid {
		self.mutex.Unlock()
		return
	}
	// Double-check that the job wasn't reset while jobid was being read.
	if !self._existsNoLock("jobid") {
		self.mutex.Unlock()
		return
	}
	self.notRunningSince = time.Now()
	self.mutex.Unlock()
}

func (self *Metadata) checkedReset() error {
	self.mutex.Lock()
	if state, _ := self._getStateNoLock(""); state == "failed" {
		if len(self.contents) > 0 {
			self.contents = make(map[string]bool)
		}
		self.mutex.Unlock()
		if err := self.uncheckedReset(); err == nil {
			PrintInfo("runtime", "(reset-partial)   %s", self.fqname)
		} else {
			return err
		}
	} else {
		self.mutex.Unlock()
	}
	return nil
}

func (self *Metadata) uncheckedReset() error {
	// Remove all related files from journal directory.
	if len(self.journalPath) > 0 {
		if files, err := filepath.Glob(path.Join(self.journalPath, self.fqname+"*")); err == nil {
			for _, file := range files {
				os.Remove(file)
			}
		}
	}
	if err := self.removeAll(); err != nil {
		PrintInfo("runtime", "Cannot reset the stage because some folder contents could not be deleted.\n\nPlease resolve this error in order to continue running the pipeline: %v", err)
		return err
	}
	return self.mkdirs()
}

// Resets the metadata if the state was queued, but the job manager had not yet
// started the job locally or queued it remotely.
func (self *Metadata) restartQueuedLocal() error {
	if self.exists("queued_locally") {
		if err := self.uncheckedReset(); err == nil {
			PrintInfo("runtime", "(reset-running)   %s", self.fqname)
			return nil
		} else {
			return err
		}
	}
	return nil
}

// Resets the metadata if the state was queued, or if the state was running and
// the pid is not a process that is still running.  This is to handle cases of
// pipelines running in local mode when MRP is killed and restarted, so all
// queued jobs are no longer actually queued, and running jobs MAY have been
// killed as well (if mrp was killed by ctrl-C or by a job manager that killed
// the entire process group).  It may miss cases where a PID was reused, but
// the heartbeat will catch those cases eventually and in the ideal case the
// adaptor should have written an error anyway unless it was a SIGKILL.
func (self *Metadata) restartLocal() error {
	state, ok := self.getState("")
	if !ok {
		return nil
	}
	if state == "queued" {
		if err := self.uncheckedReset(); err == nil {
			PrintInfo("runtime", "(reset-queued)    %s", self.fqname)
		} else {
			return err
		}
	} else if state == "running" {
		data := self.readRaw("jobinfo")

		var jobInfo *JobInfo
		if err := json.Unmarshal([]byte(data), &jobInfo); err == nil &&
			jobInfo.Pid != 0 {
			if proc, err := os.FindProcess(jobInfo.Pid); err == nil && proc != nil {
				// From man 2 kill: If sig is 0, then no signal is sent, but error
				// checking is still performed; this can be used to check for the
				// existence of a process ID or process group ID.
				// If sending signal 0 to the process returns an error, either the
				// process is not running or it is owned by another user, which we
				// can assume means the PID was reused.
				if err := proc.Signal(syscall.Signal(0)); err != nil {
					if err := self.uncheckedReset(); err == nil {
						PrintInfo("runtime", "(reset-running)   %s", self.fqname)
					} else {
						return err
					}
				} else {
					PrintInfo("runtime", "Possibly running  %s", self.fqname)
				}
			}
		}
	}
	return nil
}

func (self *Metadata) checkHeartbeat() {
	if state, _ := self.getState(""); state == "running" {
		if self.lastHeartbeat.IsZero() || self.exists("heartbeat") {
			self.uncache("heartbeat")
			self.lastHeartbeat = time.Now()
		}
		if self.lastRefresh.Sub(self.lastHeartbeat) > time.Minute*heartbeatTimeout {
			self.writeRaw("errors", fmt.Sprintf(
				"%s: No heartbeat detected for %d minutes. Assuming job has failed. This may be "+
					"due to a user manually terminating the job, or the operating system or cluster "+
					"terminating it due to resource or time limits.",
				Timestamp(), heartbeatTimeout))
		}
	}
}

func (self *Metadata) serializeState() *MetadataInfo {
	names := []string{}
	self.mutex.Lock()
	for content := range self.contents {
		names = append(names, content)
	}
	self.mutex.Unlock()
	sort.Strings(names)
	return &MetadataInfo{
		Path:  self.path,
		Names: names,
	}
}

func (self *Metadata) serializePerf(numThreads int) *PerfInfo {
	if self.exists("complete") && self.exists("jobinfo") {
		data := self.readRaw("jobinfo")

		var jobInfo *JobInfo
		if err := json.Unmarshal([]byte(data), &jobInfo); err == nil {
			fpaths, _ := self.enumerateFiles()
			return reduceJobInfo(jobInfo, fpaths, numThreads)
		}
	}
	return nil
}

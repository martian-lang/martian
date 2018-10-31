// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

// Martian runtime. This is where the action happens.

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

const heartbeatTimeout = 60 // 60 minutes

type MetadataFileName string

const AnyFile MetadataFileName = "*"
const (
	AlarmFile      MetadataFileName = "alarm"
	ArgsFile       MetadataFileName = "args"
	Assert         MetadataFileName = "assert"
	ChunkDefsFile  MetadataFileName = "chunk_defs"
	ChunkOutsFile  MetadataFileName = "chunk_outs"
	CompleteFile   MetadataFileName = "complete"
	Errors         MetadataFileName = "errors"
	FinalState     MetadataFileName = "finalstate"
	Heartbeat      MetadataFileName = "heartbeat"
	InvocationFile MetadataFileName = "invocation"
	JobId          MetadataFileName = "jobid"
	JobInfoFile    MetadataFileName = "jobinfo"
	JobModeFile    MetadataFileName = "jobmode"
	Lock           MetadataFileName = "lock"
	LogFile        MetadataFileName = "log"
	MetadataZip    MetadataFileName = "metadata.zip"
	MroSourceFile  MetadataFileName = "mrosource"
	OutsFile       MetadataFileName = "outs"
	Perf           MetadataFileName = "perf"
	PerfData       MetadataFileName = "perf.data"
	ProfileOut     MetadataFileName = "profile.out"
	ProgressFile   MetadataFileName = "progress"
	QueuedLocally  MetadataFileName = "queued_locally"
	Stackvars      MetadataFileName = "stackvars"
	StageDefsFile  MetadataFileName = "stage_defs"
	StdErr         MetadataFileName = "stderr"
	StdOut         MetadataFileName = "stdout"
	TagsFile       MetadataFileName = "tags"
	TimestampFile  MetadataFileName = "timestamp"
	UiPort         MetadataFileName = "uiport"
	UuidFile       MetadataFileName = "uuid"
	VdrKill        MetadataFileName = "vdrkill"
	PartialVdr     MetadataFileName = "vdrkill.partial"
	VersionsFile   MetadataFileName = "versions"
	DisabledFile   MetadataFileName = "disabled"
)

const MetadataFilePrefix string = "_"

func (self MetadataFileName) FileName() string {
	return MetadataFilePrefix + string(self)
}

func metadataFileNameFromPath(p string) MetadataFileName {
	return MetadataFileName(path.Base(p)[len(MetadataFilePrefix):])
}

type MetadataState string

const (
	Complete      MetadataState = "complete"
	Failed        MetadataState = "failed"
	DisabledState MetadataState = "disabled"
	Running       MetadataState = "running"
	Queued        MetadataState = "queued"
	Ready         MetadataState = "ready"
	Waiting       MetadataState = ""
	ForkWaiting   MetadataState = "waiting"
)

const (
	ChunksPrefix  = "chunks_"
	SplitPrefix   = "split_"
	JoinPrefix    = "join_"
	CleanupPrefix = "cleanup_"
	RetryPrefix   = "retry_"
)

func (self MetadataState) Prefixed(prefix string) MetadataState {
	return MetadataState(string(prefix) + string(self))
}

func (self MetadataState) HasPrefix(prefix string) bool {
	return strings.HasPrefix(string(self), prefix)
}

func (self MetadataState) IsRunning() bool {
	return strings.HasSuffix(string(self), string(Running))
}

func (self MetadataState) IsQueued() bool {
	return strings.HasSuffix(string(self), string(Queued))
}

func (self MetadataState) IsFailed() bool {
	return strings.HasSuffix(string(self), string(Failed))
}

func makeUniquifier() string {
	// Take the low 16 bits worth of the pid, and the low 24 bits
	// (~6 months) of the unix time.
	trimmedTime := uint32(time.Now().Unix()) & ((^uint32(0)) >> 8)
	return fmt.Sprintf("%04x%06x", uint16(os.Getpid()), trimmedTime)
}

//=============================================================================
// Metadata
//=============================================================================

// Manages interatction with the filesystem-based "database" of Martian
// metadata for a pipeline node (pipeline, subpipeline, fork, stage, split,
// chunk, join).
type Metadata struct {
	fqname        string
	path          string
	finalPath     string
	contents      map[MetadataFileName]bool
	readCache     map[MetadataFileName]LazyArgumentMap
	curFilesPath  string
	finalFilePath string
	journalPath   string
	lastRefresh   time.Time
	lastHeartbeat time.Time
	mutex         sync.Mutex
	uniquifier    string

	// A prefix to attach when writing journal file name.
	// Empty for chunks, or SplitPrefix or JoinPrefix.
	journalPrefix string

	// If non-zero the job was not found last time the job manager was queried,
	// the chunk will be failed out if the state seems like it's still running
	// after the job manager's grace period has elapsed.
	notRunningSince time.Time
}

// Basic exportable information from a metadata object.
type MetadataInfo struct {
	// The filesystem path containing the metadata files.
	Path string `json:"path"`

	// The metadata file names which exist for this object.
	Names []string `json:"names"`
}

func NewMetadata(fqname string, p string) *Metadata {
	return &Metadata{
		fqname:        fqname,
		path:          p,
		finalPath:     p,
		contents:      make(map[MetadataFileName]bool),
		readCache:     make(map[MetadataFileName]LazyArgumentMap),
		curFilesPath:  path.Join(p, "files"),
		finalFilePath: path.Join(p, "files"),
	}
}

func NewMetadataRunWithJournalPath(fqname string, p string, filesPath string, journalPath string, runType string) *Metadata {
	self := NewMetadataWithJournalPath(fqname, p, journalPath)
	self.curFilesPath = filesPath
	self.finalFilePath = filesPath
	if runType != "main" {
		self.journalPrefix = runType + "_"
	}
	return self
}

func NewMetadataWithJournalPath(fqname string, p string, journalPath string) *Metadata {
	self := NewMetadata(fqname, p)
	self.journalPath = journalPath
	return self
}

func (self *Metadata) glob() []string {
	paths, _ := filepath.Glob(path.Join(self.path, AnyFile.FileName()))
	return paths
}

// Gets the locations of the symlinks pointing to uniquified directories.
func (self *Metadata) symlinks() []string {
	var symlinks []string
	if self.path != self.finalPath {
		if info, err := os.Lstat(self.finalPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
			symlinks = []string{self.finalPath}
		}
		if self.finalFilePath != path.Join(self.finalPath, "files") {
			if info, err := os.Lstat(self.finalFilePath); err == nil && info.Mode()&os.ModeSymlink != 0 {
				symlinks = append(symlinks, self.finalFilePath)
			}
		}
	}
	return symlinks
}

func (self *Metadata) enumerateFiles() ([]string, error) {
	return filepath.Glob(path.Join(self.curFilesPath, "*"))
}

func (self *Metadata) enumerateTemp() ([]string, error) {
	if td := self.TempDir(); td == "" {
		return nil, nil
	} else {
		return filepath.Glob(path.Join(td, "*"))
	}
}

// The path containing output files for this node.
func (self *Metadata) FilesPath() string {
	return self.curFilesPath
}

func (self *Metadata) TempDir() string {
	if p := self.path; p != "" {
		return path.Join(p, "tmp")
	} else {
		return ""
	}
}

func (self *Metadata) mkdirs() error {
	if err := util.Mkdir(self.path); err != nil {
		msg := fmt.Sprintf("Could not create directories for %s: %s", self.fqname, err.Error())
		util.LogError(err, "runtime", msg)
		self.WriteRaw(Errors, msg)
		return err
	}
	if err := util.Mkdir(self.curFilesPath); err != nil {
		msg := fmt.Sprintf("Could not create directories for %s: %s", self.fqname, err.Error())
		util.LogError(err, "runtime", msg)
		self.WriteRaw(Errors, msg)
		return err
	}
	return nil
}

func (self *Metadata) uniquify() error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.uniquifier == "" {
		self.uniquifier = makeUniquifier()
	}
	p := self.finalPath + "-u" + self.uniquifier
	if err := util.Mkdir(p); err != nil {
		msg := fmt.Sprintf("Could not create directories for %s: %s", self.fqname, err.Error())
		util.LogError(err, "runtime", msg)
		self._writeRawNoLock(Errors, msg)
		return err
	}
	self.path = p
	filesPath := path.Join(p, "files")
	if err := util.Mkdir(filesPath); err != nil {
		msg := fmt.Sprintf("Could not create file directory for %s: %s", self.fqname, err.Error())
		util.LogError(err, "runtime", msg)
		self._writeRawNoLock(Errors, msg)
		os.Remove(p)
		return err
	}
	self.curFilesPath = filesPath
	if err := util.Mkdir(self.TempDir()); err != nil {
		msg := fmt.Sprintf("Could not create temp directory for %s: %s", self.fqname, err.Error())
		util.LogError(err, "runtime", msg)
		self._writeRawNoLock(Errors, msg)
		os.Remove(filesPath)
		os.Remove(p)
		return err
	}

	if self.discoverUniquifier() != self.uniquifier {
		if relPath, err := filepath.Rel(filepath.Dir(self.finalPath), p); err != nil {
			msg := fmt.Sprintf("Could not compute relative path for %s to %s", self.finalPath, p)
			util.LogError(err, "runtime", msg)
			self._writeRawNoLock(Errors, msg)
			os.RemoveAll(p)
			return err

		} else {
			if err := os.Symlink(relPath, self.finalPath); err != nil {
				msg := fmt.Sprintf("Could not create symlink for %s: %s", self.fqname, err.Error())
				util.LogError(err, "runtime", msg)
				self._writeRawNoLock(Errors, msg)
				os.RemoveAll(p)
				return err
			}
		}
		if self.finalFilePath != path.Join(self.finalPath, "files") {
			if err := os.Remove(self.finalFilePath); err != nil && !os.IsNotExist(err) {
				msg := fmt.Sprintf("Could not remove existing directory for %s: %s", self.fqname, err.Error())
				util.LogError(err, "runtime", msg)
				self._writeRawNoLock(Errors, msg)
				os.RemoveAll(p)
				return err
			}
			if relPath, err := filepath.Rel(filepath.Dir(self.finalFilePath), filesPath); err != nil {
				msg := fmt.Sprintf("Could not compute relative path for %s to %s", self.finalFilePath, filesPath)
				util.LogError(err, "runtime", msg)
				self._writeRawNoLock(Errors, msg)
				os.RemoveAll(p)
				return err

			} else {
				if err := os.Symlink(relPath, self.finalFilePath); err != nil {
					msg := fmt.Sprintf("Could not create files symlink for %s: %s", self.fqname, err.Error())
					util.LogError(err, "runtime", msg)
					self._writeRawNoLock(Errors, msg)
					os.RemoveAll(p)
					return err
				}
			}
		}
	}
	return nil
}

func (self *Metadata) removeAll() error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if len(self.contents) > 0 {
		self.contents = make(map[MetadataFileName]bool)
	}
	if len(self.readCache) > 0 {
		self.readCache = make(map[MetadataFileName]LazyArgumentMap)
	}
	self.notRunningSince = time.Time{}
	self.lastRefresh = time.Time{}
	if err := os.RemoveAll(self.curFilesPath); err != nil {
		return err
	}
	if err := os.RemoveAll(self.path); err != nil {
		return err
	}
	// Remove final directories iff they're symlinks or empty.  If a
	// successful run wrote to it then we don't want to delete it.
	if self.finalFilePath != self.curFilesPath {
		os.Remove(self.finalFilePath)
	}
	if self.finalPath != self.path {
		os.Remove(self.finalPath)
	}
	return nil
}

// Must be called within a lock.
func (self *Metadata) _getStateNoLock() (MetadataState, bool) {
	if self._existsNoLock(Errors) {
		return Failed, true
	}
	if self._existsNoLock(Assert) {
		return Failed, true
	}
	if self._existsNoLock(CompleteFile) {
		if self._existsNoLock(JobId) {
			self._removeNoLock(JobId)
		}
		return Complete, true
	}
	if self._existsNoLock(DisabledFile) {
		return DisabledState, true
	}
	if self._existsNoLock(LogFile) {
		return Running, true
	}
	if self._existsNoLock(JobInfoFile) {
		return Queued, true
	}
	return Waiting, false
}

func (self *Metadata) getState() (MetadataState, bool) {
	self.mutex.Lock()
	state, ok := self._getStateNoLock()
	self.mutex.Unlock()
	return state, ok
}

func (self *Metadata) _cacheNoLock(name MetadataFileName) {
	self.contents[name] = true
	// cache is usually called on write or update
	delete(self.readCache, name)
}

func (self *Metadata) cache(name MetadataFileName, uniquifier string) {
	self.mutex.Lock()
	if self.uniquifier == uniquifier {
		self._cacheNoLock(name)
	} else if self.uniquifier != "" {
		util.LogInfo("runtime",
			"There appears to be more than one instance of %s running (Saw ID '%s', expected '%s').",
			self.fqname, uniquifier, self.uniquifier)
	}
	self.mutex.Unlock()
}

func (self *Metadata) _uncacheNoLock(name MetadataFileName) {
	delete(self.contents, name)
	delete(self.readCache, name)
}

func (self *Metadata) uncache(name MetadataFileName) {
	self.mutex.Lock()
	self._uncacheNoLock(name)
	self.mutex.Unlock()
}

// Attempt to determine if this metadata object was already
// uniquified and reset paths appropriately.
func (self *Metadata) discoverUniquify() {
	self.uniquifier = self.discoverUniquifier()
	if self.uniquifier != "" {
		self.path = self.finalPath + "-u" + self.uniquifier
		self.curFilesPath = path.Join(self.path, "files")
	}
}

// Returns the uniquifier for this pipestance, if any, by
// examining the symlink from finalPath.
func (self *Metadata) discoverUniquifier() string {
	if rel, err := os.Readlink(self.finalPath); err == nil {
		dest := path.Join(path.Dir(self.finalPath), rel)
		if strings.HasPrefix(dest, self.finalPath+"-u") {
			return dest[len(self.finalPath)+2:]
		}
	}
	return ""
}

func (self *Metadata) loadCache() {
	self.discoverUniquify()
	paths := self.glob()
	self.mutex.Lock()
	if len(self.contents) > 0 {
		self.contents = make(map[MetadataFileName]bool)
	}
	if len(self.readCache) > 0 {
		self.readCache = make(map[MetadataFileName]LazyArgumentMap)
	}
	for _, p := range paths {
		self.contents[metadataFileNameFromPath(p)] = true
	}
	self.notRunningSince = time.Time{}
	self.lastRefresh = time.Time{}
	self.mutex.Unlock()
}

// Get the absolute path to the named file in the stage's files path.
func (self *Metadata) FilePath(name string) string {
	return path.Join(self.curFilesPath, name)
}

// Get the absolute path to the given metadata file.
func (self *Metadata) MetadataFilePath(name MetadataFileName) string {
	return path.Join(self.path, name.FileName())
}

func (self *Metadata) _existsNoLock(name MetadataFileName) bool {
	_, ok := self.contents[name]
	return ok
}

func (self *Metadata) exists(name MetadataFileName) bool {
	self.mutex.Lock()
	ok := self._existsNoLock(name)
	self.mutex.Unlock()
	return ok
}

func (self *Metadata) readRawBytes(name MetadataFileName) ([]byte, error) {
	return ioutil.ReadFile(self.MetadataFilePath(name))
}

func (self *Metadata) readRawSafe(name MetadataFileName) (string, error) {
	bytes, err := self.readRawBytes(name)
	return string(bytes), err
}

func (self *Metadata) readRaw(name MetadataFileName) string {
	s, _ := self.readRawSafe(name)
	return s
}

func (self *Metadata) readFromCache(name MetadataFileName) (LazyArgumentMap, bool) {
	self.mutex.Lock()
	i, ok := self.readCache[name]
	self.mutex.Unlock()
	return i, ok
}

func (self *Metadata) saveToCache(name MetadataFileName, value LazyArgumentMap) {
	self.mutex.Lock()
	self.readCache[name] = value
	self.mutex.Unlock()
}

func (self *Metadata) openFile(name MetadataFileName) (*os.File, error) {
	return os.Open(self.MetadataFilePath(name))
}

func (self *Metadata) read(name MetadataFileName, limit int64) (LazyArgumentMap, error) {
	v, ok := self.readFromCache(name)
	if ok {
		return v, nil
	}
	p := self.MetadataFilePath(name)
	if f, err := os.Open(p); err != nil {
		if !os.IsNotExist(err) {
			util.LogError(err, "runtime",
				"Could not open %s",
				p)
		}
		return nil, err
	} else {
		if err := func(p string, f *os.File, limit int64, v *LazyArgumentMap) error {
			defer f.Close()
			if limit > 0 {
				if info, err := f.Stat(); err != nil {
					return err
				} else if info.Size() > limit {
					return fmt.Errorf(
						"Insufficient memory to read %s\n"+
							"File is %d bytes, read size limited to %d bytes.",
						p, info.Size(), limit)
				}
			}
			dec := json.NewDecoder(f)
			return dec.Decode(v)
		}(p, f, limit, &v); err == nil {
			self.saveToCache(name, v)
			return v, nil
		} else {
			return nil, err
		}
	}
}

// Reads the content of the given metadata file and deserializes it into
// the given object.
func (self *Metadata) ReadInto(name MetadataFileName, target interface{}) error {
	if b, err := self.readRawBytes(name); err != nil {
		return err
	} else {
		return json.Unmarshal(b, target)
	}
}

func (self *Metadata) _writeRawNoLock(name MetadataFileName, text string) error {
	err := ioutil.WriteFile(self.MetadataFilePath(name), []byte(text), 0644)
	self._cacheNoLock(name)
	if err != nil {
		msg := fmt.Sprintf("Could not write %s for %s: %s", name, self.fqname, err.Error())
		util.LogError(err, "runtime", msg)
		if name != Errors {
			self._writeRawNoLock(Errors, msg)
		}
	}
	return err
}

func (self *Metadata) WriteRaw(name MetadataFileName, text string) error {
	return self.WriteRawBytes(name, []byte(text))
}

// Writes the given raw data into the given metadata file.
func (self *Metadata) WriteRawBytes(name MetadataFileName, text []byte) error {
	err := ioutil.WriteFile(self.MetadataFilePath(name), text, 0644)
	self.cache(name, self.uniquifier)
	if err != nil {
		msg := fmt.Sprintf("Could not write %s for %s: %s", name, self.fqname, err.Error())
		util.LogError(err, "runtime", msg)
		if name != Errors {
			self.WriteRaw(Errors, msg)
		}
	}
	return err
}

func (self *Metadata) appendRaw(name MetadataFileName, text string) error {
	self.cache(name, self.uniquifier)
	if f, err := os.OpenFile(self.MetadataFilePath(name),
		os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
		return err
	} else if _, err := f.Write([]byte(text)); err != nil {
		f.Close()
		return err
	} else {
		return f.Close()
	}
}

// Add text to the Alarm file for this node.
func (self *Metadata) AppendAlarm(text string) error {
	if err := self.appendRaw(AlarmFile, text); err != nil {
		msg := fmt.Sprintf("Could not write alarm for %s: %s",
			self.fqname, err.Error())
		util.LogError(err, "runtime", msg)
		self.WriteRaw(Errors, msg)
		return err
	}
	if self.journalPrefix != "" {
		return self.UpdateJournal(AlarmFile)
	} else {
		return nil
	}
}

// Serializes the given object and writes it to the given metadata file.
func (self *Metadata) Write(name MetadataFileName, object interface{}) error {
	bytes, _ := json.MarshalIndent(object, "", "    ")
	return self.WriteRawBytes(name, bytes)
}

// Writes the current timestamp into the given metadata file.  Generally used
// for sentinel files.
func (self *Metadata) WriteTime(name MetadataFileName) error {
	return self.WriteRaw(name, util.Timestamp())
}

// Serializes the given object and writes it to the given metadata file in a
// way that ensures the file is updated atomically and will never be observed
// in a partially-written form.
func (self *Metadata) WriteAtomic(name MetadataFileName, object interface{}) error {
	bytes, err := json.MarshalIndent(object, "", "    ")
	if err != nil {
		return err
	}
	fname := self.MetadataFilePath(name)
	tmpName := fname + ".tmp"
	if err := ioutil.WriteFile(tmpName, bytes, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpName, fname); err == nil || os.IsNotExist(err) {
		return nil
	} else {
		return err
	}
}

// Writes a journal file corresponding to the given metadata file.  This is
// used to notify the runtime of the existence of a new or updated file.
//
// The journal is a performance optimization to prevent the runtime from
// needing to constantly scan the entire database for changes.  Instead, it
// only scans the journal.  This means that when a metadata file is created
// or modified (except by the runtime itself), the change won't be "noticed"
// until the journal is updated.
func (self *Metadata) UpdateJournal(name MetadataFileName) error {
	fname := path.Join(self.journalPath, self.fqname+"."+self.journalPrefix+string(name))
	if err := ioutil.WriteFile(fname+".tmp", []byte(util.Timestamp()), 0644); err != nil {
		return err
	}
	if err := os.Rename(fname+".tmp", fname); err == nil || os.IsNotExist(err) {
		return nil
	} else {
		return err
	}
}

func (self *Metadata) remove(name MetadataFileName) error {
	self.uncache(name)
	return os.Remove(self.MetadataFilePath(name))
}
func (self *Metadata) _removeNoLock(name MetadataFileName) error {
	self._uncacheNoLock(name)
	return os.Remove(self.MetadataFilePath(name))
}

func (self *Metadata) clearReadCache() {
	self.mutex.Lock()
	if len(self.readCache) > 0 {
		self.readCache = make(map[MetadataFileName]LazyArgumentMap)
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
		if state, _ := self._getStateNoLock(); state == Running || state == Queued {
			// The job is not running but the metadata thinks it still is.
			// The check for metadata updates was completed since the time that
			// the queue query completed.  This job has failed.  Write an error.
			self._writeRawNoLock(Errors, fmt.Sprintf(
				"According to the job manager, the job for %s was not queued "+
					"or running, since at least %s.",
				self.fqname, notRunningSince.Format(util.TIMEFMT)))
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
	if !self.exists(JobId) {
		return
	}
	if st, _ := self.getState(); st != Running && st != Queued {
		return
	}
	self.mutex.Lock()
	if !self.notRunningSince.IsZero() {
		self.mutex.Unlock()
		return
	}
	if self.readRaw(JobId) != jobid {
		self.mutex.Unlock()
		return
	}
	// Double-check that the job wasn't reset while jobid was being read.
	if !self._existsNoLock(JobId) {
		self.mutex.Unlock()
		return
	}
	self.notRunningSince = time.Now()
	self.mutex.Unlock()
}

func (self *Metadata) checkedReset() error {
	self.mutex.Lock()
	if state, _ := self._getStateNoLock(); state == Failed {
		if len(self.contents) > 0 {
			self.contents = make(map[MetadataFileName]bool)
		}
		self.mutex.Unlock()
		if err := self.uncheckedReset(); err == nil {
			util.PrintInfo("runtime", "(reset-partial)   %s", self.fqname)
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
		journalPrefix := path.Join(self.journalPath, self.fqname)
		if self.uniquifier != "" {
			journalPrefix += ".u" + self.uniquifier
		}
		if files, err := filepath.Glob(journalPrefix + "*"); err == nil {
			for _, file := range files {
				os.Remove(file)
			}
		}
	}
	if err := self.removeAll(); err != nil {
		util.PrintInfo("runtime", "Cannot reset the stage because some folder contents could not be deleted.\n\nPlease resolve this error in order to continue running the pipeline: %v", err)
		return err
	}
	if self.uniquifier == "" {
		return self.mkdirs()
	} else {
		self.uniquifier = ""
		return self.uniquify()
	}
}

// Resets the metadata if the state was queued, but the job manager had not yet
// started the job locally or queued it remotely.
func (self *Metadata) restartQueuedLocal() error {
	if self.exists(QueuedLocally) {
		if err := self.uncheckedReset(); err == nil {
			util.PrintInfo("runtime", "(reset-running)   %s", self.fqname)
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
	state, ok := self.getState()
	if !ok {
		return nil
	}
	if state == Queued {
		if err := self.uncheckedReset(); err == nil {
			util.PrintInfo("runtime", "(reset-queued)    %s", self.fqname)
		} else {
			return err
		}
	} else if state == Running {
		var jobInfo *JobInfo
		if err := self.ReadInto(JobInfoFile, jobInfo); err == nil &&
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
						util.PrintInfo("runtime", "(reset-running)   %s", self.fqname)
					} else {
						return err
					}
				} else {
					util.PrintInfo("runtime", "Possibly running  %s", self.fqname)
				}
			}
		}
	}
	return nil
}

func (self *Metadata) checkHeartbeat() {
	if state, _ := self.getState(); state == Running {
		if self.lastHeartbeat.IsZero() || self.exists(Heartbeat) {
			self.uncache(Heartbeat)
			self.lastHeartbeat = time.Now()
		}
		if self.lastRefresh.Sub(self.lastHeartbeat) > time.Minute*heartbeatTimeout {
			self.WriteRaw("errors", fmt.Sprintf(
				"%s: No heartbeat detected for %d minutes. Assuming job has failed. This may be "+
					"due to a user manually terminating the job, or the operating system or cluster "+
					"terminating it due to resource or time limits.",
				util.Timestamp(), heartbeatTimeout))
		}
	}
}

func (self *Metadata) serializeState() *MetadataInfo {
	self.mutex.Lock()
	names := make([]string, 0, len(self.contents))
	for content := range self.contents {
		names = append(names, string(content))
	}
	self.mutex.Unlock()
	sort.Strings(names)
	return &MetadataInfo{
		Path:  self.finalPath,
		Names: names,
	}
}

func (self *Metadata) serializePerf(numThreads int) *PerfInfo {
	if self.exists(CompleteFile) && self.exists(JobInfoFile) {
		jobInfo := JobInfo{}
		if err := self.ReadInto(JobInfoFile, &jobInfo); err == nil {
			fpaths, _ := self.enumerateFiles()
			return reduceJobInfo(&jobInfo, fpaths, numThreads)
		}
	}
	return nil
}

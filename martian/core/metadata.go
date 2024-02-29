// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package core

// Martian runtime. This is where the action happens.

import (
	"encoding/json"
	"fmt"
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

// MimeType returns the expected MIME type for this metadata file.
//
// For most cases this will be either "text/plain" or "application/json".
//
// For unknown metadata types, an empty string is returned.
func (self MetadataFileName) MimeType() string {
	switch self {
	case ArgsFile, OutsFile,
		JobInfoFile,
		StageDefsFile, ChunkDefsFile, ChunkOutsFile,
		VdrKill, PartialVdr, FinalState,
		TagsFile, VersionsFile, Perf:
		return "application/json"
	case LogFile, StdErr, StdOut,
		InvocationFile, MroSourceFile,
		Assert, AlarmFile, Errors, Stackvars,
		ProgressFile:
		return "text/plain;charset=UTF-8"
	case MetadataZip:
		return "application/zip"
	case CompleteFile, Heartbeat, TimestampFile,
		JobId, QueuedLocally,
		JobModeFile, Lock, UiPort, UuidFile,
		DisabledFile:
		return "text/plain"
	case PerfData:
		return "application/octet-stream"
	default:
		return ""
	}
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
	return MetadataState(prefix + string(self))
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
	fqname string

	path string
	// This is the human-logical path, e.g. `join`.  This will be the same as
	// `path` unless the the directory is being uniquified.
	finalPath string
	contents  map[MetadataFileName]struct{}
	readCache map[MetadataFileName]LazyArgumentMap
	// This is where the stage code writes files.
	curFilesPath string
	// This is the "canonical" path to the files, which may be a symlink to
	// `curFilesPath`.  This will be `finalPath/files` except for the "fake"
	// `join` metadata of a stage that doesn't split, where it will point up to
	// the fork's top-level `files`, which will in turn be a symlink to the
	// singular chunk's `files`.
	finalFilePath string
	journalPath   string
	lastRefresh   time.Time
	lastHeartbeat time.Time
	// An arbitrary string appended to the path that gets changed any time the
	// stage needs to be restarted, in case the previous attempt didn't actually
	// terminate properly (perhaps it left orphened subprocesses) so that
	// the new run doesn't get clobbered by the old one.
	uniquifier string

	// If non-zero the job was not found last time the job manager was queried,
	// the chunk will be failed out if the state seems like it's still running
	// after the job manager's grace period has elapsed.
	notRunningSince time.Time

	// A prefix to attach when writing journal file name.
	// Empty for chunks, or SplitPrefix or JoinPrefix.
	journalPrefix string

	mutex sync.Mutex
}

// Basic exportable information from a metadata object.
type MetadataInfo struct {
	// The filesystem path containing the metadata files.
	Path string `json:"path"`

	// The metadata file names which exist for this object.
	Names []string `json:"names"`
}

func NewMetadata(fqname string, p string) *Metadata {
	fp := path.Join(p, "files")
	return &Metadata{
		fqname:        fqname,
		path:          p,
		finalPath:     p,
		contents:      make(map[MetadataFileName]struct{}),
		readCache:     make(map[MetadataFileName]LazyArgumentMap),
		curFilesPath:  fp,
		finalFilePath: fp,
	}
}

func NewMetadataRunWithJournalPath(fqname, p, filesPath, journalPath, runType string) *Metadata {
	self := NewMetadata(fqname, p)
	self.journalPath = path.Join(journalPath, fqname)
	self.curFilesPath = filesPath
	self.finalFilePath = filesPath
	if runType != "main" {
		self.journalPrefix = runType + "_"
	}
	return self
}

func newMetadataWithJournalPath(fqname, journalName, p, journalPath string) *Metadata {
	self := NewMetadata(fqname, p)
	self.journalPath = path.Join(journalPath, journalName)
	return self
}

// glob returns the set of metadata files for the path represented by this object.
func (self *Metadata) glob(exclude ...MetadataFileName) ([]string, error) {
	paths, err := util.Readdirnames(self.path)
	if err != nil {
		return nil, err
	}
	matches := make([]string, 0, len(paths))
	for _, p := range paths {
		if suffix, found := strings.CutPrefix(p, MetadataFilePrefix); found {
			if !isExcludedMetadataName(suffix, exclude...) {
				matches = append(matches, path.Join(self.path, p))
			}
		}
	}
	return matches, nil
}

func isExcludedMetadataName(suffix string, exclude ...MetadataFileName) bool {
	for _, e := range exclude {
		if string(e) == suffix {
			return true
		}
	}
	return false
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

// enumerateFiles returns the list of items in the files/ directory associated
// with this metadata object.
//
// These are files generated by stage code.
func (self *Metadata) enumerateFiles() ([]string, error) {
	paths, err := util.Readdirnames(self.curFilesPath)
	for i, p := range paths {
		paths[i] = path.Join(self.curFilesPath, p)
	}
	return paths, err
}

func (self *Metadata) enumerateTemp() ([]string, error) {
	if td := self.TempDir(); td == "" {
		return nil, nil
	} else {
		paths, err := util.Readdirnames(td)
		for i, p := range paths {
			paths[i] = path.Join(td, p)
		}
		return paths, err
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

func (self *Metadata) writeError(msg string, src error) {
	msg = msg + " (" + self.fqname + ")"
	util.LogError(src, "runtime", "%s", msg)
	if err := self.WriteRaw(Errors, msg+": "+src.Error()); err != nil {
		util.PrintError(err, "runtime", "Could not write error message")
	}
}

func (self *Metadata) writeErrorNoLock(msg string, src error) {
	msg = msg + self.fqname
	util.LogError(src, "runtime", "%s", msg)
	if err := self._writeRawNoLock(Errors, msg+": "+src.Error()); err != nil {
		util.PrintError(err, "runtime", "Could not write error message")
	}
}

func (self *Metadata) mkdirs() error {
	if err := util.Mkdir(self.path); err != nil {
		self.writeError("Could not create directories for ", err)
		return err
	}
	if err := util.Mkdir(self.curFilesPath); err != nil {
		self.writeError("Could not create directories for ", err)
		return err
	}
	return nil
}

func (self *Metadata) mkForkDirs() error {
	if err := util.MkdirAll(self.path); err != nil {
		self.writeError("Could not create directories for ", err)
		return err
	}
	if err := util.Mkdir(self.curFilesPath); err != nil {
		self.writeError("Could not create directories for ", err)
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
		self.writeErrorNoLock("Could not create directories for ", err)
		return err
	}
	self.path = p
	filesPath := path.Join(p, "files")
	if err := util.Mkdir(filesPath); err != nil {
		self.writeErrorNoLock("Could not create file directory for ", err)
		os.Remove(p)
		return err
	}
	self.curFilesPath = filesPath
	if err := util.Mkdir(self.TempDir()); err != nil {
		self.writeErrorNoLock("Could not create temp directory for ", err)
		os.Remove(filesPath)
		os.Remove(p)
		return err
	}

	if self.discoverUniquifier() != self.uniquifier {
		if relPath, err := filepath.Rel(filepath.Dir(self.finalPath), p); err != nil {
			msg := fmt.Sprintf(
				"Could not compute relative path for %s to %s for stage ",
				self.finalPath, p)
			self.writeErrorNoLock(msg, err)
			os.RemoveAll(p)
			return err
		} else {
			if err := os.Symlink(relPath, self.finalPath); err != nil {
				self.writeErrorNoLock("Could not create symlink for ", err)
				os.RemoveAll(p)
				return err
			}
		}
		if self.finalFilePath != path.Join(self.finalPath, "files") {
			if err := os.Remove(self.finalFilePath); err != nil && !os.IsNotExist(err) {
				self.writeErrorNoLock("Could not remove existing directory for ", err)
				os.RemoveAll(p)
				return err
			}
			if relPath, err := filepath.Rel(filepath.Dir(self.finalFilePath), filesPath); err != nil {
				msg := fmt.Sprintf(
					"Could not compute relative path for %s to %s for stage ",
					self.finalFilePath, filesPath)
				self.writeErrorNoLock(msg, err)
				os.RemoveAll(p)
				return err
			} else {
				if err := os.Symlink(relPath, self.finalFilePath); err != nil {
					self.writeErrorNoLock("Could not create files symlink for ", err)
					os.RemoveAll(p)
					return err
				}
			}
		}
	}
	return nil
}

// removeAll deletes all output files, the directory containing them, and the
// symlinks pointing to them.  If includeMeta is true, also delete the metadata
// directory itself.
func (self *Metadata) removeAll(includeMeta bool) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if len(self.contents) > 0 {
		self.contents = make(map[MetadataFileName]struct{})
	}
	if len(self.readCache) > 0 {
		self.readCache = make(map[MetadataFileName]LazyArgumentMap)
	}
	self.notRunningSince = time.Time{}
	self.lastRefresh = time.Time{}
	if err := os.RemoveAll(self.curFilesPath); err != nil {
		return err
	}
	if includeMeta {
		if err := os.RemoveAll(self.path); err != nil {
			return err
		}
	} else if t := self.TempDir(); t != "" {
		// If we're deleting the entire `self.path` then this gets taken care of
		// that way, but otherwise we still want to remove it.
		if err := os.RemoveAll(self.TempDir()); err != nil {
			return err
		}
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
			if err := self._removeNoLock(JobId); err != nil {
				util.LogError(err, "runtime", "Could not remove file.")
			}
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
	self.contents[name] = struct{}{}
	// cache is usually called on write or update
	delete(self.readCache, name)
}

func (self *Metadata) cache(name MetadataFileName, uniquifier string) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.uniquifier == uniquifier {
		self._cacheNoLock(name)
	} else if self.uniquifier != "" {
		util.LogInfo("runtime",
			"There appears to be more than one instance of %s running "+
				"(Saw ID '%s', expected '%s').",
			self.fqname, uniquifier, self.uniquifier)
	}
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
	paths, err := self.glob()
	if err != nil {
		return
	}
	self.mutex.Lock()
	if len(self.contents) > 0 {
		self.contents = make(map[MetadataFileName]struct{}, len(paths))
	}
	if len(self.readCache) > 0 {
		self.readCache = make(map[MetadataFileName]LazyArgumentMap)
	}
	for _, p := range paths {
		self.contents[metadataFileNameFromPath(p)] = struct{}{}
	}
	self.notRunningSince = time.Time{}
	self.lastRefresh = time.Time{}
	self.mutex.Unlock()
}

// Looks for new files in the metadata directory and updates the cache
// accordingly.  It does not remove files from the cache, because for
// example heartbeat files aren't expected to be seen there anyway.
func (self *Metadata) poll() {
	paths, _ := self.glob()
	if len(paths) == 0 {
		return
	}
	self.mutex.Lock()
	for _, p := range paths {
		self.contents[metadataFileNameFromPath(p)] = struct{}{}
	}
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
	return os.ReadFile(self.MetadataFilePath(name))
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
	err := os.WriteFile(self.MetadataFilePath(name), []byte(text), 0644)
	self._cacheNoLock(name)
	if err != nil {
		msg := fmt.Sprintf("Could not write %s for %s: %s", name, self.fqname, err.Error())
		util.LogError(err, "runtime", msg)
		if name != Errors {
			// ignore errors here, since what are we going to do about it?
			_ = self._writeRawNoLock(Errors, msg)
		}
	}
	return err
}

func (self *Metadata) WriteRaw(name MetadataFileName, text string) error {
	return self.WriteRawBytes(name, []byte(text))
}

// Writes the given raw data into the given metadata file.
func (self *Metadata) WriteRawBytes(name MetadataFileName, text []byte) error {
	err := os.WriteFile(self.MetadataFilePath(name), text, 0644)
	self.cache(name, self.uniquifier)
	if err != nil {
		msg := fmt.Sprintf("Could not write %s for %s: %s", name, self.fqname, err.Error())
		util.LogError(err, "runtime", "%s", msg)
		if name != Errors {
			self.WriteErrorString(msg)
		}
	}
	return err
}

func (self *Metadata) WriteErrorString(msg string) {
	if err := self.WriteRaw(Errors, msg); err != nil {
		util.LogError(err, "runtime", "Could not write errors file.")
	}
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
		util.LogError(err, "runtime", "%s", msg)
		self.WriteErrorString(msg)
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
	return writeAtomic(fname, bytes)
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
	fname := self.journalPath + "." + self.journalPrefix + string(name)
	if err := os.WriteFile(fname,
		[]byte(util.Timestamp()), 0644); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func (self *Metadata) remove(name MetadataFileName) error {
	self.uncache(name)
	err := os.Remove(self.MetadataFilePath(name))
	if os.IsNotExist(err) {
		// Workaround for an issue one heavily loaded NFS servers.  If a request
		// is taking a long time, the client will re-send the request.  The
		// server is supposed to note that the request is a duplicate and
		// de-duplicate it, but if it's heavily loaded its duplicate request
		// cache might have already been flushed, in which case the second
		// request will see ENOENT and fail.
		return nil
	}
	return err
}
func (self *Metadata) _removeNoLock(name MetadataFileName) error {
	self._uncacheNoLock(name)
	err := os.Remove(self.MetadataFilePath(name))
	if os.IsNotExist(err) {
		return nil
	}
	return err
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
			jobid := self.readRaw(JobId)
			if jobid != "" {
				util.LogInfo("runtime",
					"Job %s is not in the job manager queue.  Failing %s.",
					jobid, self.fqname)
			}
			// The job is not running but the metadata thinks it still is.
			// The check for metadata updates was completed since the time that
			// the queue query completed.  This job has failed.  Write an error.
			err := self._writeRawNoLock(Errors, fmt.Sprintf(
				"According to the job manager, the job for %s was not queued "+
					"or running, since at least %s.",
				self.fqname, notRunningSince.Format(util.TIMEFMT)))
			if err != nil {
				util.LogError(err, "runtime",
					"Error writing error message about cluster-mode job not running.")
			}
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
	// Check whether the job has changed state but we just didn't see the
	// journal update for whatever reason.
	self.poll()
	if st, _ := self.getState(); st != Running && st != Queued {
		return
	}
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if !self.notRunningSince.IsZero() {
		return
	}
	if self.readRaw(JobId) != jobid {
		return
	}
	// Double-check that the job wasn't reset while jobid was being read.
	if !self._existsNoLock(JobId) {
		return
	}
	self.notRunningSince = time.Now()
}

func (self *Metadata) checkedReset() error {
	self.mutex.Lock()
	if state, _ := self._getStateNoLock(); state == Failed {
		if len(self.contents) > 0 {
			self.contents = make(map[MetadataFileName]struct{})
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

func (self *Metadata) journalFile() string {
	if self.uniquifier == "" {
		return self.journalPath
	} else if p := self.journalPath; p == "" {
		return ""
	} else {
		return p + ".u" + self.uniquifier
	}
}

func (self *Metadata) uncheckedReset() error {
	// Remove all related files from journal directory.
	if len(self.journalPath) > 0 {
		dir, base := filepath.Split(self.journalFile())
		paths, _ := util.Readdirnames(dir)
		for _, p := range paths {
			if strings.HasPrefix(p, base) {
				os.Remove(path.Join(dir, p))
			}
		}
	}
	if err := self.removeAll(true); err != nil {
		util.PrintInfo("runtime",
			"Cannot reset the stage because some folder contents could not "+
				"be deleted.\n\nPlease resolve this error in order to "+
				"continue running the pipeline:\n\t%v",
			err)
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
		var jobInfo JobInfo
		if err := self.ReadInto(JobInfoFile, &jobInfo); err == nil &&
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
			// Check if the state changed but we just missed the journal.
			self.poll()
			if state, _ := self.getState(); state != Running {
				return
			}
			self.WriteErrorString(fmt.Sprintf(
				"%s: No heartbeat detected for %d minutes. "+
					"Assuming job has failed. This may be "+
					"due to a user manually terminating the job, "+
					"or the operating system or cluster "+
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

func (self *Metadata) serializePerf(numThreads float64) *PerfInfo {
	if self.exists(CompleteFile) && self.exists(JobInfoFile) {
		jobInfo := JobInfo{}
		if err := self.ReadInto(JobInfoFile, &jobInfo); err == nil {
			fpaths, _ := self.enumerateFiles()
			return reduceJobInfo(&jobInfo, fpaths, numThreads)
		}
	}
	return nil
}

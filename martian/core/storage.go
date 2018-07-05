// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

//
// Martian runtime storage tracking and recovery.
//

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

type VdrEvent struct {
	Timestamp  time.Time
	DeltaBytes int64
}

type PartialVdrKillReport struct {
	VDRKillReport `json:"report,omitempty"`
	Split         bool `json:"ran_split,omitempty"`
	Chunks        bool `json:"ran_chunks,omitempty"`
	Join          bool `json:"ran_join,omitempty"`
}

//
// Volatile Disk Recovery
//
type VDRKillReport struct {
	Count     uint        `json:"count"`
	Size      uint64      `json:"size"`
	Timestamp string      `json:"timestamp"`
	Paths     []string    `json:"paths"`
	Errors    []string    `json:"errors"`
	Events    []*VdrEvent `json:"events,omitempty"`
}

type VDRByTimestamp []*VDRKillReport

func (self VDRByTimestamp) Len() int {
	return len(self)
}

func (self VDRByTimestamp) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self VDRByTimestamp) Less(i, j int) bool {
	return self[i].Timestamp < self[j].Timestamp
}

// Merge events with the same timestamp.
func (pr *VDRKillReport) mergeEvents() {
	allEvents := pr.Events
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Timestamp.Before(
			allEvents[j].Timestamp)
	})
	result := make([]*VdrEvent, 0, len(allEvents))
	for _, ev := range allEvents {
		last := len(result) - 1
		if last < 0 ||
			result[last].Timestamp.Truncate(time.Second) !=
				ev.Timestamp.Truncate(time.Second) ||
			(ev.DeltaBytes < 0) != (result[last].DeltaBytes < 0) {
			result = append(result, ev)
		} else {
			result[last].DeltaBytes += ev.DeltaBytes
		}
	}
	pr.Events = result
}

func mergeVDRKillReports(killReports []*VDRKillReport) *VDRKillReport {
	allKillReport := &VDRKillReport{}
	var allEvents []*VdrEvent
	if len(killReports) > 0 {
		allEvents = make([]*VdrEvent, 0, len(killReports)+len(killReports[0].Events))
	}
	for _, killReport := range killReports {
		if killReport == nil {
			continue
		}
		allKillReport.Size += killReport.Size
		allKillReport.Count += killReport.Count
		allKillReport.Errors = append(allKillReport.Errors, killReport.Errors...)
		allKillReport.Paths = append(allKillReport.Paths, killReport.Paths...)
		allEvents = append(allEvents, killReport.Events...)
		if allKillReport.Timestamp == "" || allKillReport.Timestamp < killReport.Timestamp {
			allKillReport.Timestamp = killReport.Timestamp
		}
	}
	allKillReport.Events = allEvents
	allKillReport.mergeEvents()
	return allKillReport
}

func (self *Fork) partialVdrKill() (*VDRKillReport, bool) {
	self.storageLock.Lock()
	defer self.storageLock.Unlock()
	if state := self.getState(); state.IsFailed() {
		return nil, false
	} else if state == DisabledState {
		return self.vdrKill(nil), true
	} else if rep, ok := self.getVdrKillReport(); ok {
		if self.node.rt.Config.Debug {
			util.LogInfo("storage",
				"%s is already VDRed",
				self.node.GetFQName())
		}
		return rep, ok
	} else {
		partial := self.getPartialKillReport()
		if self.Split() &&
			(partial == nil || !partial.Split) &&
			(state == Complete ||
				state == Complete.Prefixed(SplitPrefix) ||
				state.HasPrefix(ChunksPrefix) ||
				state.HasPrefix(JoinPrefix)) {
			partial = self.cleanSplitTemp(partial)
		}
		if (partial == nil || !partial.Chunks) &&
			(state == Complete ||
				state == Complete.Prefixed(ChunksPrefix) ||
				state.HasPrefix(JoinPrefix)) {
			partial = self.cleanChunkTemp(partial)
		}
		if (partial == nil ||
			!partial.Join) &&
			(state == Complete ||
				state == Complete.Prefixed(JoinPrefix)) {
			partial = self.cleanJoinTemp(partial)
		}
		if state == Complete {
			doneNodes := make([]Nodable, 0, len(self.filePostNodes))
			for node := range self.filePostNodes {
				if node != nil {
					if st := node.getNode().getState(); st == Complete || st == DisabledState {
						doneNodes = append(doneNodes, node)
					}
				}
			}
			self.removeFilePostNodes(doneNodes)
			if len(self.filePostNodes) == 0 {
				if self.node.rt.Config.Debug {
					util.LogInfo("storage",
						"Running full vdr on %s",
						self.node.GetFQName())
				}
				return self.vdrKill(partial), true
			} else {
				if self.node.rt.Config.Debug {
					for node, args := range self.filePostNodes {
						a := make([]string, 0, len(args))
						for arg := range args {
							a = append(a, arg)
						}
						sort.Strings(a)
						if node != nil {
							util.LogInfo("storage",
								"%s is keeping argument(s) %s of %s alive",
								node.GetFQName(),
								strings.Join(a, ","),
								self.node.GetFQName())
						} else {
							util.LogInfo("storage",
								"Outs %s of %s are being kept alive by the top-level.",
								strings.Join(a, ","),
								self.node.GetFQName())
						}
					}
				}
				if self.node.strictVolatile {
					return self.vdrKillSome(partial, false)
				}
			}
		}
		if partial != nil {
			if self.node.rt.Config.Debug {
				util.LogInfo("storage",
					"Partial vdr of %s in phase %v",
					self.node.GetFQName(), state)
			}
			return &partial.VDRKillReport, false
		} else {
			if self.node.rt.Config.Debug {
				util.LogInfo("storage",
					"No partial vdr of %s in phase %v",
					self.node.GetFQName(), state)
			}
			return nil, false
		}
	}
}

func (self *Fork) vdrKillSome(partial *PartialVdrKillReport, done bool) (*VDRKillReport, bool) {
	if self.fileParamMap == nil {
		self.cacheParamFileMap(nil)
	} else {
		self.updateParamFileCache()
	}
	if self.node.rt.Config.VdrMode == "disable" ||
		!self.node.rt.overrides.GetOverride(self.node, "force_volatile", true).(bool) {
		if partial == nil {
			return nil, false
		}
		if partial != nil {
			return &partial.VDRKillReport, false
		} else {
			return nil, false
		}
	}
	killPaths := make([]string, 0, len(self.fileParamMap))
	for file, keepAliveArgs := range self.fileParamMap {
		if keepAliveArgs.args == nil {
			killPaths = append(killPaths, file)
		}
	}
	if len(killPaths) == 0 {
		if done {
			if partial != nil {
				partial.VDRKillReport.mergeEvents()
				self.metadata.Write(VdrKill, &partial.VDRKillReport)
			} else {
				self.metadata.Write(VdrKill,
					VDRKillReport{Timestamp: util.Timestamp()})
			}
			self.deletePartialKill()
		}
		if partial == nil {
			return nil, false
		} else {
			return &partial.VDRKillReport, false
		}
	}
	if partial == nil {
		partial = new(PartialVdrKillReport)
	}
	sort.Strings(killPaths)
	collapsedPaths := make([]string, 0, len(killPaths))

	var event VdrEvent
	for _, fpath := range killPaths {
		entry := self.fileParamMap[fpath]
		event.DeltaBytes -= entry.size
		partial.Size += uint64(entry.size)
		partial.Count += uint(entry.count)
		if len(collapsedPaths) == 0 || !pathIsInside(fpath, collapsedPaths[len(collapsedPaths)-1]) {
			collapsedPaths = append(collapsedPaths, fpath)
		} else {
			other := self.fileParamMap[collapsedPaths[len(collapsedPaths)-1]]
			other.size += entry.size
			other.count += entry.count
			delete(self.fileParamMap, fpath)
		}
	}
	partial.Paths = append(partial.Paths, collapsedPaths...)
	partial.Events = append(partial.Events, &event)
	util.EnterCriticalSection()
	defer util.ExitCriticalSection()
	for _, fpath := range collapsedPaths {
		if err := os.RemoveAll(fpath); err != nil {
			partial.Errors = append(partial.Errors, err.Error())
		}
		delete(self.fileParamMap, fpath)
	}
	event.Timestamp = time.Now()
	partial.Timestamp = util.Timestamp()

	if len(self.fileParamMap) == 0 || done || len(self.filePostNodes) == 0 {
		partial.VDRKillReport.mergeEvents()
		self.metadata.Write(VdrKill, &partial.VDRKillReport)
		self.deletePartialKill()
		if self.node.rt.Config.Debug {
			util.LogInfo("storage", "VDR of %s complete",
				self.node.GetFQName())
		}
		return &partial.VDRKillReport, true
	} else {
		self.writePartialKill(partial)
		if self.node.rt.Config.Debug {
			util.LogInfo("storage",
				"VDR of %s still waiting on %d nodes, "+
					"keeping %d files alive through %d arguments.",
				self.node.GetFQName(),
				len(self.filePostNodes),
				len(self.fileParamMap),
				len(self.fileArgs))
		}

		return &partial.VDRKillReport, false
	}
}

func pathIsInside(test, parent string) bool {
	parent = filepath.Clean(parent)
	for name := filepath.Clean(test); len(name) >= len(parent); name = path.Dir(name) {
		if name == parent {
			return true
		}
	}
	return false
}

// Returns all of the logical file names which may refer to the same file as
// the give path name.
func getLogicalFileNames(name string) []string {
	var names []string
	if info, err := os.Lstat(name); err == nil {
		names = append(names, name)
		if resolved, err := filepath.EvalSymlinks(name); err == nil &&
			name != resolved {
			names = append(names, resolved)
		}
		for info.Mode()&os.ModeSymlink != 0 {
			if dest, err := os.Readlink(name); err != nil {
				break
			} else {
				names = append(names, dest)
				if destInfo, err := os.Lstat(dest); err != nil {
					break
				} else {
					name = dest
					info = destInfo
				}
			}
		}
	}
	return names
}

type vdrFileCache struct {
	size  int64
	count int
	args  map[string]struct{}
}

// Returns the set of arguments from fileArgs which actually refer to files,
// and, for each one, the set of files to which they refer.
func getArgsToFilesMap(fileArgs map[string]map[Nodable]struct{},
	outs LazyArgumentMap,
	debug bool, fqname string) map[string]map[string]struct{} {
	argToFiles := make(map[string]map[string]struct{}, len(fileArgs))
	// Get the set of files each argument refers to.
	for arg := range fileArgs {
		for _, name := range getMaybeFileNames(outs[arg]) {
			for _, fullName := range getLogicalFileNames(name) {
				fileSet := argToFiles[arg]
				if fileSet == nil {
					fileSet = map[string]struct{}{fullName: struct{}{}}
					argToFiles[arg] = fileSet
				} else {
					fileSet[fullName] = struct{}{}
				}
				if debug {
					util.LogInfo("storage",
						"Argument %s of %s references file %s.",
						arg, fqname, fullName)
				}
			}
		}
	}
	return argToFiles
}

// Add files from fpath to filesToArgs.  If they are present in argToFiles,
// add the appropriate argument list.
func addFilesToArgsMappings(fpath string, debug bool, fqname string,
	filesToArgs map[string]*vdrFileCache,
	argToFiles map[string]map[string]struct{}) {
	util.Walk(fpath, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if _, ok := filesToArgs[fpath]; ok {
			if info.IsDir() {
				return filepath.SkipDir
			} else {
				return nil
			}
		}
		entry := &vdrFileCache{
			size:  info.Size(),
			count: 1,
		}
		filesToArgs[fpath] = entry
		seenNames := make(map[string]struct{})
		for _, name := range getLogicalFileNames(fpath) {
			if _, ok := seenNames[name]; ok {
				return nil
			}
			seenNames[name] = struct{}{}
			for arg, files := range argToFiles {
				for file := range files {
					if pathIsInside(file, name) || pathIsInside(name, file) {
						if debug {
							util.LogInfo("storage",
								"Argument %s of %s references file\n%s",
								arg, fqname, fpath)
							util.LogInfo("storage",
								"The direct reference is to\n%s\ncontained by\n%s",
								file, name)
						}

						if entry.args == nil {
							entry.args = map[string]struct{}{arg: struct{}{}}
						} else {
							entry.args[arg] = struct{}{}
						}
						break
					}
				}
			}
		}
		if len(entry.args) == 0 && debug {
			util.LogInfo("storage",
				"%s does not reference file\n%s",
				fqname, fpath)
		}
		return nil
	})
}

// Gets the set of files generated by this stage, and the set of arguments
// which are keeping those files live.  Files with nothing keeping them alive
// are still added to the set, so VDR knows what it can kill.
func (self *Fork) cacheParamFileMap(outs LazyArgumentMap) {
	if outs == nil {
		outs = self.metadata.read(OutsFile)
	}
	if outs == nil {
		return
	}
	argToFiles := getArgsToFilesMap(
		self.fileArgs,
		outs,
		self.node.rt.Config.Debug,
		self.node.GetFQName())
	// Remove "file" args which don't actually refer to existing files.
	for arg := range self.fileArgs {
		if _, ok := argToFiles[arg]; !ok {
			self.removeFileArg(arg)
		}
	}
	filesToArgs := make(map[string]*vdrFileCache, len(self.fileArgs))
	addMetadata := func(md *Metadata) {
		files, _ := md.enumerateFiles()
		for _, fpath := range files {
			addFilesToArgsMappings(fpath,
				self.node.rt.Config.Debug,
				self.node.GetFQName(),
				filesToArgs, argToFiles)
		}
	}
	addMetadata(self.split_metadata)
	addMetadata(self.join_metadata)
	for _, chunk := range self.chunks {
		addMetadata(chunk.metadata)
	}
	// Take out arguments which don't refer to any files owned by this stage.
	for arg := range self.fileArgs {
		any := false
		for _, entry := range filesToArgs {
			if args := entry.args; args != nil {
				if _, ok := args[arg]; ok {
					any = true
					break
				}
			}
		}
		if !any {
			self.removeFileArg(arg)
		}
	}
	self.fileParamMap = filesToArgs
}

func (self *Fork) updateParamFileCache() {
	for file, keepAliveArgs := range self.fileParamMap {
		if keepAliveArgs.args != nil {
			// Check whether args were removed because nodes completed since
			// things were cached.
			for arg := range keepAliveArgs.args {
				if _, ok := self.fileArgs[arg]; !ok {
					if self.node.rt.Config.Debug {
						util.LogInfo("storage",
							"File %s of %s is no longer being kept alive by %s.",
							file, self.node.GetFQName(), arg)
					}
					delete(keepAliveArgs.args, arg)
				}
			}
			if len(keepAliveArgs.args) == 0 {
				if self.node.rt.Config.Debug {
					util.LogInfo("storage",
						"File %s of %s is no longer required.",
						file, self.node.GetFQName())
				}
				keepAliveArgs.args = nil
			}
		}
	}
}

func (metadata *Metadata) getStartTime() time.Time {
	var jobInfo JobInfo
	if err := metadata.ReadInto(JobInfoFile, &jobInfo); err != nil && os.IsNotExist(err) {
		// Stages which don't split/join still have metadata for the
		// split/join, and still need accurate timestamps.
		if info, _ := os.Stat(metadata.path); info != nil {
			return util.FileCreateTime(info).Truncate(time.Second)
		} else {
			return time.Time{}
		}
	} else if err != nil || jobInfo.WallClockInfo == nil {
		return time.Time{}
	} else {
		t, _ := time.ParseInLocation(util.TIMEFMT, jobInfo.WallClockInfo.Start, time.Local)
		return t
	}
}

func (self *Fork) cleanSplitTemp(partial *PartialVdrKillReport) *PartialVdrKillReport {
	if tempPaths, err := self.split_metadata.enumerateTemp(); err != nil {
		return partial
	} else if filesPaths, err := self.split_metadata.enumerateFiles(); err != nil {
		return partial
	} else {
		if partial == nil {
			partial = new(PartialVdrKillReport)
		}
		partial.Split = true
		var startEvent, cleanupEvent VdrEvent
		startEvent.Timestamp = self.split_metadata.getStartTime()
		for _, p := range tempPaths {
			util.Walk(p, func(tpath string, info os.FileInfo, err error) error {
				if err == nil {
					partial.Size += uint64(info.Size())
					partial.Count++
					startEvent.DeltaBytes += int64(info.Size())
					cleanupEvent.DeltaBytes -= int64(info.Size())
				} else {
					partial.Errors = append(partial.Errors, err.Error())
				}
				return nil
			})
		}
		for _, p := range filesPaths {
			util.Walk(p, func(tpath string, info os.FileInfo, err error) error {
				if err == nil {
					startEvent.DeltaBytes += int64(info.Size())
				} else {
					partial.Errors = append(partial.Errors, err.Error())
				}
				return nil
			})
		}
		// Add metadata file sizes.
		for _, md := range self.split_metadata.glob() {
			if info, err := os.Lstat(md); err != nil {
				partial.Errors = append(partial.Errors, err.Error())
			} else {
				startEvent.DeltaBytes += int64(info.Size())
			}
		}
		if self.node.rt.Config.Debug {
			util.LogInfo("storage",
				"%d bytes of split files for %s",
				startEvent.DeltaBytes, self.node.GetFQName())
		}
		if startEvent.DeltaBytes != 0 {
			partial.Events = append(partial.Events, &startEvent)
		}
		if cleanupEvent.DeltaBytes != 0 {
			partial.Paths = append(partial.Paths, tempPaths...)
			// Critical section to avoid loosing accounting info.
			util.EnterCriticalSection()
			defer util.ExitCriticalSection()
		}
		if td := self.split_metadata.TempDir(); td != "" {
			if err := os.RemoveAll(self.split_metadata.TempDir()); err != nil {
				partial.Errors = append(partial.Errors, err.Error())
			}
			if cleanupEvent.DeltaBytes != 0 {
				cleanupEvent.Timestamp = time.Now()
				partial.Events = append(partial.Events, &cleanupEvent)
			}
		}

		self.writePartialKill(partial)
		return partial
	}
}

func (self *Fork) cleanChunkTemp(partial *PartialVdrKillReport) *PartialVdrKillReport {
	temps := make([]string, 0, len(self.chunks))
	files := make([]string, 0, len(self.chunks))
	var start time.Time
	for _, chunk := range self.chunks {
		if tempPaths, err := chunk.metadata.enumerateTemp(); err != nil {
			return partial
		} else if filesPaths, err := chunk.metadata.enumerateFiles(); err != nil {
			return partial
		} else {
			if ts := chunk.metadata.getStartTime(); ts.After(start) {
				start = ts
			}
			temps = append(temps, tempPaths...)
			files = append(files, filesPaths...)
		}
	}
	if partial == nil {
		partial = new(PartialVdrKillReport)
	}
	partial.Chunks = true
	var startEvent, cleanupEvent VdrEvent
	if start.IsZero() {
		if len(partial.Events) > 0 {
			startEvent.Timestamp = partial.Events[len(partial.Events)-1].Timestamp
		}
	} else {
		startEvent.Timestamp = start
	}
	for _, p := range temps {
		util.Walk(p, func(tpath string, info os.FileInfo, err error) error {
			if err == nil {
				partial.Size += uint64(info.Size())
				partial.Count++
				startEvent.DeltaBytes += int64(info.Size())
				cleanupEvent.DeltaBytes -= int64(info.Size())
			} else {
				partial.Errors = append(partial.Errors, err.Error())
			}
			return nil
		})
	}
	for _, p := range files {
		util.Walk(p, func(tpath string, info os.FileInfo, err error) error {
			if err == nil {
				startEvent.DeltaBytes += int64(info.Size())
			} else {
				partial.Errors = append(partial.Errors, err.Error())
			}
			return nil
		})
	}
	// Add metadata file sizes.
	for _, chunk := range self.chunks {
		for _, md := range chunk.metadata.glob() {
			if info, err := os.Lstat(md); err != nil {
				partial.Errors = append(partial.Errors, err.Error())
			} else {
				startEvent.DeltaBytes += int64(info.Size())
			}
		}
	}

	if self.node.rt.Config.Debug {
		util.LogInfo("storage",
			"%d bytes of chunk files for %s",
			startEvent.DeltaBytes, self.node.GetFQName())
	}
	if startEvent.DeltaBytes != 0 {
		partial.Events = append(partial.Events, &startEvent)
	}
	if cleanupEvent.DeltaBytes != 0 {
		partial.Paths = append(partial.Paths, temps...)
		// Critical section to avoid loosing accounting info.
		util.EnterCriticalSection()
		defer util.ExitCriticalSection()
	}

	for _, chunk := range self.chunks {
		if td := chunk.metadata.TempDir(); td != "" {
			if err := os.RemoveAll(td); err != nil {
				partial.Errors = append(partial.Errors, err.Error())
			}
		}
	}
	if cleanupEvent.DeltaBytes != 0 {
		cleanupEvent.Timestamp = time.Now()
		partial.Events = append(partial.Events, &cleanupEvent)
	}

	self.writePartialKill(partial)
	return partial
}

func (self *Fork) cleanJoinTemp(partial *PartialVdrKillReport) *PartialVdrKillReport {
	if tempPaths, err := self.join_metadata.enumerateTemp(); err != nil {
		return partial
	} else if filesPaths, err := self.join_metadata.enumerateFiles(); err != nil {
		return partial
	} else {
		if partial == nil {
			partial = new(PartialVdrKillReport)
		}
		partial.Join = true
		var startEvent, cleanupEvent VdrEvent
		if start := self.join_metadata.getStartTime(); start.IsZero() {
			if len(partial.Events) > 0 {
				startEvent.Timestamp = partial.Events[len(partial.Events)-1].Timestamp
			}
		} else {
			startEvent.Timestamp = start
		}

		for _, p := range tempPaths {
			util.Walk(p, func(tpath string, info os.FileInfo, err error) error {
				if err == nil {
					partial.Size += uint64(info.Size())
					partial.Count++
					startEvent.DeltaBytes += int64(info.Size())
					cleanupEvent.DeltaBytes -= int64(info.Size())
				} else {
					partial.Errors = append(partial.Errors, err.Error())
				}
				return nil
			})
		}
		for _, p := range filesPaths {
			util.Walk(p, func(tpath string, info os.FileInfo, err error) error {
				if err == nil {
					startEvent.DeltaBytes += int64(info.Size())
				} else {
					partial.Errors = append(partial.Errors, err.Error())
				}
				return nil
			})
		}
		for _, md := range self.join_metadata.glob() {
			if info, err := os.Lstat(md); err == nil {
				startEvent.DeltaBytes += int64(info.Size())
			} else {
				partial.Errors = append(partial.Errors, err.Error())
			}
		}
		if self.node.rt.Config.Debug {
			util.LogInfo("storage",
				"%d bytes of join files for %s",
				startEvent.DeltaBytes, self.node.GetFQName())
		}

		if startEvent.DeltaBytes != 0 {
			partial.Events = append(partial.Events, &startEvent)
		}
		if cleanupEvent.DeltaBytes != 0 {
			partial.Paths = append(partial.Paths, tempPaths...)
			// Critical section to avoid loosing accounting info.
			util.EnterCriticalSection()
			defer util.ExitCriticalSection()
		}
		if td := self.join_metadata.TempDir(); td != "" {
			if err := os.RemoveAll(td); err != nil {
				partial.Errors = append(partial.Errors, err.Error())
			}
			if cleanupEvent.DeltaBytes != 0 {
				cleanupEvent.Timestamp = time.Now()
				partial.Events = append(partial.Events, &cleanupEvent)
			}
		}

		self.writePartialKill(partial)

		return partial
	}
}

// Clean up all files (if volatile) or chunk files (otherwise).  Must be called
// through partialVdrKill in order to ensure accounting information is
// correctly preserved.
func (self *Fork) vdrKill(partialKill *PartialVdrKillReport) *VDRKillReport {
	if self.node.rt.Config.VdrMode == "disable" {
		return nil
	}
	if killReport, ok := self.getVdrKillReport(); ok {
		return killReport
	}

	var killPaths []string
	// For volatile nodes, kill fork-level files.
	if self.node.rt.overrides.GetOverride(self.node, "force_volatile", self.node.volatile).(bool) {
		rep, _ := self.vdrKillSome(partialKill, true)
		return rep
	} else if self.Split() && self.node.rt.overrides.GetOverride(self.node, "force_volatile", true).(bool) {
		// If the node splits, kill chunk-level files.
		// Must check for split here, otherwise we'll end up deleting
		// output files of non-volatile nodes because single-chunk nodes
		// get their output redirected to the one chunk's files path.
		for _, chunk := range self.chunks {
			if paths, err := chunk.metadata.enumerateFiles(); err == nil {
				killPaths = append(killPaths, paths...)
			}
		}
	}
	killReport := &VDRKillReport{
		Paths: make([]string, 0, len(killPaths)),
	}
	// Sum up the path size.
	for _, p := range killPaths {
		util.Walk(p, func(_ string, info os.FileInfo, err error) error {
			if err == nil {
				killReport.Size += uint64(info.Size())
				killReport.Count++
			} else {
				killReport.Errors = append(killReport.Errors, err.Error())
			}
			return nil
		})
		killReport.Paths = append(killReport.Paths, p)
	}
	// Critical section to avoid loosing accounting info.
	util.EnterCriticalSection()
	defer util.ExitCriticalSection()
	// Actually delete the paths.
	for _, p := range killPaths {
		os.RemoveAll(p)
	}
	// update timestamp to mark actual kill time
	killReport.Timestamp = util.Timestamp()
	if killReport.Size > 0 {
		killReport.Events = append(killReport.Events, &VdrEvent{
			Timestamp:  time.Now().Round(time.Second),
			DeltaBytes: -int64(killReport.Size),
		})
	}
	if partialKill != nil {
		if self.node.rt.Config.Debug {
			util.LogInfo("storage",
				"VDR kill on %s with %d storage events in the partial report.",
				self.node.GetFQName(), len(partialKill.Events))
		}
		killReport = mergeVDRKillReports([]*VDRKillReport{killReport, &partialKill.VDRKillReport})
		self.deletePartialKill()
	} else {
		if self.node.rt.Config.Debug {
			util.LogInfo("storage",
				"VDR kill on %s with no partial report.",
				self.node.GetFQName())
		}
	}
	self.metadata.Write(VdrKill, killReport)
	return killReport
}

/* Is self or any of its ancestors symlinked? */
func (self *Node) vdrCheckSymlink() (string, error) {

	/* Nope! Got all the way to the top.
	 * (We don't care of the top-level directory is a symlink)
	 */
	if self.parent == nil {
		return "", nil
	}
	statinfo, err := os.Lstat(self.path)

	if err != nil {
		return "", err
	}
	/* Yep! Found a symlink */
	if (statinfo.Mode() & os.ModeSymlink) != 0 {
		return self.path, nil
	}

	return self.parent.getNode().vdrCheckSymlink()
}

func (self *Node) vdrKill() (*VDRKillReport, bool) {

	/*
	 * Refuse to VDR a node if it, or any of its ancestors are symlinked.
	 */
	if symlink, err := self.vdrCheckSymlink(); symlink != "" {
		util.LogInfo("runtime", "Refuse to VDR across a symlink %s: %v", symlink, self.fqname)
		return nil, true
	} else if err != nil {
		util.LogError(err, "runtime", "Error reading node directory: %v", self.fqname)
		return nil, false
	}

	allDone := true
	killReports := make([]*VDRKillReport, 0, len(self.forks))
	for _, fork := range self.forks {
		if report, done := fork.partialVdrKill(); !done {
			allDone = false
		} else if report != nil {
			killReports = append(killReports, report)
		}
	}
	return mergeVDRKillReports(killReports), allDone
}

type StorageEvent struct {
	Timestamp time.Time
	Delta     int64
	Name      string
}

func NewStorageEvent(timestamp time.Time, delta int64, fqname string) *StorageEvent {
	self := &StorageEvent{}
	self.Timestamp = timestamp
	self.Delta = delta
	self.Name = fqname
	return self
}

type StorageEventByTimestamp []*StorageEvent

func (self StorageEventByTimestamp) Len() int {
	return len(self)
}

func (self StorageEventByTimestamp) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self StorageEventByTimestamp) Less(i, j int) bool {
	return self[i].Timestamp.Before(self[j].Timestamp)
}

// Combine duplicate storage events.
func (self StorageEventByTimestamp) Collapse() StorageEventByTimestamp {
	if len(self) <= 1 {
		return self
	}
	sort.Sort(self)
	result := make(StorageEventByTimestamp, 1, len(self))
	result[0] = self[0]
	for _, e := range self[1:] {
		last := result[len(result)-1]
		if last.Name == e.Name && e.Timestamp.Sub(last.Timestamp) < time.Second {
			last.Delta += e.Delta
		} else {
			result = append(result, e)
		}
	}
	return result
}

// this is due to the fact that the VDR bytes/total bytes
// reported at the fork level is the sum of chunk + split
// + join plus any additional files.  The additional
// files that are unique to the fork cannot be resolved
// unless you sub out chunk/split/join and then child
// stages.
type ForkStorageEvent struct {
	Name         string
	ChildNames   []string
	TotalBytes   uint64
	ChunkBytes   uint64
	ForkBytes    uint64
	ForkVDRBytes uint64
	Timestamp    time.Time
	VDRTimestamp time.Time
}

func NewForkStorageEvent(timestamp time.Time, totalBytes uint64, vdrBytes uint64, fqname string) *ForkStorageEvent {
	self := &ForkStorageEvent{ChildNames: []string{}}
	self.Name = fqname
	self.TotalBytes = totalBytes // sum total of bytes in fork and children
	self.ForkBytes = self.TotalBytes
	self.ForkVDRBytes = vdrBytes // VDR bytes in forkN/files
	self.Timestamp = timestamp
	return self
}

func (self *Pipestance) VDRKill() *VDRKillReport {
	var killReports []*VDRKillReport
	if nodes := self.node.allNodes(); len(nodes) > 0 {
		killReports = make([]*VDRKillReport, 0, len(nodes))
		for _, node := range self.node.allNodes() {
			if killReport, _ := node.vdrKill(); killReport != nil {
				killReports = append(killReports, killReport)
			}
		}
	}
	killReport := mergeVDRKillReports(killReports)
	self.metadata.Write(VdrKill, killReport)
	return killReport
}

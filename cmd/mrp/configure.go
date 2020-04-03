//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

// Martian pipeline runner.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"runtime/trace"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/martian-lang/martian/martian/api"
	"github.com/martian-lang/martian/martian/core"
	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"

	"github.com/dustin/go-humanize"
	"github.com/martian-lang/docopt.go"
)

// We need to be able to recreate pipestances and share the new pipestance
// object between the runloop and the UI.
type pipestanceHolder struct {
	pipestance       *core.Pipestance
	factory          core.PipestanceFactory
	info             *api.PipestanceInfo
	maxRetries       int
	remainingRetries int
	authKey          string
	enableUI         bool
	showedFailed     bool
	lastRegister     time.Time
	cleanupLock      sync.Mutex
	lock             sync.Mutex
	readOnly         bool
	retryWait        time.Duration
	server           *http.Server
	lastLogCheck     time.Time
}

func (self *pipestanceHolder) getPipestance() *core.Pipestance {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.pipestance
}

func (self *pipestanceHolder) setPipestance(newPipe *core.Pipestance) {
	self.pipestance = newPipe
}

// Decrements the retry count if it is positive, or returns false.
func (self *pipestanceHolder) consumeRetry() bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.remainingRetries <= 0 {
		return false
	} else {
		self.remainingRetries--
		return true
	}
}

// Restart the pipestance and set remaining retries back to maximum.
func (self *pipestanceHolder) reset(ctx context.Context) error {
	self.lock.Lock()
	self.remainingRetries = self.maxRetries
	self.showedFailed = false
	self.lock.Unlock()
	return self.restart(ctx)
}

// Restart the pipestance.
func (self *pipestanceHolder) restart(outerCtx context.Context) error {
	ctx, task := trace.NewTask(outerCtx, "restart")
	defer task.End()
	if self.readOnly {
		return fmt.Errorf("mrp instances started with --inspect cannot restart pipelines.")
	}
	self.lock.Lock()
	defer self.lock.Unlock()
	ps, err := self.factory.ReattachToPipestance(ctx)
	if err == nil {
		err = ps.Reset()
		if err != nil {
			ps.Unlock()
			return err
		}
		ps.LoadMetadata(ctx)
		self.setPipestance(ps)
	}
	return err
}

func (self *pipestanceHolder) UpdateState(state core.MetadataState) chan struct{} {
	oldState := self.info.State
	self.info.State = state
	if oldState != state || time.Since(self.lastRegister) > 10*time.Minute {
		return self.Register()
	}
	return nil
}

func (self *pipestanceHolder) UpdateError(message string) {
	self.lock.Lock()
	self.info.LastErrorMessage = message
	self.lock.Unlock()
}

func (self *pipestanceHolder) Register() chan struct{} {
	if !self.enableUI {
		return nil
	}
	if enterpriseHost := os.Getenv("MARTIAN_ENTERPRISE"); enterpriseHost != "" {
		u := url.URL{
			Scheme: "http",
			Host:   enterpriseHost,
			Path:   api.QueryRegisterEnterprise,
		}
		form := self.info.AsForm()
		form.Set("authkey", self.authKey)
		self.lastRegister = time.Now()
		complete := make(chan struct{})
		go func() {
			defer close(complete)
			if res, err := http.PostForm(u.String(), form); err == nil {
				defer func() {
					// Clear out the response buffer and close it.
					// Ignore the content.
					b := make([]byte, 1024)
					for _, err := res.Body.Read(b); err == nil; _, err = res.Body.Read(b) {
					}
					res.Body.Close()
				}()
				if res.StatusCode >= http.StatusBadRequest {
					util.LogInfo("mrenter", "Registration failed with %s.", res.Status)
				}
			} else {
				util.LogError(err, "mrenter", "Registration to %s failed", u.Host)
			}
		}()
		return complete
	} else {
		return nil
	}
}

func (self *pipestanceHolder) HandleSignal(os.Signal) {
	if self.enableUI && !self.readOnly {
		// Don't use getPipestance() here, because that can result in
		// a deadlock.  getPipestance takes the lock protecting
		// self.pipestance so you don't get a stale pointer if it's in
		// the process of being switched out.  However, the pipestance
		// is changed only when it is restarted during auto-restart.
		// Part of instantiation involves the pipestance registering a
		// signal handler, which takes a lock on the signal handler
		// mutex, which this method executes inside, so that restart
		// will never complete.  Also, it won't matter for what we then
		// use the pipestance for.
		if ps := self.pipestance; ps != nil {
			ps.ClearUiPort()
		}
	}
	if srv := self.server; srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}
}

const WAIT_SECS = 6

// Remove any buffered items from the channel.
func flushChannel(c <-chan struct{}) {
	for {
		select {
		case <-c:
		default:
			return
		}
	}
}

//=============================================================================
// Pipestance runner.
//=============================================================================
func runLoop(pipestanceBox *pipestanceHolder, stepSecs time.Duration,
	vdrMode core.VdrMode, noExit bool, localJobDone <-chan struct{}) {
	pipestanceBox.getPipestance().LoadMetadata(context.Background())

	t := time.NewTimer(0)
	if !t.Stop() {
		<-t.C
	}
	for {
		flushChannel(localJobDone)
		hadProgress := loopBody(pipestanceBox, vdrMode, noExit)

		if !hadProgress {
			// Wait for a either stepSecs or until a local job finishes.
			t.Reset(stepSecs)
			select {
			case <-t.C:
			case <-localJobDone:
				if !t.Stop() {
					<-t.C
				}
			}
			if !pipestanceBox.lastLogCheck.IsZero() &&
				time.Since(pipestanceBox.lastLogCheck) > time.Minute {
				if err := util.VerifyLogFile(); err != nil {
					util.PrintError(err, "runtime",
						"Pipestance directory seems to have disappeared.")
					util.Suicide(false)
				}
			}
			// During the idle portion of the run loop is a good time to
			// run the GC.  We do this after the sleep because StepNodes
			// launches jobs on goroutines, and it's better to give them
			// time to get to the point where they're waiting on the
			// subprocess (or, in cluster mode, possibly finish waiting)
			// before the GC runs.
			runtime.GC()
		}
	}
}

func loopBody(pipestanceBox *pipestanceHolder,
	vdrMode core.VdrMode, noExit bool) bool {
	pipestance := pipestanceBox.getPipestance()
	ctx, task := trace.NewTask(context.Background(), "update")
	defer task.End()
	pipestance.RefreshState(ctx)

	// Check for completion states.
	state := pipestance.GetState(ctx)
	if state == core.Complete || state == core.DisabledState {
		pipestanceBox.UpdateState(state.Prefixed(core.CleanupPrefix))
		cleanupCompleted(pipestance, pipestanceBox, vdrMode, noExit, ctx)
		return false
	} else if state == core.Failed {
		if pipestanceBox.showedFailed {
			pipestanceBox.UpdateState(state)
		} else {
			pipestanceBox.UpdateState(state.Prefixed(core.CleanupPrefix))
		}
		if !attemptRetry(pipestance, pipestanceBox, ctx) {
			pipestance.Unlock()
			cleanupFailed(pipestance, pipestanceBox, noExit, ctx)
		}
		return false
	} else {
		pipestanceBox.UpdateState(state)
		// If we went from failed to something else, allow the failure message to
		// be shown once if we fail again.
		pipestanceBox.showedFailed = false

		// Check job heartbeats.
		pipestance.CheckHeartbeats(ctx)

		// Step all nodes.
		return pipestance.StepNodes(ctx)
	}
}

func attemptRetry(pipestance *core.Pipestance, pipestanceBox *pipestanceHolder,
	outerCtx context.Context) bool {
	ctx, task := trace.NewTask(outerCtx, "attemptRetry")
	defer task.End()

	if pipestanceBox.readOnly {
		return false
	}
	canRetry := false
	var transient_log string
	if pipestanceBox.consumeRetry() {
		canRetry, transient_log = pipestance.IsErrorTransient()
	}
	if transient_log != "" && !pipestanceBox.showedFailed {
		pipestanceBox.UpdateError(transient_log)
	}
	if canRetry {
		pipestanceBox.UpdateState(core.Failed.Prefixed(core.RetryPrefix))
		if pipestanceBox.retryWait > 0 {
			util.LogInfo("runtime",
				"Waiting %s before attempting a retry.",
				pipestanceBox.retryWait.String())
			time.Sleep(pipestanceBox.retryWait)
		}
		// Heartbeat failures often come in clusters.  Look for any others
		// which have come in since failure was detected so that all of
		// those failures get batched up into a single retry.
		pipestance.RefreshState(ctx)
		pipestance.CheckHeartbeats(ctx)
		// Check that no non-transient failures happened in the mean time.
		canRetry, transient_log = pipestance.IsErrorTransient()
		if !canRetry {
			if transient_log != "" && !pipestanceBox.showedFailed {
				pipestanceBox.UpdateError(transient_log)
			}
			return false
		}

		pipestance.Unlock()
		if transient_log != "" {
			util.LogInfo("runtime",
				"Transient error detected.  Log content:\n\n%s\n",
				transient_log)
		}
		util.LogInfo("runtime", "Attempting retry.")
		if err := pipestanceBox.restart(ctx); err != nil {
			util.LogInfo("runtime", "Retry failed:\n%v\n", err)
			// Let the next loop around actually handle the failure.
		}
	}
	return canRetry
}

func cleanupCompleted(pipestance *core.Pipestance, pipestanceBox *pipestanceHolder,
	vdrMode core.VdrMode, noExit bool, ctx context.Context) {
	r := trace.StartRegion(ctx, "cleanupCompleted")
	defer r.End()
	if pipestanceBox.readOnly {
		pipestanceBox.UpdateState(core.Complete)
		util.Println("Pipestance completed successfully, staying alive because --inspect given.\n")
		return
	}
	pipestanceBox.cleanupLock.Lock()
	defer pipestanceBox.cleanupLock.Unlock()
	if vdrMode == core.VdrDisable {
		util.LogInfo("runtime", "VDR disabled. No files killed.")
	} else {
		killReport := pipestance.VDRKill()
		util.LogInfo("runtime", "VDR killed %d files, %s.",
			killReport.Count, humanize.Bytes(killReport.Size))
	}
	trace.WithRegion(ctx, "PostProcess", pipestance.PostProcess)
	pipestance.Unlock()
	pipestance.OnFinishHook(ctx)
	updateComplete := pipestanceBox.UpdateState(core.Complete)
	if noExit {
		util.Println("Pipestance completed successfully, staying alive because --noexit given.\n")
		runtime.GC()
		// Don't return; otherwise we'll repeatedly try to clean up.
		go completedLogCheck()
		runtime.Goexit()
	} else {
		if pipestanceBox.enableUI {
			// Give time for web ui client to get last update.
			util.Println("Waiting %d seconds for UI to do final refresh.", WAIT_SECS)
			time.Sleep(time.Second * time.Duration(WAIT_SECS))
		}
		util.Println("Pipestance completed successfully!\n")
		if updateComplete != nil {
			<-updateComplete
		}
		util.Suicide(true)
	}
}

// Check to see if the pipestance directory was deleted.  If it was, exit.
// This is used when mrp is lanuched with `--noexit` to make sure mrp doesn't
// outlive its usefulness.
func completedLogCheck() {
	for {
		time.Sleep(time.Minute)
		if err := util.VerifyLogFile(); err != nil {
			util.PrintError(err, "runtime",
				"Pipestance directory seems to have disappeared.")
			util.Suicide(true)
		}
	}
}

func cleanupFailed(pipestance *core.Pipestance, pipestanceBox *pipestanceHolder,
	noExit bool, ctx context.Context) {
	r := trace.StartRegion(ctx, "cleanupFailed")
	defer r.End()
	if pipestanceBox.readOnly {
		pipestanceBox.UpdateState(core.Failed)
		if !pipestanceBox.showedFailed {
			pipestanceBox.showedFailed = true
			util.Println("Pipestance failed, staying alive because --inspect given.\n")
		}
		return
	}
	pipestanceBox.cleanupLock.Lock()
	defer pipestanceBox.cleanupLock.Unlock()
	defer func() { pipestanceBox.showedFailed = true }()
	var serverUpdate chan struct{}
	if !pipestanceBox.showedFailed {
		pipestance.OnFinishHook(ctx)
		if _, _, _, log, kind, errPaths := pipestance.GetFatalError(); kind == "assert" {
			// Print preflight check failures.
			util.Println("\n[%s] %s\n", "error", log)
			if log != "" {
				pipestanceBox.UpdateError(log)
			} else {
				pipestanceBox.UpdateError(fmt.Sprintf("Assertion failed.  See logs at:\n%s",
					strings.Join(errPaths, "\n")))
			}
			serverUpdate = pipestanceBox.UpdateState(core.Failed)
			if serverUpdate != nil {
				<-serverUpdate
			}
			util.Suicide(false)
		} else if len(errPaths) > 0 {
			// Build relative path to _errors file
			errPath, _ := filepath.Rel(filepath.Dir(pipestance.GetPath()), errPaths[0])

			if log != "" {
				util.Println("\n[%s] Pipestance failed. Error log at:\n%s\n\nLog message:\n%s\n",
					"error", errPath, log)
				pipestanceBox.UpdateError(fmt.Sprintf("Pipestance failed. Full log at:\n%s\n%s",
					strings.Join(errPaths, "\n"), log))
			} else {
				// Print path to _errors metadata file in failed stage.
				util.Println("\n[%s] Pipestance failed. Please see log at:\n%s\n", "error", errPath)
				pipestanceBox.UpdateError(fmt.Sprintf("Pipestance failed. See logs at:\n%s",
					strings.Join(errPaths, "\n")))
			}
			serverUpdate = pipestanceBox.UpdateState(core.Failed)
		}
	} else {
		serverUpdate = pipestanceBox.UpdateState(core.Failed)
	}
	if noExit {
		// If pipestance failed but we're staying alive, only print this once
		// as long as we stay failed.
		if !pipestanceBox.showedFailed {
			util.Println("Pipestance failed, staying alive because --noexit given.\n")
		}
	} else {
		if pipestanceBox.enableUI {
			// Give time for web ui client to get last update.
			util.Println("Waiting %d seconds for UI to do final refresh.", WAIT_SECS)
			time.Sleep(time.Second * time.Duration(WAIT_SECS))
			util.Println("Pipestance failed. Use --noexit option to keep UI running after failure.\n")
		}
		if serverUpdate != nil {
			<-serverUpdate
		}
		util.Suicide(false)
	}
}

// List of environment variables which might be useful in debugging.
var loggedEnvs = map[string]bool{
	"COMMD_PORT":   true,
	"CWD":          true,
	"ENVIRONMENT":  true, // SGE
	"EXE":          true,
	"HOME":         true,
	"HOST":         true,
	"HOSTNAME":     true,
	"HOSTTYPE":     true, // LSF
	"HYDRA_ROOT":   true,
	"LANG":         true,
	"LIBRARY_PATH": true,
	"LOGNAME":      true,
	"NHOSTS":       true, // SGE
	"NQUEUES":      true, // SGE
	"NSLOTS":       true, // SGE
	"PATH":         true,
	"PID":          true,
	"PWD":          true,
	"SHELL":        true,
	"SHLVL":        true,
	"SPOOLDIR":     true, // LSF
	"TERM":         true,
	"TMPDIR":       true,
	"USER":         true,
	"WAFDIR":       true,
	"_":            true,
}

// List of environment variable prefixes which might be useful in debugging.
// These are accepted for variables of the form "KEY_*"
var loggedEnvPrefixes = map[string]bool{
	"BASH":    true,
	"CONDA":   true,
	"DYLD":    true, // Linker
	"EC2":     true,
	"EGO":     true, // LSF
	"JAVA":    true,
	"JOB":     true, // SGE
	"LC":      true,
	"LD":      true, // Linker
	"LS":      true, // LSF
	"LSB":     true, // LSF
	"LSF":     true, // LSF
	"MRO":     true, // Martian
	"MALLOC":  true, // jemalloc
	"MARTIAN": true,
	"MYSYS2":  true, // Anaconda
	"PBS":     true, // PBS
	"PD":      true,
	"RUST":    true,
	"SBATCH":  true, // Slurm
	"SELINUX": true, // Linux
	"SGE":     true,
	"SLURM":   true,
	"SSH":     true,
	"TENX":    true,
	"XDG":     true,
}

// Returns true if the environment variable should be logged.
func logEnv(env string) bool {
	if loggedEnvs[env] {
		return true
	}
	// Various important PYTHON environment variables don't have a _ separator.
	if strings.HasPrefix(env, "PYTHON") {
		return true
	}
	if idx := strings.Index(env, "_"); idx >= 0 {
		return loggedEnvPrefixes[env[:idx]]
	} else {
		return loggedEnvPrefixes[env]
	}
}

type mrpConfiguration struct {
	psid           string
	invocationPath string
	pipestancePath string
	tags           []string
	readOnly       bool
	retries        int
	retryWait      time.Duration
	enableUI       bool
	config         core.RuntimeOptions
	mroPaths       []string
	mroVersion     string
	uiport         string
	authKey        string
	requireAuth    bool
	noExit         bool
}

func parseMroFlags(opts map[string]interface{}, doc string, martianOptions []string, martianArguments []string) {
	// Parse doc string for accepted arguments
	// All accepted arguments start with `--` and contain only lowercase
	// letters and dashes.
	allowedOptions := make(map[string]struct{}, strings.Count(doc, "\n"))
	dd := doc
	for i := strings.Index(dd, "--"); i >= 0; i = strings.Index(dd, "--") {
		dd = dd[i:]
		if j := strings.IndexAny(dd, "= "); j > 2 {
			allowedOptions[dd[:j]] = struct{}{}
			dd = dd[j:]
		} else {
			break
		}
	}
	// Filter options to ones which are allowed.
	newMartianOptions := make([]string, 0, len(martianOptions)+len(martianArguments))
	for _, option := range martianOptions {
		o := option
		if i := strings.IndexRune(option, '='); i > 0 {
			o = option[:i]
		}
		if _, ok := allowedOptions[o]; ok {
			newMartianOptions = append(newMartianOptions, option)
		}
	}
	newMartianOptions = append(newMartianOptions, martianArguments...)
	defopts, err := docopt.Parse(doc, newMartianOptions, false, "", true, false)
	if err != nil {
		util.LogInfo("environ", "EnvironError: MROFLAGS environment variable has incorrect format\n")
		fmt.Println(doc)
		os.Exit(1)
	}
	for id, defval := range defopts {
		// Only use options
		if !strings.HasPrefix(id, "--") {
			continue
		}
		if val, ok := opts[id].(bool); (ok && !val) || (!ok && opts[id] == nil) {
			opts[id] = defval
		}
	}
}

func configure() mrpConfiguration {
	//=========================================================================
	// Commandline argument and environment variables.
	//=========================================================================
	// Parse commandline.
	doc := `Martian Pipeline Runner.

Usage:
    mrp <call.mro> <pipestance_name> [options]
    mrp -h | --help | --version

Options:
    --jobmode=MODE      Job manager to use. Valid options:
                            local (default), sge, lsf, or a .template file
    --localcores=NUM    Set max cores the pipeline may request at one time.
                            Only applies to local jobs.
    --localmem=NUM      Set max GB the pipeline may request at one time.
                            Only applies to local jobs.
    --localvmem=NUM     Set max virtual address space in GB for the pipeline.
                            Only applies to local jobs.
    --mempercore=NUM    Reserve enough threads for each job to ensure enough
                        memory will be available, assuming each core on your
                        cluster has at least this much memory available.
                            Only applies in cluster jobmodes.
    --maxjobs=NUM       Set max jobs submitted to cluster at one time.
                            Only applies in cluster jobmodes.
    --jobinterval=NUM   Set delay between submitting jobs to cluster, in ms.
                            Only applies in cluster jobmodes.
    --limit-loadavg     Avoid scheduling jobs when the system loadavg is high.
                            Only applies to local jobs.

    --vdrmode=MODE      Enables Volatile Data Removal. Valid options:
                            post, rolling (default), or disable

    --nopreflight       Skips preflight stages.
    --strict=MODE       Determines how mrp reports cases where it needs to fall
                        back on backwards compatibility for mro checks. Allowed
                        values: disable (default), log, alarm, or error.
    --uiport=NUM        Serve UI at http://<hostname>:NUM
    --disable-ui        Do not serve the UI.
    --disable-auth      Do not require authentication for reading the web UI.
    --require-auth      Always require authentication (this is the default
                        if --uiport is not set).
    --auth-key=KEY      Set the authentication key required for accessing the
                        web UI.
    --noexit            Keep UI running after pipestance completes or fails.
    --onfinish=EXEC     Run this when pipeline finishes, success or fail.
    --zip               Zip metadata files after pipestance completes.
    --tags=TAGS         Tag pipestance with comma-separated key:value pairs.

    --profile=MODE      Enables stage performance profiling. Valid options:
                            disable (default), cpu, mem, or line
    --stackvars         Print local variables in stage code stack trace.
    --monitor           Kill jobs that exceed requested memory resources.
    --inspect           Inspect pipestance without resetting failed stages.
    --debug             Enable debug logging for local job manager.
    --stest             Substitute real stages with stress-testing stage.
    --autoretry=NUM     Automatically retry failed runs up to NUM times.
    --retry-wait=SECS   Wait SECS seconds after a failure before attempting
                        automatic retry.  Defaults to 1 second.
    --overrides=JSON    JSON file supplying custom run conditions per stage.
    --psdir=PATH        The path to the pipestance directory.  The default is
                        to use <pipestance_name>.
    --never-local       Ignore 'local' modifiers on non-preflight stages.

    -h --help           Show this message.
    --version           Show version.`
	c := mrpConfiguration{
		config:  core.DefaultRuntimeOptions(),
		retries: core.DefaultRetries(),
	}
	config := &c.config
	opts, _ := docopt.Parse(doc, nil, true, config.MartianVersion, false)
	util.Println("Martian Runtime - %s", config.MartianVersion)
	util.LogInfo("build  ", "Built with Go version %s", runtime.Version())
	util.LogInfo("cmdline", "%s", strings.Join(os.Args, " "))
	util.LogInfo("pid    ", "%d", os.Getpid())

	for _, env := range os.Environ() {
		pair := strings.Split(env, "=")
		if len(pair) == 2 && logEnv(pair[0]) {
			util.LogInfo("environ", "%s", env)
		}
	}

	martianFlags := ""
	if martianFlags = os.Getenv("MROFLAGS"); len(martianFlags) > 0 {
		martianOptions := strings.Split(martianFlags, " ")
		parseMroFlags(opts, doc, martianOptions, []string{"call.mro", "pipestance"})
		util.LogInfo("environ", "MROFLAGS=%s", martianFlags)
	}

	if value := opts["--strict"]; value != nil {
		level := syntax.ParseEnforcementLevel(value.(string))
		syntax.SetEnforcementLevel(level)
		util.LogInfo("options", "--strict=%s", level.String())
	}

	// Requested cores and memory.
	if value := opts["--localcores"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			config.LocalCores = value
			util.LogInfo("options", "--localcores=%d", config.LocalCores)
		} else {
			util.PrintError(err, "options",
				"Could not parse --localcores value \"%s\"", opts["--localcores"].(string))
			os.Exit(1)
		}
	}
	if value := opts["--localmem"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			config.LocalMem = value
			util.LogInfo("options", "--localmem=%d", config.LocalMem)
		} else {
			util.PrintError(err, "options",
				"Could not parse --localmem value \"%s\"", opts["--localmem"].(string))
			os.Exit(1)
		}
	}
	if value := opts["--localvmem"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			config.LocalVMem = value
			util.LogInfo("options", "--localvmem=%d", config.LocalVMem)
		} else {
			util.PrintError(err, "options",
				"Could not parse --localvmem value \"%s\"", opts["--localvmem"].(string))
			os.Exit(1)
		}
	}
	if value := opts["--mempercore"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			config.MemPerCore = value
			util.LogInfo("options", "--mempercore=%d", config.MemPerCore)
		} else {
			util.PrintError(err, "options",
				"Could not parse --mempercore value \"%s\"", opts["--mempercore"].(string))
			os.Exit(1)
		}
	}

	// Special to resources mappings
	if value := os.Getenv("MRO_JOBRESOURCES"); len(value) > 0 {
		config.ResourceSpecial = value
		util.LogInfo("options", "MRO_JOBRESOURCES=%s", config.ResourceSpecial)
	}

	// Flag for full stage reset, default is chunk-granular
	if value := os.Getenv("MRO_FULLSTAGERESET"); len(value) > 0 {
		config.FullStageReset = true
		util.LogInfo("options", "MRO_FULLSTAGERESET=true")
	}

	// Compute MRO path.
	mro_dir, _ := filepath.Abs(path.Dir(os.Args[1]))
	c.mroPaths = util.ParseMroPath(mro_dir)
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		c.mroPaths = util.ParseMroPath(value)
	}
	c.mroVersion, _ = util.GetMroVersion(c.mroPaths)
	util.LogInfo("environ", "MROPATH=%s", util.FormatMroPath(c.mroPaths))
	util.LogInfo("version", "MRO Version=%s", c.mroVersion)

	// Compute job manager.
	if value := opts["--jobmode"]; value != nil {
		config.JobMode = value.(string)
	}
	util.LogInfo("options", "--jobmode=%s", config.JobMode)

	if value := opts["--never-local"]; value != nil {
		if nl, ok := value.(bool); ok && nl {
			config.NeverLocal = true
			util.LogInfo("options", "--never-local")
		}
	}

	if config.JobMode != "local" {
		// Max parallel jobs.
		config.MaxJobs = 64
		if value := opts["--maxjobs"]; value != nil {
			if value, err := strconv.Atoi(value.(string)); err == nil {
				config.MaxJobs = value
			} else {
				util.PrintError(err, "options",
					"Could not parse --maxjobs value \"%s\"",
					opts["--maxjobs"].(string))
				os.Exit(1)
			}
		}
		util.LogInfo("options", "--maxjobs=%d", config.MaxJobs)

		// frequency (in milliseconds) that jobs will be sent to the queue
		// (this is a minimum bound, as it may take longer to emit jobs)
		config.JobFreqMillis = 100
		if value := opts["--jobinterval"]; value != nil {
			if value, err := strconv.Atoi(value.(string)); err == nil {
				config.JobFreqMillis = value
			} else {
				util.PrintError(err, "options",
					"Could not parse --jobinterval value \"%s\"",
					opts["--jobinterval"].(string))
				os.Exit(1)
			}
		}
		util.LogInfo("options", "--jobinterval=%d", config.JobFreqMillis)
	}

	// Compute vdrMode.
	if value := opts["--vdrmode"]; value != nil {
		config.VdrMode = core.VdrMode(value.(string))
	}
	util.LogInfo("options", "--vdrmode=%s", config.VdrMode)
	core.VerifyVDRMode(config.VdrMode)

	// Compute onfinish
	if value := opts["--onfinish"]; value != nil {
		config.OnFinishHandler = value.(string)
		core.VerifyOnFinish(config.OnFinishHandler)
	}

	// Compute profiling mode.
	if value := opts["--profile"]; value != nil {
		config.ProfileMode = core.ProfileMode(value.(string))
	}

	// Compute UI port.
	if value := opts["--uiport"]; value != nil {
		c.uiport = value.(string)
	} else {
		c.requireAuth = true
	}
	if len(c.uiport) > 0 {
		util.LogInfo("options", "--uiport=%s", c.uiport)
	}

	c.enableUI = (opts["--disable-ui"] == nil || !opts["--disable-ui"].(bool))
	if !c.enableUI {
		util.LogInfo("options", "--disable-ui")
	}
	if value := opts["--disable-auth"]; value != nil && value.(bool) {
		c.requireAuth = false
		util.LogInfo("options", "--disable-auth")
	}
	if value := opts["--require-auth"]; value != nil && value.(bool) {
		c.requireAuth = true
		util.LogInfo("options", "--require-auth")
	}
	if value := opts["--auth-key"]; value != nil {
		c.authKey = value.(string)
		util.LogInfo("options", "--auth-key=%s", c.authKey)
	} else if c.enableUI {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			util.PrintError(err, "webserv",
				"Failed to generate an authentication key.")
			os.Exit(1)
		}
		c.authKey = base64.RawURLEncoding.EncodeToString(key)
	}

	// Parse tags.
	if value := opts["--tags"]; value != nil {
		c.tags = util.ParseTagsOpt(value.(string))
	} else {
		c.tags = []string{}
	}
	for _, tag := range c.tags {
		util.LogInfo("options", "--tag='%s'", tag)
	}

	// Parse supplied overrides file.
	if v := opts["--overrides"]; v != nil {
		var err error
		config.Overrides, err = core.ReadOverrides(v.(string))
		if err != nil {
			util.PrintError(err, "startup", "Failed to parse overrides file")
			os.Exit(1)

		}
	}

	// Compute stackVars flag.
	config.StackVars = opts["--stackvars"].(bool)
	util.LogInfo("options", "--stackvars=%v", config.StackVars)

	config.Zip = opts["--zip"].(bool)
	util.LogInfo("options", "--zip=%v", config.Zip)

	config.LimitLoadavg = opts["--limit-loadavg"].(bool)
	util.LogInfo("options", "--limit-loadavg=%v", config.LimitLoadavg)

	c.noExit = opts["--noexit"].(bool)
	util.LogInfo("options", "--noexit=%v", c.noExit)

	config.SkipPreflight = opts["--nopreflight"].(bool)
	util.LogInfo("options", "--nopreflight=%v", config.SkipPreflight)

	c.psid = opts["<pipestance_name>"].(string)
	c.invocationPath = opts["<call.mro>"].(string)
	cwd, _ := os.Getwd()
	c.pipestancePath = path.Join(cwd, c.psid)
	if value := opts["--psdir"]; value != nil {
		if p, ok := value.(string); ok && p != "" {
			if filepath.IsAbs(p) {
				c.pipestancePath = p
			} else {
				c.pipestancePath = path.Join(cwd, p)
			}
		}
	}
	config.Monitor = opts["--monitor"].(bool)
	c.readOnly = opts["--inspect"].(bool)
	config.Debug = opts["--debug"].(bool)
	config.StressTest = opts["--stest"].(bool)
	if value := opts["--autoretry"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			c.retries = value
			util.LogInfo("options", "--autoretry=%d", c.retries)
		} else {
			util.PrintError(err, "options",
				"Could not parse --autoretry value \"%s\"", opts["--autoretry"].(string))
			os.Exit(1)
		}
	}
	if c.retries > 0 && config.FullStageReset {
		c.retries = 0
		util.Println(
			"\nWARNING: ignoring autoretry when MRO_FULLSTAGERESET is set.\n")
		util.LogInfo("options", "autoretry disabled due to MRO_FULLSTAGERESET.\n")
	}
	c.retryWait = time.Second
	if c.retries > 0 {
		if value := opts["--retry-wait"]; value != nil {
			if value, err := strconv.Atoi(value.(string)); err == nil {
				c.retryWait = time.Duration(value) * time.Second
				util.LogInfo("options", "--retry-wait=%d", c.retries)
			} else {
				util.PrintError(err, "options",
					"Could not parse --retry-wait value \"%s\"", opts["--retry-wait"].(string))
				os.Exit(1)
			}
		}
	}
	return c
}

func (pipestanceBox *pipestanceHolder) Configure(c *mrpConfiguration, invocationSrc string) (
	bool, *core.Runtime) {
	//=========================================================================
	// Configure Martian runtime.
	//=========================================================================
	rt := c.config.NewRuntime()

	factory := core.NewRuntimePipestanceFactory(rt,
		invocationSrc, c.invocationPath, c.psid, c.mroPaths, c.pipestancePath, c.mroVersion,
		nil, true, c.readOnly, c.tags)
	reattaching := false
	pipestance, err := factory.InvokePipeline()
	if err != nil {
		if _, ok := err.(*core.PipestanceExistsError); ok {
			if pipestance, err = factory.ReattachToPipestance(context.Background()); err == nil {
				c.config.MartianVersion, c.mroVersion, _ = pipestance.GetVersions()
				reattaching = true
			} else {
				util.DieIf(err)
			}
		} else {
			util.DieIf(err)
		}
	}
	pipestanceBox.pipestance = pipestance
	pipestanceBox.factory = factory
	pipestanceBox.maxRetries = c.retries
	pipestanceBox.remainingRetries = c.retries
	pipestanceBox.readOnly = c.readOnly
	pipestanceBox.retryWait = c.retryWait

	return reattaching, rt
}

func (c *mrpConfiguration) checkSpace() {
	if bSize, inodes, fstype, err := core.GetAvailableSpace(c.pipestancePath); err != nil {
		util.PrintError(err, "filesys", "Error reading pipestance filesystem information.")
	} else {
		util.LogInfo("filesys", "Pipestance path %s",
			c.pipestancePath)
		util.LogInfo("filesys", "Pipestance filesystem type %s",
			fstype)
		util.LogInfo("filesys", "%s and %s inodes available.",
			humanize.Bytes(bSize), humanize.Comma(int64(inodes)))
	}
	if fstype, opts, err := core.GetMountOptions(c.pipestancePath); err != nil {
		util.LogError(err, "filesys", "Could not read pipestance filesystem mount options.")
	} else {
		util.LogInfo("filesys", "Pipestance filesystem %s mount options: %s",
			fstype, opts)
	}

	binPath := util.RelPath("")
	if _, _, fstype, err := core.GetAvailableSpace(binPath); err != nil {
		util.PrintError(err, "filesys", "Error reading source filesystem information.")
	} else {
		util.LogInfo("filesys", "Bin path %s", binPath)
		util.LogInfo("filesys", "Bin filesystem type %s",
			fstype)
	}
	if fstype, opts, err := core.GetMountOptions(binPath); err != nil {
		util.LogError(err, "filesys", "Could not read source filesystem mount options.")
	} else {
		util.LogInfo("filesys", "Bin filesystem %s mount options: %s",
			fstype, opts)
	}

}

func (c *mrpConfiguration) getListener(hostname string, pipestanceBox *pipestanceHolder) net.Listener {
	// Attempt to open the UI port.  If the port was not automatically
	// assigned, fail mrp if it cannot be opened.  Otherwise, log a message
	// and continue.
	var listener net.Listener
	if c.enableUI {
		var err error
		dieWithoutUi := true
		if c.uiport == "" {
			c.uiport = "0"
			dieWithoutUi = false
		}
		if listener, err = net.Listen("tcp",
			fmt.Sprintf(":%s", c.uiport)); err != nil {
			util.PrintError(err, "webserv", "Cannot open port %s", c.uiport)
			if dieWithoutUi {
				os.Exit(1)
			} else {
				util.PrintError(err, "webserv", "UI disabled")
				listener = nil
			}
		} else {
			u := url.URL{
				Scheme: "http",
				Host:   listener.Addr().String(),
			}
			c.uiport = u.Port()
			u.Host = net.JoinHostPort(hostname, c.uiport)
			if c.authKey != "" {
				q := u.Query()
				q.Set("auth", c.authKey)
				u.RawQuery = q.Encode()
			}
			// Print this here because the log makes more sense when this appears before
			// the runloop messages start to appear.
			util.Println("Serving UI at %s\n", u.String())
			pipestanceBox.enableUI = true
			pipestanceBox.authKey = c.authKey
			util.RegisterSignalHandler(pipestanceBox)
			if !c.readOnly {
				pipestanceBox.pipestance.RecordUiPort(u.String())
			}
		}
	} else {
		util.LogInfo("webserv", "UI disabled.")
	}

	return listener
}

func main() {
	util.SetupSignalHandlers()
	c := configure()

	// Validate psid.
	util.DieIf(util.ValidateID(c.psid))

	// Get hostname and username.
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	user, err := user.Current()
	username := "unknown"
	if err == nil {
		username = user.Username
	}

	//=========================================================================
	// Invoke pipestance or Reattach if exists.
	//=========================================================================
	data, err := ioutil.ReadFile(c.invocationPath)
	util.DieIf(err)
	invocationSrc := string(data)

	// Attempt to reattach to the pipestance.
	var pipestanceBox pipestanceHolder
	reattaching, rt := pipestanceBox.Configure(&c, invocationSrc)
	pipestance := pipestanceBox.pipestance

	if !c.readOnly {
		// Start writing (including cached entries) to log file.
		util.LogTee(path.Join(c.pipestancePath, "_log"))
		pipestanceBox.lastLogCheck = time.Now()
	}
	c.checkSpace()

	uuid, _ := pipestanceBox.pipestance.GetUuid()
	listener := c.getListener(hostname, &pipestanceBox)

	//=========================================================================
	// Collect pipestance static info.
	//=========================================================================
	cwd, _ := os.Getwd()
	pipestanceBox.info = &api.PipestanceInfo{
		Hostname:     hostname,
		Username:     username,
		Cwd:          cwd,
		Binpath:      util.RelPath(os.Args[0]),
		Cmdline:      strings.Join(os.Args, " "),
		Pid:          os.Getpid(),
		Start:        pipestance.GetTimestamp(),
		Version:      c.config.MartianVersion,
		Pname:        pipestance.GetPname(),
		PsId:         c.psid,
		State:        pipestance.GetState(context.Background()),
		JobMode:      c.config.JobMode,
		MaxCores:     rt.JobManager.GetMaxCores(),
		MaxMemGB:     rt.JobManager.GetMaxMemGB(),
		InvokePath:   c.invocationPath,
		InvokeSource: invocationSrc,
		MroPath:      util.FormatMroPath(c.mroPaths),
		ProfileMode:  c.config.ProfileMode,
		Port:         c.uiport,
		MroVersion:   c.mroVersion,
		Uuid:         uuid,
		PsPath:       c.pipestancePath,
	}

	if reattaching {
		// If it already exists, try to reattach to it.
		if !c.readOnly {
			if err = pipestance.Reset(); err == nil {
				err = pipestance.RestartLocalJobs(c.config.JobMode)
			}
			util.DieIf(err)
		}
	} else if !c.config.SkipPreflight && !c.readOnly {
		util.Println("Running preflight checks (please wait)...")
	}

	//=========================================================================
	// Start web server.
	//=========================================================================
	if listener != nil {
		go runWebServer(listener, rt, &pipestanceBox, c.requireAuth)
	}

	//=========================================================================
	// Start run loop.
	//=========================================================================
	stepSecs := 3 * time.Second
	go runLoop(&pipestanceBox, stepSecs, c.config.VdrMode, c.noExit,
		rt.LocalJobManager.Done())

	// Let daemons take over.
	runtime.Goexit()
}

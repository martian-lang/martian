//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//

// Martian pipeline runner.
package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"martian/api"
	"martian/core"
	"martian/syntax"
	"martian/util"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

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
	lastRegister     time.Time
	cleanupLock      sync.Mutex
	lock             sync.Mutex
	readOnly         bool
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
func (self *pipestanceHolder) reset() error {
	self.lock.Lock()
	self.remainingRetries = self.maxRetries
	self.lock.Unlock()
	return self.restart()
}

// Restart the pipestance.
func (self *pipestanceHolder) restart() error {
	if self.readOnly {
		return fmt.Errorf("mrp instances started with --inspect cannot restart pipelines.")
	}
	self.lock.Lock()
	defer self.lock.Unlock()
	ps, err := self.factory.ReattachToPipestance()
	if err == nil {
		err = ps.Reset()
		if err != nil {
			ps.Unlock()
			return err
		}
		ps.LoadMetadata()
		self.setPipestance(ps)
	}
	return err
}

func (self *pipestanceHolder) UpdateState(state core.MetadataState) {
	oldState := self.info.State
	self.info.State = state
	if oldState != state || time.Since(self.lastRegister) > 10*time.Minute {
		self.Register()
	}
}

func (self *pipestanceHolder) Register() {
	if !self.enableUI {
		return
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
		go func() {
			if res, err := http.PostForm(u.String(), form); err == nil {
				if res.StatusCode >= http.StatusBadRequest {
					util.LogInfo("mrenter", "Registration failed with %s.", res.Status)
				}
			} else {
				util.LogError(err, "mrenter", "Registration to %s failed", u.Host)
			}
		}()
	}
}

const WAIT_SECS = 6

//=============================================================================
// Pipestance runner.
//=============================================================================
func runLoop(pipestanceBox *pipestanceHolder, stepSecs int, vdrMode string,
	noExit bool) {
	showedFailed := false
	pipestanceBox.getPipestance().LoadMetadata()

	for {
		pipestance := pipestanceBox.getPipestance()
		pipestance.RefreshState()

		// Check for completion states.
		state := pipestance.GetState()
		hadProgress := false
		if state == core.Complete || state == core.DisabledState {
			pipestanceBox.UpdateState(state.Prefixed(core.CleanupPrefix))
			cleanupCompleted(pipestance, pipestanceBox, vdrMode, noExit)
			return
		} else if state == core.Failed {
			pipestanceBox.UpdateState(state.Prefixed(core.CleanupPrefix))
			if !attemptRetry(pipestance, pipestanceBox) {
				pipestance.Unlock()
				cleanupFailed(pipestance, pipestanceBox, showedFailed, noExit)
				showedFailed = true
			}
		} else {
			pipestanceBox.UpdateState(state)
			// If we went from failed to something else, allow the failure message to
			// be shown once if we fail again.
			showedFailed = false

			// Check job heartbeats.
			pipestance.CheckHeartbeats()

			// Step all nodes.
			hadProgress = pipestance.StepNodes()
		}

		if !hadProgress {
			// Wait for a bit.
			time.Sleep(time.Second * time.Duration(stepSecs))
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

func attemptRetry(pipestance *core.Pipestance, pipestanceBox *pipestanceHolder) bool {
	if pipestanceBox.readOnly {
		return false
	}
	canRetry := false
	var transient_log string
	if pipestanceBox.consumeRetry() {
		canRetry, transient_log = pipestance.IsErrorTransient()
	}
	if canRetry {
		pipestanceBox.UpdateState(core.Failed.Prefixed(core.RetryPrefix))
		pipestance.Unlock()
		if transient_log != "" {
			util.LogInfo("runtime", "Transient error detected.  Log content:\n\n%s\n", transient_log)
		}
		util.LogInfo("runtime", "Attempting retry.")
		if err := pipestanceBox.restart(); err != nil {
			util.LogInfo("runtime", "Retry failed:\n%v\n", err)
			// Let the next loop around actually handle the failure.
		}
	}
	return canRetry
}

func cleanupCompleted(pipestance *core.Pipestance, pipestanceBox *pipestanceHolder,
	vdrMode string, noExit bool) {
	if pipestanceBox.readOnly {
		pipestanceBox.UpdateState(core.Complete)
		util.Println("Pipestance completed successfully, staying alive because --inspect given.\n")
		return
	}
	pipestanceBox.cleanupLock.Lock()
	defer pipestanceBox.cleanupLock.Unlock()
	if vdrMode == "disable" {
		util.LogInfo("runtime", "VDR disabled. No files killed.")
	} else {
		killReport := pipestance.VDRKill()
		util.LogInfo("runtime", "VDR killed %d files, %s.",
			killReport.Count, humanize.Bytes(killReport.Size))
	}
	pipestance.PostProcess()
	pipestance.Unlock()
	pipestance.OnFinishHook()
	pipestanceBox.UpdateState(core.Complete)
	if noExit {
		util.Println("Pipestance completed successfully, staying alive because --noexit given.\n")
		runtime.GC()
	} else {
		if pipestanceBox.enableUI {
			// Give time for web ui client to get last update.
			util.Println("Waiting %d seconds for UI to do final refresh.", WAIT_SECS)
			time.Sleep(time.Second * time.Duration(WAIT_SECS))
			pipestance.ClearUiPort()
		}
		util.Println("Pipestance completed successfully!\n")
		os.Exit(0)
	}
}

func cleanupFailed(pipestance *core.Pipestance, pipestanceBox *pipestanceHolder,
	showedFailed bool, noExit bool) {
	if pipestanceBox.readOnly {
		pipestanceBox.UpdateState(core.Failed)
		if !showedFailed {
			util.Println("Pipestance failed, staying alive because --inspect given.\n")
		}
		return
	}
	pipestanceBox.cleanupLock.Lock()
	defer pipestanceBox.cleanupLock.Unlock()
	if !showedFailed {
		pipestance.OnFinishHook()
		if _, preflight, _, log, kind, errPaths := pipestance.GetFatalError(); kind == "assert" {
			// Print preflight check failures.
			util.Println("\n[%s] %s\n", "error", log)
			pipestance.ClearUiPort()
			pipestanceBox.UpdateState(core.Failed)
			if preflight {
				os.Exit(3)
			} else {
				os.Exit(1)
			}
		} else if len(errPaths) > 0 {
			// Build relative path to _errors file
			errPath, _ := filepath.Rel(filepath.Dir(pipestance.GetPath()), errPaths[0])

			if log != "" {
				util.Println("\n[%s] Pipestance failed. Error log at:\n%s\n\nLog message:\n%s\n",
					"error", errPath, log)
			} else {
				// Print path to _errors metadata file in failed stage.
				util.Println("\n[%s] Pipestance failed. Please see log at:\n%s\n", "error", errPath)
			}
			pipestanceBox.UpdateState(core.Failed)
		}
	} else {
		pipestanceBox.UpdateState(core.Failed)
	}
	if noExit {
		// If pipestance failed but we're staying alive, only print this once
		// as long as we stay failed.
		if !showedFailed {
			util.Println("Pipestance failed, staying alive because --noexit given.\n")
		}
	} else {
		if pipestanceBox.enableUI {
			// Give time for web ui client to get last update.
			util.Println("Waiting %d seconds for UI to do final refresh.", WAIT_SECS)
			time.Sleep(time.Second * time.Duration(WAIT_SECS))
			util.Println("Pipestance failed. Use --noexit option to keep UI running after failure.\n")
			pipestance.ClearUiPort()
		}
		os.Exit(1)
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
	"MARTIAN": true,
	"MYSYS2":  true, // Anaconda
	"PBS":     true, // PBS
	"PD":      true,
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

func main() {
	util.SetupSignalHandlers()

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
    --mempercore=NUM    Specify min GB per core on your cluster.
                            Only applies in cluster jobmodes.
    --maxjobs=NUM       Set max jobs submitted to cluster at one time.
                            Only applies in cluster jobmodes.
    --jobinterval=NUM   Set delay between submitting jobs to cluster, in ms.
                            Only applies in cluster jobmodes.
    --limit-loadavg     Avoid scheduling jobs when the system loadavg is high.
                            Only applies to local jobs.

    --vdrmode=MODE      Enables Volatile Data Removal. Valid options:
                            post (default), rolling, or disable

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
    --overrides=JSON    JSON file supplying custom run conditions per stage.

    -h --help           Show this message.
    --version           Show version.`
	config := core.DefaultRuntimeOptions()
	opts, _ := docopt.Parse(doc, nil, true, config.MartianVersion, false)
	util.Println("Martian Runtime - %s", config.MartianVersion)
	util.LogInfo("build  ", "Built with Go version %s", runtime.Version())
	util.LogInfo("cmdline", strings.Join(os.Args, " "))
	util.LogInfo("pid    ", strconv.Itoa(os.Getpid()))

	for _, env := range os.Environ() {
		pair := strings.Split(env, "=")
		if len(pair) == 2 && logEnv(pair[0]) {
			util.LogInfo("environ", env)
		}
	}

	martianFlags := ""
	if martianFlags = os.Getenv("MROFLAGS"); len(martianFlags) > 0 {
		martianOptions := strings.Split(martianFlags, " ")
		util.ParseMroFlags(opts, doc, martianOptions, []string{"call.mro", "pipestance"})
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
	cwd, _ := os.Getwd()
	mro_dir, _ := filepath.Abs(path.Dir(os.Args[1]))
	mroPaths := util.ParseMroPath(mro_dir)
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPaths = util.ParseMroPath(value)
	}
	mroVersion, _ := util.GetMroVersion(mroPaths)
	util.LogInfo("environ", "MROPATH=%s", util.FormatMroPath(mroPaths))
	util.LogInfo("version", "MRO Version=%s", mroVersion)

	// Compute job manager.
	if value := opts["--jobmode"]; value != nil {
		config.JobMode = value.(string)
	}
	util.LogInfo("options", "--jobmode=%s", config.JobMode)

	// Max parallel jobs.
	if config.JobMode != "local" {
		config.MaxJobs = 64
	}
	if value := opts["--maxjobs"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			config.MaxJobs = value
		} else {
			util.PrintError(err, "options", "Could not parse --maxjobs value \"%s\"", opts["--maxjobs"].(string))
			os.Exit(1)
		}
	}
	util.LogInfo("options", "--maxjobs=%d", config.MaxJobs)

	// frequency (in milliseconds) that jobs will be sent to the queue
	// (this is a minimum bound, as it may take longer to emit jobs)
	if config.JobMode != "local" {
		config.JobFreqMillis = 100
	}
	if value := opts["--jobinterval"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			config.JobFreqMillis = value
		} else {
			util.PrintError(err, "options", "Could not parse --jobinterval value \"%s\"", opts["--jobinterval"].(string))
			os.Exit(1)
		}
	}
	util.LogInfo("options", "--jobinterval=%d", config.JobFreqMillis)

	// Compute vdrMode.
	if value := opts["--vdrmode"]; value != nil {
		config.VdrMode = value.(string)
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
	util.LogInfo("options", "--profile=%s", config.ProfileMode)
	core.VerifyProfileMode(config.ProfileMode)

	// Compute UI port.
	requireAuth := true
	uiport := ""
	if value := opts["--uiport"]; value != nil {
		uiport = value.(string)
		requireAuth = false
	} else if os.Getenv("MRVHOST") != "" {
		requireAuth = false
	}
	if len(uiport) > 0 {
		util.LogInfo("options", "--uiport=%s", uiport)
	}

	enableUI := (opts["--disable-ui"] == nil || !opts["--disable-ui"].(bool))
	if !enableUI {
		util.LogInfo("options", "--disable-ui")
	}
	if value := opts["--disable-auth"]; value != nil && value.(bool) {
		requireAuth = false
		util.LogInfo("options", "--disable-auth")
	}
	if value := opts["--require-auth"]; value != nil && value.(bool) {
		requireAuth = true
		util.LogInfo("options", "--require-auth")
	}
	var authKey string
	if value := opts["--authkey"]; value != nil {
		authKey = value.(string)
		util.LogInfo("options", "--authkey=%s", authKey)
	} else if enableUI {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			util.PrintError(err, "webserv", "Failed to generate an authentication key.")
			os.Exit(1)
		}
		authKey = base64.RawURLEncoding.EncodeToString(key)
	}

	// Parse tags.
	tags := []string{}
	if value := opts["--tags"]; value != nil {
		tags = util.ParseTagsOpt(value.(string))
	}
	for _, tag := range tags {
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

	noExit := opts["--noexit"].(bool)
	util.LogInfo("options", "--noexit=%v", noExit)

	config.SkipPreflight = opts["--nopreflight"].(bool)
	util.LogInfo("options", "--nopreflight=%v", config.SkipPreflight)

	psid := opts["<pipestance_name>"].(string)
	invocationPath := opts["<call.mro>"].(string)
	pipestancePath := path.Join(cwd, psid)
	stepSecs := 3
	checkSrc := true
	config.Monitor = opts["--monitor"].(bool)
	readOnly := opts["--inspect"].(bool)
	config.Debug = opts["--debug"].(bool)
	config.StressTest = opts["--stest"].(bool)
	envs := map[string]string{}
	retries := core.DefaultRetries()
	if value := opts["--autoretry"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			retries = value
			util.LogInfo("options", "--autoretry=%d", retries)
		} else {
			util.PrintError(err, "options",
				"Could not parse --autoretry value \"%s\"", opts["--autoretry"].(string))
			os.Exit(1)
		}
	}
	if retries > 0 && config.FullStageReset {
		retries = 0
		util.Println(
			"\nWARNING: ignoring autoretry when MRO_FULLSTAGERESET is set.\n")
		util.LogInfo("options", "autoretry disabled due to MRO_FULLSTAGERESET.\n")
	}
	// Validate psid.
	util.DieIf(util.ValidateID(psid))

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
	// Configure Martian runtime.
	//=========================================================================
	rt := config.NewRuntime()
	rt.MroCache.CacheMros(mroPaths)

	//=========================================================================
	// Invoke pipestance or Reattach if exists.
	//=========================================================================
	data, err := ioutil.ReadFile(invocationPath)
	util.DieIf(err)
	invocationSrc := string(data)
	executingPreflight := !config.SkipPreflight

	factory := core.NewRuntimePipestanceFactory(rt,
		invocationSrc, invocationPath, psid, mroPaths, pipestancePath, mroVersion,
		envs, checkSrc, readOnly, tags)

	// Attempt to reattach to the pipestance.
	reattaching := false
	pipestance, err := factory.InvokePipeline()
	if err != nil {
		if _, ok := err.(*core.PipestanceExistsError); ok {
			if pipestance, err = factory.ReattachToPipestance(); err == nil {
				config.MartianVersion, mroVersion, _ = pipestance.GetVersions()
				reattaching = true
			} else {
				util.DieIf(err)
			}
		} else {
			util.DieIf(err)
		}
	}
	pipestanceBox := pipestanceHolder{
		pipestance:       pipestance,
		factory:          factory,
		maxRetries:       retries,
		remainingRetries: retries,
		readOnly:         readOnly,
	}

	if !readOnly {
		// Start writing (including cached entries) to log file.
		util.LogTee(path.Join(pipestancePath, "_log"))
	}

	uuid, _ := pipestance.GetUuid()

	// Attempt to open the UI port.  If the port was not automatically
	// assigned, fail mrp if it cannot be opened.  Otherwise, log a message
	// and continue.
	var listener net.Listener
	if enableUI {
		var err error
		dieWithoutUi := true
		if uiport == "" {
			uiport = "0"
			dieWithoutUi = false
		}
		if listener, err = net.Listen("tcp",
			fmt.Sprintf(":%s", uiport)); err != nil {
			util.PrintError(err, "webserv", "Cannot open port %s", uiport)
			if dieWithoutUi {
				os.Exit(1)
			} else {
				util.PrintError(err, "webserv", "UI disabled")
				enableUI = false
				listener = nil
			}
		} else {
			u := url.URL{
				Scheme: "http",
				Host:   listener.Addr().String(),
			}
			uiport = u.Port()
			u.Host = net.JoinHostPort(hostname, uiport)
			if authKey != "" {
				q := u.Query()
				q.Set("auth", authKey)
				u.RawQuery = q.Encode()
			}
			// Print this here because the log makes more sense when this appears before
			// the runloop messages start to appear.
			util.Println("Serving UI at %s\n", u.String())
			if !readOnly {
				pipestance.RecordUiPort(u.String())
			}
			pipestanceBox.authKey = authKey
			pipestanceBox.enableUI = true
		}
	} else {
		util.LogInfo("webserv", "UI disabled.")
	}

	//=========================================================================
	// Collect pipestance static info.
	//=========================================================================
	pipestanceBox.info = &api.PipestanceInfo{
		Hostname:     hostname,
		Username:     username,
		Cwd:          cwd,
		Binpath:      util.RelPath(os.Args[0]),
		Cmdline:      strings.Join(os.Args, " "),
		Pid:          os.Getpid(),
		Start:        pipestance.GetTimestamp(),
		Version:      config.MartianVersion,
		Pname:        pipestance.GetPname(),
		PsId:         psid,
		State:        pipestance.GetState(),
		JobMode:      config.JobMode,
		MaxCores:     rt.JobManager.GetMaxCores(),
		MaxMemGB:     rt.JobManager.GetMaxMemGB(),
		InvokePath:   invocationPath,
		InvokeSource: invocationSrc,
		MroPath:      util.FormatMroPath(mroPaths),
		ProfileMode:  config.ProfileMode,
		Port:         uiport,
		MroVersion:   mroVersion,
		Uuid:         uuid,
	}

	if reattaching {
		// If it already exists, try to reattach to it.
		if !readOnly {
			if err = pipestance.Reset(); err == nil {
				err = pipestance.RestartLocalJobs(config.JobMode)
			}
			util.DieIf(err)
		}
	} else if executingPreflight && !readOnly {
		util.Println("Running preflight checks (please wait)...")
	}

	//=========================================================================
	// Start web server.
	//=========================================================================
	if listener != nil {
		go runWebServer(listener, rt, &pipestanceBox, requireAuth)
	}

	//=========================================================================
	// Start run loop.
	//=========================================================================
	go runLoop(&pipestanceBox, stepSecs, config.VdrMode, noExit)

	// Let daemons take over.
	runtime.Goexit()
}

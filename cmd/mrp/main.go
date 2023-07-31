//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

// Command mrp is the martian pipeline runner.
package main

import (
	"context"
	"crypto/tls"
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
	"strings"
	"sync"
	"time"

	"github.com/martian-lang/martian/martian/api"
	"github.com/martian-lang/martian/martian/core"
	"github.com/martian-lang/martian/martian/util"

	"github.com/dustin/go-humanize"
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
	https            bool
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
		return self.Register(false)
	}
	return nil
}

func (self *pipestanceHolder) UpdateError(message string) {
	self.lock.Lock()
	self.info.LastErrorMessage = message
	self.lock.Unlock()
}

// Register sends state information to enterprise, if available.
//
// It will only do this if the UI is enabled, unless force is true.
func (self *pipestanceHolder) Register(force bool) chan struct{} {
	if !self.enableUI && !force {
		return nil
	}
	if enterpriseHost := os.Getenv("MARTIAN_ENTERPRISE"); enterpriseHost != "" {
		u := url.URL{
			Scheme: "http",
			Host:   strings.TrimPrefix(enterpriseHost, "http://"),
			Path:   api.QueryRegisterEnterprise,
		}
		if strings.HasPrefix(enterpriseHost, "https://") {
			u.Host = enterpriseHost[len("https://"):]
			u.Scheme = "https"
		}
		form := self.info.AsForm()
		form.Set("authkey", self.authKey)
		if self.https {
			form.Set("https", "true")
		}
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
			_ = ps.ClearUiPort()
		}
	}
}

func (pipestanceBox *pipestanceHolder) Configure(c *mrpConfiguration, invocationSrc string) (
	bool, *core.Runtime) {
	//=========================================================================
	// Configure Martian runtime.
	//=========================================================================
	rt, err1 := c.config.NewRuntime()

	factory := core.NewRuntimePipestanceFactory(rt,
		invocationSrc, c.invocationPath, c.psid, c.mroPaths, c.pipestancePath, c.mroVersion,
		nil, true, c.readOnly, c.tags)
	reattaching := false
	pipestance, err := factory.InvokePipeline()
	pipestanceBox.pipestance = pipestance
	pipestanceBox.factory = factory
	pipestanceBox.maxRetries = c.retries
	pipestanceBox.remainingRetries = c.retries
	pipestanceBox.readOnly = c.readOnly
	pipestanceBox.retryWait = c.retryWait
	pipestanceBox.https = c.cert != nil
	// Delay reporting of this error to here so that we have a chance to
	// populate the pipestance, so we can get the UUID for reporting purposes
	// if necessary.  We do need to check it before we try to get anything
	// from the job manager, however.
	if err1 != nil {
		util.PrintInfo("jobmngr", "%v", err)
		pipestanceBox.reportConfigFailure(err)
		// Not using util.DieIf here because it would log the error redundantly.
		os.Exit(1)
	}
	pipestanceBox.info.MaxCores = rt.JobManager.GetMaxCores()
	pipestanceBox.info.MaxMemGB = rt.JobManager.GetMaxMemGB()

	if err != nil {
		if _, ok := err.(*core.PipestanceExistsError); ok {
			pipestance, err = factory.ReattachToPipestance(context.Background())
			pipestanceBox.pipestance = pipestance
			if err == nil {
				c.config.MartianVersion, c.mroVersion, _ = pipestance.GetVersions()
				reattaching = true
			} else {
				pipestanceBox.reportAndDieIf(err)
			}
		} else {
			pipestanceBox.reportAndDieIf(err)
		}
	}
	pipestanceBox.info.Uuid, _ = pipestance.GetUuid()
	pipestanceBox.info.Start = pipestance.GetTimestamp()
	pipestanceBox.info.Pname = pipestance.GetPname()
	pipestanceBox.info.State = pipestance.GetState(context.Background())

	return reattaching, rt
}

// reportAndDieIf is shortand for reportConfigFailure followed by util.DieIf.
func (pipestanceBox *pipestanceHolder) reportAndDieIf(err error) {
	if err == nil {
		return
	}
	pipestanceBox.reportConfigFailure(err)
	util.DieIf(err)
}

// reportConfigFailure reports startup failures to enterprise if possible.
func (pipestanceBox *pipestanceHolder) reportConfigFailure(err error) {
	enterpriseHost := os.Getenv("MARTIAN_ENTERPRISE")
	if enterpriseHost == "" {
		// No one to report to.
		return
	}
	// We must have a UUID for reporting, but this may be called before the
	// point at which the UUID was populated in the info object.
	if pipestanceBox.info.Uuid == "" {
		if pipestance := pipestanceBox.pipestance; pipestance != nil {
			pipestanceBox.info.Uuid, _ = pipestance.GetUuid()
		}
		// If we don't have a UUID from the pipestance object, fall back by
		// trying to get it from the environment.
		if pipestanceBox.info.Uuid == "" {
			pipestanceBox.info.Uuid = os.Getenv("MRO_FORCE_UUID")
		}
		if pipestanceBox.info.Uuid == "" {
			pipestanceBox.info.Uuid = os.Getenv("MRO_UUID")
		}
		if pipestanceBox.info.Uuid == "" {
			// No UUID, can't report anything.
			return
		}
	}
	pipestanceBox.info.State = core.Failed
	if pipestanceBox.info.Start == "" {
		pipestanceBox.info.Start = util.Timestamp()
	}
	pipestanceBox.UpdateError(err.Error())
	// Set a timeout on error reporting.
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()
	select {
	case <-pipestanceBox.Register(true):
	case <-timer.C:
	}
}

func (c *mrpConfiguration) checkSpace() {
	psPath := c.pipestancePath
	info, err := os.Lstat(c.pipestancePath)
	if err != nil {
		util.LogError(err, "filesys", "Cannot stat pipestance directory.")
	} else if info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(psPath)
		if err != nil {
			util.LogError(err, "filesys",
				"Could not resolve pipestance path %s, which is a symlink.",
				psPath)
		} else {
			util.LogInfo("filesys",
				"Pipestance path %s is a symlink to %s",
				psPath, resolved)
			psPath = resolved
			info, err = os.Stat(psPath)
			if err != nil {
				util.LogError(err, "filesys",
					"Cannot stat resolved pipestance directory.")
			}
			if !info.IsDir() {
				util.PrintInfo("filesys", "Pipestance path is not a directory.")
			}
		}
	}
	if bSize, inodes, fstype, err := core.GetAvailableSpace(psPath); err != nil {
		util.PrintError(err, "filesys", "Error reading pipestance filesystem information.")
	} else {
		// Log the logical path, not the resolved one here, even though
		// we're getting the info for the resolved one because that's what
		// matters for space usage etc.
		util.LogInfo("filesys", "Pipestance path %s",
			c.pipestancePath)
		util.LogInfo("filesys", "Pipestance filesystem type %s",
			fstype)
		util.LogInfo("filesys", "%s and %s inodes available.",
			humanize.Bytes(bSize), humanize.Comma(int64(inodes)))
	}
	if fstype, opts, err := core.GetMountOptions(psPath); err != nil {
		util.LogError(err, "filesys",
			"Could not read pipestance filesystem mount options.")
	} else {
		util.LogInfo("filesys",
			"Pipestance filesystem %s mount options: %s",
			fstype, opts)
	}
	if info != nil {
		if uid, gid, ok := util.GetFileOwner(info); ok {
			util.LogInfo("filesys",
				"Pipestance directory permissions %s owned by uid %d gid %d",
				info.Mode().String(), uid, gid)
		} else {
			util.LogInfo("filesys", "Pipestance permissions: %s",
				info.Mode().String())
		}
	}

	binPath := util.RelPath("")
	if _, _, fstype, err := core.GetAvailableSpace(binPath); err != nil {
		util.PrintError(err, "filesys",
			"Error reading source filesystem information.")
	} else {
		util.LogInfo("filesys", "Bin path %s", binPath)
		util.LogInfo("filesys", "Bin filesystem type %s",
			fstype)
	}
	if fstype, opts, err := core.GetMountOptions(binPath); err != nil {
		util.LogError(err, "filesys",
			"Could not read source filesystem mount options.")
	} else {
		util.LogInfo("filesys", "Bin filesystem %s mount options: %s",
			fstype, opts)
	}
	if exe, err := os.Executable(); err != nil {
		util.LogError(err, "filesys", "Could not get executable path.")
	} else if info, err := os.Stat(exe); err != nil {
		util.LogError(err, "filesys", "Could not stat executable.")
	} else if uid, gid, ok := util.GetFileOwner(info); ok {
		util.LogInfo("filesys", "Executable file permissions %s owned by uid %d gid %d",
			info.Mode().String(), uid, gid)
	} else {
		util.LogInfo("filesys", "Executable file permissions %s",
			info.Mode().String())
	}
}

func logUids(username string) {
	uid := os.Getuid()
	euid := os.Geteuid()
	gid := os.Getgid()
	egid := os.Getegid()
	if euid != uid || egid != gid {
		util.LogInfo("user   ", "User %s uid %d (real = %d) / gid %d (real = %d)",
			username, euid, uid, egid, gid)
	} else {
		util.LogInfo("user   ", "User %s uid %d / gid %d",
			username, uid, gid)
	}
}

func (c *mrpConfiguration) getListener(hostname string,
	pipestanceBox *pipestanceHolder, conf *tls.Config) net.Listener {
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
				pipestanceBox.reportConfigFailure(err)
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
			if conf != nil {
				listener = tls.NewListener(listener, conf)
				u.Scheme = "https"
			}
			c.uiport = u.Port()
			pipestanceBox.info.Port = c.uiport
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
			if !c.readOnly {
				util.RegisterSignalHandler(pipestanceBox)
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
	cwd, _ := os.Getwd()
	pipestanceBox := pipestanceHolder{
		info: &api.PipestanceInfo{
			Hostname:     hostname,
			Username:     username,
			Cwd:          cwd,
			Binpath:      util.RelPath(os.Args[0]),
			Cmdline:      strings.Join(os.Args, " "),
			Pid:          os.Getpid(),
			Version:      c.config.MartianVersion,
			PsId:         c.psid,
			JobMode:      c.config.JobMode,
			InvokePath:   c.invocationPath,
			InvokeSource: invocationSrc,
			MroPath:      util.FormatMroPath(c.mroPaths),
			ProfileMode:  c.config.ProfileMode,
			Port:         c.uiport,
			MroVersion:   c.mroVersion,
			PsPath:       c.pipestancePath,
		},
	}
	reattaching, rt := pipestanceBox.Configure(&c, invocationSrc)
	pipestance := pipestanceBox.pipestance

	util.LogSysInfo()
	if !c.readOnly {
		// Start writing (including cached entries) to log file.
		util.LogTee(path.Join(c.pipestancePath, "_log"))
		pipestanceBox.lastLogCheck = time.Now()
	}
	c.checkSpace()
	logUids(username)

	listener := c.getListener(hostname, &pipestanceBox, c.cert)

	if reattaching {
		// If it already exists, try to reattach to it.
		if !c.readOnly {
			if err = pipestance.Reset(); err == nil {
				err = pipestance.RestartLocalJobs(c.config.JobMode)
			}
			pipestanceBox.reportAndDieIf(err)
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

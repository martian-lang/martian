//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian pipeline runner.
//
package main

import (
	"fmt"
	"github.com/docopt/docopt.go"
	"github.com/dustin/go-humanize"
	"io/ioutil"
	"martian/core"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//=============================================================================
// Pipestance runner.
//=============================================================================
func runLoop(pipestance *core.Pipestance, stepSecs int, vdrMode string,
	noExit bool, enableUI bool) {
	showedFailed := false
	WAIT_SECS := 6

	pipestance.LoadMetadata()

	for {
		pipestance.RefreshState()

		// Check for completion states.
		state := pipestance.GetState()
		if state == "complete" {
			if vdrMode == "disable" {
				core.LogInfo("runtime", "VDR disabled. No files killed.")
			} else {
				core.LogInfo("runtime", "Starting VDR kill...")
				killReport := pipestance.VDRKill()
				core.LogInfo("runtime", "VDR killed %d files, %s.",
					killReport.Count, humanize.Bytes(killReport.Size))
			}
			pipestance.Unlock()
			pipestance.PostProcess()
			if noExit {
				core.Println("Pipestance is complete, staying alive because --noexit given.")
				break
			} else {
				if enableUI {
					// Give time for web ui client to get last update.
					core.Println("Waiting %d seconds for UI to do final refresh.", WAIT_SECS)
					time.Sleep(time.Second * time.Duration(WAIT_SECS))
				}
				core.Println("Pipestance is complete, exiting.")
				os.Exit(0)
			}
		} else if state == "failed" {
			pipestance.Unlock()
			if !showedFailed {
				if fqname, _, log, kind, _ := pipestance.GetFatalError(); kind == "assert" {
					// Print pre-flight check failures.
					core.Println("\n[%s] %s", core.Colorize("error", core.ANSI_MAGENTA), log)
					os.Exit(2)
				} else {
					// Convert fqname into path
					errpath := strings.Replace(fqname[3:], ".", "/", -1)
					core.Println("\n[%s] Pipestance failed. Please see log at:\n%s\n", core.Colorize("error", core.ANSI_MAGENTA), core.Colorize(fmt.Sprintf("%s/_errors", errpath), core.ANSI_CYAN))
				}
			}
			if noExit {
				// If pipestance failed but we're staying alive, only print this once
				// as long as we stay failed.
				if !showedFailed {
					showedFailed = true
					core.Println("Pipestance failed, staying alive because --noexit given.")
				}
			} else {
				if enableUI {
					// Give time for web ui client to get last update.
					core.Println("Waiting %d seconds for UI to do final refresh.", WAIT_SECS)
					time.Sleep(time.Second * time.Duration(WAIT_SECS))
					core.Println("Pipestance failed, exiting. Use --noexit option to keep UI running after failure.")
				}
				os.Exit(1)
			}
		} else {
			// If we went from failed to something else, allow the failure message to
			// be shown once if we fail again.
			showedFailed = false
		}

		// Step all nodes.
		pipestance.StepNodes()

		// Wait for a bit.
		time.Sleep(time.Second * time.Duration(stepSecs))
	}
}

func main() {
	core.SetupSignalHandlers()

	//=========================================================================
	// Commandline argument and environment variables.
	//=========================================================================
	// Parse commandline.
	doc := `Martian Pipeline Runner.

Usage: 
    mrp <call.mro> <pipestance_name> [options]
    mrp -h | --help | --version

Options:
    --uiport=<num>       Serve UI at http://localhost:<num>
    --jobmode=<name>     Run jobs on custom or local job manager.
                           Valid job managers are local, sge or .template file
                           Defaults to local.
    --vdrmode=<name>     Enables Volatile Data Removal.
                           Valid options are rolling, post and disable.
                           Defaults to post.
    --nodump             Turns off debug dump tarball generation.
    --noexit             Keep UI running after pipestance completes or fails.
    --profile            Enable stage performance profiling.
    --stackvars          Print local variables in stage code stack trace.
    --localcores=<num>   Set max cores the pipeline may request at one time.
                           (Only applies in local jobmode)
    --localmem=<num>     Set max GB the pipeline may request at one time.
                           (Only applies in local jobmode)
    --mempercore=<num>   Set max GB each job may use at one time.
                           Defaults to 4 GB.
                           (Only applies in non-local jobmodes)
    --inspect            Inspect pipestance without resetting failed stages.
    --debug              Enable debug logging for local job manager.
    --stest              Substitute real stages with stress-testing stage.
    -h --help            Show this message.
    --version            Show version.`
	martianVersion := core.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, martianVersion, false)
	core.Println("Martian Runtime - %s", martianVersion)
	core.LogInfo("cmdline", strings.Join(os.Args, " "))

	martianFlags := ""
	if martianFlags = os.Getenv("MROFLAGS"); len(martianFlags) > 0 {
		martianOptions := strings.Split(martianFlags, " ")
		core.ParseMroFlags(opts, doc, martianOptions, []string{"call.mro", "pipestance"})
	}

	// Requested cores and memory.
	reqCores := -1
	if value := opts["--localcores"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			reqCores = value
		}
	}
	reqMem := -1
	if value := opts["--localmem"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			reqMem = value
		}
	}
	reqMemPerCore := -1
	if value := opts["--mempercore"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			reqMemPerCore = value
		}
	}

	// Compute MRO path.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPath := cwd
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPath = value
	}
	mroVersion := core.GetGitTag(mroPath)
	core.LogInfo("environ", "MROPATH=%s", mroPath)
	core.LogInfo("version", "MRO Version=%s", mroVersion)

	// Compute job manager.
	jobMode := "local"
	if value := opts["--jobmode"]; value != nil {
		jobMode = value.(string)
	}
	core.LogInfo("options", "--jobmode=%s", jobMode)
	core.VerifyJobManager(jobMode)

	// Compute vdrMode.
	vdrMode := "post"
	if value := opts["--vdrmode"]; value != nil {
		vdrMode = value.(string)
	}
	core.LogInfo("environ", "--vdrmode=%s", vdrMode)
	core.VerifyVDRMode(vdrMode)

	// Compute UI port.
	uiport := ""
	enableUI := false
	if value := opts["--uiport"]; value != nil {
		uiport = value.(string)
		enableUI = true
	}
	if enableUI {
		core.LogInfo("environ", "uiport = %s", uiport)
	}
	core.LogInfo("options", "--port=%s", uiport)

	// Compute profiling flag.
	profile := opts["--profile"].(bool)
	core.LogInfo("options", "--profile=%v", profile)

	// Compute stackVars flag.
	stackVars := opts["--stackvars"].(bool)
	core.LogInfo("environ", "stackvars = %v", stackVars)

	noExit := opts["--noexit"].(bool)
	core.LogInfo("options", "--noexit=%v", noExit)

	psid := opts["<pipestance_name>"].(string)
	invocationPath := opts["<call.mro>"].(string)
	pipestancePath := path.Join(cwd, psid)
	stepSecs := 3
	checkSrc := true
	readOnly := false
	inspect := opts["--inspect"].(bool)
	debug := opts["--debug"].(bool)
	stest := opts["--stest"].(bool)

	// Validate psid.
	core.DieIf(core.ValidateID(psid))

	//=========================================================================
	// Configure Martian runtime.
	//=========================================================================
	rt := core.NewRuntimeWithCores(jobMode, vdrMode, mroPath, martianVersion, mroVersion,
		reqCores, reqMem, reqMemPerCore, -1, profile, stackVars, debug, stest)

	// Print this here because the log makes more sense when this appears before
	// the runloop messages start to appear.
	if enableUI {
		core.Println("Serving UI at http://localhost:%s", uiport)
	} else {
		core.LogInfo("webserv", "UI disabled.")
	}

	//=========================================================================
	// Invoke pipestance or Reattach if exists.
	//=========================================================================
	data, err := ioutil.ReadFile(invocationPath)
	core.DieIf(err)
	invocationSrc := string(data)
	pipestance, err := rt.InvokePipeline(invocationSrc, invocationPath, psid, pipestancePath)
	if err != nil {
		if _, ok := err.(*core.PipestanceExistsError); ok {
			// If it already exists, try to reattach to it.
			if pipestance, err = rt.ReattachToPipestance(psid, pipestancePath, invocationSrc, checkSrc, readOnly); err == nil {
				if !inspect {
					err = pipestance.Reset()
				}
			}
		}
		core.DieIf(err)
	}
	core.Println("\nRunning pre-flight checks (15 seconds)...")
	logfile := path.Join(pipestancePath, "_log")
	core.LogTee(logfile)

	//=========================================================================
	// Collect pipestance static info.
	//=========================================================================
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	user, err := user.Current()
	username := "unknown"
	if err == nil {
		username = user.Username
	}
	info := map[string]string{
		"hostname":   hostname,
		"username":   username,
		"cwd":        cwd,
		"binpath":    core.RelPath(os.Args[0]),
		"cmdline":    strings.Join(os.Args, " "),
		"pid":        strconv.Itoa(os.Getpid()),
		"start":      time.Now().Format(time.RFC822),
		"version":    martianVersion,
		"pname":      pipestance.GetPname(),
		"psid":       psid,
		"state":      pipestance.GetState(),
		"jobmode":    jobMode,
		"maxcores":   strconv.Itoa(rt.JobManager.GetMaxCores()),
		"maxmemgb":   strconv.Itoa(rt.JobManager.GetMaxMemGB()),
		"invokepath": invocationPath,
		"invokesrc":  invocationSrc,
		"MROPATH":    mroPath,
		"MROPROFILE": fmt.Sprintf("%v", profile),
		"MROPORT":    uiport,
		"mroversion": mroVersion,
	}

	//=========================================================================
	// Register with mrv.
	//=========================================================================
	if mrvhost := os.Getenv("MRVHOST"); len(mrvhost) > 0 {
		u := url.URL{
			Scheme: "http",
			Host:   mrvhost,
			Path:   "/register",
		}
		form := url.Values{}
		for k, v := range info {
			form.Add(k, v)
		}
		if res, err := http.PostForm(u.String(), form); err == nil {
			if content, err := ioutil.ReadAll(res.Body); err == nil {
				if res.StatusCode == 200 {
					uiport = string(content)
				}
			} else {
				core.LogError(err, "mrvcli", "Could not read response from mrv %s.", u.String())
			}
		} else {
			core.LogError(err, "mrvcli", "HTTP request failed %s.", u.String())
		}
	}

	//=========================================================================
	// Start web server.
	//=========================================================================
	if enableUI && len(uiport) > 0 {
		go runWebServer(uiport, rt, pipestance, info)
	}

	//=========================================================================
	// Start run loop.
	//=========================================================================
	go runLoop(pipestance, stepSecs, vdrMode, noExit, enableUI)

	// Let daemons take over.
	done := make(chan bool)
	<-done
}

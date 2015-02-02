//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian pipeline runner.
//
package main

import (
	"fmt"
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

	"github.com/docopt/docopt.go"
	"github.com/dustin/go-humanize"
)

//=============================================================================
// Pipestance runner.
//=============================================================================
func runLoop(pipestance *core.Pipestance, stepSecs int, disableVDR bool,
	noExit bool, noUI bool) {
	showedFailed := false
	WAIT_SECS := 6

	pipestance.LoadMetadata()

	for {
		pipestance.RefreshState()

		// Check for completion states.
		state := pipestance.GetState()
		if state == "complete" {
			pipestance.Cleanup()
			pipestance.Immortalize()
			if warnings, ok := pipestance.GetWarnings(); ok {
				core.Println("%s", warnings)
			}
			if disableVDR {
				core.LogInfo("runtime",
					"VDR disabled by --novdr option. No files killed.")
			} else {
				core.LogInfo("runtime", "Starting VDR kill...")
				killReport := pipestance.VDRKill()
				core.LogInfo("runtime", "VDR killed %d files, %s.",
					killReport.Count, humanize.Bytes(killReport.Size))
			}
			if noExit {
				core.Println("Pipestance is complete, staying alive because --noexit given.")
				break
			} else {
				if !noUI {
					// Give time for web ui client to get last update.
					core.Println("Waiting %d seconds for UI to do final refresh.", WAIT_SECS)
					time.Sleep(time.Second * time.Duration(WAIT_SECS))
				}
				core.Println("Pipestance is complete, exiting.")
				os.Exit(0)
			}
		} else if state == "failed" {
			if !showedFailed {
				if warnings, ok := pipestance.GetWarnings(); ok {
					core.Println(warnings)
				}
				if fqname, _, log, kind, _ := pipestance.GetFatalError(); kind == "assert" {
					// Print pre-flight check failures.
					core.Println("\n[\033[35merror\033[0m] %s", log)
					os.Exit(2)
				} else {
					// Convert fqname into path
					errpath := strings.Replace(fqname[3:], ".", "/", -1)
					core.Println("\n[\033[35merror\033[0m] Pipestance failed. Please see log at:\n\033[36m%s/_errors\033[0m\n", errpath)
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
				if !noUI {
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
    --port=<num>       Serve UI at http://localhost:<num>
                         Defaults to 3600 if not otherwise specified.
    --jobmode=<name>   Run jobs on custom or local job manager.
                         Valid job managers are local, sge or .template file
                         Defaults to local.
    --nodump           Turns off debug dump tarball generation.
    --noexit           Keep UI running after pipestance completes or fails.
    --noui             Disable UI.
    --novdr            Disable Volatile Data Removal.
    --profile          Enable stage performance profiling.
    --localvars        Print local variables in stage code stack trace.
    --maxcores=<num>   Set max cores the pipeline may request at one time.
                         (Only applies in local jobmode)
    --maxmem=<num>     Set max GB the pipeline may request at one time.
                         (Only applies in local jobmode)
    --debug            Enable debug logging for local job manager.
    --stest            Substitute real stages with stress-testing stage.
    -h --help          Show this message.
    --version          Show version.`
	martianVersion := core.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, martianVersion, false)
	core.Println("Martian Runtime (%s)", martianVersion)
	core.LogInfo("cmdline", strings.Join(os.Args, " "))

	martianFlags := ""
	if martianFlags = os.Getenv("MROFLAGS"); len(martianFlags) > 0 {
		martianOptions := strings.Split(martianFlags, " ")
		core.ParseMroFlags(opts, doc, martianOptions, []string{"call.mro", "pipestance"})
	}

	// Requested cores.
	reqCores := -1
	if value := opts["--maxcores"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			reqCores = value
		}
	}
	reqMem := -1
	if value := opts["--maxmem"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			reqMem = value
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

	// Compute UI port.
	uiport := "3600"
	noUI := false
	if value := opts["--port"]; value != nil {
		uiport = value.(string)
	}
	if opts["--noui"].(bool) {
		uiport = ""
		noUI = true
	}
	core.LogInfo("options", "--port=%s", uiport)

	// Compute profiling flag.
	profile := opts["--profile"].(bool)
	core.LogInfo("options", "--profile=%v", profile)

	// Compute localVars flag.
	localVars := opts["--localvars"].(bool)
	core.LogInfo("options", "--localvars=%v", localVars)

	// Setup invocation-specific values.
	disableVDR := opts["--novdr"].(bool)
	core.LogInfo("options", "--novdr=%v", disableVDR)

	noExit := opts["--noexit"].(bool)
	core.LogInfo("options", "--noexit=%v", noExit)

	psid := opts["<pipestance_name>"].(string)
	invocationPath := opts["<call.mro>"].(string)
	pipestancePath := path.Join(cwd, psid)
	stepSecs := 3
	debug := opts["--debug"].(bool)
	stest := opts["--stest"].(bool)

	// Validate psid.
	core.DieIf(core.ValidateID(psid))

	//=========================================================================
	// Configure Martian runtime.
	//=========================================================================
	rt := core.NewRuntimeWithCores(jobMode, mroPath, martianVersion, mroVersion,
		reqCores, reqMem, profile, localVars, debug, stest)

	// Print this here because the log makes more sense when this appears before
	// the runloop messages start to appear.
	if noUI {
		core.LogInfo("webserv", "UI disabled by --noui option.")
	} else {
		core.Println("Serving UI at http://localhost:%s", uiport)
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
			if pipestance, err = rt.ReattachToPipestance(psid, pipestancePath); err == nil {
				err = pipestance.RestartAssertedNodes()
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
		"mrobranch":  core.GetGitBranch(mroPath),
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
	if !noUI && len(uiport) > 0 {
		go runWebServer(uiport, rt, pipestance, info)
	}

	//=========================================================================
	// Start run loop.
	//=========================================================================
	go runLoop(pipestance, stepSecs, disableVDR, noExit, noUI)

	// Let daemons take over.
	done := make(chan bool)
	<-done
}

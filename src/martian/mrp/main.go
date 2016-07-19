//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian pipeline runner.
//
package main

import (
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

	"github.com/10XDev/docopt.go"
	"github.com/dustin/go-humanize"
)

//=============================================================================
// Pipestance runner.
//=============================================================================
func runLoop(pipestance *core.Pipestance, stepSecs int, vdrMode string,
	noExit bool, enableUI bool) {
	showedFailed := false
	showedComplete := false
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
				killReport := pipestance.VDRKill()
				core.LogInfo("runtime", "VDR killed %d files, %s.",
					killReport.Count, humanize.Bytes(killReport.Size))
			}
			pipestance.Unlock()
			pipestance.PostProcess()
			if !showedComplete {
				pipestance.OnFinishHook()
				showedComplete = true
			}
			if noExit {
				core.Println("Pipestance completed successfully, staying alive because --noexit given.")
				break
			} else {
				if enableUI {
					// Give time for web ui client to get last update.
					core.Println("Waiting %d seconds for UI to do final refresh.", WAIT_SECS)
					time.Sleep(time.Second * time.Duration(WAIT_SECS))
				}
				core.Println("Pipestance completed successfully!")
				os.Exit(0)
			}
		} else if state == "failed" {
			pipestance.Unlock()
			if !showedFailed {
				pipestance.OnFinishHook()
				if _, preflight, _, log, kind, errPaths := pipestance.GetFatalError(); kind == "assert" {
					// Print preflight check failures.
					core.Println("\n[%s] %s\n", "error", log)
					if preflight {
						os.Exit(2)
					} else {
						os.Exit(1)
					}
				} else if len(errPaths) > 0 {
					// Build relative path to _errors file
					errPath, _ := filepath.Rel(filepath.Dir(pipestance.GetPath()), errPaths[0])

					// Print path to _errors metadata file in failed stage.
					core.Println("\n[%s] Pipestance failed. Please see log at:\n%s\n", "error", errPath)
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
					core.Println("Pipestance failed. Use --noexit option to keep UI running after failure.")
				}
				os.Exit(1)
			}
		} else {
			// If we went from failed to something else, allow the failure message to
			// be shown once if we fail again.
			showedFailed = false

			// Check job heartbeats.
			pipestance.CheckHeartbeats()

			// Step all nodes.
			pipestance.StepNodes()
		}

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
    --jobmode=MODE      Job manager to use. Valid options:
                            local (default), sge, lsf, or a .template file
    --localcores=NUM    Set max cores the pipeline may request at one time.
                            Only applies when --jobmode=local.
    --localmem=NUM      Set max GB the pipeline may request at one time.
                            Only applies when --jobmode=local.
    --mempercore=NUM    Specify min GB per core on your cluster.
                            Only applies in cluster jobmodes.
    --maxjobs=NUM       Set max jobs submitted to cluster at one time.
                            Only applies in cluster jobmodes.
    --jobinterval=NUM   Set delay between submitting jobs to cluster, in ms.
                            Only applies in cluster jobmodes.

    --vdrmode=MODE      Enables Volatile Data Removal. Valid options:
                            post (default), rolling, or disable

    --nopreflight       Skips preflight stages.
    --uiport=NUM        Serve UI at http://<hostname>:NUM
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

    -h --help           Show this message.
    --version           Show version.`
	martianVersion := core.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, martianVersion, false)
	core.Println("Martian Runtime - %s", martianVersion)
	core.LogInfo("cmdline", strings.Join(os.Args, " "))

	martianFlags := ""
	if martianFlags = os.Getenv("MROFLAGS"); len(martianFlags) > 0 {
		martianOptions := strings.Split(martianFlags, " ")
		core.ParseMroFlags(opts, doc, martianOptions, []string{"call.mro", "pipestance"})
		core.LogInfo("environ", "MROFLAGS=%s", martianFlags)
	}

	// Requested cores and memory.
	reqCores := -1
	if value := opts["--localcores"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			reqCores = value
			core.LogInfo("options", "--localcores=%d", reqCores)
		}
	}
	reqMem := -1
	if value := opts["--localmem"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			reqMem = value
			core.LogInfo("options", "--localmem=%d", reqMem)
		}
	}
	reqMemPerCore := -1
	if value := opts["--mempercore"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			reqMemPerCore = value
			core.LogInfo("options", "--mempercore=%d", reqMemPerCore)
		}
	}

	// Max parallel jobs.
	maxJobs := -1
	if value := opts["--maxjobs"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			maxJobs = value
			core.LogInfo("options", "--maxjobs=%d", maxJobs)
		}
	}
	// frequency (in milliseconds) that jobs will be sent to the queue
	// (this is a minimum bound, as it may take longer to emit jobs)
	jobFreqMillis := -1
	if value := opts["--jobinterval"]; value != nil {
		if value, err := strconv.Atoi(value.(string)); err == nil {
			jobFreqMillis = value
			core.LogInfo("options", "--jobinterval=%d", jobFreqMillis)
		}
	}

	// Special to resources mappings
	jobResources := ""
	if value := os.Getenv("MRO_JOBRESOURCES"); len(value) > 0 {
		jobResources = value
		core.LogInfo("options", "MRO_JOBRESOURCES=%s", jobResources)
	}

	// Compute MRO path.
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	mroPaths := core.ParseMroPath(cwd)
	if value := os.Getenv("MROPATH"); len(value) > 0 {
		mroPaths = core.ParseMroPath(value)
	}
	mroVersion, _ := core.GetMroVersion(mroPaths)
	core.LogInfo("environ", "MROPATH=%s", core.FormatMroPath(mroPaths))
	core.LogInfo("version", "MRO Version=%s", mroVersion)

	// Compute job manager.
	jobMode := "local"
	if value := opts["--jobmode"]; value != nil {
		jobMode = value.(string)
	}
	core.LogInfo("options", "--jobmode=%s", jobMode)

	// Compute vdrMode.
	vdrMode := "post"
	if value := opts["--vdrmode"]; value != nil {
		vdrMode = value.(string)
	}
	core.LogInfo("options", "--vdrmode=%s", vdrMode)
	core.VerifyVDRMode(vdrMode)

	// Compute onfinish
	onfinish := ""
	if value := opts["--onfinish"]; value != nil {
		onfinish = value.(string)
		core.VerifyOnFinish(onfinish)
	}

	// Compute profiling mode.
	profileMode := "disable"
	if value := opts["--profile"]; value != nil {
		profileMode = value.(string)
	}
	core.LogInfo("options", "--profile=%s", profileMode)
	core.VerifyProfileMode(profileMode)

	// Compute UI port.
	uiport := ""
	enableUI := false
	if value := opts["--uiport"]; value != nil {
		uiport = value.(string)
		enableUI = true
	}
	if enableUI {
		core.LogInfo("options", "--uiport=%s", uiport)
	}

	// Parse tags.
	tags := []string{}
	if value := opts["--tags"]; value != nil {
		tags = core.ParseTagsOpt(value.(string))
	}
	for _, tag := range tags {
		core.LogInfo("options", "--tag='%s'", tag)
	}

	// Compute stackVars flag.
	stackVars := opts["--stackvars"].(bool)
	core.LogInfo("options", "--stackvars=%v", stackVars)

	zip := opts["--zip"].(bool)
	core.LogInfo("options", "--zip=%v", zip)

	noExit := opts["--noexit"].(bool)
	core.LogInfo("options", "--noexit=%v", noExit)

	skipPreflight := opts["--nopreflight"].(bool)
	core.LogInfo("options", "--nopreflight=%v", skipPreflight)

	psid := opts["<pipestance_name>"].(string)
	invocationPath := opts["<call.mro>"].(string)
	pipestancePath := path.Join(cwd, psid)
	stepSecs := 3
	checkSrc := true
	readOnly := false
	enableMonitor := opts["--monitor"].(bool)
	inspect := opts["--inspect"].(bool)
	debug := opts["--debug"].(bool)
	stest := opts["--stest"].(bool)
	envs := map[string]string{}

	// Validate psid.
	core.DieIf(core.ValidateID(psid))

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
	rt := core.NewRuntimeWithCores(jobMode, vdrMode, profileMode, martianVersion,
		reqCores, reqMem, reqMemPerCore, maxJobs, jobFreqMillis, jobResources, stackVars,
		zip, skipPreflight, enableMonitor, debug, stest, onfinish)
	rt.MroCache.CacheMros(mroPaths)

	// Print this here because the log makes more sense when this appears before
	// the runloop messages start to appear.
	if enableUI {
		core.Println("Serving UI at http://%s:%s", hostname, uiport)
	} else {
		core.LogInfo("webserv", "UI disabled.")
	}

	//=========================================================================
	// Invoke pipestance or Reattach if exists.
	//=========================================================================
	data, err := ioutil.ReadFile(invocationPath)
	core.DieIf(err)
	invocationSrc := string(data)
	executingPreflight := !skipPreflight

	pipestance, err := rt.InvokePipeline(invocationSrc, invocationPath, psid, pipestancePath,
		mroPaths, mroVersion, envs, tags)
	if err != nil {
		if _, ok := err.(*core.PipestanceExistsError); ok {
			executingPreflight = false
			// If it already exists, try to reattach to it.
			if pipestance, err = rt.ReattachToPipestance(psid, pipestancePath, invocationSrc,
				mroPaths, mroVersion, envs, checkSrc, readOnly); err == nil {
				martianVersion, mroVersion, _ = pipestance.GetVersions()
				if !inspect {
					err = pipestance.Reset()
				}
			}
		}
		core.DieIf(err)
	}
	if executingPreflight {
		core.Println("\nRunning preflight checks (please wait)...")
	}

	// Start writing (including cached entries) to log file.
	core.LogTee(path.Join(pipestancePath, "_log"))

	//=========================================================================
	// Collect pipestance static info.
	//=========================================================================
	info := map[string]string{
		"hostname":   hostname,
		"username":   username,
		"cwd":        cwd,
		"binpath":    core.RelPath(os.Args[0]),
		"cmdline":    strings.Join(os.Args, " "),
		"pid":        strconv.Itoa(os.Getpid()),
		"start":      pipestance.GetTimestamp(),
		"version":    martianVersion,
		"pname":      pipestance.GetPname(),
		"psid":       psid,
		"state":      pipestance.GetState(),
		"jobmode":    jobMode,
		"maxcores":   strconv.Itoa(rt.JobManager.GetMaxCores()),
		"maxmemgb":   strconv.Itoa(rt.JobManager.GetMaxMemGB()),
		"invokepath": invocationPath,
		"invokesrc":  invocationSrc,
		"mropath":    core.FormatMroPath(mroPaths),
		"mroprofile": profileMode,
		"mroport":    uiport,
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

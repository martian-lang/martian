//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Mario pipeline runner.
//
package main

import (
	"fmt"
	"io/ioutil"
	"mario/core"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/dustin/go-humanize"
)

//=============================================================================
// Pipestance runner.
//=============================================================================
func runLoop(pipestance *core.Pipestance, stepSecs int, disableVDR bool,
	noExit bool, noDump bool, noUI bool) {
	showedFailed := false
	WAIT_SECS := 6

	for {
		pipestance.RefreshMetadata()

		// Check for completion states.
		state := pipestance.GetState()
		if state == "complete" {
			pipestance.Immortalize()
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
				core.LogInfo("runtime",
					"Pipestance is complete, staying alive because --noexit given.")
				break
			} else {
				if !noUI {
					// Give time for web ui client to get last update.
					core.LogInfo("runtime", "Waiting %d seconds for UI to do final refresh.", WAIT_SECS)
					time.Sleep(time.Second * time.Duration(WAIT_SECS))
				}
				core.LogInfo("runtime", "Pipestance is complete, exiting.")
				os.Exit(0)
			}
		} else if state == "failed" {
			if !showedFailed {
				fqname, _, log, errpaths := pipestance.GetFatalError()
				fmt.Printf("\nPipestance failed at:\n  %s\n\nError logs written to:\n", fqname)
				for _, errpath := range errpaths {
					fmt.Printf("  %s\n", errpath)
				}
				fmt.Printf("\n%s\n", log)

				if !noDump {
					// Generate debug tarball.
					fmt.Printf("Generating debug dump tarball...")
					debugFile := fmt.Sprintf("%s-debug-dump.tar.bz2", pipestance.GetPsid())
					cmd := exec.Command("tar", "jcf", debugFile, "--exclude=*files*", pipestance.GetPsid())
					if _, err := cmd.CombinedOutput(); err != nil {
						fmt.Printf("failed.\n  %s\n", err.Error())
					} else {
						fmt.Printf("complete.\n  %s\n", debugFile)
					}
					fmt.Print("\n")
				}
			}
			if noExit {
				// If pipestance failed but we're staying alive, only print this once
				// as long as we stay failed.
				if !showedFailed {
					showedFailed = true
					core.LogInfo("runtime",
						"Pipestance failed, staying alive because --noexit given.")
				}
			} else {
				if !noUI {
					// Give time for web ui client to get last update.
					core.LogInfo("runtime", "Waiting %d seconds for UI to do final refresh.", WAIT_SECS)
					time.Sleep(time.Second * time.Duration(WAIT_SECS))
				}
				core.LogInfo("runtime", "Pipestance failed, exiting. Use --noexit option to keep UI running after failure.")
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
	doc := `Mario Pipeline Runner.

Usage: 
    mrp <call.mro> <pipestance_name> [options]
    mrp -h | --help | --version

Options:
    --port=<num>     Serve UI at http://localhost:<num>
                       Overrides $MROPORT environment variable.
                       Defaults to 3600 if not otherwise specified.
    --noexit         Keep UI running after pipestance completes or fails.
    --noui           Disable UI.
    --novdr          Disable Volatile Data Removal.
    --profile        Enable stage performance profiling.
    --maxcores=<num> Set max cores the pipeline may request at one time.
    --maxmem=<num>   Set max GB the pipeline may request at one time.
    --sge            Run jobs on Sun Grid Engine instead of locally.
                     (--maxcores and --maxmem will be ignored)
    --debug          Enable debug logging for local scheduler.
    --stest          Substitute real stages with stress-testing stage.
    -h --help        Show this message.
    --version        Show version.`
	marioVersion := core.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, marioVersion, false)
	core.LogInfo("*", "Mario Run Pipeline")
	core.LogInfo("version", marioVersion)
	core.LogInfo("cmdline", strings.Join(os.Args, " "))

	// Required job mode and SGE environment variables.
	jobMode := "local"
	if opts["--sge"].(bool) {
		jobMode = "sge"
		core.EnvRequire([][]string{
			{"SGE_ROOT", "path/to/sge/root"},
			{"SGE_CLUSTER_NAME", "SGE cluster name"},
			{"SGE_CELL", "usually 'default'"},
		}, true)
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
	core.LogInfo("environ", "MROPATH = %s", mroPath)
	core.LogInfo("version", "MROPATH = %s", mroVersion)

	// Compute UI port.
	uiport := "3600"
	noUI := false
	if value := os.Getenv("MROPORT"); len(value) > 0 {
		core.LogInfo("environ", "MROPORT = %s", value)
		uiport = value
	}
	if value := opts["--port"]; value != nil {
		uiport = value.(string)
	}
	if opts["--noui"].(bool) {
		uiport = ""
		noUI = true
	}

	// Compute profiling flag.
	profile := opts["--profile"].(bool)
	if value := os.Getenv("MROPROFILE"); len(value) > 0 {
		profile = true
	}
	core.LogInfo("environ", "MROPROFILE = %v", profile)

	// Compute no debug dump flag.
	noDump := false
	if value := os.Getenv("MRONODUMP"); len(value) > 0 {
		noDump = true
	}
	core.LogInfo("environ", "MRONODUMP = %v", noDump)

	// Setup invocation-specific values.
	disableVDR := opts["--novdr"].(bool)
	noExit := opts["--noexit"].(bool)
	psid := opts["<pipestance_name>"].(string)
	invocationPath := opts["<call.mro>"].(string)
	pipestancePath := path.Join(cwd, psid)
	stepSecs := 4
	debug := opts["--debug"].(bool)
	stest := opts["--stest"].(bool)

	// Validate psid.
	core.DieIf(core.ValidateID(psid))

	//=========================================================================
	// Configure Mario runtime.
	//=========================================================================
	rt := core.NewRuntimeWithCores(jobMode, mroPath, marioVersion, mroVersion,
		reqCores, reqMem, profile, debug, stest)

	// Print this here because the log makes more sense when this appears before
	// the runloop messages start to appear.
	if noUI {
		core.LogInfo("webserv", "UI disabled by --noui option.")
	} else {
		core.LogInfo("webserv", "Serving UI at http://localhost:%s", uiport)
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
			pipestance, err = rt.ReattachToPipestance(psid, pipestancePath)
			core.DieIf(err)
		}
		core.DieIf(err)
	}

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
		"version":    marioVersion,
		"pname":      pipestance.GetPname(),
		"psid":       psid,
		"state":      pipestance.GetState(),
		"jobmode":    jobMode,
		"maxcores":   strconv.Itoa(rt.Scheduler.GetMaxCores()),
		"maxmemgb":   strconv.Itoa(rt.Scheduler.GetMaxMemGB()),
		"invokepath": invocationPath,
		"invokesrc":  invocationSrc,
		"MROPATH":    mroPath,
		"MRONODUMP":  fmt.Sprintf("%v", noDump),
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
	go runLoop(pipestance, stepSecs, disableVDR, noExit, noDump, noUI)

	// Let daemons take over.
	done := make(chan bool)
	<-done
}

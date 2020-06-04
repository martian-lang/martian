//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/martian-lang/martian/martian/core"
	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"

	"github.com/martian-lang/docopt.go"
)

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
	cert           *tls.Config
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

// Parse command-line options and MROFLAGS environment variable.
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
                            local (default)
                            A cluster job mode listed such as sge, lsf, or slurm
                            A file <jobmode>.template
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
                            post, rolling (default), strict, or disable

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
    --https-cert=FILE   Set path to a file containing the TLS certificate to use
                        for the user interface.
                            If set, https-key must also be provided.
                            The UI will then use https.
    --https-key=FILE    Set the path to the file containing the private key for
                        serving the UI over https.
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

	logEnviron(config.MartianVersion, os.Args, os.Environ(), os.Getpid())

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
	if config.ProfileMode != "" {
		util.LogInfo("options", "--profile=%s", config.ProfileMode)
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
	var certFile, keyFile string
	if value := opts["--https-cert"]; value != nil {
		if p, ok := value.(string); ok && p != "" {
			if filepath.IsAbs(p) {
				certFile = p
			} else {
				certFile = path.Join(cwd, p)
			}
			util.LogInfo("options", "--https-cert=%s", certFile)
		}
	}
	if value := opts["--https-key"]; value != nil {
		if p, ok := value.(string); ok && p != "" {
			if filepath.IsAbs(p) {
				keyFile = p
			} else {
				keyFile = path.Join(cwd, p)
			}
			util.LogInfo("options", "--https-key=%s", keyFile)
		}
	}
	if certFile != "" && keyFile == "" {
		util.PrintInfo("options", "--https-cert provided, but no --https-key.")
		os.Exit(1)
	} else if certFile == "" && keyFile != "" {
		util.PrintInfo("options", "--https-key provided, but no --https-cert.")
		os.Exit(1)
	} else if certFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			util.PrintError(err, "options", "Failed to read https key pair.")
			os.Exit(1)
		}
		c.cert = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}
	return c
}

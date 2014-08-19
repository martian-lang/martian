//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario pipeline runner.
//
package main

import (
	"bytes"
	"encoding/json"
	"github.com/docopt/docopt-go"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"html/template"
	"io/ioutil"
	"margo/core"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

//=============================================================================
// Pipestance runner.
//=============================================================================
func runLoop(pipestance *core.Pipestance, stepSecs int) {
	nodes := pipestance.Node().AllNodes()
	for {
		// Concurrently run metadata refreshes.
		//fmt.Println("===============================================================")
		//start := time.Now()
		var wg sync.WaitGroup
		wg.Add(len(nodes))
		for _, node := range nodes {
			node.RefreshMetadata(&wg)
		}
		wg.Wait()
		//fmt.Println(time.Since(start))

		// Check for completion states.
		if pipestance.GetOverallState() == "complete" {
			core.LogInfo("RUNTIME", "Starting VDR kill...")
			killReport := pipestance.VDRKill()
			core.LogInfo("RUNTIME", "VDR killed %d files, %d bytes.", killReport.Count, killReport.Size)
			// Give time for web ui client to get last update.
			time.Sleep(time.Second * 10)
			core.LogInfo("RUNTIME", "Pipestance is complete, exiting.")
			os.Exit(0)
		}

		// Step all nodes.
		for _, node := range nodes {
			node.Step()
		}

		// Wait for a bit.
		time.Sleep(time.Second * time.Duration(stepSecs))
	}
}

//=============================================================================
// Web server.
//=============================================================================
type GraphPage struct {
	Container string
	Pname     string
	Psid      string
	Admin     bool
}

type MetadataForm struct {
	Path string
	Name string
}

func runWebServer(uiport string, rt *core.Runtime, pipestance *core.Pipestance) {
	//=========================================================================
	// Configure server.
	//=========================================================================
	m := martini.New()
	r := martini.NewRouter()
	m.Use(martini.Recovery())
	m.Use(martini.Static("../web/res", martini.StaticOptions{"", true, "index.html", nil}))
	m.Use(martini.Static("../web/client", martini.StaticOptions{"", true, "index.html", nil}))
	m.MapTo(r, (*martini.Routes)(nil))
	m.Action(r.Handle)
	app := &martini.ClassicMartini{m, r}

	//=========================================================================
	// Page renderers.
	//=========================================================================
	app.Get("/", func() string {
		tmpl, _ := template.New("graph.html").Delims("[[", "]]").ParseFiles("../web/templates/graph.html")
		var doc bytes.Buffer
		tmpl.Execute(&doc, &GraphPage{"runner", pipestance.Pname(), pipestance.Psid(), true})
		return doc.String()
	})

	//=========================================================================
	// API endpoints.
	//=========================================================================

	// Get graph nodes.
	app.Get("/api/get-nodes/:container/:pname/:psid",
		func(p martini.Params) string {
			data := []interface{}{}
			for _, node := range pipestance.Node().AllNodes() {
				data = append(data, node.Serialize())
			}
			bytes, _ := json.Marshal(data)
			return string(bytes)
		})

	// Get metadata file contents.
	app.Post("/api/get-metadata/:container/:pname/:psid", binding.Bind(MetadataForm{}),
		func(body MetadataForm, p martini.Params) string {
			if strings.Index(body.Path, "..") > -1 {
				return "'..' not allowed in path."
			}
			data, err := ioutil.ReadFile(path.Join(body.Path, "_"+body.Name))
			if err != nil {
				return err.Error()
			}
			return string(data)
		})

	// Restart failed stage.
	app.Post("/api/restart/:container/:pname/:psid/:fqname",
		func(p martini.Params) string {
			node := pipestance.Node().Find(p["fqname"])
			var wg sync.WaitGroup
			node.RestartFailedMetadatas(&wg)
			wg.Wait()
			return ""
		})

	//=========================================================================
	// Start webserver.
	//=========================================================================
	http.ListenAndServe(":"+uiport, app)
}

func main() {
	runtime.GOMAXPROCS(2)

	//=========================================================================
	// Commandline argument and environment variables.
	//=========================================================================
	// Parse commandline.
	doc :=
		`Usage: 
    mrp <invocation_mro> <unique_pipestance_id> [--sge]
    mrp -h | --help | --version`
	opts, _ := docopt.Parse(doc, nil, true, "mrp", false)

	// Required Mario environment variables.
	env := core.EnvRequire([][]string{
		{"MARIO_PORT", ">2000"},
		{"MARIO_PIPELINES_PATH", "path/to/pipelines"},
	}, true)

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

	// Prepare configuration variables.
	uiport := env["MARIO_PORT"]
	psid := opts["<unique_pipestance_id>"].(string)
	invocationPath := opts["<invocation_mro>"].(string)
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	pipestancePath := path.Join(cwd, psid)
	callSrc, _ := ioutil.ReadFile(invocationPath)
	stepSecs := 3

	//=========================================================================
	// Configure Mario runtime.
	//=========================================================================
	rt := core.NewRuntime(jobMode, env["MARIO_PIPELINES_PATH"])
	_, err := rt.CompileAll()
	core.DieIf(err)

	//=========================================================================
	// Invoke pipestance or Reattach if exists.
	//=========================================================================
	pipestance, err := rt.InvokeWithSource(psid, string(callSrc), pipestancePath)
	if err != nil {
		// If it already exists, try to reattach to it.
		pipestance, err = rt.Reattach(psid, pipestancePath)
		core.DieIf(err)
	}

	//=========================================================================
	// Start run loop.
	//=========================================================================
	go runLoop(pipestance, stepSecs)

	//=========================================================================
	// Start web server.
	//=========================================================================
	go runWebServer(uiport, rt, pipestance)

	// Let daemons take over.
	done := make(chan bool)
	<-done
}

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
	"time"
)

func main() {
	runtime.GOMAXPROCS(2)

	// Command-line arguments.
	doc :=
		`Usage: 
    mrp <invocation_mro> <unique_pipestance_id> [--sge]
    mrp -h | --help | --version`
	opts, _ := docopt.Parse(doc, nil, true, "mrp", false)

	// Mario environment variables.
	env := core.EnvRequire([][]string{
		{"MARIO_PORT", ">2000"},
		{"MARIO_PIPELINES_PATH", "path/to/pipelines"},
	}, true)

	// Job mode and SGE environment variables.
	jobMode := "local"
	if opts["--sge"].(bool) {
		jobMode = "sge"
		core.EnvRequire([][]string{
			{"SGE_ROOT", "path/to/sge/root"},
			{"SGE_CLUSTER_NAME", "SGE cluster name"},
			{"SGE_CELL", "usually 'default'"},
		}, true)
	}

	// Compile MRO files.
	rt := core.NewRuntime(jobMode, env["MARIO_PIPELINES_PATH"])
	_, err := rt.CompileAll()
	core.DieIf(err)

	// psid, invocation file, and pipestance.
	uiport := env["MARIO_PORT"]
	psid := opts["<unique_pipestance_id>"].(string)
	invocationPath := opts["<invocation_mro>"].(string)
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	pipestancePath := path.Join(cwd, psid)
	callSrc, _ := ioutil.ReadFile(invocationPath)
	STEP_SECS := 3

	// Invoke pipestance.
	pipestance, pname, err := rt.InvokeWithSource(psid, string(callSrc), pipestancePath)
	if err != nil {
		// If it already exists, try to reattach to it.
		pipestance, pname, err = rt.Reattach(psid, pipestancePath)
		core.DieIf(err)
	}

	// Start the runner loop.
	go func() {
		nodes := pipestance.Node().AllNodes()
		for {
			// Concurrently run metadata refreshes.
			//fmt.Println("===============================================================")
			//start := time.Now()
			done := make(chan bool)
			count := 0
			for _, node := range nodes {
				count += node.RefreshMetadata(done)
			}
			for i := 0; i < count; i++ {
				<-done
			}
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
			time.Sleep(time.Second * time.Duration(STEP_SECS))
		}
	}()

	// Start the web server.
	m := martini.New()
	r := martini.NewRouter()
	m.Use(martini.Recovery())
	m.Use(martini.Static("../web/res", martini.StaticOptions{"", true, "index.html", nil}))
	m.Use(martini.Static("../web/client", martini.StaticOptions{"", true, "index.html", nil}))
	m.MapTo(r, (*martini.Routes)(nil))
	m.Action(r.Handle)
	app := &martini.ClassicMartini{m, r}

	// Pages
	type Graph struct {
		Container string
		Pname     string
		Psid      string
		Admin     bool
	}
	app.Get("/", func() string {
		tmpl, _ := template.New("graph.html").Delims("[[", "]]").ParseFiles("../web/templates/graph.html")
		var doc bytes.Buffer
		err = tmpl.Execute(&doc, &Graph{
			Container: "runner",
			Pname:     pname,
			Psid:      psid,
			Admin:     true,
		})
		return doc.String()
	})

	// APIs
	// Get graph nodes.
	app.Get("/api/get-nodes/:container/:pname/:psid", func(params martini.Params) string {
		data := []interface{}{}
		for _, node := range pipestance.Node().AllNodes() {
			data = append(data, node.Serialize())
		}
		bytes, _ := json.Marshal(data)
		return string(bytes)
	})

	// Get metadata contents.
	type MetadataForm struct {
		Path string
		Name string
	}
	app.Post("/api/get-metadata/:container/:pname/:psid", binding.Bind(MetadataForm{}), func(body MetadataForm, params martini.Params) string {
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
	app.Post("/api/restart/:container/:pname/:psid/:fqname", func(params martini.Params) string {
		node := pipestance.Node().Find(params["fqname"])
		done := make(chan bool)
		count := node.RestartFailedMetadatas(done)
		for i := 0; i < count; i++ {
			<-done
		}
		return ""
	})

	http.ListenAndServe(":"+uiport, app)
}

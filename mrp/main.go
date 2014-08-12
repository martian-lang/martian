//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Margo
//
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"html/template"
	"io/ioutil"
	"margo/core"
	"os"
	"path"
	"path/filepath"
	"runtime"
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
	})

	// Job mode and SGE environment variables.
	JOBMODE := "local"
	if opts["--sge"].(bool) {
		JOBMODE = "sge"
		core.EnvRequire([][]string{
			{"SGE_ROOT", "path/to/sge/root"},
			{"SGE_CLUSTER_NAME", "SGE cluster name"},
			{"SGE_CELL", "usually 'default'"},
		})
	}

	// Compile MRO files.
	rt := core.NewRuntime(JOBMODE, env["MARIO_PIPELINES_PATH"])
	_, err := rt.CompileAll()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// psid, invocation file, and pipestance.
	psid := opts["<unique_pipestance_id>"].(string)
	INVOCATION_PATH := opts["<invocation_mro>"].(string)
	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	PIPESTANCE_PATH := path.Join(cwd, psid)
	callSrc, _ := ioutil.ReadFile(INVOCATION_PATH)
	os.MkdirAll(PIPESTANCE_PATH, 0700)
	STEP_SECS := 3

	// Invoke pipestance.
	pipestance, pname, err := rt.InvokeWithSource(psid, string(callSrc), PIPESTANCE_PATH)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
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
				// Give time for web ui client to get last update.
				time.Sleep(time.Second * 10)
				fmt.Println("[RUNTIME]", core.Timestamp(), "Pipestance is complete, exiting.")
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

	// API: Pipestance Browser
	// Pages
	type Graph struct {
		Container string
		Pname     string
		Psid      string
		Admin     bool
	}
	app.Get("/", func() string {
		tmpl, err := template.New("graph.html").Delims("[[", "]]").ParseFiles("../web/templates/graph.html")
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		var doc bytes.Buffer
		err = tmpl.Execute(&doc, &Graph{
			Container: "runner",
			Pname:     pname,
			Psid:      psid,
			Admin:     true,
		})
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		return doc.String()
	})

	// APIs
	// Get graph nodes
	app.Get("/api/get-nodes/:container/:pname/:psid", func(params martini.Params) string {
		data := []interface{}{}
		for _, node := range pipestance.Node().AllNodes() {
			data = append(data, node.Serialize())
		}
		bytes, _ := json.Marshal(data)
		return string(bytes)
	})

	// Get metadata contents
	type MetadataForm struct {
		Path string `form:"path" binding:"required"`
		Name string `form:"name" binding:"required"`
	}
	app.Post("/api/get-metadata/:container/:pname/:psid", binding.Bind(MetadataForm{}), func(body MetadataForm, params martini.Params) string {
		// TODO sanitize input, check for '..'
		data, err := ioutil.ReadFile(path.Join(body.Path, "_"+body.Name))
		if err != nil {
			fmt.Println(err.Error())
		}
		return string(data)
	})

	// Restart failed stage
	app.Post("/api/restart/:container/:pname/:psid/:fqname", func(params martini.Params) string {
		// TODO sanitize input, check for '..'
		node := pipestance.Node().Find(params["fqname"])
		done := make(chan bool)
		count := node.RestartFailedMetadatas(done)
		for i := 0; i < count; i++ {
			<-done
		}
		return ""
	})

	app.Run()
}

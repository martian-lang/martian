//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// mrp webserver.
//
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"martian/core"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/gzip"
)

//=============================================================================
// Page and form structs.
//=============================================================================
type GraphPage struct {
	InstanceName string
	Container    string
	Pname        string
	Psid         string
	Admin        bool
	AdminStyle   bool
	Release      bool
}

type MetadataForm struct {
	Path string
	Name string
}

func getSerialization(rt *core.Runtime, pipestance *core.Pipestance, name string) interface{} {
	if ser, ok := rt.GetSerialization(pipestance.GetPath(), name); ok {
		return ser
	}
	return pipestance.Serialize(name)
}

func runWebServer(uiport string, rt *core.Runtime, pipestanceBox *pipestanceHolder,
	info map[string]string) {
	//=========================================================================
	// Configure server.
	//=========================================================================
	m := martini.New()
	r := martini.NewRouter()
	m.Use(martini.Recovery())
	m.Use(martini.Static(core.RelPath("../web/martian/res"),
		martini.StaticOptions{"", true, "index.html", nil}))
	m.Use(martini.Static(core.RelPath("../web/martian/client"),
		martini.StaticOptions{"", true, "index.html", nil}))
	m.MapTo(r, (*martini.Routes)(nil))
	m.Action(r.Handle)
	app := &martini.ClassicMartini{m, r}
	app.Use(gzip.All())

	//=========================================================================
	// Page renderers.
	//=========================================================================
	app.Get("/", func() string {
		pipestance := pipestanceBox.getPipestance()
		tmpl, _ := template.New("graph.html").Delims("[[", "]]").ParseFiles(core.RelPath("../web/martian/templates/graph.html"))
		var doc bytes.Buffer
		tmpl.Execute(&doc, &GraphPage{
			InstanceName: "Martian Pipeline Runner",
			Container:    "runner",
			Pname:        pipestance.GetPname(),
			Psid:         pipestance.GetPsid(),
			Admin:        true,
			AdminStyle:   false,
			Release:      core.IsRelease(),
		})
		return doc.String()
	})

	//=========================================================================
	// API endpoints.
	//=========================================================================
	// Get pipestance state: nodes and fatal error (if any).
	app.Get("/api/get-info", func(p martini.Params) string {
		pipestance := pipestanceBox.getPipestance()
		info["state"] = pipestance.GetState()
		bytes, _ := json.Marshal(info)
		return string(bytes)
	})

	// Get pipestance state: nodes and fatal error (if any).
	app.Get("/api/get-state/:container/:pname/:psid",
		func(p martini.Params) string {
			pipestance := pipestanceBox.getPipestance()
			state := map[string]interface{}{}
			info["state"] = pipestance.GetState()
			state["nodes"] = getSerialization(rt, pipestance, "finalstate")
			state["info"] = info
			bytes, _ := json.Marshal(state)
			return string(bytes)
		})

	// Get pipestance performance data: disable API endpoint for release
	if !core.IsRelease() {
		app.Get("/api/get-perf/:container/:pname/:psid",
			func(p martini.Params) string {
				pipestance := pipestanceBox.getPipestance()
				state := map[string]interface{}{}
				state["nodes"] = getSerialization(rt, pipestance, "perf")
				bytes, _ := json.Marshal(state)
				return string(bytes)
			})
	}

	// Get metadata file contents.
	app.Post("/api/get-metadata/:container/:pname/:psid", binding.Bind(MetadataForm{}),
		func(body MetadataForm, p martini.Params) string {
			pipestance := pipestanceBox.getPipestance()
			if strings.Index(body.Path, "..") > -1 {
				return "'..' not allowed in path."
			}
			data, err := rt.GetMetadata(pipestance.GetPath(), path.Join(body.Path, "_"+body.Name))
			if err != nil {
				return err.Error()
			}
			return data
		})

	// Restart failed stage.
	app.Post("/api/restart/:container/:pname/:psid",
		func(p martini.Params) string {
			pipestance := pipestanceBox.getPipestance()
			if err := pipestance.Reset(); err != nil {
				return err.Error()
			}
			return ""
		})

	//=========================================================================
	// Start webserver.
	//=========================================================================
	if err := http.ListenAndServe(":"+uiport, app); err != nil {
		// Don't continue starting if we detect another instance running.
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

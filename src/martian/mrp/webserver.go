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
	"io/ioutil"
	"martian/core"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
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
	Auth         string
}

type MetadataForm struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

func getSerialization(rt *core.Runtime, pipestance *core.Pipestance, name core.MetadataFileName) interface{} {
	if ser, ok := rt.GetSerialization(pipestance.GetPath(), name); ok {
		return ser
	}
	return pipestance.Serialize(name)
}

func runWebServer(
	listener net.Listener,
	rt *core.Runtime,
	pipestanceBox *pipestanceHolder,
	info *PipestanceInfo,
	authKey string,
	requireAuth bool) {
	server := &mrpWebServer{
		listener:      listener,
		webRoot:       findWebRoot(),
		rt:            rt,
		pipestanceBox: pipestanceBox,
		info:          info,
		authKey:       authKey,
		readAuth:      requireAuth,
	}
	server.Start()
}

func findWebRoot() string {
	return core.RelPath(path.Join("..", "web", "martian"))
}

type mrpWebServer struct {
	listener net.Listener

	// The authentication token expected in the URL.
	authKey string

	// True if authentication is required for read-only commands.
	// Authentication is always required for write commands.
	readAuth bool

	rt            *core.Runtime
	pipestanceBox *pipestanceHolder
	webRoot       string
	info          *PipestanceInfo
	graphPage     []byte
	mutex         sync.Mutex
}

func (self *mrpWebServer) Start() {
	self.makeGraphPage()

	sm := http.NewServeMux()
	self.handleApi(sm)
	self.handleStatic(sm)
	sm.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" &&
			req.URL.Path != "/index.html" &&
			req.URL.Path != "/graph.html" {
			http.NotFound(w, req)
			return
		} else {
			self.serveGraphPage(w, req)
		}
	})

	if err := http.Serve(self.listener, sm); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

// Checks that the request includes a valid authentication token, if required.
// If it does not, it writes an error to the response and returns false.
func (self *mrpWebServer) verifyAuth(w http.ResponseWriter, req *http.Request) bool {
	if err := req.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}
	if self.authKey == "" {
		return true
	}
	key := req.FormValue("auth")
	// No early abort on the check here, to prevent timing attacks.
	// (not that this is serious security anyway...)
	authKey := []byte(self.authKey)
	pass := len(self.authKey) == len(key)
	for i, c := range []byte(key) {
		if i >= len(authKey) || authKey[i] != c {
			pass = false
		}
	}
	if !pass {
		http.Error(w, "This API requires authentication.", http.StatusUnauthorized)
	}
	return pass
}

//=========================================================================
// Web endpoints.
//=========================================================================

func (self *mrpWebServer) handleStatic(sm *http.ServeMux) {
	sm.Handle("/graph.js", http.FileServer(http.Dir(path.Join(self.webRoot, "client"))))
	res := http.FileServer(http.Dir(path.Join(self.webRoot, "res")))
	sm.Handle("/css/", res)
	sm.Handle("/fonts/", res)
	sm.Handle("/js/", res)
}

func (self *mrpWebServer) graphTemplate() (*template.Template, error) {
	return template.New("graph.html").Delims(
		"[[", "]]").ParseFiles(
		path.Join(self.webRoot, "templates", "graph.html"))
}

func (self *mrpWebServer) makeGraphPage() {
	pipestance := self.pipestanceBox.getPipestance()
	var buff bytes.Buffer
	if tmpl, err := self.graphTemplate(); err != nil {
		core.Println("Error starting web server: %v", err)
	} else {
		graphParams := GraphPage{
			InstanceName: "Martian Pipeline Runner",
			Container:    "runner",
			Pname:        pipestance.GetPname(),
			Psid:         pipestance.GetPsid(),
			Admin:        true,
			AdminStyle:   false,
			Release:      core.IsRelease(),
		}
		if self.authKey != "" && self.readAuth {
			graphParams.Auth = "?auth=" + self.authKey
		}
		if err := tmpl.Execute(&buff, &graphParams); err != nil {
			core.Println("Error starting web server: %v", err)
		} else {
			self.graphPage = buff.Bytes()
		}
	}
}

func (self *mrpWebServer) serveGraphPage(w http.ResponseWriter, req *http.Request) {
	if !self.readAuth || self.verifyAuth(w, req) {
		w.Write(self.graphPage)
	}
}

//=========================================================================
// API endpoints.
//=========================================================================

func (self *mrpWebServer) handleApi(sm *http.ServeMux) {
	sm.HandleFunc("/api/get-info/", self.getInfo)
	sm.HandleFunc("/api/get-info", self.getInfo)
	sm.HandleFunc("/api/get-state/", self.getState)
	sm.HandleFunc("/api/get-state", self.getState)
	sm.HandleFunc("/api/get-perf/", self.getPerf)
	sm.HandleFunc("/api/get-perf", self.getPerf)
	sm.HandleFunc("/api/get-metadata/", self.getMetadata)
	sm.HandleFunc("/api/get-metadata", self.getMetadata)
	sm.HandleFunc("/api/restart/", self.restart)
	sm.HandleFunc("/api/restart", self.restart)
	sm.HandleFunc("/api/get-metadata-top/", self.getMetadataTop)
	sm.HandleFunc("/api/kill", self.kill)
}

// Get pipestance state: nodes and fatal error (if any).
func (self *mrpWebServer) getInfo(w http.ResponseWriter, req *http.Request) {
	if self.readAuth && !self.verifyAuth(w, req) {
		return
	}
	pipestance := self.pipestanceBox.getPipestance()
	st := pipestance.GetState()
	self.mutex.Lock()
	self.info.State = st
	bytes, err := json.Marshal(self.info)
	self.mutex.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(bytes)
}

// Get pipestance state: nodes and fatal error (if any).
func (self *mrpWebServer) getState(w http.ResponseWriter, req *http.Request) {
	if self.readAuth && !self.verifyAuth(w, req) {
		return
	}
	pipestance := self.pipestanceBox.getPipestance()
	state := map[string]interface{}{}
	st := pipestance.GetState()
	state["nodes"] = getSerialization(self.rt, pipestance, "finalstate")
	state["info"] = self.info
	self.mutex.Lock()
	self.info.State = st
	bytes, err := json.Marshal(state)
	self.mutex.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(bytes)
}

// Get pipestance performance data: disable API endpoint for release
func (self *mrpWebServer) getPerf(w http.ResponseWriter, req *http.Request) {
	if self.readAuth && !self.verifyAuth(w, req) {
		return
	}
	pipestance := self.pipestanceBox.getPipestance()
	state := map[string]interface{}{}
	state["nodes"] = getSerialization(self.rt, pipestance, "perf")
	bytes, err := json.Marshal(state)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(bytes)
}

// Get metadata file contents.
func (self *mrpWebServer) getMetadata(w http.ResponseWriter, req *http.Request) {
	// Someone thought it was a good idea to put a JSON object in the body
	// instead of making a proper REST request.
	var name, p string
	if body, err := ioutil.ReadAll(req.Body); err != nil || len(body) <= 0 {
		http.Error(w, "Request body is required.", http.StatusBadRequest)
		return
	} else {
		form := MetadataForm{}
		if err := json.Unmarshal(body, &form); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		p = path.Clean(form.Path)
		name = form.Name
	}
	if self.readAuth && !self.verifyAuth(w, req) {
		return
	}
	pipestance := self.pipestanceBox.getPipestance()
	if strings.HasPrefix(p, "..") {
		http.Error(w, "'..' not allowed in path.", http.StatusBadRequest)
		return
	}
	data, err := self.rt.GetMetadata(pipestance.GetPath(),
		path.Join(p, "_"+name))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Write([]byte(data))
}

// Get metadata from the pipestance top-level.
func (self *mrpWebServer) getMetadataTop(w http.ResponseWriter, req *http.Request) {
	if self.readAuth && !self.verifyAuth(w, req) {
		return
	}
	p := path.Clean(strings.TrimLeft(strings.TrimPrefix(
		req.URL.Path, "/api/get-metadata-top"), "/"))
	if strings.HasPrefix(p, "..") {
		http.Error(w, "'..' not allowed in path.", http.StatusBadRequest)
		return
	}
	pipestance := self.pipestanceBox.getPipestance()
	data, err := self.rt.GetMetadata(pipestance.GetPath(),
		path.Join(pipestance.GetPath(), "_"+path.Base(p)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Write([]byte(data))
}

// Restart failed stage.
func (self *mrpWebServer) restart(w http.ResponseWriter, req *http.Request) {
	if !self.verifyAuth(w, req) {
		return
	}
	pipestance := self.pipestanceBox.getPipestance()
	if err := pipestance.Reset(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Kill the pipestance.
func (self *mrpWebServer) kill(w http.ResponseWriter, req *http.Request) {
	if !self.verifyAuth(w, req) {
		return
	}
	self.pipestanceBox.getPipestance().Kill()
}

//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//
// mrp webserver.
//

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/martian-lang/martian/martian/api"
	"github.com/martian-lang/martian/martian/core"
	"github.com/martian-lang/martian/martian/util"
)

func getFinalState(rt *core.Runtime, pipestance *core.Pipestance) []*core.NodeInfo {
	var target []*core.NodeInfo
	if err := rt.GetSerializationInto(pipestance.GetPath(), core.FinalState, &target); err == nil {
		return target
	}
	return pipestance.SerializeState()
}

func getPerf(rt *core.Runtime, pipestance *core.Pipestance) []*core.NodePerfInfo {
	var target []*core.NodePerfInfo
	if err := rt.GetSerializationInto(pipestance.GetPath(), core.Perf, &target); err == nil {
		return target
	}
	return pipestance.SerializePerf()
}

func runWebServer(
	listener net.Listener,
	rt *core.Runtime,
	pipestanceBox *pipestanceHolder,
	requireAuth bool) {
	server := &mrpWebServer{
		listener:      listener,
		webRoot:       findWebRoot(),
		rt:            rt,
		pipestanceBox: pipestanceBox,
		readAuth:      requireAuth,
	}
	server.Start()
}

func findWebRoot() string {
	return util.RelPath(path.Join("..", "web", "martian"))
}

type mrpWebServer struct {
	listener net.Listener

	// True if authentication is required for read-only commands.
	// Authentication is always required for write commands.
	readAuth bool

	rt            *core.Runtime
	pipestanceBox *pipestanceHolder
	webRoot       string
	graphPage     []byte
	startTime     time.Time
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
	api.EnableDebug(sm, self.verifyAuth)

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
	if self.pipestanceBox.authKey == "" {
		return true
	}
	key := req.FormValue("auth")
	// No early abort on the check here, to prevent timing attacks.
	// (not that this is serious security anyway...)
	authKey := []byte(self.pipestanceBox.authKey)
	pass := len(self.pipestanceBox.authKey) == len(key)
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
	res := http.FileServer(http.Dir(path.Join(self.webRoot, "serve")))
	contentGzip := func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		res.ServeHTTP(w, req)
	}
	sm.HandleFunc("/graph.js", contentGzip)
	sm.HandleFunc("/favicon.ico", contentGzip)
	sm.HandleFunc("/css/", contentGzip)
	sm.HandleFunc("/js/", contentGzip)
	sm.Handle("/fonts/", res)
}

func (self *mrpWebServer) graphTemplate() (*template.Template, error) {
	return template.New("graph.html").Delims(
		"[[", "]]").ParseFiles(
		path.Join(self.webRoot, "templates", "graph.html"))
}

func (self *mrpWebServer) makeGraphPage() {
	pipestance := self.pipestanceBox.getPipestance()
	if tmpl, err := self.graphTemplate(); err != nil {
		util.Println("Error starting web server: %v", err)
	} else {
		graphParams := api.GraphPage{
			InstanceName: "Martian Pipeline Runner",
			Container:    "runner",
			Pname:        pipestance.GetPname(),
			Psid:         pipestance.GetPsid(),
			Admin:        true,
			AdminStyle:   false,
			Release:      util.IsRelease(),
		}
		if self.pipestanceBox.authKey != "" && self.readAuth {
			graphParams.Auth = "?auth=" + self.pipestanceBox.authKey
		}
		var buff bytes.Buffer
		zipper, _ := gzip.NewWriterLevel(&buff, gzip.BestCompression)
		if err := tmpl.Execute(zipper, &graphParams); err != nil {
			util.PrintError(err, "webserv", "Error starting web server.")
		} else {
			if err := zipper.Close(); err != nil {
				util.PrintError(err, "webserv", "Error starting web server.")
			} else {
				self.startTime = time.Now()
				self.graphPage = buff.Bytes()
			}
		}
	}
}

func (self *mrpWebServer) serveGraphPage(w http.ResponseWriter, req *http.Request) {
	if !self.readAuth || self.verifyAuth(w, req) {
		w.Header().Set("Content-Encoding", "gzip")
		http.ServeContent(w, req, "graph.html", self.startTime,
			bytes.NewReader(self.graphPage))
	}
}

//=========================================================================
// API endpoints.
//=========================================================================

func (self *mrpWebServer) handleApi(sm *http.ServeMux) {
	sm.HandleFunc(api.QueryGetInfo, self.getInfo)
	sm.HandleFunc(api.QueryGetInfo+"/", self.getInfo)
	sm.HandleFunc(api.QueryGetState, self.getState)
	sm.HandleFunc(api.QueryGetState+"/", self.getState)
	sm.HandleFunc(api.QueryGetPerf, self.getPerf)
	sm.HandleFunc(api.QueryGetPerf+"/", self.getPerf)
	sm.HandleFunc(api.QueryGetMetadata, self.getMetadata)
	sm.HandleFunc(api.QueryGetMetadata+"/", self.getMetadata)
	sm.HandleFunc(api.QueryRestart, self.restart)
	sm.HandleFunc(api.QueryRestart+"/", self.restart)
	sm.HandleFunc(api.QueryGetMetadataTop, self.getMetadataTop)
	sm.HandleFunc(api.QueryKill, self.kill)
}

// Get pipestance state: nodes and fatal error (if any).
func (self *mrpWebServer) getInfo(w http.ResponseWriter, req *http.Request) {
	if self.readAuth && !self.verifyAuth(w, req) {
		return
	}
	pipestance := self.pipestanceBox.getPipestance()
	st := pipestance.GetState()
	self.pipestanceBox.UpdateState(st)
	self.mutex.Lock()
	bytes, err := json.Marshal(self.pipestanceBox.info)
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
	state := api.PipestanceState{
		Nodes: getFinalState(self.rt, pipestance),
		Info:  self.pipestanceBox.info,
	}
	st := pipestance.GetState()
	self.pipestanceBox.UpdateState(st)
	self.mutex.Lock()
	bytes, err := json.Marshal(&state)
	self.mutex.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Encoding", "gzip")
	zipper, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
	zipper.Write(bytes)
	if err := zipper.Close(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Get pipestance performance data: disable API endpoint for release
func (self *mrpWebServer) getPerf(w http.ResponseWriter, req *http.Request) {
	if self.readAuth && !self.verifyAuth(w, req) {
		return
	}
	pipestance := self.pipestanceBox.getPipestance()
	state := api.PerfInfo{
		Nodes: getPerf(self.rt, pipestance),
	}
	bytes, err := json.Marshal(&state)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Encoding", "gzip")
	zipper, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
	zipper.Write(bytes)
	if err := zipper.Close(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
		form := api.MetadataForm{}
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
		req.URL.Path, api.QueryGetMetadataTop), "/"))
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
	if self.pipestanceBox.readOnly {
		http.Error(w, "mrp is in read-only mode.", http.StatusBadRequest)
		return
	}
	self.pipestanceBox.cleanupLock.Lock()
	defer self.pipestanceBox.cleanupLock.Unlock()
	if st := self.pipestanceBox.getPipestance().GetState(); st != core.Failed {
		http.Error(w, "Only failed pipestances can be restarted.", http.StatusBadRequest)
		return
	}
	if err := self.pipestanceBox.reset(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Kill the pipestance.
func (self *mrpWebServer) kill(w http.ResponseWriter, req *http.Request) {
	if !self.verifyAuth(w, req) {
		return
	}
	util.LogInfo("webserv", "Got API shutdown request.")
	go func() {
		self.pipestanceBox.cleanupLock.Lock()
		defer self.pipestanceBox.cleanupLock.Unlock()
		if !self.pipestanceBox.readOnly {
			self.pipestanceBox.getPipestance().KillWithMessage(
				"Pipstance was killed by API call from " + req.RemoteAddr)
			time.Sleep(6 * time.Second) // Make sure UI has a chance to refresh.
		}
		util.Suicide()
	}()
}

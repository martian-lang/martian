//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// mrp webserver.
//

package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/martian-lang/martian/martian/api"
	"github.com/martian-lang/martian/martian/api/webdebug"
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
	webdebug.EnableDebug(sm, self.verifyAuth)

	self.pipestanceBox.server = &http.Server{
		Handler:      sm,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 65 * time.Second,
		IdleTimeout:  time.Minute,
	}
	self.pipestanceBox.server.ErrorLog, _ = util.GetLogger("webserv")
	util.RegisterSignalHandler(self)

	if err := self.pipestanceBox.server.Serve(self.listener); err != nil {
		if err != http.ErrServerClosed {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}
}

func (self *mrpWebServer) HandleSignal(os.Signal) {
	if srv := self.pipestanceBox.server; srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
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
}

func (self *mrpWebServer) graphTemplate() (*template.Template, error) {
	return template.New("graph.html").Delims(
		"[[", "]]").ParseFiles(
		path.Join(self.webRoot, "serve", "graph.html"))
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
		w.Header().Set("Content-Type", "text/html")
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
	p := self.pipestanceBox.getPipestance().GetPath()
	sm.Handle(api.QueryGetMetadataTop, self.authorize(pathToMetadata(
		http.FileServer(http.Dir(p)))))
	sm.HandleFunc(api.QueryListMetadataTop, self.listMetadataTop)
	sm.HandleFunc(api.QueryListMetadataTop+"/", self.listMetadataTop)
	sm.HandleFunc(api.QueryKill, self.kill)
	sm.Handle(api.QueryExtras, self.authorize(noDot(
		http.FileServer(http.Dir(path.Join(p, "extras"))))))
}

func (self *mrpWebServer) authorize(source http.Handler) http.Handler {
	if self.readAuth {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if self.verifyAuth(w, r) {
				source.ServeHTTP(w, r)
			}
		})
	} else {
		return source
	}
}

// Strips a request down to the base name and prepends the metadata file
// prefix.
func pathToMetadata(source http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := path.Base(r.URL.Path); len(p) > 0 {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = core.MetadataFilePrefix + p
			if t := core.MetadataFileName(p).MimeType(); t != "" {
				w.Header().Set("Content-Type", t)
			}
			source.ServeHTTP(w, r2)
		} else {
			http.NotFound(w, r)
		}
	})
}

// Strips a request down to the base name and returns 404 if that starts with
// a '.' character.
func noDot(source http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := path.Base(r.URL.Path); len(p) > 0 && p[0] != '.' {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p
			source.ServeHTTP(w, r2)
		} else {
			http.NotFound(w, r)
		}
	})
}

// Get pipestance state: nodes and fatal error (if any).
func (self *mrpWebServer) getInfo(w http.ResponseWriter, req *http.Request) {
	if self.readAuth && !self.verifyAuth(w, req) {
		return
	}
	self.mutex.Lock()
	bytes, err := json.Marshal(self.pipestanceBox.info)
	self.mutex.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(bytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
	self.mutex.Lock()
	bytes, err := json.Marshal(&state)
	self.mutex.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := req.Context().Err(); err != nil {
		// Don't sending bytes if the request was canceled.
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "application/json")
	zipper, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
	zipper.Write(bytes)
	if err := zipper.Close(); err != nil {
		// Can't use http.Error since the header was already set.
		fmt.Fprintf(w, "\nzip error: %v", err)
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

	if err := req.Context().Err(); err != nil {
		// Don't sending bytes if the request was canceled.
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "application/json")
	zipper, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
	zipper.Write(bytes)
	if err := zipper.Close(); err != nil {
		// Can't use http.Error since the header was already set.
		fmt.Fprintf(w, "\nzip error: %v", err)
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
		path.Join(p, core.MetadataFilePrefix+name))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer data.Close()
	api.ServeMetadataFile(w, req, name, data)
}

// Get the list of metadata files from the pipestance top-level.  This is a
// whitelisted subset of actual metadata files, because some of those files,
// such as _uuid, are uninteresting, while others such as _uiport, _versions,
// _finalstate and so on are redundant with other queries.
func (self *mrpWebServer) listMetadataTop(w http.ResponseWriter, req *http.Request) {
	if self.readAuth && !self.verifyAuth(w, req) {
		return
	}
	p := self.pipestanceBox.getPipestance().GetPath()
	if result, err := api.GetFilesListing(p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else if b, err := json.Marshal(&result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(b); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
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
	if st := self.pipestanceBox.getPipestance().GetState(req.Context()); st != core.Failed {
		http.Error(w, "Only failed pipestances can be restarted.", http.StatusBadRequest)
		return
	}
	if err := self.pipestanceBox.reset(req.Context()); err != nil {
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
				"Pipestance was killed by API call from " + req.RemoteAddr)
			time.Sleep(6 * time.Second) // Make sure UI has a chance to refresh.
		}
		if info := self.pipestanceBox.info; info != nil {
			util.Suicide(info.State == core.Complete)
		} else {
			util.Suicide(false)
		}
	}()
}

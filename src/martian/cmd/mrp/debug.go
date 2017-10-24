//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//
// Debug probe tools for mrp webserver.
//

package main

import (
	"net/http"
	"net/http/pprof"
	"runtime"
	"strconv"
)

// Enables debugging endpoints for profiling and getting stack traces
// on a running mrp instance.
func (self *mrpWebServer) handleDebug(sm *http.ServeMux) {
	sm.HandleFunc("/debug/pprof/", pprof.Index)
	sm.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	sm.HandleFunc("/debug/pprof/profile", pprof.Profile)
	sm.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	sm.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Turn on block and mutex profiling at the rate/fractions given by
	// the "block" and "mutex" form paramters.  Returns the previous
	// mutex sampling rate.  Requires authentication.
	sm.HandleFunc("/debug/enable_profile", self.enableProf)
}

func (self *mrpWebServer) enableProf(w http.ResponseWriter, req *http.Request) {
	if !self.verifyAuth(w, req) {
		return
	}
	blockRate := -1
	mutexFrac := -1
	if blk := req.FormValue("block"); blk != "" {
		if b, err := strconv.Atoi(blk); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			blockRate = b
		}
	}
	if mut := req.FormValue("mutex"); mut != "" {
		if m, err := strconv.Atoi(mut); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			mutexFrac = m
		}
	}
	if blockRate >= 0 {
		runtime.SetBlockProfileRate(blockRate)
	}
	w.Write([]byte(strconv.Itoa(runtime.SetMutexProfileFraction(mutexFrac))))
}

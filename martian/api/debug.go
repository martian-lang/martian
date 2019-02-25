//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//
// Debug probe tools for processes.
//

package api

import (
	"expvar"
	"net/http"
	"net/http/pprof"
	"runtime"
	"strconv"
)

// A function which should return true if authorization succeeds,
// and should write an error to the response and return false otherwise.
type AuthFunction func(http.ResponseWriter, *http.Request) bool

// Enables debugging endpoints for profiling and getting stack traces on a
// running instance.  See `go doc net/http/pprof`.  The difference is
// this doesn't rely on the default ServeMux.
//
// In addition to the standard pprof endpoints, the /debug/enable_profile
// endpoint allows remotely enabling the mutex and blocking profiling.  Posting
// to /debug/enable_profile?block=N&mutex=M sets the rates as given (see
// runtime.SetBlockProfileRate and runtime.SetMutexProfileFraction) If
// verifyAuth is non-null, it will be called and must return true before allowing
// the call is allowed to proceed.
func EnableDebug(sm *http.ServeMux, verifyAuth AuthFunction) {
	sm.HandleFunc("/debug/pprof/", pprof.Index)
	sm.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	sm.HandleFunc("/debug/pprof/profile", pprof.Profile)
	sm.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	sm.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Turn on block and mutex profiling at the rate/fractions given by
	// the "block" and "mutex" form parameters.  Returns the previous
	// mutex sampling rate.  Requires authentication.
	sm.HandleFunc("/debug/enable_profile", authorizeThenRun(verifyAuth, enableProf))

	// Expose exported variables, but require authorization.
	sm.HandleFunc("/debug/vars",
		authorizeThenRun(verifyAuth, expvar.Handler().ServeHTTP))
}

func authorizeThenRun(auth AuthFunction, then http.HandlerFunc) http.HandlerFunc {
	if auth == nil {
		return then
	}
	return func(w http.ResponseWriter, req *http.Request) {
		if !auth(w, req) {
			return
		}
		then(w, req)
	}
}

func enableProf(w http.ResponseWriter, req *http.Request) {
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

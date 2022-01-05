//go:build debug
// +build debug

/*
File Name:  Debug.go
Copyright:  2017 Kleissner Investments s.r.o.
Author:     Peter Kleissner

Debug runtime functionality. The functions only work if the config setting DebugAPI is enabled.
*/

package main

import (
	"net/http"
	"net/http/pprof" // Warning: If the default HTTP handler is used, this installs handlers!
	"runtime"

	"github.com/PeernetOfficial/core/webapi"
)

// apiDebugBugcheck handles /debug/bugcheck
func apiDebugBugcheck(w http.ResponseWriter, r *http.Request) {

	if !config.DebugAPI {
		http.Error(w, "", http.StatusOK)
		return
	}

	http.Error(w, "Executing immediate bugcheck", http.StatusOK)

	panic("via /debug/bugcheck")
}

// apiDebugStack handles /debug/stack
func apiDebugStack(w http.ResponseWriter, r *http.Request) {

	if !config.DebugAPI {
		http.Error(w, "", http.StatusOK)
		return
	}

	buffer := make([]byte, 1*1024*1024)
	size := runtime.Stack(buffer, true)

	http.Error(w, string(buffer[:size]), http.StatusOK)
}

func attachDebugAPI(api *webapi.WebapiInstance) {
	api.AllowKeyInParam = append(api.AllowKeyInParam, []string{
		"/debug/bugcheck",
		"/debug/stack",
		"/debug/pprof",
		"/debug/pprof/cmdline",
		"/debug/pprof/profile",
		"/debug/pprof/symbol",
		"/debug/pprof/trace",
		"/debug/pprof/goroutine",
		"/debug/pprof/heap",
		"/debug/pprof/threadcreate",
		"/debug/pprof/block",
		"/debug/pprof/allocs",
		"/debug/pprof/mutex",
	}...)

	api.Router.HandleFunc("/debug/bugcheck", apiDebugBugcheck)
	api.Router.HandleFunc("/debug/stack", apiDebugStack)

	api.Router.HandleFunc("/debug/pprof", pprof.Index)
	api.Router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	api.Router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	api.Router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	api.Router.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Manually add support for paths linked to by index page at /debug/pprof/
	api.Router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	api.Router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	api.Router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	api.Router.Handle("/debug/pprof/block", pprof.Handler("block"))
	api.Router.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	api.Router.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
}

/*
To use the pprof functionality set DebugAPI in the config to true and then use the right endpoints.

For heap profiling:
* While running, download the file http://[IP:Port]/debug/pprof/heap
* Run "go tool pprof -http=127.0.0.1:80 [service].exe heap". The http parameter will open an interactive web-server.
* pprof supports the following options. Default is inuse_space.
  -inuse_space      Display in-use memory size
  -inuse_objects    Display in-use object counts
  -alloc_space      Display allocated memory size
  -alloc_objects    Display allocated object counts
* Instead of download the file in step 2, you can directly provide the full URL in step 3!

Enabling DebugAPI shall not have any performance impact; it just installs the handlers. Only once a profiler is actually used, it may impact the performance.
*/

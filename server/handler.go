package server

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/sourcegraph/vcsstore"
	"github.com/sourcegraph/vcsstore/vcsclient"
	"github.com/sqs/mux"
)

var (
	Service vcsstore.Service

	router      = vcsclient.NewRouter(nil)
	routePrefix string

	Log = log.New(ioutil.Discard, "", 0)

	// InformativeErrors is whether to report internal error messages to HTTP
	// clients. This should be set to false in publicly available servers, as
	// internal error messages may reveal sensitive information.
	InformativeErrors bool
)

// NewHandler adds routes and handlers to an existing parent router (or creates
// one if parent is nil). If wrap is non-nil, it is called on each internal
// handler before being registered as the handler for a router.
func NewHandler(parent *mux.Router, wrap func(http.Handler) http.Handler) http.Handler {
	router = vcsclient.NewRouter(parent)
	r := (*mux.Router)(router)

	if wrap == nil {
		wrap = func(h http.Handler) http.Handler { return h }
	}

	r.Get(vcsclient.RouteRoot).Handler(wrap(handler(serveRoot)))
	r.Get(vcsclient.RouteRepo).Handler(wrap(handler(serveRepo)))
	r.Get(vcsclient.RouteRepoCreateOrUpdate).Handler(wrap(handler(serveRepoCreateOrUpdate)))
	r.Get(vcsclient.RouteRepoBranch).Handler(wrap(handler(serveRepoBranch)))
	r.Get(vcsclient.RouteRepoCommit).Handler(wrap(handler(serveRepoCommit)))
	r.Get(vcsclient.RouteRepoCommitLog).Handler(wrap(handler(serveRepoCommitLog)))
	r.Get(vcsclient.RouteRepoRevision).Handler(wrap(handler(serveRepoRevision)))
	r.Get(vcsclient.RouteRepoTag).Handler(wrap(handler(serveRepoTag)))
	r.Get(vcsclient.RouteRepoTreeEntry).Handler(wrap(handler(serveRepoTreeEntry)))
	return r
}

type handler func(w http.ResponseWriter, r *http.Request) error

// handler wraps f to handle errors it returns.
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(w, r)
	if err != nil {
		c := errorHTTPStatusCode(err)
		Log.Printf("HTTP %d error serving %q: %s.", c, r.URL.RequestURI(), err)
		http.Error(w, errorBody(err), c)
	}
}

// errorBody formats an error message for the HTTP response.
func errorBody(err error) string {
	if InformativeErrors {
		data, _ := json.Marshal(&vcsclient.ErrorResponse{Message: err.Error()})
		return string(data)
	}
	return ""
}

// writeJSON writes a JSON Content-Type header and a JSON-encoded object to the
// http.ResponseWriter.
func writeJSON(w http.ResponseWriter, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return &httpError{http.StatusInternalServerError, err}
	}

	w.Header().Set("content-type", "application/json; charset=utf-8")
	_, err = w.Write(data)
	return err
}

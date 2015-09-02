package server

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/schema"
	"github.com/sourcegraph/mux"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/vcsstore"
	"sourcegraph.com/sourcegraph/vcsstore/git"
	"sourcegraph.com/sourcegraph/vcsstore/vcsclient"
)

type Handler struct {
	Service        vcsstore.Service
	GitTransporter git.GitTransporter

	router *vcsclient.Router

	Log *log.Logger

	// Debug is whether to report internal error messages to HTTP clients.
	//
	// IMPORTANT NOTE: This should be set to false in publicly available
	// servers, as internal error messages may reveal sensitive information.
	Debug bool

	middleware []Middleware
}

// NewHandler adds routes and handlers to an existing parent router (or
// creates one if parent is nil). Middleware is injected between mux and the
// handler functions (so you have access to gorilla/context for example)
func NewHandler(svc vcsstore.Service, gitTrans git.GitTransporter, parent *mux.Router, mw ...Middleware) *Handler {
	router := vcsclient.NewRouter(parent)
	r := (*mux.Router)(router)

	h := &Handler{
		Service:        svc,
		GitTransporter: gitTrans,
		router:         router,
		Log:            log.New(ioutil.Discard, "", 0),
		middleware:     mw,
	}

	handler := func(handlerFunc robustHandlerFunc) robustHandler {
		return robustHandler{h, handlerFunc}
	}

	r.Get(git.RouteGitInfoRefs).Handler(handler(h.serveInfoRefs))
	r.Get(git.RouteGitUploadPack).Handler(handler(h.serveUploadPack))
	r.Get(git.RouteGitReceivePack).Handler(handler(h.serveReceivePack))

	r.Get(vcsclient.RouteRoot).Handler(handler(h.serveRoot))
	r.Get(vcsclient.RouteRepo).Handler(handler(h.serveRepo))
	r.Get(vcsclient.RouteRepoCreateOrUpdate).Handler(handler(h.serveRepoCreateOrUpdate))
	r.Get(vcsclient.RouteRepoBlameFile).Handler(handler(h.serveRepoBlameFile))
	r.Get(vcsclient.RouteRepoBranch).Handler(handler(h.serveRepoBranch))
	r.Get(vcsclient.RouteRepoBranches).Handler(handler(h.serveRepoBranches))
	r.Get(vcsclient.RouteRepoCommit).Handler(handler(h.serveRepoCommit))
	r.Get(vcsclient.RouteRepoCommits).Handler(handler(h.serveRepoCommits))
	r.Get(vcsclient.RouteRepoCommitters).Handler(handler(h.serveRepoCommitters))
	r.Get(vcsclient.RouteRepoDiff).Handler(handler(h.serveRepoDiff))
	r.Get(vcsclient.RouteRepoCrossRepoDiff).Handler(handler(h.serveRepoCrossRepoDiff))
	r.Get(vcsclient.RouteRepoMergeBase).Handler(handler(h.serveRepoMergeBase))
	r.Get(vcsclient.RouteRepoCrossRepoMergeBase).Handler(handler(h.serveRepoCrossRepoMergeBase))
	r.Get(vcsclient.RouteRepoSearch).Handler(handler(h.serveRepoSearch))
	r.Get(vcsclient.RouteRepoRevision).Handler(handler(h.serveRepoRevision))
	r.Get(vcsclient.RouteRepoTag).Handler(handler(h.serveRepoTag))
	r.Get(vcsclient.RouteRepoTags).Handler(handler(h.serveRepoTags))
	r.Get(vcsclient.RouteRepoTreeEntry).Handler(handler(h.serveRepoTreeEntry))

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("date", time.Now().UTC().Format(http.TimeFormat))
	(*mux.Router)(h.router).ServeHTTP(w, r)
}

type robustHandlerFunc func(w http.ResponseWriter, r *http.Request) error

type robustHandler struct {
	h           *Handler
	handlerFunc robustHandlerFunc
}

// robust handler wraps f to handle errors it returns.
func (h robustHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	innerHandler := func(w http.ResponseWriter, r *http.Request) {
		err := h.handlerFunc(w, r)
		if err != nil {
			c := errorHTTPStatusCode(err)
			h.h.Log.Printf("HTTP %d error serving %q: %s.", c, r.URL.RequestURI(), err)
			w.Header().Set("cache-control", "no-cache, max-age=0") // don't cache errors
			http.Error(w, errorBody(h.h.Debug, err), c)
		}
	}
	FuncWithMiddleware(innerHandler, h.h.middleware...)(w, r)
}

// errorBody formats an error message for the HTTP response.
func errorBody(debug bool, err error) string {
	if debug {
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

var schemaDecoder = schema.NewDecoder()

func init() {
	schemaDecoder.RegisterConverter(vcs.CommitID(""), func(s string) reflect.Value {
		return reflect.ValueOf(vcs.CommitID(s))
	})
}

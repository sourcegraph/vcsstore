package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"github.com/sourcegraph/vcsstore"
	"github.com/sourcegraph/vcsstore/client"
	"github.com/sqs/mux"
)

var (
	Service vcsstore.Service

	router = client.NewRouter()

	Log = log.New(ioutil.Discard, "", 0)

	// InformativeErrors is whether to report internal error messages to HTTP
	// clients. This should be set to false in publicly available servers, as
	// internal error messages may reveal sensitive information.
	InformativeErrors bool
)

func NewHandler() http.Handler {
	r := (*mux.Router)(client.NewRouter())
	r.Get(client.RouteRoot).Handler(handler(serveRoot))
	r.Get(client.RouteRepo).Handler(handler(serveRepo))
	r.Get(client.RouteRepoBranch).Handler(handler(serveRepoBranch))
	r.Get(client.RouteRepoCommit).Handler(handler(serveRepoCommit))
	r.Get(client.RouteRepoCommitLog).Handler(handler(serveRepoCommitLog))
	r.Get(client.RouteRepoRevision).Handler(handler(serveRepoRevision))
	r.Get(client.RouteRepoTag).Handler(handler(serveRepoTag))
	r.Get(client.RouteRepoTreeEntry).Handler(handler(serveRepoTreeEntry))
	return r
}

type handler func(w http.ResponseWriter, r *http.Request) error

// handler wraps f to handle errors it returns.
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rw := newRecorder(w)
	err := h(rw, r)
	if err != nil {
		c := errorHTTPStatusCode(err)
		Log.Printf("HTTP %d error serving %q: %s.", c, r.URL.RequestURI(), err)
		if rw.Code == 0 {
			// No response written yet, so we can write a response.
			http.Error(w, errorMessage(err), c)
		}
	}
}

// errorMessage formats an error message for the HTTP response.
func errorMessage(err error) string {
	c := errorHTTPStatusCode(err)
	if InformativeErrors {
		return fmt.Sprintf("HTTP %d (%s): %s", c, http.StatusText(c), err)
	}
	return fmt.Sprintf("HTTP %d (%s)", c, http.StatusText(c))
}

// responseRecorder is an implementation of http.ResponseWriter that
// records its HTTP status code and body length.
type responseRecorder struct {
	Code       int // the HTTP response code from WriteHeader
	BodyLength int

	underlying http.ResponseWriter
}

// newRecorder returns an initialized ResponseRecorder.
func newRecorder(underlying http.ResponseWriter) *responseRecorder {
	return &responseRecorder{underlying: underlying}
}

// Header returns the header map from the underlying ResponseWriter.
func (rw *responseRecorder) Header() http.Header {
	return rw.underlying.Header()
}

// Write always succeeds and writes to rw.Body, if not nil.
func (rw *responseRecorder) Write(buf []byte) (int, error) {
	rw.BodyLength += len(buf)
	if rw.Code == 0 {
		rw.Code = http.StatusOK
	}
	return rw.underlying.Write(buf)
}

// WriteHeader sets rw.Code.
func (rw *responseRecorder) WriteHeader(code int) {
	rw.Code = code
	rw.underlying.WriteHeader(code)
}

// Hijack implements net/http.Hijacker.
func (rw *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return rw.underlying.(http.Hijacker).Hijack()
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

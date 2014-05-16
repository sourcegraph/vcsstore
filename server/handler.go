package server

import (
	"bufio"
	"encoding/json"
	"expvar"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

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

var (
	// Published expvars
	numRequests       = &expvar.Int{}
	numResponses      = &expvar.Int{}
	numResponseErrors = &expvar.Int{}

	totalTreeEntryResponseTime = &expvar.Int{}
	numTreeEntryResponses      = &expvar.Int{}

	totalResolveRevisionResponseTime = &expvar.Int{}
	numResolveRevisionResponses      = &expvar.Int{}
)

func init() {
	avgTreeEntry := expvar.Func(func() interface{} {
		total, _ := strconv.ParseInt(totalTreeEntryResponseTime.String(), 10, 64)
		count, _ := strconv.ParseInt(numTreeEntryResponses.String(), 10, 64)
		if count == 0 {
			return nil
		}
		return time.Duration(total / count).String()
	})
	avgResolveRevision := expvar.Func(func() interface{} {
		total, _ := strconv.ParseInt(totalResolveRevisionResponseTime.String(), 10, 64)
		count, _ := strconv.ParseInt(numResolveRevisionResponses.String(), 10, 64)
		if count == 0 {
			return nil
		}
		return time.Duration(total / count).String()
	})
	m := expvar.NewMap("vcsstore")
	m.Set("AvgTreeEntryResponseTime", avgTreeEntry)
	m.Set("AvgResolveRevisionResponseTime", avgResolveRevision)
	m.Set("NumRequests", numRequests)
	m.Set("NumResponses", numResponses)
	m.Set("NumResponseErrors", numResponseErrors)
}

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
	numRequests.Add(1)
	rw := newRecorder(w)
	err := h(rw, r)
	numResponses.Add(1)
	if err != nil {
		numResponseErrors.Add(1)
		c := errorHTTPStatusCode(err)
		Log.Printf("HTTP %d error serving %q: %s.", c, r.URL.RequestURI(), err)
		if rw.Code == 0 {
			// No response written yet, so we can write a response.
			http.Error(w, errorBody(err), c)
		}
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

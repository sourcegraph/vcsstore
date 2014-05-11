package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"github.com/sqs/mux"
)

func serveRepo(w http.ResponseWriter, r *http.Request) error {
	repo, cloneURL, err := getRepo(r)
	if err != nil {
		return err
	}

	return writeJSON(w, struct {
		ImplementationType string
		CloneURL           string
	}{fmt.Sprintf("%T", repo), cloneURL.String()})
}

var getRepoMu sync.Mutex

func getRepo(r *http.Request) (interface{}, *url.URL, error) {
	// TODO(sqs): only lock per-repo if there are write ops going on
	getRepoMu.Lock()
	defer getRepoMu.Unlock()

	v := mux.Vars(r)
	cloneURLStr := v["CloneURL"]
	if cloneURLStr == "" {
		// If cloneURLStr is empty, then the CloneURLEscaped route var failed to
		// be unescaped using url.QueryUnescape.
		return nil, nil, &httpError{http.StatusBadRequest, errors.New("invalid clone URL (unescaping failed)")}
	}

	cloneURL, err := url.Parse(cloneURLStr)
	if err != nil {
		return nil, nil, &httpError{http.StatusBadRequest, errors.New("invalid clone URL (parsing failed)")}
	}

	repo, err := Service.Open(v["VCS"], cloneURL)
	if err != nil {
		return nil, nil, err
	}

	return repo, cloneURL, nil
}

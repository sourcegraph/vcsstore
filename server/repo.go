package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

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

func serveRepoUpdate(w http.ResponseWriter, r *http.Request) error {
	repo, _, err := getRepo(r)
	if err != nil {
		return err
	}

	type mirrorUpdate interface {
		MirrorUpdate() error
	}
	if repo, ok := repo.(mirrorUpdate); ok {
		err := repo.MirrorUpdate()
		if err != nil {
			return err
		}

		return nil
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("MirrorUpdate not yet implemented for %T", repo)}
}

func getRepo(r *http.Request) (interface{}, *url.URL, error) {
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

	if cloneURL.Scheme == "" || cloneURL.Host == "" || cloneURL.User != nil {
		return nil, nil, errors.New("invalid clone URL")
	}

	repo, err := Service.Open(v["VCS"], cloneURL)
	if err != nil {
		return nil, nil, err
	}

	return repo, cloneURL, nil
}

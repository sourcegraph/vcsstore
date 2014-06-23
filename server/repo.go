package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/sourcegraph/vcsstore/vcsclient"
	"github.com/sqs/mux"
)

func (h *Handler) serveRepo(w http.ResponseWriter, r *http.Request) error {
	repo, cloneURL, _, err := h.getRepo(r, 0)
	if err != nil {
		return err
	}

	return writeJSON(w, struct {
		ImplementationType string
		CloneURL           string
	}{fmt.Sprintf("%T", repo), cloneURL.String()})
}

func (h *Handler) serveRepoCreateOrUpdate(w http.ResponseWriter, r *http.Request) error {
	repo, _, cloned, err := h.getRepo(r, cloneIfNotExists)
	if err != nil {
		return err
	}

	if cloned {
		w.WriteHeader(http.StatusCreated)
		return nil
	} else {
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
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("MirrorUpdate not yet implemented for %T", repo)}
}

type getRepoMode int

const (
	cloneIfNotExists = 1 << iota
)

func (h *Handler) getRepo(r *http.Request, opt getRepoMode) (repo interface{}, cloneURL *url.URL, cloned bool, err error) {
	v := mux.Vars(r)
	vcsType := v["VCS"]
	cloneURLStr := v["CloneURL"]
	if cloneURLStr == "" {
		// If cloneURLStr is empty, then the CloneURLEscaped route var failed to
		// be unescaped using url.QueryUnescape.
		return nil, nil, false, &httpError{http.StatusBadRequest, errors.New("invalid clone URL (unescaping failed)")}
	}

	cloneURL, err = url.Parse(cloneURLStr)
	if err != nil {
		return nil, nil, false, &httpError{http.StatusBadRequest, errors.New("invalid clone URL (parsing failed)")}
	}

	if cloneURL.Scheme == "" || cloneURL.Host == "" || cloneURL.User != nil {
		return nil, nil, false, errors.New("invalid clone URL")
	}

	repo, err = h.Service.Open(vcsType, cloneURL)
	if os.IsNotExist(err) && opt&cloneIfNotExists != 0 {
		cloned = true
		repo, err = h.Service.Clone(vcsType, cloneURL)
	}
	if err != nil {
		if os.IsNotExist(err) {
			err = &httpError{http.StatusNotFound, vcsclient.ErrRepoNotExist}
		}
		return nil, nil, cloned, err
	}

	return repo, cloneURL, cloned, nil
}

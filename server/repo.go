package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/sourcegraph/mux"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/vcsstore/vcsclient"
)

func (h *Handler) serveRepo(w http.ResponseWriter, r *http.Request) error {
	repo, _, done, err := h.getRepo(r)
	if err != nil {
		return err
	}
	defer done()

	return writeJSON(w, struct {
		ImplementationType string
	}{fmt.Sprintf("%T", repo)})
}

func (h *Handler) serveRepoCreateOrUpdate(w http.ResponseWriter, r *http.Request) error {
	var cloneInfo vcsclient.CloneInfo
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&cloneInfo); err != nil {
			return err
		}
	}

	var cloned bool // whether the repo was newly cloned
	repo, repoPath, _, err := h.getRepo(r)
	if errorHTTPStatusCode(err) == http.StatusNotFound {
		cloned = true
		repo, err = h.Service.Clone(repoPath, &cloneInfo)
	}
	if err != nil {
		return cloneOrUpdateError(err)
	}
	defer h.Service.Close(repoPath)

	if cloned {
		w.WriteHeader(http.StatusCreated)
		return nil
	}

	type updateEverythinger interface {
		UpdateEverything(opt vcs.RemoteOpts) error
	}
	if repo, ok := repo.(updateEverythinger); ok {
		err := repo.UpdateEverything(cloneInfo.RemoteOpts)
		if err != nil {
			return cloneOrUpdateError(err)
		}

		return nil
	}
	return &httpError{http.StatusNotImplemented, fmt.Errorf("Remote updates not yet implemented for %T", repo)}
}

func cloneOrUpdateError(err error) error {
	if err != nil {
		var c int
		switch err.Error() {
		case "authentication required but no callback set":
			c = http.StatusUnauthorized
		case "callback returned unsupported credentials type":
			c = http.StatusUnauthorized
		case "Failed to authenticate SSH session: Waiting for USERAUTH response":
			c = http.StatusForbidden
		}
		if c != 0 {
			return &httpError{err: err, statusCode: c}
		}
	}
	return err
}

type getRepoMode int

const (
	cloneIfNotExists = 1 << iota
)

func (h *Handler) getRepo(r *http.Request) (repo interface{}, repoPath string, done func(), err error) {
	return h.getRepoLabeled(r, "")
}

// getRepoLabel allows either getting the main repo in the URL or
// another one, such as the head repo for cross-repo diffs.
func (h *Handler) getRepoLabeled(r *http.Request, label string) (repo interface{}, repoPath string, done func(), err error) {
	repoPath, err = h.getRepoPath(r, label)
	if err != nil {
		return nil, "", nil, err
	}

	repo, err = h.Service.Open(repoPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = &httpError{http.StatusNotFound, vcsclient.ErrRepoNotExist}
		}
		return nil, repoPath, nil, err
	}

	done = func() {
		h.Service.Close(repoPath)
	}

	return repo, repoPath, done, nil
}

func (h *Handler) getRepoPath(r *http.Request, label string) (repoPath string, err error) {
	v := mux.Vars(r)
	repoPath = v[label+"RepoPath"]
	if repoPath == "" {
		return "", &httpError{http.StatusBadRequest, errors.New("repoPath not found")}
	}
	return repoPath, err
}

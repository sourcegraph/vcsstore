package server

import (
	"fmt"
	"net/http"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

func (h *Handler) serveRepoCommitters(w http.ResponseWriter, r *http.Request) error {
	repo, _, done, err := h.getRepo(r)
	if err != nil {
		return err
	}
	defer done()

	// TODO: implement fetching CommittersOptions from the URL query string.
	var opt vcs.CommittersOptions

	type committers interface {
		Committers(vcs.CommittersOptions) ([]*vcs.Committer, error)
	}
	if repo, ok := repo.(committers); ok {
		committers, err := repo.Committers(opt)
		if err != nil {
			return err
		}

		setShortCache(w)

		return writeJSON(w, committers)
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("GetShortLog not yet implemented for %T", repo)}
}

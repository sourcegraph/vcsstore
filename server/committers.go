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

	var opt vcs.CommittersOptions
	if err := schemaDecoder.Decode(&opt, r.URL.Query()); err != nil {
		return err
	}

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

	return &httpError{http.StatusNotImplemented, fmt.Errorf("Committers not yet implemented for %T", repo)}
}

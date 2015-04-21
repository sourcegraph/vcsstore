package server

import (
	"fmt"
	"net/http"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

func (h *Handler) serveRepoBranches(w http.ResponseWriter, r *http.Request) error {
	repo, _, done, err := h.getRepo(r)
	if err != nil {
		return err
	}
	defer done()

	var opt vcs.BranchesOptions
	if err := schemaDecoder.Decode(&opt, r.URL.Query()); err != nil {
		return err
	}

	type branches interface {
		Branches(opt vcs.BranchesOptions) ([]*vcs.Branch, error)
	}
	if repo, ok := repo.(branches); ok {
		branches, err := repo.Branches(opt)
		if err != nil {
			return err
		}

		return writeJSON(w, branches)
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("Branches not yet implemented for %T", repo)}
}

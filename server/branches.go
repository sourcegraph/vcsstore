package server

import (
	"fmt"
	"net/http"

	"github.com/sourcegraph/go-vcs/vcs"
)

func (h *Handler) serveRepoBranches(w http.ResponseWriter, r *http.Request) error {
	repo, _, _, err := h.getRepo(r, 0)
	if err != nil {
		return err
	}

	type branches interface {
		Branches() ([]*vcs.Branch, error)
	}
	if repo, ok := repo.(branches); ok {
		branches, err := repo.Branches()
		if err != nil {
			return err
		}

		return writeJSON(w, branches)
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("Branches not yet implemented for %T", repo)}
}
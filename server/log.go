package server

import (
	"fmt"
	"net/http"

	"github.com/sourcegraph/go-vcs/vcs"
)

func (h *Handler) serveRepoCommitLog(w http.ResponseWriter, r *http.Request) error {
	repo, _, _, err := h.getRepo(r, 0)
	if err != nil {
		return err
	}

	commitID, canon, err := getCommitID(r)
	if err != nil {
		return err
	}

	type commitLog interface {
		CommitLog(to vcs.CommitID) ([]*vcs.Commit, error)
	}
	if repo, ok := repo.(commitLog); ok {
		commits, err := repo.CommitLog(commitID)
		if err != nil {
			return err
		}

		if canon {
			setLongCache(w)
		}
		return writeJSON(w, commits)
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("GetCommit not yet implemented for %T", repo)}
}

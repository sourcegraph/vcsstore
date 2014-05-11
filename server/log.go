package server

import (
	"fmt"
	"net/http"

	"github.com/sourcegraph/go-vcs/vcs"
)

func serveRepoCommitLog(w http.ResponseWriter, r *http.Request) error {
	repo, _, err := getRepo(r)
	if err != nil {
		return err
	}

	commitID, err := getCommitID(r)
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

		return writeJSON(w, commits)
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("GetCommit not yet implemented for %T", repo)}
}

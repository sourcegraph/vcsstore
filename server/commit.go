package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sqs/mux"
)

func serveRepoCommit(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

	repo, cloneURL, err := getRepo(r)
	if err != nil {
		return err
	}

	commitID, err := getCommitID(r)
	if err != nil {
		return err
	}

	type getCommit interface {
		GetCommit(vcs.CommitID) (*vcs.Commit, error)
	}
	if repo, ok := repo.(getCommit); ok {
		commit, err := repo.GetCommit(commitID)
		if err != nil {
			return err
		}

		if commit.ID != commitID {
			http.Redirect(w, r, router.URLToRepoCommit(v["VCS"], cloneURL, commit.ID).String(), http.StatusSeeOther)
			return nil
		}

		return writeJSON(w, commit)
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("GetCommit not yet implemented for %T", repo)}
}

func getCommitID(r *http.Request) (vcs.CommitID, error) {
	v := mux.Vars(r)
	commitID := v["CommitID"]
	if commitID == "" {
		return "", &httpError{http.StatusBadRequest, errors.New("CommitID is empty")}
	}

	// check that it is lowercase hex
	i := strings.IndexFunc(commitID, func(c rune) bool {
		return !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'))
	})
	if i != -1 {
		return "", &httpError{http.StatusBadRequest, errors.New("CommitID must be lowercase hex")}
	}

	return vcs.CommitID(commitID), nil
}

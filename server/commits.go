package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/sourcegraph/go-vcs/vcs"
)

func (h *Handler) serveRepoCommits(w http.ResponseWriter, r *http.Request) error {
	repo, _, _, err := h.getRepo(r, 0)
	if err != nil {
		return err
	}

	var opt vcs.CommitsOptions
	// TODO(sqs): failing because Head is not a string but a typedef, make a RegisterConverter to handle vcs.CommitID type
	if err := schemaDecoder.Decode(&opt, r.URL.Query()); err != nil {
		log.Println(err)
		return err
	}

	head, canon, err := checkCommitID(string(opt.Head))
	if err != nil {
		return err
	}
	opt.Head = head

	type commits interface {
		Commits(opt vcs.CommitsOptions) ([]*vcs.Commit, error)
	}
	if repo, ok := repo.(commits); ok {
		commits, err := repo.Commits(opt)
		if err != nil {
			return err
		}

		if canon {
			setLongCache(w)
		}
		return writeJSON(w, commits)
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("Commits not yet implemented for %T", repo)}
}

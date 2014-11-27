package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/vcsstore/vcsclient"
)

func (h *Handler) serveRepoCommits(w http.ResponseWriter, r *http.Request) error {
	repo, _, err := h.getRepo(r)
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
		Commits(opt vcs.CommitsOptions) ([]*vcs.Commit, uint, error)
	}
	if repo, ok := repo.(commits); ok {
		commits, total, err := repo.Commits(opt)
		if err != nil {
			return err
		}

		if canon {
			setLongCache(w)
		}

		w.Header().Set(vcsclient.TotalCommitsHeader, strconv.FormatUint(uint64(total), 10))

		return writeJSON(w, commits)
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("Commits not yet implemented for %T", repo)}
}

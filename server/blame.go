package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sqs/mux"
)

func (h *Handler) serveRepoBlameFile(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

	repo, _, err := h.getRepo(r)
	if err != nil {
		return err
	}

	var opt vcs.BlameOptions
	if err := schemaDecoder.Decode(&opt, r.URL.Query()); err != nil {
		log.Println(err)
		return err
	}

	type blameFile interface {
		BlameFile(path string, opt *vcs.BlameOptions) ([]*vcs.Hunk, error)
	}
	if repo, ok := repo.(blameFile); ok {
		hunks, err := repo.BlameFile(v["Path"], &opt)
		if err != nil {
			return err
		}

		if opt.NewestCommit != "" {
			_, canon, err := checkCommitID(string(opt.NewestCommit))
			if err != nil {
				return err
			}
			if canon {
				setLongCache(w)
			} else {
				setShortCache(w)
			}
		}

		return writeJSON(w, hunks)
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("BlameFile not yet implemented for %T", repo)}
}
package server

import (
	"fmt"
	"net/http"

	"github.com/sqs/mux"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

func (h *Handler) serveRepoMergeBase(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

	repo, cloneURL, done, err := h.getRepo(r)
	if err != nil {
		return err
	}
	defer done()

	if repo, ok := repo.(vcs.Merger); ok {
		a, b := vcs.CommitID(v["CommitID1"]), vcs.CommitID(v["CommitID2"])

		mb, err := repo.MergeBase(a, b)
		if err != nil {
			return err
		}

		var statusCode int
		if commitIDIsCanon(string(a)) && commitIDIsCanon(string(b)) {
			setLongCache(w)
			statusCode = http.StatusMovedPermanently
		} else {
			setShortCache(w)
			statusCode = http.StatusFound
		}
		http.Redirect(w, r, h.router.URLToRepoCommit(v["VCS"], cloneURL, mb).String(), statusCode)
		return nil
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("Merger not yet implemented by %T", repo)}
}

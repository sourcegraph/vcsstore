package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sourcegraph/mux"
	"golang.org/x/tools/godoc/vfs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/vcsstore/vcsclient"
)

func (h *Handler) serveRepoTreeEntry(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

	repo, _, done, err := h.getRepo(r)
	if err != nil {
		return err
	}
	defer done()

	commitID, canon, err := getCommitID(r)
	if err != nil {
		return err
	}

	type fileSystem interface {
		FileSystem(vcs.CommitID) (vfs.FileSystem, error)
	}
	if repo, ok := repo.(fileSystem); ok {
		fs, err := repo.FileSystem(commitID)
		if err != nil {
			return err
		}

		// Check for extended range options (GetFileOptions).
		var fopt vcsclient.GetFileOptions
		if err := schemaDecoder.Decode(&fopt, r.URL.Query()); err != nil {
			return err
		}

		fr, err := vcsclient.GetFileWithOptions(fs, v["Path"], fopt)
		if err != nil {
			if os.IsNotExist(err) {
				return &httpError{http.StatusNotFound, err}
			}
			return err
		}

		if canon {
			setLongCache(w)
		} else {
			setShortCache(w)
		}
		return writeJSON(w, fr)
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("FileSystem not yet implemented for %T", repo)}
}

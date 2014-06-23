package server

import (
	"fmt"
	"net/http"

	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sqs/mux"
)

func (h *Handler) serveRepoBranch(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

	repo, cloneURL, _, err := h.getRepo(r, 0)
	if err != nil {
		return err
	}

	type resolveBranch interface {
		ResolveBranch(string) (vcs.CommitID, error)
	}
	if repo, ok := repo.(resolveBranch); ok {
		commitID, err := repo.ResolveBranch(v["Branch"])
		if err != nil {
			return err
		}

		setShortCache(w)
		http.Redirect(w, r, h.router.URLToRepoCommit(v["VCS"], cloneURL, commitID).String(), http.StatusFound)
		return nil
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("ResolveBranch not yet implemented for %T", repo)}
}

func (h *Handler) serveRepoRevision(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

	repo, cloneURL, _, err := h.getRepo(r, 0)
	if err != nil {
		return err
	}

	type resolveRevision interface {
		ResolveRevision(string) (vcs.CommitID, error)
	}
	if repo, ok := repo.(resolveRevision); ok {
		commitID, err := repo.ResolveRevision(v["RevSpec"])
		if err != nil {
			return err
		}

		setShortCache(w)
		http.Redirect(w, r, h.router.URLToRepoCommit(v["VCS"], cloneURL, commitID).String(), http.StatusFound)
		return nil
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("ResolveRevision not yet implemented for %T", repo)}
}

func (h *Handler) serveRepoTag(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

	repo, cloneURL, _, err := h.getRepo(r, 0)
	if err != nil {
		return err
	}

	type resolveTag interface {
		ResolveTag(string) (vcs.CommitID, error)
	}
	if repo, ok := repo.(resolveTag); ok {
		commitID, err := repo.ResolveTag(v["Tag"])
		if err != nil {
			return err
		}

		setShortCache(w)
		http.Redirect(w, r, h.router.URLToRepoCommit(v["VCS"], cloneURL, commitID).String(), http.StatusFound)
		return nil
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("ResolveTag not yet implemented for %T", repo)}
}

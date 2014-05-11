package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sqs/mux"
)

func serveRepoBranch(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

	repo, cloneURL, err := getRepo(r)
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

		http.Redirect(w, r, router.URLToRepoCommit(v["VCS"], cloneURL, commitID).String(), http.StatusSeeOther)
		return nil
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("ResolveBranch not yet implemented for %T", repo)}
}

func serveRepoRevision(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)
	start := time.Now()

	repo, cloneURL, err := getRepo(r)
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

		numResolveRevisionResponses.Add(1)
		totalResolveRevisionResponseTime.Add(int64(time.Since(start)))

		http.Redirect(w, r, router.URLToRepoCommit(v["VCS"], cloneURL, commitID).String(), http.StatusSeeOther)
		return nil
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("ResolveRevision not yet implemented for %T", repo)}
}

func serveRepoTag(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

	repo, cloneURL, err := getRepo(r)
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

		http.Redirect(w, r, router.URLToRepoCommit(v["VCS"], cloneURL, commitID).String(), http.StatusSeeOther)
		return nil
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("ResolveTag not yet implemented for %T", repo)}
}

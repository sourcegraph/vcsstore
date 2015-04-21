package server

import (
	"net/http"
	"net/url"
	"testing"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

func TestHandlers_NotImplemented(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	repoPath := "a.b/c"
	uris := []*url.URL{
		testHandler.router.URLToRepoBranch(repoPath, "mybranch"),
		testHandler.router.URLToRepoRevision(repoPath, "myrevspec"),
		testHandler.router.URLToRepoTag(repoPath, "mytag"),
		testHandler.router.URLToRepoCommit(repoPath, "abcd"),
		testHandler.router.URLToRepoCommits(repoPath, vcs.CommitsOptions{Head: "abcd"}),
		testHandler.router.URLToRepoTreeEntry(repoPath, "abcd", "myfile"),
		testHandler.router.URLToRepoTreeEntry(repoPath, "abcd", "."),
	}

	sm := &mockServiceForExistingRepo{
		t:        t,
		repoPath: repoPath,
		repo:     nil, // doesn't implement any repo methods
	}
	testHandler.Service = sm

	for _, uri := range uris {
		resp, err := http.Get(server.URL + uri.String())
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if got, want := resp.StatusCode, http.StatusNotImplemented; got != want {
			t.Errorf("%s: got status code %d, want %d", uri, got, want)
		}

		if !sm.opened {
			t.Errorf("!opened")
		}
	}
}

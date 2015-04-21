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

	repoID := "a.b/c"
	uris := []*url.URL{
		testHandler.router.URLToRepoBranch(repoID, "mybranch"),
		testHandler.router.URLToRepoRevision(repoID, "myrevspec"),
		testHandler.router.URLToRepoTag(repoID, "mytag"),
		testHandler.router.URLToRepoCommit(repoID, "abcd"),
		testHandler.router.URLToRepoCommits(repoID, vcs.CommitsOptions{Head: "abcd"}),
		testHandler.router.URLToRepoTreeEntry(repoID, "abcd", "myfile"),
		testHandler.router.URLToRepoTreeEntry(repoID, "abcd", "."),
	}

	sm := &mockServiceForExistingRepo{
		t:      t,
		repoID: repoID,
		repo:   nil, // doesn't implement any repo methods
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

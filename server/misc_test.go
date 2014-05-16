package server

import (
	"net/http"
	"net/url"
	"testing"
)

func TestHandlers_NotImplemented(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	uris := []*url.URL{
		router.URLToRepoBranch("git", cloneURL, "mybranch"),
		router.URLToRepoRevision("git", cloneURL, "myrevspec"),
		router.URLToRepoTag("git", cloneURL, "mytag"),
		router.URLToRepoCommit("git", cloneURL, "abcd"),
		router.URLToRepoCommitLog("git", cloneURL, "abcd"),
		router.URLToRepoTreeEntry("git", cloneURL, "abcd", "myfile"),
		router.URLToRepoTreeEntry("git", cloneURL, "abcd", "."),
	}

	sm := &mockServiceForExistingRepo{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
		repo:     nil, // doesn't implement any repo methods
	}
	Service = sm

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

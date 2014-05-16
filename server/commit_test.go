package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/sourcegraph/go-vcs/vcs"
)

func TestServeRepoCommit(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	rm := &mockGetCommit{
		t:      t,
		id:     "abcd",
		commit: &vcs.Commit{ID: "abcd"},
	}
	sm := &mockServiceForExistingRepo{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
		repo:     rm,
	}
	Service = sm

	resp, err := http.Get(server.URL + router.URLToRepoCommit("git", cloneURL, "abcd").String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if !sm.opened {
		t.Errorf("!opened")
	}
	if !rm.called {
		t.Errorf("!called")
	}

	var commit *vcs.Commit
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		t.Fatal(err)
	}

	normalizeCommit(rm.commit)
	if !reflect.DeepEqual(commit, rm.commit) {
		t.Errorf("got commit %+v, want %+v", commit, rm.commit)
	}
}

func TestServeRepoCommit_RedirectToFull(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	rm := &mockGetCommit{
		t:      t,
		id:     "ab",
		commit: &vcs.Commit{ID: "abcd"},
	}
	sm := &mockServiceForExistingRepo{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
		repo:     rm,
	}
	Service = sm

	resp, err := ignoreRedirectsClient.Get(server.URL + router.URLToRepoCommit("git", cloneURL, "ab").String())
	if err != nil && !isIgnoredRedirectErr(err) {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if !sm.opened {
		t.Errorf("!opened")
	}
	if !rm.called {
		t.Errorf("!called")
	}
	testRedirectedTo(t, resp, http.StatusSeeOther, router.URLToRepoCommit("git", cloneURL, "abcd"))
}

// TODO(sqs): Add redirects to the full commit ID for other endpoints that
// include the commit ID.

type mockGetCommit struct {
	t *testing.T

	// expected args
	id vcs.CommitID

	// return values
	commit *vcs.Commit
	err    error

	called bool
}

func (m *mockGetCommit) GetCommit(id vcs.CommitID) (*vcs.Commit, error) {
	if id != m.id {
		m.t.Errorf("mock: got id arg %q, want %q", id, m.id)
	}
	m.called = true
	return m.commit, m.err
}

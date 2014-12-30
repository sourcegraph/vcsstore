package server

import (
	"net/http"
	"net/url"
	"testing"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

func TestServeRepoMergeBase(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	rm := &mockMergeBase{
		t:         t,
		a:         "a",
		b:         "b",
		mergeBase: "abcd",
	}
	sm := &mockServiceForExistingRepo{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
		repo:     rm,
	}
	testHandler.Service = sm

	resp, err := ignoreRedirectsClient.Get(server.URL + testHandler.router.URLToRepoMergeBase("git", cloneURL, "a", "b").String())
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
	testRedirectedTo(t, resp, http.StatusFound, testHandler.router.URLToRepoCommit("git", cloneURL, "abcd"))
}

type mockMergeBase struct {
	t *testing.T

	// expected args
	a, b vcs.CommitID

	// return values
	mergeBase vcs.CommitID
	err       error

	called bool
}

func (m *mockMergeBase) MergeBase(a, b vcs.CommitID) (vcs.CommitID, error) {
	if a != m.a {
		m.t.Errorf("mock: got a == %q, want %q", a, m.a)
	}
	if b != m.b {
		m.t.Errorf("mock: got b == %q, want %q", b, m.b)
	}
	m.called = true
	return m.mergeBase, m.err
}

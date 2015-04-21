package server

import (
	"net/http"
	"testing"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

func TestServeRepoBranch(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	repoID := "a.b/c"
	rm := &mockResolveBranch{
		t:        t,
		name:     "mybranch",
		commitID: "abcd",
	}
	sm := &mockServiceForExistingRepo{
		t:      t,
		repoID: repoID,
		repo:   rm,
	}
	testHandler.Service = sm

	resp, err := ignoreRedirectsClient.Get(server.URL + testHandler.router.URLToRepoBranch(repoID, "mybranch").String())
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
	testRedirectedTo(t, resp, http.StatusFound, testHandler.router.URLToRepoCommit(repoID, "abcd"))
}

func TestServeRepoRevision(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	repoID := "a.b/c"
	rm := &mockResolveRevision{
		t:        t,
		revSpec:  "myrevspec",
		commitID: "abcd",
	}
	sm := &mockServiceForExistingRepo{
		t:      t,
		repoID: repoID,
		repo:   rm,
	}
	testHandler.Service = sm

	resp, err := ignoreRedirectsClient.Get(server.URL + testHandler.router.URLToRepoRevision(repoID, "myrevspec").String())
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
	testRedirectedTo(t, resp, http.StatusFound, testHandler.router.URLToRepoCommit(repoID, "abcd"))
}

func TestServeRepoTag(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	repoID := "a.b/c"
	rm := &mockResolveTag{
		t:        t,
		name:     "mytag",
		commitID: "abcd",
	}
	sm := &mockServiceForExistingRepo{
		t:      t,
		repoID: repoID,
		repo:   rm,
	}
	testHandler.Service = sm

	resp, err := ignoreRedirectsClient.Get(server.URL + testHandler.router.URLToRepoTag(repoID, "mytag").String())
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
	testRedirectedTo(t, resp, http.StatusFound, testHandler.router.URLToRepoCommit(repoID, "abcd"))
}

type mockResolveBranch struct {
	t *testing.T

	// expected args
	name string

	// return values
	commitID vcs.CommitID
	err      error

	called bool
}

func (m *mockResolveBranch) ResolveBranch(name string) (vcs.CommitID, error) {
	if name != m.name {
		m.t.Errorf("mock: got name arg %q, want %q", name, m.name)
	}
	m.called = true
	return m.commitID, m.err
}

type mockResolveTag struct {
	t *testing.T

	// expected args
	name string

	// return values
	commitID vcs.CommitID
	err      error

	called bool
}

func (m *mockResolveTag) ResolveTag(name string) (vcs.CommitID, error) {
	if name != m.name {
		m.t.Errorf("mock: got name arg %q, want %q", name, m.name)
	}
	m.called = true
	return m.commitID, m.err
}

type mockResolveRevision struct {
	t *testing.T

	// expected args
	revSpec string

	// return values
	commitID vcs.CommitID
	err      error

	called bool
}

func (m *mockResolveRevision) ResolveRevision(revSpec string) (vcs.CommitID, error) {
	if revSpec != m.revSpec {
		m.t.Errorf("mock: got revSpec arg %q, want %q", revSpec, m.revSpec)
	}
	m.called = true
	return m.commitID, m.err
}

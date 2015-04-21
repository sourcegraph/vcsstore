package server

import (
	"net/http"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	vcs_testing "sourcegraph.com/sourcegraph/go-vcs/vcs/testing"
)

func TestServeRepoMergeBase(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	repoPath := "a.b/c"
	rm := &mockMergeBase{
		t:         t,
		a:         "a",
		b:         "b",
		mergeBase: "abcd",
	}
	sm := &mockServiceForExistingRepo{
		t:        t,
		repoPath: repoPath,
		repo:     rm,
	}
	testHandler.Service = sm

	resp, err := ignoreRedirectsClient.Get(server.URL + testHandler.router.URLToRepoMergeBase(repoPath, "a", "b").String())
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
	testRedirectedTo(t, resp, http.StatusFound, testHandler.router.URLToRepoCommit(repoPath, "abcd"))
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

func TestServeRepoCrossRepoMergeBase(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	aRepoID := "a.b/c"
	bRepoID := "x.y/z"
	mockRepoB := vcs_testing.MockRepository{}

	rm := &mockCrossRepoMergeBase{
		t:         t,
		a:         "a",
		repoB:     mockRepoB,
		b:         "b",
		mergeBase: "abcd",
	}
	sm := &mockService{
		t: t,
		open: func(repoPath string) (interface{}, error) {
			switch repoPath {
			case aRepoID:
				return rm, nil
			case bRepoID:
				return mockRepoB, nil
			default:
				panic("unexpected repo clone: " + repoPath)
			}
		},
	}
	testHandler.Service = sm

	resp, err := ignoreRedirectsClient.Get(server.URL + testHandler.router.URLToRepoCrossRepoMergeBase(aRepoID, "a", bRepoID, "b").String())
	if err != nil && !isIgnoredRedirectErr(err) {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if !rm.called {
		t.Errorf("!called")
	}
	testRedirectedTo(t, resp, http.StatusFound, testHandler.router.URLToRepoCommit(aRepoID, "abcd"))
}

type mockCrossRepoMergeBase struct {
	t *testing.T

	// expected args
	a, b  vcs.CommitID
	repoB vcs.Repository

	// return values
	mergeBase vcs.CommitID
	err       error

	called bool
}

func (m *mockCrossRepoMergeBase) CrossRepoMergeBase(a vcs.CommitID, repoB vcs.Repository, b vcs.CommitID) (vcs.CommitID, error) {
	if a != m.a {
		m.t.Errorf("mock: got a == %q, want %q", a, m.a)
	}
	if !reflect.DeepEqual(repoB, m.repoB) {
		m.t.Errorf("mock: got repoB %v (%T), want %v (%T)", repoB, repoB, m.repoB, m.repoB)
	}
	if b != m.b {
		m.t.Errorf("mock: got b == %q, want %q", b, m.b)
	}
	m.called = true
	return m.mergeBase, m.err
}

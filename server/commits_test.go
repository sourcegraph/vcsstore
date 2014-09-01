package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/sourcegraph/go-vcs/vcs"
)

func TestServeRepoCommits(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	opt := vcs.CommitsOptions{Head: "abcd", N: 2, Skip: 3}

	rm := &mockCommits{
		t:       t,
		opt:     opt,
		commits: []*vcs.Commit{{ID: "abcd"}, {ID: "wxyz"}},
	}
	sm := &mockServiceForExistingRepo{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
		repo:     rm,
	}
	testHandler.Service = sm

	resp, err := http.Get(server.URL + testHandler.router.URLToRepoCommits("git", cloneURL, opt).String())
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

	var commits []*vcs.Commit
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		t.Fatal(err)
	}

	for _, c := range rm.commits {
		normalizeCommit(c)
	}
	if !reflect.DeepEqual(commits, rm.commits) {
		t.Errorf("got commits %+v, want %+v", commits, rm.commits)
	}
}

type mockCommits struct {
	t *testing.T

	// expected args
	opt vcs.CommitsOptions

	// return values
	commits []*vcs.Commit
	err     error

	called bool
}

func (m *mockCommits) Commits(opt vcs.CommitsOptions) ([]*vcs.Commit, error) {
	if opt != m.opt {
		m.t.Errorf("mock: got opt %+v, want %+v", opt, m.opt)
	}
	m.called = true
	return m.commits, m.err
}

func normalizeCommit(c *vcs.Commit) {
	c.Author.Date = c.Author.Date.In(time.UTC)
	if c.Committer != nil {
		c.Committer.Date = c.Committer.Date.In(time.UTC)
	}
}

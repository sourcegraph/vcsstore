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

func TestServeRepoCommitLog(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	rm := &mockCommitLog{
		t:       t,
		to:      "abcd",
		commits: []*vcs.Commit{{ID: "abcd"}, {ID: "wxyz"}},
	}
	sm := &mockService{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
		repo:     rm,
	}
	Service = sm

	resp, err := http.Get(server.URL + router.URLToRepoCommitLog("git", cloneURL, "abcd").String())
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

type mockCommitLog struct {
	t *testing.T

	// expected args
	to vcs.CommitID

	// return values
	commits []*vcs.Commit
	err     error

	called bool
}

func (m *mockCommitLog) CommitLog(to vcs.CommitID) ([]*vcs.Commit, error) {
	if to != m.to {
		m.t.Errorf("mock: got to arg %q, want %q", to, m.to)
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

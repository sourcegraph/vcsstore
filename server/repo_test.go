package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	godoc_vfs "code.google.com/p/go.tools/godoc/vfs"
	"code.google.com/p/go.tools/godoc/vfs/mapfs"
	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sourcegraph/vcsstore/client"
)

func TestServeRepo(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	sm := &mockService{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
	}
	Service = sm

	resp, err := http.Get(server.URL + router.URLToRepo("git", cloneURL).String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if !sm.opened {
		t.Errorf("!opened")
	}
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("got code %d, want %d", got, want)
		logResponseBody(t, resp)
	}
}

func TestServeRepo_NotImplemented(t *testing.T) {
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

	sm := &mockService{
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

func TestServeRepoBranch(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	rm := &mockResolveBranch{
		t:        t,
		name:     "mybranch",
		commitID: "abcd",
	}
	sm := &mockService{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
		repo:     rm,
	}
	Service = sm

	resp, err := ignoreRedirectsClient.Get(server.URL + router.URLToRepoBranch("git", cloneURL, "mybranch").String())
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

func TestServeRepoRevision(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	rm := &mockResolveRevision{
		t:        t,
		revSpec:  "myrevspec",
		commitID: "abcd",
	}
	sm := &mockService{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
		repo:     rm,
	}
	Service = sm

	resp, err := ignoreRedirectsClient.Get(server.URL + router.URLToRepoRevision("git", cloneURL, "myrevspec").String())
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

func TestServeRepoTag(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	rm := &mockResolveTag{
		t:        t,
		name:     "mytag",
		commitID: "abcd",
	}
	sm := &mockService{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
		repo:     rm,
	}
	Service = sm

	resp, err := ignoreRedirectsClient.Get(server.URL + router.URLToRepoTag("git", cloneURL, "mytag").String())
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

func TestServeRepoCommit(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	rm := &mockGetCommit{
		t:      t,
		id:     "abcd",
		commit: &vcs.Commit{ID: "abcd"},
	}
	sm := &mockService{
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
	sm := &mockService{
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

func TestServeRepoTreeEntry_File(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	rm := &mockFileSystem{
		t:  t,
		at: "abcd",
		fs: vfs{mapfs.New(map[string]string{"myfile": "mydata"})},
	}
	sm := &mockService{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
		repo:     rm,
	}
	Service = sm

	resp, err := http.Get(server.URL + router.URLToRepoTreeEntry("git", cloneURL, "abcd", "myfile").String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("got status code %d, want %d", got, want)
	}

	if !sm.opened {
		t.Errorf("!opened")
	}
	if !rm.called {
		t.Errorf("!called")
	}

	var e *client.TreeEntry
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		t.Fatal(err)
	}

	wantEntry := &client.TreeEntry{
		Name:     "myfile",
		Type:     client.FileEntry,
		Size:     6,
		Contents: []byte("mydata"),
	}
	normalizeTreeEntry(wantEntry)

	if !reflect.DeepEqual(e, wantEntry) {
		t.Errorf("got tree entry %+v, want %+v", e, wantEntry)
	}
}

func TestServeRepoTreeEntry_Dir(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	rm := &mockFileSystem{
		t:  t,
		at: "abcd",
		fs: vfs{mapfs.New(map[string]string{"myfile": "mydata", "mydir/f": ""})},
	}
	sm := &mockService{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
		repo:     rm,
	}
	Service = sm

	resp, err := http.Get(server.URL + router.URLToRepoTreeEntry("git", cloneURL, "abcd", ".").String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("got status code %d, want %d", got, want)
	}

	if !sm.opened {
		t.Errorf("!opened")
	}
	if !rm.called {
		t.Errorf("!called")
	}

	var e *client.TreeEntry
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		t.Fatal(err)
	}

	wantEntry := &client.TreeEntry{
		Name: ".",
		Type: client.DirEntry,
		Entries: []*client.TreeEntry{
			{
				Name: "mydir",
				Type: client.DirEntry,
			},
			{
				Name: "myfile",
				Type: client.FileEntry,
				Size: 6,
			},
		},
	}
	normalizeTreeEntry(wantEntry)

	if !reflect.DeepEqual(e, wantEntry) {
		t.Errorf("got tree entry %+v, want %+v", e, wantEntry)
	}
}

type mockService struct {
	t *testing.T

	// expected args
	vcs      string
	cloneURL *url.URL

	// return values
	repo interface{}
	err  error

	opened bool
}

func (m *mockService) Open(vcs string, cloneURL *url.URL) (interface{}, error) {
	if vcs != m.vcs {
		m.t.Errorf("mock: got vcs arg %q, want %q", vcs, m.vcs)
	}
	if cloneURL.String() != m.cloneURL.String() {
		m.t.Errorf("mock: got cloneURL arg %q, want %q", cloneURL, m.cloneURL)
	}
	m.opened = true
	return m.repo, m.err
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

type mockFileSystem struct {
	t *testing.T

	// expected args
	at vcs.CommitID

	// return values
	fs  vcs.FileSystem
	err error

	called bool
}

func (m *mockFileSystem) FileSystem(at vcs.CommitID) (vcs.FileSystem, error) {
	if at != m.at {
		m.t.Errorf("mock: got at arg %q, want %q", at, m.at)
	}
	m.called = true
	return m.fs, m.err
}

// vfs wraps a godoc/vfs.FileSystem to implement vcs.FileSystem.
type vfs struct{ godoc_vfs.FileSystem }

// Open implements vcs.FileSystem (using the underlying godoc/vfs.FileSystem
// Open method, which returns an interface with the same methods but at a
// different import path).
func (fs vfs) Open(name string) (vcs.ReadSeekCloser, error) { return fs.FileSystem.Open("/" + name) }
func (fs vfs) Lstat(path string) (os.FileInfo, error)       { return fs.FileSystem.Lstat("/" + path) }
func (fs vfs) Stat(path string) (os.FileInfo, error)        { return fs.FileSystem.Stat("/" + path) }
func (fs vfs) ReadDir(path string) ([]os.FileInfo, error)   { return fs.FileSystem.ReadDir("/" + path) }

func normalizeCommit(c *vcs.Commit) {
	c.Author.Date = c.Author.Date.In(time.UTC)
	if c.Committer != nil {
		c.Committer.Date = c.Committer.Date.In(time.UTC)
	}
}

func normalizeTreeEntry(e *client.TreeEntry) {
	e.ModTime = e.ModTime.In(time.UTC)
	for _, e := range e.Entries {
		normalizeTreeEntry(e)
	}
}

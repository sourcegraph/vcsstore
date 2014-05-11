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
	"github.com/sourcegraph/vcsstore/vcsclient"
)

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

	var e *vcsclient.TreeEntry
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		t.Fatal(err)
	}

	wantEntry := &vcsclient.TreeEntry{
		Name:     "myfile",
		Type:     vcsclient.FileEntry,
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

	var e *vcsclient.TreeEntry
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		t.Fatal(err)
	}

	wantEntry := &vcsclient.TreeEntry{
		Name: ".",
		Type: vcsclient.DirEntry,
		Entries: []*vcsclient.TreeEntry{
			{
				Name: "mydir",
				Type: vcsclient.DirEntry,
			},
			{
				Name: "myfile",
				Type: vcsclient.FileEntry,
				Size: 6,
			},
		},
	}
	normalizeTreeEntry(wantEntry)

	if !reflect.DeepEqual(e, wantEntry) {
		t.Errorf("got tree entry %+v, want %+v", e, wantEntry)
	}
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

func normalizeTreeEntry(e *vcsclient.TreeEntry) {
	e.ModTime = e.ModTime.In(time.UTC)
	for _, e := range e.Entries {
		normalizeTreeEntry(e)
	}
}

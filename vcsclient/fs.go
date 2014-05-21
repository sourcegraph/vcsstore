package vcsclient

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/sourcegraph/go-vcs/vcs"
)

type FileSystem interface {
	vcs.FileSystem
	Get(path string) (*TreeEntry, error)
}

type repositoryFS struct {
	at   vcs.CommitID
	repo *repository
}

var _ FileSystem = &repositoryFS{}

func (fs *repositoryFS) Open(name string) (vcs.ReadSeekCloser, error) {
	e, err := fs.Get(name)
	if err != nil {
		return nil, err
	}

	return nopCloser{bytes.NewReader(e.Contents)}, nil
}

func (fs *repositoryFS) Lstat(path string) (os.FileInfo, error) {
	e, err := fs.Get(path)
	if err != nil {
		return nil, err
	}

	return e.Stat()
}

func (fs *repositoryFS) Stat(path string) (os.FileInfo, error) {
	// TODO(sqs): follow symlinks (as Stat specification requires)
	e, err := fs.Get(path)
	if err != nil {
		return nil, err
	}

	return e.Stat()
}

func (fs *repositoryFS) ReadDir(path string) ([]os.FileInfo, error) {
	e, err := fs.Get(path)
	if err != nil {
		return nil, err
	}

	fis := make([]os.FileInfo, len(e.Entries))
	for i, e := range e.Entries {
		fis[i], err = e.Stat()
		if err != nil {
			return nil, err
		}
	}

	return fis, nil
}

func (fs *repositoryFS) String() string {
	return fmt.Sprintf("%s repository %s commit %s (client)", fs.repo.vcsType, fs.repo.cloneURL, fs.at)
}

// Get returns the whole TreeEntry struct for a tree entry.
func (fs *repositoryFS) Get(path string) (*TreeEntry, error) {
	url, err := fs.url(path)
	if err != nil {
		return nil, err
	}

	req, err := fs.repo.client.NewRequest("GET", url.String())
	if err != nil {
		return nil, err
	}

	var entry *TreeEntry
	_, err = fs.repo.client.Do(req, &entry)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

// url generates the URL to RouteRepoTreeEntry for the given path (all other
// route vars are taken from repositoryFS fields).
func (fs *repositoryFS) url(path string) (*url.URL, error) {
	return fs.repo.url(RouteRepoTreeEntry, map[string]string{
		"CommitID": string(fs.at),
		"Path":     path,
	})
}

type nopCloser struct {
	io.ReadSeeker
}

func (nc nopCloser) Close() error { return nil }

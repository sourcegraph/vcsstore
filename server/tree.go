package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sourcegraph/vcsstore/client"
	"github.com/sqs/mux"
)

func serveRepoTreeEntry(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)
	start := time.Now()

	repo, _, err := getRepo(r)
	if err != nil {
		return err
	}

	commitID, err := getCommitID(r)
	if err != nil {
		return err
	}

	type fileSystem interface {
		FileSystem(vcs.CommitID) (vcs.FileSystem, error)
	}
	if repo, ok := repo.(fileSystem); ok {
		fs, err := repo.FileSystem(commitID)
		if err != nil {
			return err
		}

		path := v["Path"]
		fi, err := fs.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return &httpError{http.StatusNotFound, err}
			}
			return err
		}

		e := newTreeEntry(fi)

		if fi.Mode().IsDir() {
			entries, err := fs.ReadDir(path)
			if err != nil {
				return err
			}

			e.Entries = make([]*client.TreeEntry, len(entries))
			for i, fi := range entries {
				e.Entries[i] = newTreeEntry(fi)
			}
			sort.Sort(treeEntries(e.Entries))
		} else if fi.Mode().IsRegular() {
			f, err := fs.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			contents, err := ioutil.ReadAll(f)
			if err != nil {
				return err
			}

			e.Contents = contents
		}

		numTreeEntryResponses.Add(1)
		totalTreeEntryResponseTime.Add(int64(time.Since(start)))

		return writeJSON(w, e)
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("FileSystem not yet implemented for %T", repo)}
}

func newTreeEntry(fi os.FileInfo) *client.TreeEntry {
	e := &client.TreeEntry{
		Name:    fi.Name(),
		Size:    int(fi.Size()),
		ModTime: fi.ModTime(),
	}
	if fi.Mode().IsDir() {
		e.Type = client.DirEntry
	} else if fi.Mode().IsRegular() {
		e.Type = client.FileEntry
	} else if fi.Mode()&os.ModeSymlink != 0 {
		e.Type = client.SymlinkEntry
	}
	return e
}

type treeEntries []*client.TreeEntry

func (v treeEntries) Len() int           { return len(v) }
func (v treeEntries) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v treeEntries) Less(i, j int) bool { return v[i].Name < v[j].Name }

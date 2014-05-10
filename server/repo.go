package server

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sourcegraph/vcsstore/client"
	"github.com/sqs/mux"
)

func serveRepo(w http.ResponseWriter, r *http.Request) error {
	repo, cloneURL, err := getRepo(r)
	if err != nil {
		return err
	}

	return writeJSON(w, struct {
		ImplementationType string
		CloneURL           string
	}{fmt.Sprintf("%T", repo), cloneURL.String()})
}

func serveRepoBranch(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

	repo, cloneURL, err := getRepo(r)
	if err != nil {
		return err
	}

	type resolveBranch interface {
		ResolveBranch(string) (vcs.CommitID, error)
	}
	if repo, ok := repo.(resolveBranch); ok {
		commitID, err := repo.ResolveBranch(v["Branch"])
		if err != nil {
			return err
		}

		http.Redirect(w, r, router.URLToRepoCommit(v["VCS"], cloneURL, commitID).String(), http.StatusSeeOther)
		return nil
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("ResolveBranch not yet implemented for %T", repo)}
}

func serveRepoCommit(w http.ResponseWriter, r *http.Request) error {
	repo, _, err := getRepo(r)
	if err != nil {
		return err
	}

	commitID, err := getCommitID(r)
	if err != nil {
		return err
	}

	type getCommit interface {
		GetCommit(vcs.CommitID) (*vcs.Commit, error)
	}
	if repo, ok := repo.(getCommit); ok {
		commit, err := repo.GetCommit(commitID)
		if err != nil {
			return err
		}

		return writeJSON(w, commit)
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("GetCommit not yet implemented for %T", repo)}
}

func serveRepoCommitLog(w http.ResponseWriter, r *http.Request) error {
	repo, _, err := getRepo(r)
	if err != nil {
		return err
	}

	commitID, err := getCommitID(r)
	if err != nil {
		return err
	}

	type commitLog interface {
		CommitLog(to vcs.CommitID) ([]*vcs.Commit, error)
	}
	if repo, ok := repo.(commitLog); ok {
		commits, err := repo.CommitLog(commitID)
		if err != nil {
			return err
		}

		return writeJSON(w, commits)
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("GetCommit not yet implemented for %T", repo)}
}

func serveRepoRevision(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

	repo, cloneURL, err := getRepo(r)
	if err != nil {
		return err
	}

	type resolveRevision interface {
		ResolveRevision(string) (vcs.CommitID, error)
	}
	if repo, ok := repo.(resolveRevision); ok {
		commitID, err := repo.ResolveRevision(v["RevSpec"])
		if err != nil {
			return err
		}

		http.Redirect(w, r, router.URLToRepoCommit(v["VCS"], cloneURL, commitID).String(), http.StatusSeeOther)
		return nil
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("ResolveRevision not yet implemented for %T", repo)}
}

func serveRepoTag(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

	repo, cloneURL, err := getRepo(r)
	if err != nil {
		return err
	}

	type resolveTag interface {
		ResolveTag(string) (vcs.CommitID, error)
	}
	if repo, ok := repo.(resolveTag); ok {
		commitID, err := repo.ResolveTag(v["Tag"])
		if err != nil {
			return err
		}

		http.Redirect(w, r, router.URLToRepoCommit(v["VCS"], cloneURL, commitID).String(), http.StatusSeeOther)
		return nil
	}

	return &httpError{http.StatusNotImplemented, fmt.Errorf("ResolveTag not yet implemented for %T", repo)}
}

func serveRepoTreeEntry(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

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

var getRepoMu sync.Mutex

func getRepo(r *http.Request) (interface{}, *url.URL, error) {
	// TODO(sqs): only lock per-repo if there are write ops going on
	getRepoMu.Lock()
	defer getRepoMu.Unlock()

	v := mux.Vars(r)
	cloneURLStr := v["CloneURL"]
	if cloneURLStr == "" {
		// If cloneURLStr is empty, then the CloneURLEscaped route var failed to
		// be unescaped using url.QueryUnescape.
		return nil, nil, &httpError{http.StatusBadRequest, errors.New("invalid clone URL (unescaping failed)")}
	}

	cloneURL, err := url.Parse(cloneURLStr)
	if err != nil {
		return nil, nil, &httpError{http.StatusBadRequest, errors.New("invalid clone URL (parsing failed)")}
	}

	repo, err := Service.Open(v["VCS"], cloneURL)
	if err != nil {
		return nil, nil, err
	}

	return repo, cloneURL, nil
}

func getCommitID(r *http.Request) (vcs.CommitID, error) {
	v := mux.Vars(r)
	commitID := v["CommitID"]
	if commitID == "" {
		return "", &httpError{http.StatusBadRequest, errors.New("CommitID is empty")}
	}

	// check that it is lowercase hex
	i := strings.IndexFunc(commitID, func(c rune) bool {
		return !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'))
	})
	if i != -1 {
		return "", &httpError{http.StatusBadRequest, errors.New("CommitID must be lowercase hex")}
	}

	return vcs.CommitID(commitID), nil
}

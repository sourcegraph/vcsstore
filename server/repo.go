package server

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/sourcegraph/go-vcs/vcs"
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

	type branchResolver interface {
		ResolveBranch(string) (vcs.CommitID, error)
	}
	if repo, ok := repo.(branchResolver); ok {
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
	repo, cloneURL, err := getRepo(r)
	if err != nil {
		return err
	}

	_ = repo
	_ = cloneURL

	return nil
}

func serveRepoRevision(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)

	repo, cloneURL, err := getRepo(r)
	if err != nil {
		return err
	}

	type revisionResolver interface {
		ResolveRevision(string) (vcs.CommitID, error)
	}
	if repo, ok := repo.(revisionResolver); ok {
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

	type tagResolver interface {
		ResolveTag(string) (vcs.CommitID, error)
	}
	if repo, ok := repo.(tagResolver); ok {
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

	type fileSystem interface {
		FileSystem(vcs.CommitID) (vcs.FileSystem, error)
	}
	if repo, ok := repo.(fileSystem); ok {
		fs, err := repo.FileSystem(vcs.CommitID(v["CommitID"]))
		if err != nil {
			return err
		}

		type entry struct {
			Name     string
			Size     int
			Type     string
			ModTime  time.Time
			Contents []byte   `json:",omitempty"`
			Entries  []*entry `json:",omitempty"`
		}
		makeEntry := func(fi os.FileInfo) *entry {
			e := &entry{
				Name:    fi.Name(),
				Size:    int(fi.Size()),
				ModTime: fi.ModTime(),
			}
			if fi.Mode().IsDir() {
				e.Type = "dir"
			} else if fi.Mode().IsRegular() {
				e.Type = "file"
			} else if fi.Mode()&os.ModeSymlink != 0 {
				e.Type = "symlink"
			}
			return e
		}

		path := v["Path"]
		fi, err := fs.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return &httpError{http.StatusNotFound, err}
			}
			return err
		}

		e := makeEntry(fi)

		if fi.Mode().IsDir() {
			entries, err := fs.ReadDir(path)
			if err != nil {
				return err
			}

			e.Entries = make([]*entry, len(entries))
			for i, fi := range entries {
				e.Entries[i] = makeEntry(fi)
			}
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

var getRepoMu sync.Mutex

func getRepo(r *http.Request) (vcs.Repository, *url.URL, error) {
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

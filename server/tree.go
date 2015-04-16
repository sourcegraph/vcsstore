package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sort"

	"github.com/gorilla/schema"
	"github.com/sourcegraph/mux"
	"golang.org/x/tools/godoc/vfs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/vcsstore/fileutil"
	"sourcegraph.com/sourcegraph/vcsstore/vcsclient"
)

func (h *Handler) serveRepoTreeEntry(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)
	var opt vcsclient.GetFileOptions
	schema.NewDecoder().Decode(&opt, r.URL.Query())

	repo0, _, done, err := h.getRepo(r)
	if err != nil {
		return err
	}
	defer done()

	commitID, canon, err := getCommitID(r)
	if err != nil {
		return err
	}

	repo, ok := repo0.(interface {
		FileSystem(vcs.CommitID) (vfs.FileSystem, error)
	})
	if !ok {
		return &httpError{http.StatusNotImplemented, fmt.Errorf("FileSystem not yet implemented for %T", repo)}
	}

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
	var respVal interface{} = e // the value written to the resp body as JSON

	if fi.Mode().IsDir() {
		if opt.FullTree {
			path = ""
		}
		ee, err := readDir(fs, path, opt.FullTree)
		if err != nil {
			return err
		}
		e.Entries = ee
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

		// Check for extended range options (GetFileOptions).
		var fopt vcsclient.GetFileOptions
		if err := schemaDecoder.Decode(&fopt, r.URL.Query()); err != nil {
			return err
		}
		if empty := (vcsclient.GetFileOptions{}); fopt != empty {
			fr, _, err := fileutil.ComputeFileRange(contents, fopt)
			if err != nil {
				return err
			}

			// Trim to only requested range.
			e.Contents = e.Contents[fr.StartByte:fr.EndByte]
			respVal = &vcsclient.FileWithRange{
				TreeEntry: e,
				FileRange: *fr,
			}
		}
	}

	if canon {
		setLongCache(w)
	} else {
		setShortCache(w)
	}
	return writeJSON(w, respVal)
}

// readDir uses the passed vfs.FileSystem to read from starting at the base path. If
// shouldRecurse is set to true, it will recurse into all sub-folders and return the
// full sub-tree.
func readDir(fs vfs.FileSystem, base string, shouldRecurse bool) ([]*vcsclient.TreeEntry, error) {
	entries, err := fs.ReadDir(base)
	if err != nil {
		return nil, err
	}
	te := make([]*vcsclient.TreeEntry, len(entries))
	for i, fi := range entries {
		te[i] = newTreeEntry(fi)
		if fi.Mode().IsDir() && shouldRecurse {
			ee, err := readDir(fs, path.Join(base, fi.Name()), true)
			if err != nil {
				return nil, err
			}
			te[i].Entries = ee
		}
	}
	sort.Sort(vcsclient.TreeEntriesByTypeByName(te))
	return te, nil
}

func newTreeEntry(fi os.FileInfo) *vcsclient.TreeEntry {
	e := &vcsclient.TreeEntry{
		Name:    fi.Name(),
		Size:    int(fi.Size()),
		ModTime: fi.ModTime(),
	}
	if fi.Mode().IsDir() {
		e.Type = vcsclient.DirEntry
	} else if fi.Mode().IsRegular() {
		e.Type = vcsclient.FileEntry
	} else if fi.Mode()&os.ModeSymlink != 0 {
		e.Type = vcsclient.SymlinkEntry
	}
	return e
}

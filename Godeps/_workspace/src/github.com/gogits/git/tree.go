package git

import (
	"errors"
	"path"
	"strings"
)

var (
	ErrNotExist = errors.New("error not exist")
)

type TreeWalkCallback func(string, *TreeEntry) int

// A tree is a flat directory listing.
type Tree struct {
	Id   sha1
	repo *Repository

	// parent tree
	ptree *Tree

	entries       Entries
	entriesParsed bool
}

// The entries will be traversed in the specified order,
// children subtrees will be automatically loaded as required, and the
// callback will be called once per blob with the current (relative) root
// for the blob and the blob data itself.
//
// If the callback returns a positive value, the passed blob will be skipped
// on the traversal (in pre mode). A negative value stops the walk.
//
// Walk will panic() if an error occurs
func (t *Tree) walk(callback TreeWalkCallback) error {
	t._walk(callback, "")
	return nil
}

func (t *Tree) _walk(cb TreeWalkCallback, dirname string) bool {
	for _, te := range t.ListEntries() {
		cont := cb(dirname, te)
		switch {
		case cont < 0:
			return false
		case cont == 0:
			// descend if it is a tree
			if te.Type == ObjectTree {
				commit, err := t.repo.getCommit(te.Id)
				if err != nil {
					panic(err)
				}
				t, err := t.repo.getTree(commit.Id)
				if err != nil {
					panic(err)
				}
				if t._walk(cb, path.Join(dirname, te.name)) == false {
					return false
				}
			}
		case cont > 0:
			// do nothing, don't descend into the tree
		}
	}
	return true
}

func (t *Tree) SubTree(rpath string) (*Tree, error) {
	if len(rpath) == 0 {
		return t, nil
	}

	paths := strings.Split(rpath, "/")
	var err error
	var g = t
	var p = t
	var te *TreeEntry
	for _, name := range paths {
		te, err = p.GetTreeEntryByPath(name)
		if err != nil {
			return nil, err
		}
		g, err = t.repo.getTree(te.Id)
		if err != nil {
			return nil, err
		}
		g.ptree = p
		p = g
	}
	return g, nil
}

func (t *Tree) ListEntries() Entries {
	if t.entriesParsed {
		return t.entries
	}

	t.entriesParsed = true

	_, _, data, err := t.repo.getRawObject(t.Id)
	if err != nil {
		return nil
	}

	t.entries, err = parseTreeData(t, data)
	if err != nil {
		return nil
	}

	return t.entries
}

func NewTree(repo *Repository, id sha1) *Tree {
	tree := new(Tree)
	tree.Id = id
	tree.repo = repo
	return tree
}

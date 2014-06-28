package git

import (
	"path"
	"strings"
)

func (t *Tree) GetTreeEntryByPath(rpath string) (*TreeEntry, error) {
	if len(rpath) == 0 {
		return nil, ErrNotExist
	}

	parts := strings.Split(path.Clean(rpath), "/")
	var err error
	tree := t
	for i, name := range parts {
		if i == len(parts)-1 {
			for _, v := range tree.ListEntries() {
				if v.name == name {
					return v, nil
				}
			}
		} else {
			tree, err = tree.SubTree(name)
			if err != nil {
				return nil, err
			}
		}
	}

	return nil, ErrNotExist
}

func (t *Tree) GetBlobByPath(rpath string) (*Blob, error) {
	entry, err := t.GetTreeEntryByPath(rpath)
	if err != nil {
		return nil, err
	}

	if !entry.IsDir() {
		return entry.Blob(), nil
	}

	return nil, ErrNotExist
}

// Copyright 2013 The hgo Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hgo

import (
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

// BranchHeads contains a mapping from branch names to head changeset IDs, and a
// mapping from head changeset IDs to slices of branch names.
type BranchHeads struct {
	IdByName map[string]string
	ById     map[string][]string
}

func newBranchHeads() *BranchHeads {
	return &BranchHeads{
		ById:     map[string][]string{},
		IdByName: map[string]string{},
	}
}

func (dest *BranchHeads) copy(src *BranchHeads) {
	for k, v := range src.ById {
		dest.ById[k] = v
	}
	for k, v := range src.IdByName {
		dest.IdByName[k] = v
	}
}

// BranchHeads parses and returns the repository's branch heads.
func (r *Repository) BranchHeads() (*BranchHeads, error) {
	bh := newBranchHeads()

	names := []string{"branchheads-served", "branchheads-base", "branchheads"}
	for _, name := range names {
		f, err := r.open(".hg/cache/" + name)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		defer f.Close()

		err = bh.parseFile(f)
		if err != nil {
			return nil, err
		}
	}

	return bh, nil
}

func (bh *BranchHeads) parseFile(r io.Reader) error {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	lines := strings.Split(string(buf), "\n")
	m := make(map[string]string, len(lines)-1)

	for i, line := range lines {
		if i == 0 {
			// first line is current numbered, don't include
			continue
		}
		branch := strings.SplitN(strings.TrimSpace(line), " ", 2)
		if len(branch) != 2 {
			continue
		}
		m[branch[1]] = branch[0]
	}
	// unify
	for name, id := range m {
		bh.ById[id] = append(bh.ById[id], name)
		bh.IdByName[name] = id
	}
	return nil
}

// Associate a new branch with a changeset ID.
func (bh *BranchHeads) Add(name, id string) {
	bh.ById[id] = append(bh.ById[id], name)
	bh.IdByName[name] = id
}

// For each changeset ID within the ById member, sort the branch names
// associated with it in increasing order. This function should be called
// if one or more branches have been inserted using Add.
func (bh *BranchHeads) Sort() {
	for _, v := range bh.ById {
		switch len(v) {
		case 1:
		case 2:
			if v[0] > v[1] {
				v[0], v[1] = v[1], v[0]
			}
		default:
			sort.Strings(v)
		}
	}
}

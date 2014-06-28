package git

import (
	"bytes"
	"errors"
)

// Parse tree information from the (uncompressed) raw
// data from the tree object.
func parseTreeData(tree *Tree, data []byte) ([]*TreeEntry, error) {
	entries := make([]*TreeEntry, 0, 10)
	l := len(data)
	pos := 0
	for pos < l {
		entry := new(TreeEntry)
		entry.ptree = tree
		spacepos := bytes.IndexByte(data[pos:], ' ')
		switch string(data[pos : pos+spacepos]) {
		case "100644":
			entry.mode = ModeBlob
			entry.Type = ObjectBlob
		case "100755":
			entry.mode = ModeExec
			entry.Type = ObjectBlob
		case "120000":
			entry.mode = ModeSymlink
			entry.Type = ObjectBlob
		case "160000":
			entry.mode = ModeCommit
			entry.Type = ObjectCommit
		case "40000":
			entry.mode = ModeTree
			entry.Type = ObjectTree
		default:
			return nil, errors.New("unknown type: " + string(data[pos:pos+spacepos]))
		}
		pos += spacepos + 1
		zero := bytes.IndexByte(data[pos:], 0)
		entry.name = string(data[pos : pos+zero])
		pos += zero + 1
		id, err := NewId(data[pos : pos+20])
		if err != nil {
			return nil, err
		}
		entry.Id = id
		pos = pos + 20
		entries = append(entries, entry)
	}
	return entries, nil
}

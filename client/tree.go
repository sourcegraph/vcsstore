package client

import "time"

type TreeEntryType string

const (
	FileEntry    TreeEntryType = "file"
	DirEntry     TreeEntryType = "dir"
	SymlinkEntry TreeEntryType = "symlink"
)

type TreeEntry struct {
	Name     string
	Type     TreeEntryType
	Size     int
	ModTime  time.Time
	Contents []byte       `json:",omitempty"`
	Entries  []*TreeEntry `json:",omitempty"`
}

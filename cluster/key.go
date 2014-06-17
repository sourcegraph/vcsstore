package cluster

import (
	"net/url"
	"strings"
)

// Keyer is implemented by any value that has a Key method, which defines the
type Keyer interface {
	Key() string
}

type CloneURL struct {
	// VCS is the type of version control system of the repository: "git", "hg", etc.
	VCS string

	*url.URL
}

func (s CloneURL) Key() string {
	return s.VCS + "/" + url.QueryEscape(strings.Replace(s.URL.String(), "//", "/", -1))
}

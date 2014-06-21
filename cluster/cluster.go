// Package cluster uses datad to distribute VCS data storage across multiple
// machines.
package cluster

import (
	"net/url"
	"strings"
)

func RepositoryKey(vcsType string, cloneURL *url.URL) string {
	return strings.Join([]string{vcsType, cloneURL.Scheme, cloneURL.Host, cloneURL.Path}, "/")
}

func DecodeRepositoryKey(key string) (vcsType string, cloneURL *url.URL, err error) {
	parts := strings.SplitN(key, "/", 4)
	if len(parts) != 4 {
		tmp := make([]string, 4)
		copy(tmp, parts)
		parts = tmp
	}
	vcsType = parts[0]
	cloneURL = &url.URL{Scheme: parts[1], Host: parts[2], Path: parts[3]}
	return vcsType, cloneURL, nil
}

package vcsstore

import (
	"net/url"
	"path/filepath"
	"strings"
)

// EncodeRepositoryPath encodes the VCS type and clone URL of a repository into
// a path suitable for use in a URL. The encoded path may be decoded with
// DecodeRepositoryPath, which is roughly the inverse operation (except for
// calling filepath.Clean on the URL path).
func EncodeRepositoryPath(vcsType string, cloneURL *url.URL) string {
	return strings.Join([]string{vcsType, cloneURL.Scheme, cloneURL.Host, strings.TrimPrefix(filepath.Clean(cloneURL.Path), "/")}, "/")
}

// DecodeRepositoryPath decodes a repository path encoded using RepositoryPath.
func DecodeRepositoryPath(path string) (vcsType string, cloneURL *url.URL, err error) {
	parts := strings.SplitN(path, "/", 4)
	if len(parts) != 4 {
		tmp := make([]string, 4)
		copy(tmp, parts)
		parts = tmp
	}
	vcsType = parts[0]
	cloneURL = &url.URL{Scheme: parts[1], Host: parts[2], Path: parts[3]}
	return vcsType, cloneURL, nil
}

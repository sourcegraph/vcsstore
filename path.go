package vcsstore

import "path/filepath"

func EncodeRepositoryPath(repoID string) (path string) {
	return filepath.Clean(repoID)
}

func DecodeRepositoryPath(path string) (repoID string) {
	return path
}

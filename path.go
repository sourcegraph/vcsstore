package vcsstore

import (
	"fmt"
	"os"
	"path/filepath"
)

func EncodeRepositoryPath(repoID string) (path string) {
	return filepath.Clean(repoID)
}

func DecodeRepositoryPath(path string) (repoID string) {
	return path
}

func vcsTypeFromDir(cloneDir string) (vcsType string, err error) {
	if _, err := os.Stat(filepath.Join(cloneDir, ".git")); err == nil {
		// git non-bare
		return "git", nil
	} else if _, err := os.Stat(filepath.Join(cloneDir, "objects")); err == nil {
		// git bare
		return "git", nil
	} else if _, err := os.Stat(filepath.Join(cloneDir, ".hg")); err == nil {
		return "hg", nil
	} else {
		if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
			return "", err
		} else {
			return "", fmt.Errorf("could not determine VCS from dir %s", cloneDir)
		}
	}
}

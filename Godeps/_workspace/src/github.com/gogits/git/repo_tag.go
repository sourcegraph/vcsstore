package git

import (
	"path/filepath"
)

func IsTagExist(repoPath, tagName string) bool {
	tagPath := filepath.Join(repoPath, "refs/tags", tagName)
	return isFile(tagPath)
}

func (repo *Repository) IsTagExist(tagName string) bool {
	return IsTagExist(repo.Path, tagName)
}

// GetTags returns all tags of given repository.
func (repo *Repository) GetTags() ([]string, error) {
	return repo.readRefDir("refs/tags", "")
}

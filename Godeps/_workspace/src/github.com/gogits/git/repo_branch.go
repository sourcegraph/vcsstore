package git

import (
	"os"
	"path/filepath"
	"strings"
)

func IsBranchExist(repoPath, branchName string) bool {
	branchPath := filepath.Join(repoPath, "refs/heads", branchName)
	return isFile(branchPath)
}

func (repo *Repository) IsBranchExist(branchName string) bool {
	return IsBranchExist(repo.Path, branchName)
}

func (repo *Repository) readRefDir(prefix, relPath string) ([]string, error) {
	dirPath := filepath.Join(repo.Path, prefix, relPath)
	f, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fis, err := f.Readdir(0)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(fis))
	for _, fi := range fis {
		if strings.Contains(fi.Name(), ".DS_Store") {
			continue
		}

		relFileName := filepath.Join(relPath, fi.Name())
		if fi.IsDir() {
			subnames, err := repo.readRefDir(prefix, relFileName)
			if err != nil {
				return nil, err
			}
			names = append(names, subnames...)
			continue
		}

		names = append(names, relFileName)
	}

	return names, nil
}

// GetBranches returns all branches of given repository.
func (repo *Repository) GetBranches() ([]string, error) {
	return repo.readRefDir("refs/heads", "")
}

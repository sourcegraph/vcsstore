package vcsstore

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestEncodeAndDecodeRepositoryPath(t *testing.T) {
	repos := []struct {
		repoPath string
		want     string
	}{
		{"foo.com/bar/baz", "foo.com/bar/baz"},
		{"github.com/sourcegraph/go-sourcegraph", "github.com/sourcegraph/go-sourcegraph"},
	}
	for _, repo := range repos {
		encPath := EncodeRepositoryPath(repo.repoPath)
		if encPath != repo.want {
			t.Errorf("got encoded path == %q, want %q", encPath, repo.want)
		}

		repoPath := DecodeRepositoryPath(encPath)
		if repoPath != repo.repoPath {
			t.Errorf("got repoPath == %q, want %q", repoPath, repo.repoPath)
		}
	}
}

func TestVCSTypeFromDir(t *testing.T) {
	tests := []struct {
		initCmd    string
		expVCSType string
	}{{"git init", "git"}, {"git init --bare", "git"}, {"hg init", "hg"}}

	for _, test := range tests {
		func() {
			repoDir, err := ioutil.TempDir("", "TestVCSTypeFromDir")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(repoDir)

			cmdArgs := strings.Fields(test.initCmd)
			cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
			cmd.Dir = repoDir
			if err := cmd.Run(); err != nil {
				t.Fatal(err)
			}

			vcsType, err := vcsTypeFromDir(repoDir)
			if err != nil {
				t.Errorf("unexpected error calling vcsTypeFromDir: %s", err)
			} else if vcsType != test.expVCSType {
				t.Errorf("expected VCS type %s, but got %s", test.expVCSType, vcsType)
			}
		}()
	}
}

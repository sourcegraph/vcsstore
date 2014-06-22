package vcsstore

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sourcegraph/go-vcs/vcs"
)

type Service interface {
	// Open opens a repository. If it doesn't exist. an
	// os.ErrNotExist-satisfying error is returned. If opening succeeds, the
	// repository is returned.
	Open(vcs string, cloneURL *url.URL) (interface{}, error)

	// Clone clones the repository if a clone doesn't yet exist locally.
	// Otherwise, it opens the repository. If no errors occur, the repository is
	// returned.
	Clone(vcs string, cloneURL *url.URL) (interface{}, error)
}

type Config struct {
	// StorageDir is where cloned repositories are stored. If empty, the current
	// working directory is used.
	StorageDir string

	Log *log.Logger

	DebugLog *log.Logger
}

// CloneDir validates vcsType and cloneURL. If they are valid, cloneDir returns
// the local directory that the repository should be cloned to (which it may
// already exist at). If invalid, cloneDir returns a non-nil error.
func (c *Config) CloneDir(vcsType string, cloneURL *url.URL) (string, error) {
	if !isLowercaseLetter(vcsType) {
		return "", errors.New("invalid VCS type")
	}
	if cloneURL.Scheme == "" || cloneURL.Host == "" {
		return "", errors.New("invalid clone URL")
	}

	return filepath.Join(c.StorageDir, EncodeRepositoryPath(vcsType, cloneURL)), nil
}

func NewService(c *Config) Service {
	if c == nil {
		c = &Config{
			StorageDir: ".",
			Log:        log.New(os.Stderr, "vcsstore: ", log.LstdFlags),
			DebugLog:   log.New(ioutil.Discard, "", 0),
		}
	}
	return &service{
		Config: *c,
		repoMu: make(map[repoKey]*sync.Mutex),
	}
}

type service struct {
	Config

	// repoMu prevents more than one goroutine from simultaneously cloning the
	// same repository.
	repoMu map[repoKey]*sync.Mutex

	// repoMuMu synchronizes access to repoMu.
	repoMuMu sync.Mutex
}

type repoKey struct {
	vcsType  string
	cloneURL string
}

func (s *service) Open(vcsType string, cloneURL *url.URL) (interface{}, error) {
	cloneDir, err := s.CloneDir(vcsType, cloneURL)
	if err != nil {
		return nil, err
	}
	return s.open(vcsType, cloneDir)
}

func (s *service) open(vcsType, cloneDir string) (interface{}, error) {
	if fi, err := os.Stat(cloneDir); err != nil {
		return nil, err
	} else if !fi.Mode().IsDir() {
		return nil, fmt.Errorf("clone path %q is not a directory", cloneDir)
	}
	return vcs.OpenMirror(vcsType, cloneDir)
}

func (s *service) Clone(vcsType string, cloneURL *url.URL) (interface{}, error) {
	cloneDir, err := s.CloneDir(vcsType, cloneURL)
	if err != nil {
		return nil, err
	}

	// See if the clone directory exists and return immediately (without
	// locking) if so.
	if r, err := s.open(vcsType, cloneDir); !os.IsNotExist(err) {
		if err == nil {
			s.DebugLog.Printf("Clone(%s, %s): repository already exists at %s", vcsType, cloneURL, cloneDir)
		} else {
			s.DebugLog.Printf("Clone(%s, %s): opening existing repository at %s failed: %s", vcsType, cloneURL, cloneDir, err)
		}
		return r, err
	}

	// The local clone directory doesn't exist, so we need to clone the repository.
	mu := s.Mutex(vcsType, cloneURL)
	mu.Lock()
	defer mu.Unlock()

	// Check again after obtaining the lock, so we don't clone multiple times.
	if r, err := s.open(vcsType, cloneDir); !os.IsNotExist(err) {
		if err == nil {
			s.DebugLog.Printf("Clone(%s, %s): after obtaining clone lock, repository already exists at %s", vcsType, cloneURL, cloneDir)
		} else {
			s.DebugLog.Printf("Clone(%s, %s): after obtaining clone lock, opening existing repository at %s failed: %s", vcsType, cloneURL, cloneDir, err)
		}
		return r, err
	}

	start := time.Now()
	msg := fmt.Sprintf("%s %s to %s", vcsType, cloneURL.String(), cloneDir)
	s.Log.Print("Cloning ", msg, "...")
	defer func() {
		s.Log.Print("Finished cloning ", msg, " in ", time.Since(start))
	}()

	// "Atomically" clone the repository. First, clone it to a temporary sibling
	// directory. Once the clone is complete, "atomically"
	// rename it to the intended cloneDir.
	//
	// "Atomically" is in quotes because this operation is not really atomic. It
	// depends on the underlying FS. For now, for our purposes, it performs well
	// enough on local ext4 and on GlusterFS.
	parentDir := filepath.Dir(cloneDir)
	if err := os.MkdirAll(parentDir, 0700); err != nil {
		return nil, err
	}

	cloneTmpDir, err := ioutil.TempDir(parentDir, "_tmp_"+filepath.Base(cloneDir)+"-")
	if err != nil {
		return nil, err
	}
	s.DebugLog.Printf("Clone(%s, %s): cloning to temporary sibling dir %s", vcsType, cloneURL, cloneTmpDir)
	defer os.RemoveAll(cloneTmpDir)

	_, err = vcs.CloneMirror(vcsType, cloneURL.String(), cloneTmpDir)
	if err != nil {
		return nil, err
	}
	s.DebugLog.Printf("Clone(%s, %s): cloned to temporary sibling dir %s; now renaming to intended clone dir %s", vcsType, cloneURL, cloneTmpDir, cloneDir)

	if err := os.Rename(cloneTmpDir, cloneDir); err != nil {
		s.DebugLog.Printf("Clone(%s, %s): Rename(%s -> %s) failed: %s", vcsType, cloneURL, cloneTmpDir, cloneDir)
		return nil, err
	}

	return s.open(vcsType, cloneDir)
}

func (s *service) Mutex(vcsType string, cloneURL *url.URL) *sync.Mutex {
	s.repoMuMu.Lock()
	defer s.repoMuMu.Unlock()

	k := repoKey{vcsType, cloneURL.String()}
	if mu, ok := s.repoMu[k]; ok {
		return mu
	}
	s.repoMu[k] = &sync.Mutex{}
	return s.repoMu[k]
}

func isLowercaseLetter(s string) bool {
	return strings.IndexFunc(s, func(c rune) bool {
		return !(c >= 'a' && c <= 'z')
	}) == -1
}

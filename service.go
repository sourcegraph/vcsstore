package vcsstore

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sourcegraph/go-vcs/vcs"
)

var (
	// RepositoryPath is called to determine the directory, relative to a
	// Config's StorageDir, to which the repository should be cloned to. The
	// default implementation stores repositories in directories of the form
	// "vcs-type/escaped-clone-url".
	RepositoryPath = func(vcsType string, cloneURL *url.URL) string {
		return filepath.Join(vcsType, url.QueryEscape(cloneURL.String()))
	}
)

// HashedRepositoryPath may be assigned to RepositoryPath to use paths
// of the form "xx/yy/zzzzzzzz" where xx and yy are the first 4 characters of
// some hash of vcsType and cloneURL, and zzzzzzzz is the full hash (minus the
// first 4 characters).
func HashedRepositoryPath(vcsType string, cloneURL *url.URL) string {
	h := sha1.New()
	h.Write([]byte(vcsType))
	h.Write([]byte(cloneURL.String()))
	s := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s/%s/%s/%s", vcsType, s[:2], s[2:4], s[4:])
}

type Service interface {
	Open(vcs string, cloneURL *url.URL) (interface{}, error)
}

type Config struct {
	// StorageDir is where cloned repositories are stored. If empty, the current
	// working directory is used.
	StorageDir string

	Log *log.Logger
}

func NewService(c *Config) Service {
	if c == nil {
		c = &Config{
			StorageDir: ".",
			Log:        log.New(os.Stderr, "vcsstore: ", log.LstdFlags),
		}
	}
	return &service{
		Config: *c,
		repoMu: make(map[repoKey]*sync.Mutex),
	}
}

type service struct {
	Config

	// repoMu synchronizes access to repository data on the filesystem.
	repoMu map[repoKey]*sync.Mutex

	// repoMuMu synchronizes access to repoMu.
	repoMuMu sync.Mutex
}

type repoKey struct {
	vcsType  string
	cloneURL string
}

func (s *service) Open(vcsType string, cloneURL *url.URL) (interface{}, error) {
	if !isLowercaseLetter(vcsType) {
		return nil, errors.New("invalid VCS type")
	}
	if cloneURL.Scheme == "" || cloneURL.Host == "" {
		return nil, errors.New("invalid clone URL")
	}

	cloneDir := filepath.Join(s.StorageDir, RepositoryPath(vcsType, cloneURL))

	mu := s.Mutex(vcsType, cloneURL)
	mu.Lock()
	defer mu.Unlock()

	_, err := os.Stat(cloneDir)
	if os.IsNotExist(err) {
		// Repository hasn't been cloned locally. Try cloning and opening it.
		if err := os.MkdirAll(filepath.Join(s.StorageDir, vcsType), 0700); err != nil {
			return nil, err
		}
		start := time.Now()
		msg := fmt.Sprintf("%s %s to %s", vcsType, cloneURL.String(), cloneDir)
		s.Log.Print("Cloning ", msg, "...")
		defer func() {
			s.Log.Print("Finished cloning ", msg, " in ", time.Since(start))
		}()
		return vcs.CloneMirror(vcsType, cloneURL.String(), cloneDir)
	} else if err != nil {
		return nil, err
	}

	return vcs.OpenMirror(vcsType, cloneDir)
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

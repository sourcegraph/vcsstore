package vcsstore

import (
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

	cloneDir := filepath.Join(s.StorageDir, vcsType, url.QueryEscape(cloneURL.String()))

	mu := s.Mutex(vcsType, cloneURL)
	mu.Lock()
	defer mu.Unlock()

	_, err := os.Stat(cloneDir)
	if os.IsNotExist(err) {
		// Repository hasn't been cloned locally. Try cloning and opening it.
		if err := os.Mkdir(filepath.Join(s.StorageDir, vcsType), 0700); err != nil && !os.IsExist(err) {
			return nil, err
		}
		start := time.Now()
		msg := fmt.Sprintf("%s %s to %s", vcsType, cloneURL.String(), cloneDir)
		s.Log.Print("Cloning ", msg, "...")
		time.Sleep(time.Second * 2)
		defer s.Log.Print("Finished cloning ", msg, " in ", time.Since(start))
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

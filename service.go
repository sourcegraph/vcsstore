package vcsstore

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/sourcegraph/go-vcs/vcs"
)

type Service interface {
	Open(vcs string, cloneURL *url.URL) (vcs.Repository, error)
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
	return &service{*c}
}

type service struct {
	Config
}

func (s *service) Open(vcsType string, cloneURL *url.URL) (vcs.Repository, error) {
	cloneDir := filepath.Join(s.StorageDir, url.QueryEscape(cloneURL.String()))

	_, err := os.Stat(cloneDir)
	if os.IsNotExist(err) {
		// Repository hasn't been cloned locally. Try cloning and opening it.
		start := time.Now()
		msg := fmt.Sprintf("%s %s to %s", vcsType, cloneURL.String(), cloneDir)
		s.Log.Print("Cloning ", msg, "...")
		defer s.Log.Print("Finished cloning ", msg, " in ", time.Since(start))
		return vcs.CloneMirror(vcsType, cloneURL.String(), cloneDir)
	} else if err != nil {
		return nil, err
	}

	return vcs.OpenMirror(vcsType, cloneDir)
}

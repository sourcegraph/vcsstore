package cluster

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/datad"
	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sourcegraph/vcsstore"
)

// A Server tracks repositories.
type Server struct {
	// Datad is the underlying datad client to use.
	Datad *datad.Client

	Log *log.Logger

	conf *vcsstore.Config
	svc  vcsstore.Service
}

// NewServer
func NewServer(dc *datad.Client, c *vcsstore.Config, s vcsstore.Service) *Server {
	return &Server{dc, log.New(os.Stderr, "vcsstore cluster: ", log.Ltime), c, s}
}

func (s *Server) ProviderHandler() http.Handler {
	return datad.NewProviderHandler(s)
}

// TODO(sqs): the master commit id is not fully representative of the version of
// the repo. it should probably be some hash of all of the head commit ids.
func (s *Server) KeyVersion(key string) (string, error) {
	key = strings.TrimPrefix(key, "/")

	vcsType, cloneURL, err := DecodeRepositoryKey(key)
	if err != nil {
		return "", err
	}

	repo, err := s.svc.Clone(vcsType, cloneURL)
	if err != nil {
		return "", err
	}

	return repositoryVersion(repo)
}

func (s *Server) KeyVersions(keyPrefix string) (map[string]string, error) {
	keyPrefix = strings.TrimPrefix(keyPrefix, "/")
	keyPrefix = filepath.Clean(keyPrefix)
	if strings.HasPrefix(keyPrefix, "..") || strings.HasPrefix(keyPrefix, "/") {
		return nil, errors.New("invalid keyPrefix")
	}
	topDir := filepath.Join(s.conf.StorageDir, keyPrefix)

	kvs := make(map[string]string)

	err := filepath.Walk(topDir, func(path string, info os.FileInfo, err error) error {
		// Ignore errors for broken symlinks.
		if err != nil {
			if info == nil {
				return err
			}
			if info.Mode()&os.ModeSymlink == 0 {
				return err
			}
		}

		if info.Mode().IsDir() {
			vcsTypes := []string{"git", "hg"}
			for _, vcsType := range vcsTypes {
				repo, err := vcs.OpenMirror(vcsType, path)
				if err == nil {
					key, err := filepath.Rel(s.conf.StorageDir, path)
					if err != nil {
						return err
					}

					version, err := repo.ResolveBranch("master")
					if err != nil {
						return err
					}

					kvs[key] = string(version)
					return filepath.SkipDir
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return kvs, nil
}

func (s *Server) Update(key, version string) error {
	key = strings.TrimPrefix(key, "/")
	vcsType, cloneURL, err := DecodeRepositoryKey(key)
	if err != nil {
		return err
	}

	repo, err := s.svc.Clone(vcsType, cloneURL)
	if err != nil {
		return err
	}

	curVersion, err := repositoryVersion(repo)
	if err != nil {
		return err
	}

	if version == curVersion {
		// Already at requested version.
		return nil
	}

	return updateRepository(repo)
}

func repositoryVersion(repo interface{}) (string, error) {
	type resolveBranch interface {
		ResolveBranch(string) (vcs.CommitID, error)
	}
	if repo, ok := repo.(resolveBranch); ok {
		commitID, err := repo.ResolveBranch("master")
		if err != nil {
			return "", err
		}
		return string(commitID), nil
	}
	return "", errors.New("failed to get version for repo")
}

func updateRepository(repo interface{}) error {
	type mirrorUpdate interface {
		MirrorUpdate() error
	}
	if repo, ok := repo.(mirrorUpdate); ok {
		return repo.MirrorUpdate()
	}
	return errors.New("failed to update repo")
}

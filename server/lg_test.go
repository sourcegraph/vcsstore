// +build lgtest

package server

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/sourcegraph/go-vcs/vcs"
	_ "github.com/sourcegraph/go-vcs/vcs/git"
	"github.com/sourcegraph/vcsstore"
	"github.com/sourcegraph/vcsstore/vcsclient"
)

var (
	sshKeyFile  = flag.String("sshkey", "", "ssh private key file for clone remote")
	privateRepo = flag.String("privrepo", "ssh://git@github.com/sourcegraph/private-repo.git", "a private, SSH-accessible repo to test cloning")
)

func TestClone_lg(t *testing.T) {
	storageDir, err := ioutil.TempDir("", "vcsstore-test")
	if err != nil {
		t.Fatal(err)
	}

	conf := &vcsstore.Config{
		StorageDir: storageDir,
		Log:        log.New(os.Stderr, "", 0),
		DebugLog:   log.New(os.Stderr, "", log.LstdFlags),
	}

	h := NewHandler(vcsstore.NewService(conf), nil, nil)
	h.Log = log.New(os.Stderr, "", 0)
	h.Debug = true

	srv := httptest.NewServer(h)
	defer srv.Close()

	baseURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c := vcsclient.New(baseURL, nil)
	repoURL, err := url.Parse(*privateRepo)
	if err != nil {
		t.Fatal(err)
	}
	repo, err := c.Repository("git", repoURL)
	if err != nil {
		t.Fatal(err)
	}

	var opt vcs.RemoteOpts
	if *sshKeyFile != "" {
		key, err := ioutil.ReadFile(*sshKeyFile)
		if err != nil {
			log.Fatal(err)
		}
		opt.SSH = &vcs.SSHConfig{PrivateKey: key}
	}

	if repo, ok := repo.(vcsclient.RepositoryCloneUpdater); ok {
		// Clones the first time.
		if err := repo.CloneOrUpdate(opt); err != nil {
			t.Fatal(err)
		}

		// Updates the second time.
		if err := repo.CloneOrUpdate(opt); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Fatalf("Remote cloning is not implemented for %T.", repo)
	}
}

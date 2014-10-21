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
	_ "github.com/sourcegraph/go-vcs/vcs/hg"
	_ "github.com/sourcegraph/go-vcs/vcs/hgcmd"
	"github.com/sourcegraph/vcsstore"
	"github.com/sourcegraph/vcsstore/vcsclient"
)

var (
	sshKeyFile  = flag.String("sshkey", "", "ssh private key file for clone remote")
	privateRepo = flag.String("privrepo", "ssh://git@github.com/sourcegraph/private-repo.git", "a private, SSH-accessible repo to test cloning")
)

func TestCloneGitHTTPS_lg(t *testing.T) {
	t.Parallel()
	testClone_lg(t, "git", "https://github.com/sgtest/empty-repo.git", vcs.RemoteOpts{})
}

func TestCloneGitGit_lg(t *testing.T) {
	t.Parallel()
	testClone_lg(t, "git", "git://github.com/sgtest/empty-repo.git", vcs.RemoteOpts{})
}

func TestCloneGitSSH_lg(t *testing.T) {
	t.Parallel()
	if *sshKeyFile == "" {
		t.Skip("no ssh key specified")
	}

	var opt vcs.RemoteOpts
	if *sshKeyFile != "" {
		key, err := ioutil.ReadFile(*sshKeyFile)
		if err != nil {
			log.Fatal(err)
		}
		opt.SSH = &vcs.SSHConfig{PrivateKey: key}
	}

	testClone_lg(t, "git", *privateRepo, opt)
}

func TestCloneHgHTTPS_lg(t *testing.T) {
	t.Parallel()
	testClone_lg(t, "hg", "https://bitbucket.org/sqs/go-vcs-hgtest", vcs.RemoteOpts{})
}

func testClone_lg(t *testing.T, vcsType, repoURLStr string, opt vcs.RemoteOpts) {
	storageDir, err := ioutil.TempDir("", "vcsstore-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(storageDir)

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
	repoURL, err := url.Parse(repoURLStr)
	if err != nil {
		t.Fatal(err)
	}
	repo, err := c.Repository(vcsType, repoURL)
	if err != nil {
		t.Fatal(err)
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

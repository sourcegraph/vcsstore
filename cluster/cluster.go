package cluster

import (
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/go-etcd/etcd"
	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sourcegraph/vcsstore/vcsclient"
)

type Client struct {
	dir *Directory

	c *etcd.Client

	// HTTP client used to communicate with the vcsstore API.
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	c := etcd.NewClient([]string{"http://localhost:4001"})
	return &Client{
		dir: &Directory{c},
		c:   c,
	}
}

func (c *Client) Repository(vcsType string, cloneURL *url.URL) (vcs.Repository, error) {
	key := CloneURL{vcsType, cloneURL}
	owner, err := c.dir.Owner(key)
	if err == ErrNoOwner {
		// This repository doesn't exist yet. Pick a machine at random from
		// the cluster to designate the owner.
		ms := c.c.GetCluster()
		log.Printf("### etcd cluster is: %v", ms)
		if len(ms) == 1 {
			owner = ms[0]
		} else {
			owner = ms[rand.Intn(len(ms)-1)]
		}
		owner = strings.Replace(owner, ":4001", ":3000", -1)

		log.Printf("### etcd vcsclient: setting owner %q for %s %s", owner, vcsType, cloneURL)
		err := c.dir.SetOwner(key, owner)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	ownerBaseURL, err := url.Parse(owner)
	if err != nil {
		return nil, err
	}
	ownerBaseURL.Path = "/api/vcs/"

	log.Printf("### etcd vcsclient: owner is %q for %s %s", owner, vcsType, cloneURL)

	vc := vcsclient.New(ownerBaseURL, c.httpClient)
	return vc.Repository(vcsType, cloneURL)
}

package cluster

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/sourcegraph/datad"
	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sourcegraph/vcsstore"
	"github.com/sourcegraph/vcsstore/vcsclient"
)

// A Client accesses repositories distributed across a datad cluster.
type Client struct {
	// datad is the underlying datad client to use.
	datad *datad.Client

	// transport is the underlying HTTP transport to use.
	transport http.RoundTripper
}

// NewClient creates a new client to access repositories distributed in a datad
// cluster.
func NewClient(dc *datad.Client, t http.RoundTripper) *Client {
	return &Client{dc, t}
}

var _ vcsclient.RepositoryOpener = &Client{}

func (c *Client) TransportForRepository(vcsType string, cloneURL *url.URL) (http.RoundTripper, error) {
	key := vcsstore.EncodeRepositoryPath(vcsType, cloneURL)
	return c.datad.TransportForKey(key, c.transport)
}

// Repository implements vcsclient.RepositoryOpener.
func (c *Client) Repository(vcsType string, cloneURL *url.URL) (vcs.Repository, error) {
	repo, err := c.Clone(vcsType, cloneURL)
	if err != nil {
		return nil, err
	}

	if repo, ok := repo.(vcs.Repository); ok {
		return repo, nil
	}

	return nil, errors.New("repository does not support this operation")
}

// Open implements vcsstore.Service and opens a repository. If the repository
// does not exist in the cluster, an os.ErrNotExist-satisfying error is
// returned.
func (c *Client) Open(vcsType string, cloneURL *url.URL) (interface{}, error) {
	key := vcsstore.EncodeRepositoryPath(vcsType, cloneURL)

	t, err := c.TransportForRepository(vcsType, cloneURL)
	if err != nil {
		return nil, err
	}

	vc := vcsclient.New(nil, &http.Client{Transport: t})
	repo, err := vc.Repository(vcsType, cloneURL)
	if err != nil {
		return nil, err
	}
	return &repository{c.datad, key, repo}, nil
}

// Clone implements vcsstore.Service and clones a repository.
func (c *Client) Clone(vcsType string, cloneURL *url.URL) (interface{}, error) {
	key := vcsstore.EncodeRepositoryPath(vcsType, cloneURL)

	_, err := c.datad.Update(key)
	if err != nil {
		return nil, err
	}

	// TODO(sqs): add option for waiting for clone (triggered by Update) to
	// complete?

	return c.Open(vcsType, cloneURL)
}

var (
	_ vcsclient.RepositoryOpener = &Client{}
	_ vcsstore.Service           = &Client{}
)

// repository wraps a vcsclient.repository to make CloneRemote also add the
// repository key to the datad registry.
type repository struct {
	datad    *datad.Client
	datadKey string
	vcs.Repository
}

func (r *repository) CloneRemote() error {
	_, err := r.datad.Update(r.datadKey)
	if err != nil {
		return nil
	}

	// TODO(sqs): make the datad transport look up new nodes in the registry if
	// the key doesn't have any nodes. otherwise, for new repositories, the
	// transport given to this `type repository` will have no nodes. it's a
	// chicken-and-the-egg problem: it won't have nodes until it's updated here.
	// so, as a hack, we are calling update for each call to
	// (*Client).Repository, but that is inefficient.

	if rrc, ok := r.Repository.(vcsclient.RepositoryRemoteCloner); ok {
		return rrc.CloneRemote()
	}

	return nil
}

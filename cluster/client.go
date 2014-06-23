package cluster

import (
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

func (c *Client) Repository(vcsType string, cloneURL *url.URL) (vcs.Repository, error) {
	key := vcsstore.EncodeRepositoryPath(vcsType, cloneURL)

	_, err := c.datad.Update(key)
	if err != nil {
		return nil, err
	}

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

var _ vcsclient.RepositoryOpener = &Client{}

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

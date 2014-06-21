package cluster

import (
	"net/http"
	"net/url"

	"github.com/sourcegraph/datad"
	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sourcegraph/vcsstore/vcsclient"
)

// A Client accesses repositories distributed across multiple machines using
// datad.
type Client struct {
	// Datad is the underlying datad client to use.
	Datad *datad.Client
}

// NewClient creates a new client to access repositories distributed in a datad
// cluster.
func NewClient(dc *datad.Client) *Client {
	return &Client{dc}
}

var _ vcsclient.RepositoryOpener = &Client{}

func (c *Client) Repository(vcsType string, cloneURL *url.URL) (vcs.Repository, error) {
	t, err := c.Datad.DataTransport(RepositoryKey(vcsType, cloneURL), nil)
	if err != nil {
		return nil, err
	}

	vc := vcsclient.New(nil, &http.Client{Transport: t})
	return vc.Repository(vcsType, cloneURL)
}

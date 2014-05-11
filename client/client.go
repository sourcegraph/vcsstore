package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/sourcegraph/go-vcs/vcs"
)

const (
	libraryVersion = "0.0.1"
	userAgent      = "vcsstore-client/" + libraryVersion
)

// A Client communicates with the vcsstore API.
type Client struct {
	// Base URL for API requests, which should have a trailing slash.
	BaseURL *url.URL

	// Router used to generate URLs for the vcsstore API.
	router *Router

	// User agent used for HTTP requests to the vcsstore API.
	UserAgent string

	// HTTP client used to communicate with the vcsstore API.
	httpClient *http.Client

	// HTTP client that is identical to httpClient except it does not follow
	// redirects.
	ignoreRedirectsHTTPClient *http.Client
}

// New returns a new vcsstore API client that communicates with an HTTP server
// at the base URL. If httpClient is nil, http.DefaultClient is used.
func New(base *url.URL, httpClient *http.Client) *Client {
	if httpClient == nil {
		cloned := *http.DefaultClient
		httpClient = &cloned
	}

	ignoreRedirectsHTTPClient := *httpClient
	ignoreRedirectsHTTPClient.CheckRedirect = func(r *http.Request, via []*http.Request) error { return errIgnoredRedirect }

	c := &Client{
		BaseURL:                   base,
		router:                    NewRouter(),
		UserAgent:                 userAgent,
		httpClient:                httpClient,
		ignoreRedirectsHTTPClient: &ignoreRedirectsHTTPClient,
	}

	return c
}

func (c *Client) Repository(vcsType string, cloneURL *url.URL) vcs.Repository {
	return &repository{
		client:   c,
		vcsType:  vcsType,
		cloneURL: cloneURL,
	}
}

// NewRequest creates an API request. A relative URL can be provided in urlStr,
// in which case it is resolved relative to the BaseURL of the Client. Relative
// URLs should always be specified without a preceding slash. If specified, the
// value pointed to by body is JSON encoded and included as the request body.
func (c *Client) NewRequest(method, urlStr string) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	u := c.BaseURL.ResolveReference(rel)

	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", c.UserAgent)
	return req, nil
}

// Do sends an API request and returns the API response.  The API response is
// decoded and stored in the value pointed to by v, or returned as an error if
// an API error has occurred.
func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = CheckResponse(resp, false)
	if err != nil {
		// even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		return resp, err
	}

	if v != nil {
		if bp, ok := v.(*[]byte); ok {
			*bp, err = ioutil.ReadAll(resp.Body)
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("error reading response from %s %s: %s", req.Method, req.URL.RequestURI(), err)
	}
	return resp, nil
}

// doIgnoringRedirects sends an API request and returns the HTTP response. If
// it encounters an HTTP redirect, it does not follow it.
func (c *Client) doIgnoringRedirects(req *http.Request) (*http.Response, error) {
	resp, err := c.ignoreRedirectsHTTPClient.Do(req)
	if err != nil && !isIgnoredRedirectErr(err) {
		return nil, err
	}
	defer resp.Body.Close()

	return resp, CheckResponse(resp, true)
}

var errIgnoredRedirect = errors.New("not following redirect")

func isIgnoredRedirectErr(err error) bool {
	if err, ok := err.(*url.Error); ok && err.Err == errIgnoredRedirect {
		return true
	}
	return false
}

type RepositoryOpener interface {
	Repository(vcsType string, cloneURL *url.URL) vcs.Repository
}

type MockRepositoryOpener struct{ Return vcs.Repository }

var _ RepositoryOpener = MockRepositoryOpener{}

func (m MockRepositoryOpener) Repository(vcsType string, cloneURL *url.URL) vcs.Repository {
	return m.Return
}

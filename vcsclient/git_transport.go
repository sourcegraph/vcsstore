package vcsclient

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	"sourcegraph.com/sourcegraph/vcsstore/git"
)

type gitTransport struct {
	client   *Client
	cloneURL *url.URL
}

var _ git.GitTransport = &gitTransport{}

func (t *gitTransport) InfoRefs(w io.Writer, service string) error {
	rp := &repository{client: t.client, vcsType: "git", cloneURL: t.cloneURL}
	urlQuery := struct {
		Service string `url:"service"`
	}{
		Service: service,
	}
	u, err := rp.url(git.RouteGitInfoRefs, nil, urlQuery)
	if err != nil {
		return err
	}
	u = t.client.BaseURL.ResolveReference(u)

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	_, err = t.client.Do(req, &out)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, &out)
	if err != nil {
		return err
	}
	return nil
}

func (t *gitTransport) ReceivePack(w io.Writer, rdr io.Reader, opt git.GitTransportOpt) error {
	rp := &repository{client: t.client, vcsType: "git", cloneURL: t.cloneURL}
	u, err := rp.url(git.RouteGitReceivePack, nil, nil)
	if err != nil {
		return err
	}
	u = t.client.BaseURL.ResolveReference(u)

	req, err := http.NewRequest("POST", u.String(), rdr)
	if err != nil {
		return err
	}
	req.Header.Set("content-encoding", opt.ContentEncoding)

	var out bytes.Buffer
	_, err = t.client.Do(req, &out)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, &out)

	return nil
}

func (t *gitTransport) UploadPack(w io.Writer, rdr io.Reader, opt git.GitTransportOpt) error {
	rp := &repository{client: t.client, vcsType: "git", cloneURL: t.cloneURL}
	u, err := rp.url(git.RouteGitUploadPack, nil, nil)
	if err != nil {
		return err
	}
	u = t.client.BaseURL.ResolveReference(u)

	req, err := http.NewRequest("POST", u.String(), rdr)
	if err != nil {
		return err
	}
	req.Header.Set("content-encoding", opt.ContentEncoding)

	var out bytes.Buffer
	_, err = t.client.Do(req, &out)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, &out)

	return nil
}

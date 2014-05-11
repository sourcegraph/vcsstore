package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/sourcegraph/go-vcs/vcs"
	muxpkg "github.com/sqs/mux"
)

type repository struct {
	client   *Client
	vcsType  string
	cloneURL *url.URL
}

func (r *repository) ResolveBranch(name string) (vcs.CommitID, error) {
	url, err := r.url(RouteRepoBranch, map[string]string{"Branch": name})
	if err != nil {
		return "", err
	}

	req, err := r.client.NewRequest("GET", url.String())
	if err != nil {
		return "", err
	}

	resp, err := r.client.doIgnoringRedirects(req)
	if err != nil {
		return "", err
	}

	return r.parseCommitIDInURL(resp.Header.Get("location"))
}

func (r *repository) ResolveRevision(spec string) (vcs.CommitID, error) {
	url, err := r.url(RouteRepoRevision, map[string]string{"RevSpec": spec})
	if err != nil {
		return "", err
	}

	req, err := r.client.NewRequest("GET", url.String())
	if err != nil {
		return "", err
	}

	resp, err := r.client.doIgnoringRedirects(req)
	if err != nil {
		return "", err
	}

	return r.parseCommitIDInURL(resp.Header.Get("location"))
}

func (r *repository) ResolveTag(name string) (vcs.CommitID, error) {
	url, err := r.url(RouteRepoTag, map[string]string{"Tag": name})
	if err != nil {
		return "", err
	}

	req, err := r.client.NewRequest("GET", url.String())
	if err != nil {
		return "", err
	}

	resp, err := r.client.doIgnoringRedirects(req)
	if err != nil {
		return "", err
	}

	return r.parseCommitIDInURL(resp.Header.Get("location"))
}

func (r *repository) parseCommitIDInURL(urlStr string) (vcs.CommitID, error) {
	url, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	var info muxpkg.RouteMatch
	match := (*muxpkg.Router)(r.client.router).Match(&http.Request{Method: "GET", URL: url}, &info)
	if !match || info.Vars["CommitID"] == "" {
		return "", errors.New("failed to determine CommitID from URL")
	}

	return vcs.CommitID(info.Vars["CommitID"]), nil
}

func (r *repository) GetCommit(id vcs.CommitID) (*vcs.Commit, error) {
	url, err := r.url(RouteRepoCommit, map[string]string{"CommitID": string(id)})
	if err != nil {
		return nil, err
	}

	req, err := r.client.NewRequest("GET", url.String())
	if err != nil {
		return nil, err
	}

	var commit *vcs.Commit
	_, err = r.client.Do(req, &commit)
	if err != nil {
		return nil, err
	}

	return commit, nil
}

func (r *repository) CommitLog(to vcs.CommitID) ([]*vcs.Commit, error) {
	url, err := r.url(RouteRepoCommitLog, map[string]string{"CommitID": string(to)})
	if err != nil {
		return nil, err
	}

	req, err := r.client.NewRequest("GET", url.String())
	if err != nil {
		return nil, err
	}

	var commits []*vcs.Commit
	_, err = r.client.Do(req, &commits)
	if err != nil {
		return nil, err
	}

	return commits, nil
}

func (r *repository) FileSystem(at vcs.CommitID) (vcs.FileSystem, error) {
	return nil, nil
}

// url generates the URL to the named vcsstore API endpoint, using the
// specified route variables and query options.
func (r *repository) url(routeName string, routeVars map[string]string) (*url.URL, error) {
	route := (*muxpkg.Router)(r.client.router).Get(routeName)
	if route == nil {
		return nil, fmt.Errorf("no API route named %q", route)
	}

	routeVarsList := make([]string, 2*len(routeVars))
	i := 0
	for name, val := range routeVars {
		routeVarsList[i*2] = name
		routeVarsList[i*2+1] = val
		i++
	}
	routeVarsList = append(routeVarsList, "CloneURL", r.cloneURL.String(), "VCS", r.vcsType)
	url, err := route.URL(routeVarsList...)
	if err != nil {
		return nil, err
	}

	// make the route URL path relative to BaseURL by trimming the leading "/"
	url.Path = strings.TrimPrefix(url.Path, "/")

	return url, nil
}

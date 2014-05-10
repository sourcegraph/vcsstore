package client

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/kr/pretty"
	"github.com/sqs/mux"
)

func TestMatch(t *testing.T) {
	router := (*mux.Router)(NewRouter())

	const (
		cloneURL        = "git://example.com/my/repo.git"
		cloneURLEscaped = "git%3A$$example.com$my$repo.git"
	)

	tests := []struct {
		path          string
		wantNoMatch   bool
		wantRouteName string
		wantVars      map[string]string
		wantPath      string
	}{
		// Root
		{
			path:          "/",
			wantRouteName: RouteRoot,
		},

		// Repo
		{
			path:          "/repos/git/" + cloneURLEscaped,
			wantRouteName: RouteRepo,
			wantVars:      map[string]string{"VCS": "git", "CloneURL": cloneURL},
		},

		// Repo revisions
		{
			path:          "/repos/git/" + cloneURLEscaped + "/branches/mybranch",
			wantRouteName: RouteRepoBranch,
			wantVars:      map[string]string{"VCS": "git", "CloneURL": cloneURL, "Branch": "mybranch"},
		},
		{
			path:          "/repos/git/" + cloneURLEscaped + "/tags/mytag",
			wantRouteName: RouteRepoTag,
			wantVars:      map[string]string{"VCS": "git", "CloneURL": cloneURL, "Tag": "mytag"},
		},
		{
			path:          "/repos/git/" + cloneURLEscaped + "/revs/myrevspec",
			wantRouteName: RouteRepoRevision,
			wantVars:      map[string]string{"VCS": "git", "CloneURL": cloneURL, "RevSpec": "myrevspec"},
		},
		{
			path:          "/repos/git/" + cloneURLEscaped + "/commits/mycommitid",
			wantRouteName: RouteRepoCommit,
			wantVars:      map[string]string{"VCS": "git", "CloneURL": cloneURL, "CommitID": "mycommitid"},
		},

		// Repo commit log
		{
			path:          "/repos/git/" + cloneURLEscaped + "/commits/mycommitid/log",
			wantRouteName: RouteRepoCommitLog,
			wantVars:      map[string]string{"VCS": "git", "CloneURL": cloneURL, "CommitID": "mycommitid"},
		},

		// Repo tree
		{
			path:          "/repos/git/" + cloneURLEscaped + "/commits/mycommitid/tree",
			wantRouteName: RouteRepoTreeEntry,
			wantVars:      map[string]string{"VCS": "git", "CloneURL": cloneURL, "CommitID": "mycommitid", "Path": "."},
		},
		{
			path:          "/repos/git/" + cloneURLEscaped + "/commits/mycommitid/tree/",
			wantRouteName: RouteRepoTreeEntry,
			wantVars:      map[string]string{"VCS": "git", "CloneURL": cloneURL, "CommitID": "mycommitid", "Path": "."},
			wantPath:      "/repos/git/" + cloneURLEscaped + "/commits/mycommitid/tree",
		},
		{
			path:          "/repos/git/" + cloneURLEscaped + "/commits/mycommitid/tree/a/b",
			wantRouteName: RouteRepoTreeEntry,
			wantVars:      map[string]string{"VCS": "git", "CloneURL": cloneURL, "CommitID": "mycommitid", "Path": "a/b"},
		},
		{
			path:          "/repos/git/" + cloneURLEscaped + "/commits/mycommitid/tree/a/b/",
			wantRouteName: RouteRepoTreeEntry,
			wantVars:      map[string]string{"VCS": "git", "CloneURL": cloneURL, "CommitID": "mycommitid", "Path": "a/b"},
			wantPath:      "/repos/git/" + cloneURLEscaped + "/commits/mycommitid/tree/a/b",
		},
	}

	for _, test := range tests {
		var routeMatch mux.RouteMatch
		match := router.Match(&http.Request{Method: "GET", URL: &url.URL{Path: test.path}}, &routeMatch)

		if match && test.wantNoMatch {
			t.Errorf("%s: got match (route %q), want no match", test.path, routeMatch.Route.GetName())
		}
		if !match && !test.wantNoMatch {
			t.Errorf("%s: got no match, wanted match", test.path)
		}
		if !match || test.wantNoMatch {
			continue
		}

		if routeName := routeMatch.Route.GetName(); routeName != test.wantRouteName {
			t.Errorf("%s: got matched route %q, want %q", test.path, routeName, test.wantRouteName)
		}

		if diff := pretty.Diff(routeMatch.Vars, test.wantVars); len(diff) > 0 {
			t.Errorf("%s: vars don't match expected:\n%s", test.path, strings.Join(diff, "\n"))
		}

		// Check that building the URL yields the original path.
		var pairs []string
		for k, v := range test.wantVars {
			pairs = append(pairs, k, v)
		}
		path, err := routeMatch.Route.URLPath(pairs...)
		if err != nil {
			t.Errorf("%s: URLPath(%v) failed: %s", test.path, pairs, err)
			continue
		}
		var wantPath string
		if test.wantPath != "" {
			wantPath = test.wantPath
		} else {
			wantPath = test.path
		}
		if path.Path != wantPath {
			t.Errorf("got generated path %q, want %q", path, wantPath)
		}
	}
}

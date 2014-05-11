package vcsclient

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/go-vcs/vcs"
	muxpkg "github.com/sqs/mux"
)

const (
	// Route names
	RouteRepo          = "repo"
	RouteRepoBranch    = "repo.branch"
	RouteRepoCommit    = "repo.commit"
	RouteRepoCommitLog = "repo.commit.log"
	RouteRepoRevision  = "repo.rev"
	RouteRepoTag       = "repo.tag"
	RouteRepoTreeEntry = "repo.tree-entry"
	RouteRepoUpdate    = "repo.update"
	RouteRoot          = "root"
)

type Router muxpkg.Router

// NewRouter creates a new router that matches and generates URLs that the HTTP
// handler recognizes.
func NewRouter() *Router {
	r := muxpkg.NewRouter()
	r.StrictSlash(true)

	r.Path("/").Methods("GET").Name(RouteRoot)

	unescapeRepoVars := func(req *http.Request, match *muxpkg.RouteMatch, r *muxpkg.Route) {
		esc := strings.Replace(match.Vars["CloneURLEscaped"], "$", "%2F", -1)
		match.Vars["CloneURL"], _ = url.QueryUnescape(esc)
		delete(match.Vars, "CloneURLEscaped")
	}
	escapeRepoVars := func(vars map[string]string) map[string]string {
		esc := url.QueryEscape(vars["CloneURL"])
		vars["CloneURLEscaped"] = strings.Replace(esc, "%2F", "$", -1)
		delete(vars, "CloneURL")
		return vars
	}

	repoPath := "/repos/{VCS}/{CloneURLEscaped:[^/]+}"
	r.Path(repoPath).Methods("GET").PostMatchFunc(unescapeRepoVars).BuildVarsFunc(escapeRepoVars).Name(RouteRepo)
	repo := r.PathPrefix(repoPath).PostMatchFunc(unescapeRepoVars).BuildVarsFunc(escapeRepoVars).Subrouter()
	repo.Path("/update").Methods("PUT").Name(RouteRepoUpdate)
	repo.Path("/branches/{Branch}").Methods("GET").Name(RouteRepoBranch)
	repo.Path("/revs/{RevSpec}").Methods("GET").Name(RouteRepoRevision)
	repo.Path("/tags/{Tag}").Methods("GET").Name(RouteRepoTag)
	commitPath := "/commits/{CommitID}"
	repo.Path(commitPath).Methods("GET").Name(RouteRepoCommit)
	commit := repo.PathPrefix(commitPath).Subrouter()
	commit.Path("/log").Methods("GET").Name(RouteRepoCommitLog)

	// cleanTreeVars modifies the Path route var to be a clean filepath. If it
	// is empty, it is changed to ".".
	cleanTreeVars := func(req *http.Request, match *muxpkg.RouteMatch, r *muxpkg.Route) {
		path := filepath.Clean(strings.TrimPrefix(match.Vars["Path"], "/"))
		if path == "" || path == "." {
			match.Vars["Path"] = "."
		} else {
			match.Vars["Path"] = path
		}
	}
	// prepareTreeVars prepares the Path route var to generate a clean URL.
	prepareTreeVars := func(vars map[string]string) map[string]string {
		if path := vars["Path"]; path == "." {
			vars["Path"] = ""
		} else {
			vars["Path"] = "/" + filepath.Clean(path)
		}
		return vars
	}
	commit.Path("/tree{Path:(?:/.*)*}").Methods("GET").PostMatchFunc(cleanTreeVars).BuildVarsFunc(prepareTreeVars).Name(RouteRepoTreeEntry)

	return (*Router)(r)
}

func (r *Router) URLToRepo(vcsType string, cloneURL *url.URL) *url.URL {
	return r.URLTo(RouteRepo, "VCS", vcsType, "CloneURL", cloneURL.String())
}

func (r *Router) URLToRepoUpdate(vcsType string, cloneURL *url.URL) *url.URL {
	return r.URLTo(RouteRepoUpdate, "VCS", vcsType, "CloneURL", cloneURL.String())
}

func (r *Router) URLToRepoBranch(vcsType string, cloneURL *url.URL, branch string) *url.URL {
	return r.URLTo(RouteRepoBranch, "VCS", vcsType, "CloneURL", cloneURL.String(), "Branch", branch)
}

func (r *Router) URLToRepoRevision(vcsType string, cloneURL *url.URL, revSpec string) *url.URL {
	return r.URLTo(RouteRepoRevision, "VCS", vcsType, "CloneURL", cloneURL.String(), "RevSpec", revSpec)
}

func (r *Router) URLToRepoTag(vcsType string, cloneURL *url.URL, tag string) *url.URL {
	return r.URLTo(RouteRepoTag, "VCS", vcsType, "CloneURL", cloneURL.String(), "Tag", tag)
}

func (r *Router) URLToRepoCommit(vcsType string, cloneURL *url.URL, commitID vcs.CommitID) *url.URL {
	return r.URLTo(RouteRepoCommit, "VCS", vcsType, "CloneURL", cloneURL.String(), "CommitID", string(commitID))
}

func (r *Router) URLToRepoCommitLog(vcsType string, cloneURL *url.URL, commitID vcs.CommitID) *url.URL {
	return r.URLTo(RouteRepoCommitLog, "VCS", vcsType, "CloneURL", cloneURL.String(), "CommitID", string(commitID))
}

func (r *Router) URLToRepoTreeEntry(vcsType string, cloneURL *url.URL, commitID vcs.CommitID, path string) *url.URL {
	return r.URLTo(RouteRepoTreeEntry, "VCS", vcsType, "CloneURL", cloneURL.String(), "CommitID", string(commitID), "Path", path)
}

func (r *Router) URLTo(route string, vars ...string) *url.URL {
	url, err := (*muxpkg.Router)(r).Get(route).URL(vars...)
	if err != nil {
		panic(err.Error())
	}
	return url
}

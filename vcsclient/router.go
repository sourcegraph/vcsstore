package vcsclient

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/go-vcs/vcs"
	muxpkg "github.com/sqs/mux"
)

const (
	// Route names
	RouteRepo          = "vcsstore:repo"
	RouteRepoBranch    = "vcsstore:repo.branch"
	RouteRepoCommit    = "vcsstore:repo.commit"
	RouteRepoCommitLog = "vcsstore:repo.commit.log"
	RouteRepoRevision  = "vcsstore:repo.rev"
	RouteRepoTag       = "vcsstore:repo.tag"
	RouteRepoTreeEntry = "vcsstore:repo.tree-entry"
	RouteRepoUpdate    = "vcsstore:repo.update"
	RouteRoot          = "vcsstore:root"
)

type Router muxpkg.Router

// NewRouter creates a new router that matches and generates URLs that the HTTP
// handler recognizes.
func NewRouter(parent *muxpkg.Router) *Router {
	if parent == nil {
		parent = muxpkg.NewRouter()
	}

	parent.Path("/").Methods("GET").Name(RouteRoot)

	// Encode the repository clone URL as its base64.URLEncoding-encoded string.
	// Add the repository URI after it (as a convenience) so that it's possible
	// to tell which repository is being requested by inspecting the URL, but
	// don't heed this friendly URI when decoding.
	unescapeRepoVars := func(req *http.Request, match *muxpkg.RouteMatch, r *muxpkg.Route) {
		s := match.Vars["CloneURLEscaped"]
		delete(match.Vars, "CloneURLEscaped")
		i := strings.Index(s, "!")
		if i == -1 {
			return
		}
		urlBytes, _ := base64.URLEncoding.DecodeString(s[:i])
		match.Vars["CloneURL"] = string(urlBytes)
	}
	escapeRepoVars := func(vars map[string]string) map[string]string {
		enc := base64.URLEncoding.EncodeToString([]byte(vars["CloneURL"]))
		vars["CloneURLEscaped"] = enc + "!" + strings.Map(func(c rune) rune {
			if c == '/' {
				return '_'
			}
			if c == '.' || c == '_' || (c >= '0' && c <= 'z') {
				return c
			}
			return '-'
		}, vars["CloneURL"])
		delete(vars, "CloneURL")
		return vars
	}

	repoPath := "/repos/{VCS}/{CloneURLEscaped:[^/]+}"
	parent.Path(repoPath).Methods("GET").PostMatchFunc(unescapeRepoVars).BuildVarsFunc(escapeRepoVars).Name(RouteRepo)
	repo := parent.PathPrefix(repoPath).PostMatchFunc(unescapeRepoVars).BuildVarsFunc(escapeRepoVars).Subrouter()
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

	return (*Router)(parent)
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

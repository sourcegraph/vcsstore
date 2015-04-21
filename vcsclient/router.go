package vcsclient

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/google/go-querystring/query"
	muxpkg "github.com/sourcegraph/mux"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/vcsstore/git"
)

const (
	// Route names
	RouteRepo                   = "vcs:repo"
	RouteRepoBlameFile          = "vcs:repo.blame-file"
	RouteRepoBranch             = "vcs:repo.branch"
	RouteRepoBranches           = "vcs:repo.branches"
	RouteRepoCommit             = "vcs:repo.commit"
	RouteRepoCommits            = "vcs:repo.commits"
	RouteRepoCreateOrUpdate     = "vcs:repo.create-or-update"
	RouteRepoDiff               = "vcs:repo.diff"
	RouteRepoCrossRepoDiff      = "vcs:repo.cross-repo-diff"
	RouteRepoMergeBase          = "vcs:repo.merge-base"
	RouteRepoCrossRepoMergeBase = "vcs:repo.cross-repo-merge-base"
	RouteRepoRevision           = "vcs:repo.rev"
	RouteRepoSearch             = "vcs:repo.search"
	RouteRepoTag                = "vcs:repo.tag"
	RouteRepoTags               = "vcs:repo.tags"
	RouteRepoTreeEntry          = "vcs:repo.tree-entry"
	RouteRoot                   = "vcs:root"
)

type Router muxpkg.Router

// NewRouter creates a new router that matches and generates URLs that the HTTP
// handler recognizes.
func NewRouter(parent *muxpkg.Router) *Router {
	if parent == nil {
		parent = muxpkg.NewRouter()
	}

	parent.Path("/").Methods("GET").Name(RouteRoot)

	const repoURIPattern = "(?:[^./][^/]*)(?:/[^./][^/]*){1,}"

	repoPath := "/{RepoID:" + repoURIPattern + "}"
	parent.Path(repoPath).Methods("GET").Name(RouteRepo)
	parent.Path(repoPath).Methods("POST").Name(RouteRepoCreateOrUpdate)

	repo := parent.PathPrefix(repoPath).Subrouter()

	// attach git transport endpoints
	repoGit := repo.PathPrefix("/.git").Subrouter()
	git.NewRouter(repoGit)

	repo.Path("/.blame/{Path:.+}").Methods("GET").Name(RouteRepoBlameFile)
	repo.Path("/.diff/{Base}..{Head}").Methods("GET").Name(RouteRepoDiff)
	repo.Path("/.cross-repo-diff/{Base}..{HeadRepoID:" + repoURIPattern + "}:{Head}").Methods("GET").Name(RouteRepoCrossRepoDiff)
	repo.Path("/.branches").Methods("GET").Name(RouteRepoBranches)
	repo.Path("/.branches/{Branch:.+}").Methods("GET").Name(RouteRepoBranch)
	repo.Path("/.revs/{RevSpec:.+}").Methods("GET").Name(RouteRepoRevision)
	repo.Path("/.tags").Methods("GET").Name(RouteRepoTags)
	repo.Path("/.tags/{Tag:.+}").Methods("GET").Name(RouteRepoTag)
	repo.Path("/.merge-base/{CommitIDA}/{CommitIDB}").Methods("GET").Name(RouteRepoMergeBase)
	repo.Path("/.cross-repo-merge-base/{CommitIDA}/{BRepoID:" + repoURIPattern + "}/{CommitIDB}").Methods("GET").Name(RouteRepoCrossRepoMergeBase)
	repo.Path("/.commits").Methods("GET").Name(RouteRepoCommits)
	commitPath := "/.commits/{CommitID}"
	repo.Path(commitPath).Methods("GET").Name(RouteRepoCommit)
	commit := repo.PathPrefix(commitPath).Subrouter()

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
	commit.Path("/search").Methods("GET").Name(RouteRepoSearch)

	return (*Router)(parent)
}

func (r *Router) URLToRepo(repoID string) *url.URL {
	return r.URLTo(RouteRepo, "RepoID", repoID)
}

func (r *Router) URLToRepoBlameFile(repoID string, path string, opt *vcs.BlameOptions) *url.URL {
	u := r.URLTo(RouteRepoBlameFile, "RepoID", repoID, "Path", path)
	if opt != nil {
		q, err := query.Values(opt)
		if err != nil {
			panic(err.Error())
		}
		u.RawQuery = q.Encode()
	}
	return u
}

func (r *Router) URLToRepoDiff(repoID string, base, head vcs.CommitID, opt *vcs.DiffOptions) *url.URL {
	u := r.URLTo(RouteRepoDiff, "RepoID", repoID, "Base", string(base), "Head", string(head))
	if opt != nil {
		q, err := query.Values(opt)
		if err != nil {
			panic(err.Error())
		}
		u.RawQuery = q.Encode()
	}
	return u
}

func (r *Router) URLToRepoCrossRepoDiff(baseRepoID string, base vcs.CommitID, headRepoID string, head vcs.CommitID, opt *vcs.DiffOptions) *url.URL {
	u := r.URLTo(RouteRepoCrossRepoDiff, "RepoID", baseRepoID, "Base", string(base), "HeadRepoID", headRepoID, "Head", string(head))
	if opt != nil {
		q, err := query.Values(opt)
		if err != nil {
			panic(err.Error())
		}
		u.RawQuery = q.Encode()
	}
	return u
}

func (r *Router) URLToRepoBranch(repoID string, branch string) *url.URL {
	return r.URLTo(RouteRepoBranch, "RepoID", repoID, "Branch", branch)
}

func (r *Router) URLToRepoBranches(repoID string, opt vcs.BranchesOptions) *url.URL {
	u := r.URLTo(RouteRepoBranches, "RepoID", repoID)
	q, err := query.Values(opt)
	if err != nil {
		panic(err.Error())
	}
	u.RawQuery = q.Encode()
	return u
}

func (r *Router) URLToRepoRevision(repoID string, revSpec string) *url.URL {
	return r.URLTo(RouteRepoRevision, "RepoID", repoID, "RevSpec", revSpec)
}

func (r *Router) URLToRepoTag(repoID string, tag string) *url.URL {
	return r.URLTo(RouteRepoTag, "RepoID", repoID, "Tag", tag)
}

func (r *Router) URLToRepoTags(repoID string) *url.URL {
	return r.URLTo(RouteRepoTags, "RepoID", repoID)
}

func (r *Router) URLToRepoCommit(repoID string, commitID vcs.CommitID) *url.URL {
	return r.URLTo(RouteRepoCommit, "RepoID", repoID, "CommitID", string(commitID))
}

func (r *Router) URLToRepoCommits(repoID string, opt vcs.CommitsOptions) *url.URL {
	u := r.URLTo(RouteRepoCommits, "RepoID", repoID)
	q, err := query.Values(opt)
	if err != nil {
		panic(err.Error())
	}
	u.RawQuery = q.Encode()
	return u
}

func (r *Router) URLToRepoTreeEntry(repoID string, commitID vcs.CommitID, path string) *url.URL {
	return r.URLTo(RouteRepoTreeEntry, "RepoID", repoID, "CommitID", string(commitID), "Path", path)
}

func (r *Router) URLToRepoSearch(repoID string, at vcs.CommitID, opt vcs.SearchOptions) *url.URL {
	u := r.URLTo(RouteRepoSearch, "RepoID", repoID, "CommitID", string(at))
	q, err := query.Values(opt)
	if err != nil {
		panic(err.Error())
	}
	u.RawQuery = q.Encode()
	return u
}

func (r *Router) URLToRepoMergeBase(repoID string, a, b vcs.CommitID) *url.URL {
	return r.URLTo(RouteRepoMergeBase, "RepoID", repoID, "CommitIDA", string(a), "CommitIDB", string(b))
}

func (r *Router) URLToRepoCrossRepoMergeBase(repoID string, a vcs.CommitID, bRepoID string, b vcs.CommitID) *url.URL {
	return r.URLTo(RouteRepoCrossRepoMergeBase, "RepoID", repoID, "CommitIDA", string(a), "BRepoID", bRepoID, "CommitIDB", string(b))
}

func (r *Router) URLTo(route string, vars ...string) *url.URL {
	url, err := (*muxpkg.Router)(r).Get(route).URL(vars...)
	if err != nil {
		panic(err.Error())
	}
	return url
}

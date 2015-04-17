package git

import (
	"net/http"
	"strings"

	"github.com/sourcegraph/mux"
)

const (
	RouteGitInfoRefs    = "git.info-refs"
	RouteGitUploadPack  = "git.upload-pack"
	RouteGitReceivePack = "git.receive-pack"
)

// New creates a new Git HTTP router.
func NewRouter(base *mux.Router) *mux.Router {
	if base == nil {
		base = mux.NewRouter()
	}

	var gitMatcher mux.MatcherFunc = func(req *http.Request, rt *mux.RouteMatch) bool {
		userAgent := req.Header.Get("User-Agent")
		if strings.HasPrefix(strings.ToLower(userAgent), "git/") {
			return true
		}
		return false
	}

	gm := base.MatcherFunc(gitMatcher).Subrouter()
	gm.Path("/{URI:(?:.*)}/info/refs").Methods("GET").Name(RouteGitInfoRefs)
	gm.Path("/{URI:(?:.*)}/git-upload-pack").Methods("POST").Name(RouteGitUploadPack)
	gm.Path("/{URI:(?:.*)}/git-receive-pack").Methods("POST").Name(RouteGitReceivePack)

	return base
}

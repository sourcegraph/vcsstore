package vcsclient

import (
	"net/http"
	"net/url"
	"testing"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

func TestRepository_MergeBase(t *testing.T) {
	setup()
	defer teardown()

	cloneURL, _ := url.Parse("git://a.b/c")
	repo_, _ := vcsclient.Repository("git", cloneURL)
	repo := repo_.(*repository)

	want := vcs.CommitID("abcd")

	var called bool
	mux.HandleFunc(urlPath(t, RouteRepoMergeBase, repo, map[string]string{"VCS": "git", "CloneURL": cloneURL.String(), "CommitID1": "a", "CommitID2": "b"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		http.Redirect(w, r, urlPath(t, RouteRepoCommit, repo, map[string]string{"CommitID": "abcd"}), http.StatusFound)
	})

	commitID, err := repo.MergeBase("a", "b")
	if err != nil {
		t.Errorf("Repository.MergeBase returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if commitID != want {
		t.Errorf("Repository.MergeBase returned %+v, want %+v", commitID, want)
	}
}

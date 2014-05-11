package vcsclient

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/sourcegraph/go-vcs/vcs"
)

func TestRepository_ResolveBranch(t *testing.T) {
	setup()
	defer teardown()

	cloneURL, _ := url.Parse("git://a.b/c")
	repo := vcsclient.Repository("git", cloneURL).(*repository)

	want := vcs.CommitID("abcd")

	var called bool
	mux.HandleFunc(urlPath(t, RouteRepoBranch, repo, map[string]string{"VCS": "git", "CloneURL": cloneURL.String(), "Branch": "mybranch"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		http.Redirect(w, r, urlPath(t, RouteRepoCommit, repo, map[string]string{"CommitID": "abcd"}), http.StatusSeeOther)
	})

	commitID, err := repo.ResolveBranch("mybranch")
	if err != nil {
		t.Errorf("Repository.ResolveBranch returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if commitID != want {
		t.Errorf("Repository.ResolveBranch returned %+v, want %+v", commitID, want)
	}
}

func TestRepository_ResolveRevision(t *testing.T) {
	setup()
	defer teardown()

	cloneURL, _ := url.Parse("git://a.b/c")
	repo := vcsclient.Repository("git", cloneURL).(*repository)

	want := vcs.CommitID("abcd")

	var called bool
	mux.HandleFunc(urlPath(t, RouteRepoRevision, repo, map[string]string{"VCS": "git", "CloneURL": cloneURL.String(), "RevSpec": "myrevspec"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		http.Redirect(w, r, urlPath(t, RouteRepoCommit, repo, map[string]string{"CommitID": "abcd"}), http.StatusSeeOther)
	})

	commitID, err := repo.ResolveRevision("myrevspec")
	if err != nil {
		t.Errorf("Repository.ResolveRevision returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if commitID != want {
		t.Errorf("Repository.ResolveRevision returned %+v, want %+v", commitID, want)
	}
}

func TestRepository_ResolveTag(t *testing.T) {
	setup()
	defer teardown()

	cloneURL, _ := url.Parse("git://a.b/c")
	repo := vcsclient.Repository("git", cloneURL).(*repository)

	want := vcs.CommitID("abcd")

	var called bool
	mux.HandleFunc(urlPath(t, RouteRepoTag, repo, map[string]string{"VCS": "git", "CloneURL": cloneURL.String(), "Tag": "mytag"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		http.Redirect(w, r, urlPath(t, RouteRepoCommit, repo, map[string]string{"CommitID": "abcd"}), http.StatusSeeOther)
	})

	commitID, err := repo.ResolveTag("mytag")
	if err != nil {
		t.Errorf("Repository.ResolveTag returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if commitID != want {
		t.Errorf("Repository.ResolveTag returned %+v, want %+v", commitID, want)
	}
}

func TestRepository_CommitLog(t *testing.T) {
	setup()
	defer teardown()

	cloneURL, _ := url.Parse("git://a.b/c")
	repo := vcsclient.Repository("git", cloneURL).(*repository)

	want := []*vcs.Commit{{ID: "abcd"}}
	normalizeTime(&want[0].Author.Date)

	var called bool
	mux.HandleFunc(urlPath(t, RouteRepoCommitLog, repo, map[string]string{"CommitID": "abcd"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	commits, err := repo.CommitLog("abcd")
	if err != nil {
		t.Errorf("Repository.CommitLog returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(commits, want) {
		t.Errorf("Repository.CommitLog returned %+v, want %+v", commits, want)
	}
}

func TestRepository_GetCommit(t *testing.T) {
	setup()
	defer teardown()

	cloneURL, _ := url.Parse("git://a.b/c")
	repo := vcsclient.Repository("git", cloneURL).(*repository)

	want := &vcs.Commit{ID: "abcd"}
	normalizeTime(&want.Author.Date)

	var called bool
	mux.HandleFunc(urlPath(t, RouteRepoCommit, repo, map[string]string{"CommitID": "abcd"}), func(w http.ResponseWriter, r *http.Request) {
		called = true
		testMethod(t, r, "GET")

		writeJSON(w, want)
	})

	commit, err := repo.GetCommit("abcd")
	if err != nil {
		t.Errorf("Repository.GetCommit returned error: %v", err)
	}

	if !called {
		t.Fatal("!called")
	}

	if !reflect.DeepEqual(commit, want) {
		t.Errorf("Repository.GetCommit returned %+v, want %+v", commit, want)
	}
}

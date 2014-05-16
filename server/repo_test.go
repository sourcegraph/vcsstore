package server

import (
	"net/http"
	"net/url"
	"os"
	"testing"
)

func TestServeRepo(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	sm := &mockServiceForExistingRepo{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
	}
	Service = sm

	resp, err := http.Get(server.URL + router.URLToRepo("git", cloneURL).String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if !sm.opened {
		t.Errorf("!opened")
	}
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("got code %d, want %d", got, want)
		logResponseBody(t, resp)
	}
}

func TestServeRepoCreateOrUpdate_CreateNew(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	rm := struct{}{} // trivial mock repository
	var calledOpen, calledClone bool
	sm := &mockService{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
		open: func(vcs string, cloneURL *url.URL) (interface{}, error) {
			// Simulate that the repository doesn't exist locally.
			calledOpen = true
			return nil, os.ErrNotExist
		},
		clone: func(vcs string, cloneURL *url.URL) (interface{}, error) {
			calledClone = true
			return rm, nil
		},
	}
	Service = sm

	req, err := http.NewRequest("POST", server.URL+router.URLToRepo("git", cloneURL).String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if !calledOpen {
		t.Errorf("!calledOpen")
	}
	if !calledClone {
		t.Errorf("!calledClone")
	}
	if got, want := resp.StatusCode, http.StatusCreated; got != want {
		t.Errorf("got code %d, want %d", got, want)
		logResponseBody(t, resp)
	}
}

func TestServeRepoCreateOrUpdate_UpdateExisting(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	rm := &mockMirrorUpdate{t: t}
	sm := &mockServiceForExistingRepo{
		t:        t,
		vcs:      "git",
		cloneURL: cloneURL,
		repo:     rm,
	}
	Service = sm

	req, err := http.NewRequest("POST", server.URL+router.URLToRepo("git", cloneURL).String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if !sm.opened {
		t.Errorf("!opened")
	}
	if !rm.called {
		t.Errorf("!called")
	}
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("got code %d, want %d", got, want)
		logResponseBody(t, resp)
	}
}

type mockMirrorUpdate struct {
	t *testing.T

	// return values
	err error

	called bool
}

func (m *mockMirrorUpdate) MirrorUpdate() error {
	m.called = true
	return m.err
}

type mockServiceForExistingRepo struct {
	t *testing.T

	// expected args
	vcs      string
	cloneURL *url.URL

	// return values
	repo interface{}
	err  error

	opened bool
}

func (m *mockServiceForExistingRepo) Open(vcs string, cloneURL *url.URL) (interface{}, error) {
	if vcs != m.vcs {
		m.t.Errorf("mock: got vcs arg %q, want %q", vcs, m.vcs)
	}
	if cloneURL.String() != m.cloneURL.String() {
		m.t.Errorf("mock: got cloneURL arg %q, want %q", cloneURL, m.cloneURL)
	}
	m.opened = true
	return m.repo, m.err
}

func (m *mockServiceForExistingRepo) Clone(vcs string, cloneURL *url.URL) (interface{}, error) {
	m.t.Errorf("mock: unexpectedly called Clone for repo that exists (%s %s)", vcs, cloneURL)
	return m.repo, m.err
}

type mockService struct {
	t *testing.T

	// expected args
	vcs      string
	cloneURL *url.URL

	// mockable methods
	open  func(vcs string, cloneURL *url.URL) (interface{}, error)
	clone func(vcs string, cloneURL *url.URL) (interface{}, error)
}

func (m *mockService) Open(vcs string, cloneURL *url.URL) (interface{}, error) {
	if vcs != m.vcs {
		m.t.Errorf("mock: got vcs arg %q, want %q", vcs, m.vcs)
	}
	if cloneURL.String() != m.cloneURL.String() {
		m.t.Errorf("mock: got cloneURL arg %q, want %q", cloneURL, m.cloneURL)
	}
	return m.open(vcs, cloneURL)
}

func (m *mockService) Clone(vcs string, cloneURL *url.URL) (interface{}, error) {
	if vcs != m.vcs {
		m.t.Errorf("mock: got vcs arg %q, want %q", vcs, m.vcs)
	}
	if cloneURL.String() != m.cloneURL.String() {
		m.t.Errorf("mock: got cloneURL arg %q, want %q", cloneURL, m.cloneURL)
	}
	return m.clone(vcs, cloneURL)
}

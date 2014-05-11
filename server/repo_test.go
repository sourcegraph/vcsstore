package server

import (
	"net/http"
	"net/url"
	"testing"
)

func TestServeRepo(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	cloneURL, _ := url.Parse("git://a.b/c")
	sm := &mockService{
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

type mockService struct {
	t *testing.T

	// expected args
	vcs      string
	cloneURL *url.URL

	// return values
	repo interface{}
	err  error

	opened bool
}

func (m *mockService) Open(vcs string, cloneURL *url.URL) (interface{}, error) {
	if vcs != m.vcs {
		m.t.Errorf("mock: got vcs arg %q, want %q", vcs, m.vcs)
	}
	if cloneURL.String() != m.cloneURL.String() {
		m.t.Errorf("mock: got cloneURL arg %q, want %q", cloneURL, m.cloneURL)
	}
	m.opened = true
	return m.repo, m.err
}

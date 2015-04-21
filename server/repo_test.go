package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/vcsstore"
	"sourcegraph.com/sourcegraph/vcsstore/vcsclient"
)

func TestServeRepo(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	repoID := "a.b/c"
	sm := &mockServiceForExistingRepo{
		t: t,

		repoID: repoID,
	}
	testHandler.Service = sm

	resp, err := http.Get(server.URL + testHandler.router.URLToRepo(repoID).String())
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

func TestServeRepo_DoesNotExist(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	repoID := "a.b/c"
	var calledOpen bool
	sm := &mockService{
		t: t,

		repoID: repoID,
		open: func(repoID string) (interface{}, error) {
			// Simulate that the repository doesn't exist locally.
			calledOpen = true
			return nil, os.ErrNotExist
		},
		clone: func(repoID string, opt *vcsclient.CloneInfo) (interface{}, error) {
			t.Fatal("unexpectedly called Clone")
			panic("unreachable")
		},
	}
	testHandler.Service = sm

	req, err := http.NewRequest("GET", server.URL+testHandler.router.URLToRepo(repoID).String(), nil)
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
	if got, want := resp.StatusCode, http.StatusNotFound; got != want {
		t.Errorf("got code %d, want %d", got, want)
		logResponseBody(t, resp)
	}
}

func TestServeRepoCreateOrUpdate_CreateNew_noBody(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	repoID := "a.b/c"
	rm := struct{}{} // trivial mock repository
	var calledOpen, calledClone bool
	sm := &mockService{
		t: t,

		repoID: repoID,
		open: func(repoID string) (interface{}, error) {
			// Simulate that the repository doesn't exist locally.
			calledOpen = true
			return nil, os.ErrNotExist
		},
		clone: func(repoID string, opt *vcsclient.CloneInfo) (interface{}, error) {
			calledClone = true
			return rm, nil
		},
	}
	testHandler.Service = sm

	req, err := http.NewRequest("POST", server.URL+testHandler.router.URLToRepo(repoID).String(), nil)
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

func TestServeRepoCreateOrUpdate_CreateNew_withBody(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	repoID := "a.b/c"
	opt := vcsclient.CloneInfo{RemoteOpts: vcs.RemoteOpts{SSH: &vcs.SSHConfig{User: "u"}}}
	rm := struct{}{} // trivial mock repository
	var calledOpen, calledClone bool
	sm := &mockService{
		t: t,

		repoID: repoID,
		opt:    opt,
		open: func(repoID string) (interface{}, error) {
			// Simulate that the repository doesn't exist locally.
			calledOpen = true
			return nil, os.ErrNotExist
		},
		clone: func(repoID string, opt *vcsclient.CloneInfo) (interface{}, error) {
			calledClone = true
			return rm, nil
		},
	}
	testHandler.Service = sm

	body, _ := json.Marshal(opt)
	req, err := http.NewRequest("POST", server.URL+testHandler.router.URLToRepo(repoID).String(), bytes.NewReader(body))
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

func TestServeRepoCreateOrUpdate_UpdateExisting_noBody(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	repoID := "a.b/c"
	rm := &mockUpdateEverythinger{t: t}
	sm := &mockServiceForExistingRepo{
		t: t,

		repoID: repoID,
		repo:   rm,
	}
	testHandler.Service = sm

	req, err := http.NewRequest("POST", server.URL+testHandler.router.URLToRepo(repoID).String(), nil)
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

func TestServeRepoCreateOrUpdate_UpdateExisting_withBody(t *testing.T) {
	setupHandlerTest()
	defer teardownHandlerTest()

	repoID := "a.b/c"
	opt := vcsclient.CloneInfo{RemoteOpts: vcs.RemoteOpts{SSH: &vcs.SSHConfig{User: "u"}}}
	rm := &mockUpdateEverythinger{t: t, opt: opt}
	sm := &mockServiceForExistingRepo{
		t: t,

		repoID: repoID,
		repo:   rm,
	}
	testHandler.Service = sm

	body, _ := json.Marshal(opt)
	req, err := http.NewRequest("POST", server.URL+testHandler.router.URLToRepo(repoID).String(), bytes.NewReader(body))
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

type mockUpdateEverythinger struct {
	t *testing.T

	// expected args
	opt vcsclient.CloneInfo

	// return values
	err error

	called bool
}

func (m *mockUpdateEverythinger) UpdateEverything(opt *vcsclient.CloneInfo) error {
	m.called = true
	if !reflect.DeepEqual(opt, &m.opt) {
		m.t.Errorf("mock: got opt %+v, want %+v", asJSON(opt), asJSON(m.opt))
	}
	return m.err
}

type mockServiceForExistingRepo struct {
	t *testing.T

	// expected args
	repoID string

	// return values
	repo interface{}
	err  error

	opened bool
}

var _ vcsstore.Service = (*mockServiceForExistingRepo)(nil)

func (m *mockServiceForExistingRepo) Open(repoID string) (interface{}, error) {
	if m.repoID != "" && repoID != m.repoID {
		m.t.Errorf("mock: got repoID arg %q, want %q", repoID, m.repoID)
	}
	m.opened = true
	return m.repo, m.err
}

func (m *mockServiceForExistingRepo) Clone(repoID string, opt *vcsclient.CloneInfo) (interface{}, error) {
	m.t.Errorf("mock: unexpectedly called Clone for repo that exists (%s)", repoID)
	return m.repo, m.err
}

func (m *mockServiceForExistingRepo) Close(repoID string) {}

type mockService struct {
	t *testing.T

	// expected args
	repoID string
	opt    vcsclient.CloneInfo

	// mockable methods
	open  func(repoID string) (interface{}, error)
	clone func(repoID string, opt *vcsclient.CloneInfo) (interface{}, error)
}

var _ vcsstore.Service = (*mockService)(nil)

func (m *mockService) Open(repoID string) (interface{}, error) {
	if m.repoID != "" && repoID != m.repoID {
		m.t.Errorf("mock: got repoID arg %q, want %q", repoID, m.repoID)
	}
	return m.open(repoID)
}

func (m *mockService) Clone(repoID string, opt *vcsclient.CloneInfo) (interface{}, error) {
	if m.repoID != "" && repoID != m.repoID {
		m.t.Errorf("mock: got repoID arg %q, want %q", repoID, m.repoID)
	}
	if !reflect.DeepEqual(opt, &m.opt) {
		m.t.Errorf("mock: got opt %+v, want %+v", asJSON(opt), asJSON(m.opt))
	}
	return m.clone(repoID, opt)
}

func (m *mockService) Close(repoID string) {}

func asJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

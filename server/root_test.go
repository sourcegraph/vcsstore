package server

import (
	"log"
	"net/http"
	"testing"
)

func TestHandler_serveRoot(t *testing.T) {
	setupHandlerTest()

	resp, err := http.Get(urlTo(RouteRoot))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("got code %d, want %d", got, want)
	}
}

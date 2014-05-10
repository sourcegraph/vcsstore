package server

import "net/http"

func serveRoot(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("vcsstore"))
	return nil
}

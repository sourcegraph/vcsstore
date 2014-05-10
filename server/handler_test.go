package server

import "net/http/httptest"

var (
	handler *Handler
	server  *httptest.Server
)

func setupHandlerTest() {
	handler = NewHandler(nil)
	server = httptest.NewServer(handler)
}

func urlTo(route string, vars ...string) string {
	return server.URL + uriTo(handler.router, route, vars...).RequestURI()
}

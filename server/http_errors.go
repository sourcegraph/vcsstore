package server

import "net/http"

type httpError struct {
	statusCode int   // HTTP status code.
	err        error // Optional reason for the HTTP error.
}

func (err httpError) Error() string {
	if err.err != nil {
		return err.err.Error()
	}
	return http.StatusText(err.statusCode)
}

func (err httpError) httpStatusCode() int { return err.statusCode }

// errorHTTPStatusCode returns the HTTP error code that most closely describes err.
func errorHTTPStatusCode(err error) int {
	type httpStatusCoder interface {
		httpStatusCode() int
	}
	if err, ok := err.(httpStatusCoder); ok {
		return err.httpStatusCode()
	}
	return http.StatusInternalServerError
}

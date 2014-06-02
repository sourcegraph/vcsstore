package server

import "net/http"

var longCacheControl = "max-age=31536000, public"

func setLongCache(w http.ResponseWriter) {
	numLongCachedResponses.Add(1)
	w.Header().Set("cache-control", longCacheControl)
}

package git

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sourcegraph/mux"
)

var RepoRootDir string

func NewHandler(base *mux.Router) *mux.Router {
	router := NewRouter(base)

	router.Get(RouteGitInfoRefs).Handler(handler(serveInfoRefs))
	router.Get(RouteGitUploadPack).Handler(handler(serveUploadPack))
	router.Get(RouteGitReceivePack).Handler(handler(serveReceivePack))

	return router
}

func handler(f func(w http.ResponseWriter, r *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusOK)
		}
	})
}

func serveInfoRefs(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)
	uri := v["URI"]
	rawService := r.URL.Query().Get("service")

	var service string
	if strings.HasPrefix(rawService, "git-") {
		service = rawService[len("git-"):]
	}

	t := NewLocalGitTransport(uriToFilePath(uri))

	var refsBuf bytes.Buffer
	err := t.InfoRefs(&refsBuf, service)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))
	w.WriteHeader(http.StatusOK)
	w.Write(packetWrite("# service=git-" + service + "\n"))
	w.Write(packetFlush())
	w.Write(refsBuf.Bytes())
	return nil
}

func serveReceivePack(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)
	uri := v["URI"]

	var opt GitTransportOpt
	opt.ContentEncoding = r.Header.Get("content-encoding")

	t := NewLocalGitTransport(uriToFilePath(uri))
	w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
	return t.ReceivePack(w, r.Body, opt)
}

func serveUploadPack(w http.ResponseWriter, r *http.Request) error {
	v := mux.Vars(r)
	uri := v["URI"]

	t := NewLocalGitTransport(uriToFilePath(uri))

	var opt GitTransportOpt
	opt.ContentEncoding = r.Header.Get("content-encoding")
	w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
	if err := t.UploadPack(w, r.Body, opt); err != nil {
		return err
	}
	return nil
}

// uriToFilePath maps a repository URI to its file path on disk.
func uriToFilePath(uri string) string {
	// TODO(beyang): this is insecure ("..")
	return filepath.Join(RepoRootDir, uri)
}

// Helpers copied from githttp
func hdrNocache(w http.ResponseWriter) {
	w.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func hdrCacheForever(w http.ResponseWriter) {
	now := time.Now().Unix()
	expires := now + 31536000
	w.Header().Set("Date", fmt.Sprintf("%d", now))
	w.Header().Set("Expires", fmt.Sprintf("%d", expires))
	w.Header().Set("Cache-Control", "public, max-age=31536000")
}

func packetFlush() []byte {
	return []byte("0000")
}

func packetWrite(str string) []byte {
	s := strconv.FormatInt(int64(len(str)+4), 16)

	if len(s)%4 != 0 {
		s = strings.Repeat("0", 4-len(s)%4) + s
	}

	return []byte(s + str)
}

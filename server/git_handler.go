package server

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sourcegraph.com/sourcegraph/vcsstore/git"
)

func (h *Handler) serveInfoRefs(w http.ResponseWriter, r *http.Request) error {
	repoID, err := h.getRepoCloneURL(r, "")
	if err != nil {
		return err
	}
	rawService := r.URL.Query().Get("service")

	var service string
	if strings.HasPrefix(rawService, "git-") {
		service = rawService[len("git-"):]
	}

	t, err := h.GitTransporter.GitTransport(repoID)
	if err != nil {
		return err
	}

	var refsBuf bytes.Buffer
	err = t.InfoRefs(&refsBuf, service)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))
	w.WriteHeader(http.StatusOK)
	w.Write(refsBuf.Bytes())
	return nil
}

func (h *Handler) serveReceivePack(w http.ResponseWriter, r *http.Request) error {
	repoID, err := h.getRepoCloneURL(r, "")
	if err != nil {
		return err
	}

	var opt git.GitTransportOpt
	opt.ContentEncoding = r.Header.Get("content-encoding")

	t, err := h.GitTransporter.GitTransport(repoID)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
	return t.ReceivePack(w, r.Body, opt)
}

func (h *Handler) serveUploadPack(w http.ResponseWriter, r *http.Request) error {
	repoID, err := h.getRepoCloneURL(r, "")
	if err != nil {
		return err
	}

	t, err := h.GitTransporter.GitTransport(repoID)
	if err != nil {
		return err
	}

	var opt git.GitTransportOpt
	opt.ContentEncoding = r.Header.Get("content-encoding")
	w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
	if err := t.UploadPack(w, r.Body, opt); err != nil {
		return err
	}
	return nil
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

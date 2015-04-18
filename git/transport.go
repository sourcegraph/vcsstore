package git

import (
	"io"
	"net/url"
)

type GitTransporter interface {
	GitTransport(vcsType string, cloneURL *url.URL) (GitTransport, error)
}

// GitTransport represents a git repository with all the functions to
// support the "smart" transfer protocol.
type GitTransport interface {
	// InfoRefs writes the output of git-info-refs to w. If this
	// function returns an error, then nothing is written to w.
	InfoRefs(w io.Writer, service string) error

	// ReceivePack writes the output of git-receive-pack to w, reading
	// from rc. If this function returns an error, then nothing is
	// written to w.
	ReceivePack(w io.Writer, rc io.ReadCloser, opt GitTransportOpt) error

	// UploadPack writes the output of git-upload-pack to w, reading
	// from rc. If this function returns an error, then nothing is
	// written to w.
	UploadPack(w io.Writer, rc io.ReadCloser, opt GitTransportOpt) error
}

type GitTransportOpt struct {
	ContentEncoding string
}

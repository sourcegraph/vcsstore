package server

import (
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"sourcegraph.com/sourcegraph/vcsstore"
	"sourcegraph.com/sourcegraph/vcsstore/git"

	githttp "github.com/sourcegraph/go-git-http"
)

func NewGitTransporter(conf *vcsstore.Config) git.GitTransporter {
	return &localGitTransporter{conf}
}

type localGitTransporter struct {
	*vcsstore.Config
}

var _ git.GitTransporter = (*localGitTransporter)(nil)

func (t *localGitTransporter) GitTransport(repoPath string) (git.GitTransport, error) {
	cloneDir, err := t.Config.CloneDir(repoPath)
	if err != nil {
		return nil, err
	}
	return &localGitTransport{dir: cloneDir}, nil
}

// localGitTransport is a git repository hosted on local disk
type localGitTransport struct {
	dir string
}

func (r *localGitTransport) InfoRefs(w io.Writer, service string) error {
	if service != "upload-pack" && service != "receive-pack" {
		return fmt.Errorf("unrecognized git service \"%s\"", service)
	}
	w.Write(packetWrite("# service=git-" + service + "\n"))
	w.Write(packetFlush())

	cmd := exec.Command("git", service, "--stateless-rpc", "--advertise-refs", ".")
	cmd.Dir = r.dir
	cmd.Stdout, cmd.Stderr = w, os.Stderr
	return cmd.Run()
}

func (r *localGitTransport) ReceivePack(w io.Writer, rdr io.Reader, opt git.GitTransportOpt) error {
	return r.servicePack("receive-pack", w, rdr, opt)
}

func (r *localGitTransport) UploadPack(w io.Writer, rdr io.Reader, opt git.GitTransportOpt) error {
	return r.servicePack("upload-pack", w, rdr, opt)
}

func (r *localGitTransport) servicePack(service string, w io.Writer, rdr io.Reader, opt git.GitTransportOpt) error {
	var err error
	switch opt.ContentEncoding {
	case "gzip":
		rdr, err = gzip.NewReader(rdr)
	case "deflate":
		rdr = flate.NewReader(rdr)
	}
	if err != nil {
		return err
	}

	rpcReader := &githttp.RpcReader{
		Reader: rdr,
		Rpc:    service,
	}

	cmd := exec.Command("git", service, "--stateless-rpc", ".")
	cmd.Dir = r.dir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	// Scan's git command's output for errors
	gitReader := &githttp.GitReader{
		Reader: stdout,
	}

	// Copy input to git binary
	io.Copy(stdin, rpcReader)

	// Write git binary's output to http response
	io.Copy(w, gitReader)

	// Wait till command has completed
	mainError := cmd.Wait()
	if mainError == nil {
		mainError = gitReader.GitError
	}
	for _, e := range rpcReader.Events {
		log.Printf("EVENT: %q\n", e)
	}
	return mainError
}

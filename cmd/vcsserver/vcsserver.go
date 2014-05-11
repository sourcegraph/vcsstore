package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/sourcegraph/vcsstore"
	"github.com/sourcegraph/vcsstore/server"
)

var (
	bindAddr   = flag.String("http", ":9090", "HTTP listen address")
	storageDir = flag.String("storage", "/tmp/vcsstore", "storage root dir for VCS repos")
	verbose    = flag.Bool("v", true, "show verbose output")
	debug      = flag.Bool("d", false, "debug mode (don't use on publicly available servers)")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "vcsserver mirrors and serves VCS repositories.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\n")
		fmt.Fprintf(os.Stderr, "\tvcsserver [options]\n\n")
		fmt.Fprintf(os.Stderr, "The options are:\n\n")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
	}

	err := os.MkdirAll(*storageDir, 0700)
	if err != nil {
		log.Fatalf("Error creating directory %q: %s.", *storageDir, err)
	}

	var logw io.Writer
	if *verbose {
		logw = os.Stderr
	} else {
		logw = ioutil.Discard
	}

	conf := &vcsstore.Config{
		StorageDir: *storageDir,
		Log:        log.New(logw, "vcsstore: ", log.LstdFlags),
	}
	server.Service = vcsstore.NewService(conf)
	server.Log = log.New(logw, "server: ", log.LstdFlags)
	server.InformativeErrors = *debug

	http.Handle("/", server.NewHandler())

	fmt.Fprintf(os.Stderr, "Starting server on %s\n", *bindAddr)
	err = http.ListenAndServe(*bindAddr, nil)
	if err != nil {
		log.Fatalf("HTTP server failed to start: %s.", err)
	}
}

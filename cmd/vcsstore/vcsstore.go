package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/sourcegraph/vcsstore"
	"github.com/sourcegraph/vcsstore/server"
	"github.com/sourcegraph/vcsstore/vcsclient"
)

var (
	storageDir = flag.String("s", "/tmp/vcsstore", "storage root dir for VCS repos")
	verbose    = flag.Bool("v", true, "show verbose output")

	defaultPort = "9090"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, `vcsstore caches and serves information about VCS repositories.

Usage:

        vcsstore [options] command [arg...]

The commands are:
`)
		for _, c := range subcommands {
			fmt.Fprintf(os.Stderr, "    %-14s %s\n", c.Name, c.Description)
		}
		fmt.Fprintln(os.Stderr, `
Use "vcsstore command -h" for more information about a command.

The global options are:
`)
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
	}

	subcmd := flag.Arg(0)
	extraArgs := flag.Args()[1:]
	for _, c := range subcommands {
		if c.Name == subcmd {
			c.Run(extraArgs)
			return
		}

	}

	fmt.Fprintf(os.Stderr, "vcsstore: unknown subcommand %q\n", subcmd)
	fmt.Fprintln(os.Stderr, `Run "vcsstore -h" for usage.`)
	os.Exit(1)
}

type subcommand struct {
	Name        string
	Description string
	Run         func(args []string)
}

var subcommands = []subcommand{
	{"serve", "start an HTTP server to serve VCS repository data", serveCmd},
	{"repo", "display information about a repository", repoCmd},
	{"clone", "clones a repository on the server", cloneCmd},
}

func serveCmd(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	debug := fs.Bool("d", false, "debug mode (don't use on publicly available servers)")
	bindAddr := fs.String("http", ":"+defaultPort, "HTTP listen address")
	hashedPath := fs.Bool("hashed-path", false, "use nested dirs based on VCS/repo hash instead of flat (HashedRepositoryPath)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: vcsstore serve [options]

Starts an HTTP server that serves information about VCS repositories.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 0 {
		fs.Usage()
	}

	if *hashedPath {
		vcsstore.RepositoryPath = vcsstore.HashedRepositoryPath
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
	if *debug {
		conf.DebugLog = log.New(logw, "vcsstore DEBUG: ", log.LstdFlags)
	}
	server.Service = vcsstore.NewService(conf)
	server.Log = log.New(logw, "server: ", log.LstdFlags)
	server.InformativeErrors = *debug

	http.Handle("/", server.NewHandler(nil, nil))

	fmt.Fprintf(os.Stderr, "Starting server on %s\n", *bindAddr)
	err = http.ListenAndServe(*bindAddr, nil)
	if err != nil {
		log.Fatalf("HTTP server failed to start: %s.", err)
	}
}

func repoCmd(args []string) {
	fs := flag.NewFlagSet("repo", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: vcsstore repo [options] vcs-type clone-url

Displays the directory to which a repository would be cloned.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 2 {
		fs.Usage()
	}

	vcsType := fs.Arg(0)
	cloneURL, err := url.Parse(fs.Arg(1))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("RepositoryPath:      ", filepath.Join(*storageDir, vcsstore.RepositoryPath(vcsType, cloneURL)))
	fmt.Println("HashedRepositoryPath:", filepath.Join(*storageDir, vcsstore.HashedRepositoryPath(vcsType, cloneURL)))
	fmt.Println("URL:                 ", vcsclient.NewRouter(nil).URLToRepo(vcsType, cloneURL))
}

func cloneCmd(args []string) {
	fs := flag.NewFlagSet("clone", flag.ExitOnError)
	urlStr := fs.String("url", "http://localhost:"+defaultPort, "base URL to a running vcsstore API server")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: vcsstore clone [options] vcs-type clone-url

Clones a repository on the server. Once finished, the repository will be
available to the client via the vcsstore API.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 2 {
		fs.Usage()
	}

	baseURL, err := url.Parse(*urlStr)
	if err != nil {
		log.Fatal(err)
	}

	vcsType := fs.Arg(0)
	cloneURL, err := url.Parse(fs.Arg(1))
	if err != nil {
		log.Fatal(err)
	}

	c := vcsclient.New(baseURL, nil)
	repo := c.Repository(vcsType, cloneURL)

	if repo, ok := repo.(vcsclient.RepositoryRemoteCloner); ok {
		err := repo.CloneRemote()
		if err != nil {
			log.Fatal("Clone: ", err)
		}
	} else {
		log.Fatalf("Remote cloning is not implemented for %T.", repo)
	}

	fmt.Printf("%-5s %-45s cloned OK\n", vcsType, cloneURL)
}

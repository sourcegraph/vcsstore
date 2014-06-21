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

	"github.com/coreos/go-etcd/etcd"
	"github.com/sourcegraph/datad"
	"github.com/sourcegraph/vcsstore"
	"github.com/sourcegraph/vcsstore/cluster"
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
	{"serve-cluster", "start a datad provider and HTTP server", serveClusterCmd},
	{"repo", "display information about a repository", repoCmd},
	{"clone", "clones a repository on the server", cloneCmd},
}

func serveCmd(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	debug := fs.Bool("d", false, "debug mode (don't use on publicly available servers)")
	bindAddr := fs.String("http", ":"+defaultPort, "HTTP listen address")
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

	err := os.MkdirAll(*storageDir, 0700)
	if err != nil {
		log.Fatalf("Error creating directory %q: %s.", *storageDir, err)
	}

	conf := &vcsstore.Config{StorageDir: *storageDir}

	startServer(conf, *debug, *bindAddr)
}

func serveClusterCmd(args []string) {
	fs := flag.NewFlagSet("serve-cluster", flag.ExitOnError)
	debug := fs.Bool("d", false, "debug mode (don't use on publicly available servers)")
	bindAddr := fs.String("http", "0.0.0.0:"+defaultPort, "HTTP listen address")
	datadBindAddr := fs.String("datad-http", "0.0.0.0:4388", "datad provider HTTP listen address")
	etcdEndpoint := fs.String("etcd", "http://127.0.0.1:4001", "etcd endpoint")
	etcdKeyPrefix := fs.String("etcd-key-prefix", datad.DefaultKeyPrefix, "keyspace for datad registry and provider list in etcd")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: vcsstore serve-cluster [options]

Starts an HTTP server that serves information about VCS repositories, and a
datad provider server.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() != 0 {
		fs.Usage()
	}

	err := os.MkdirAll(*storageDir, 0700)
	if err != nil {
		log.Fatalf("Error creating directory %q: %s.", *storageDir, err)
	}

	conf := &vcsstore.Config{
		StorageDir:     *storageDir,
		RepositoryPath: cluster.RepositoryKey,
	}

	go startServer(conf, *debug, *bindAddr)

	fmt.Fprintf(os.Stderr, "Connecting to etcd at %s (key prefix %q)\n", *etcdEndpoint, *etcdKeyPrefix)
	datadBackend := datad.NewEtcdBackend(*etcdKeyPrefix, etcd.NewClient([]string{*etcdEndpoint}))
	datadClient := datad.NewClient(datadBackend)
	clusterServer := cluster.NewServer(datadClient, conf, vcsstore.NewService(conf))

	err = datadClient.AddProvider("http://"+*datadBindAddr, "http://"+*bindAddr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(os.Stderr, "Starting datad provider on %s\n", *datadBindAddr)
	log.Fatal(http.ListenAndServe(*datadBindAddr, clusterServer.ProviderHandler()))
}

func startServer(conf *vcsstore.Config, debug bool, bindAddr string) {
	var logw io.Writer
	if *verbose {
		logw = os.Stderr
	} else {
		logw = ioutil.Discard
	}

	conf.Log = log.New(logw, "vcsstore: ", log.LstdFlags)

	if debug {
		conf.DebugLog = log.New(logw, "vcsstore DEBUG: ", log.LstdFlags)
	}
	server.Service = vcsstore.NewService(conf)
	server.Log = log.New(logw, "server: ", log.LstdFlags)
	server.InformativeErrors = debug

	http.Handle("/", server.NewHandler(nil, nil))

	fmt.Fprintf(os.Stderr, "Starting server on %s\n", bindAddr)
	err := http.ListenAndServe(bindAddr, nil)
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

	fmt.Println("URL:  ", vcsclient.NewRouter(nil).URLToRepo(vcsType, cloneURL))
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
	repo, err := c.Repository(vcsType, cloneURL)
	if err != nil {
		log.Fatal("Open repository: ", err)
	}

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

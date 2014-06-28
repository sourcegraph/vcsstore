# vcsstore

vcsstore stores VCS repositories and makes them accessible via HTTP.

## Install

* [Install libgit2](https://github.com/libgit2/libgit2). The latest version vcsstore has been tested with is
  7851e595ad832b532e6edc6ac5fb0e43db24fc6a
* `go get github.com/sourcegraph/vcsstore`
* `cd $GOPATH/src/github.com/sourcegraph/vcsstore`
* `godep go install ./...`
* `vcsstore`

vcsstore can also be called as a library.

## Related reading

* [How We Made GitHub Fast (GitHub blog post)](https://github.com/blog/530-how-we-made-github-fast)

## Authors

* Quinn Slack <sqs@sourcegraph.com>

FROM ubuntu:14.04

RUN apt-get update -q
RUN apt-get install -qy build-essential curl git mercurial pkg-config

# Install Go
RUN curl -Ls https://golang.org/dl/go1.3.3.linux-amd64.tar.gz | tar -C /usr/local -xz
ENV PATH /usr/local/go/bin:$PATH
ENV GOBIN /usr/local/bin

# Install libgit2 (for git2go); use pinned version from 2014-05-11 (because it is known to work; there's nothing otherwise special about this commit).
RUN apt-get install -qy cmake libssh2-1-dev libssl-dev

ENV GOPATH /opt
RUN go get github.com/tools/godep
ADD . /opt/src/github.com/sourcegraph/vcsstore
WORKDIR /opt/src/github.com/sourcegraph/vcsstore
RUN make build-libgit2
RUN godep go install -v ./cmd/vcsstore

EXPOSE 80
VOLUME ["/mnt/vcsstore"]
CMD ["-v", "-s=/mnt/vcsstore", "serve", "-http=:80"]
ENTRYPOINT ["/usr/local/bin/vcsstore"]

FROM ubuntu:14.04

RUN apt-get update -q
RUN apt-get install -qy build-essential curl git mercurial pkg-config

# Install Go
RUN curl -s https://storage.googleapis.com/golang/go1.3beta1.linux-amd64.tar.gz | tar -C /usr/local -xz
ENV PATH /usr/local/go/bin:$PATH
ENV GOBIN /usr/local/bin
ENV GOPATH /usr/local/lib/go

# Install libgit2 (for git2go); use pinned version from 2014-05-11 (because it is known to work; there's nothing otherwise special about this commit).
RUN apt-get install -qy cmake libssh2-1-dev
ADD https://github.com/libgit2/libgit2/tarball/e18d5e52e385c0cc2ad8d9d4fdd545517f170a11 /tmp/libgit2.tgz
RUN cd /tmp && tar xzf /tmp/libgit2.tgz && cd /tmp/libgit2-libgit2-e18d5e5 && mkdir build && cd build && cmake .. -DCMAKE_INSTALL_PREFIX=/usr -DBUILD_CLAR=OFF -DTHREADSAFE=ON && cmake --build . --target install

# Install sgx
RUN ln -s /usr/bin/nodejs /usr/local/bin/node
ADD . /opt/src/github.com/sourcegraph/vcsstore
WORKDIR /opt/src/github.com/sourcegraph/vcsstore
ENV GOPATH /opt:$GOPATH
RUN go get -d -v -t github.com/sourcegraph/vcsstore/cmd/vcsstore
RUN cd /opt/src/github.com/sqs/mux && git checkout custom
RUN cd /opt/src/github.com/knieriem/hgo && git pull -f git://github.com/sqs/hgo.git branches_support
RUN go install -v github.com/sourcegraph/vcsstore/cmd/vcsstore

EXPOSE 80
VOLUME ["/mnt/vcsstore"]
CMD ["-v", "-s=/mnt/vcsstore", "serve", "-http=:80", "-hashed-path"]
ENTRYPOINT ["/usr/local/bin/vcsstore"]

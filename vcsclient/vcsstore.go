package vcsclient

import "sourcegraph.com/sourcegraph/vcsstore/git"

type VCSStore interface {
	RepositoryOpener
	git.GitTransporter
}

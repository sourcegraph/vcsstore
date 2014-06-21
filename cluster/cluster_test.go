package cluster

import (
	"net/url"
	"testing"
)

func TestEncodeAndDecodeRepoKey(t *testing.T) {
	keys := []struct {
		vcsType     string
		cloneURLStr string
	}{
		{"git", "git://foo.com/bar/baz.git"},
	}
	for _, key := range keys {
		cloneURL, err := url.Parse(key.cloneURLStr)
		if err != nil {
			t.Fatal(err)
		}

		encKey := RepositoryKey(key.vcsType, cloneURL)
		vcsType, cloneURL2, err := DecodeRepositoryKey(encKey)
		if err != nil {
			t.Errorf("decodeRepoKey(%q): %s", encKey, err)
			continue
		}
		if vcsType != key.vcsType {
			t.Errorf("got vcsType == %q, want %q", vcsType, key.vcsType)
		}
		if cloneURL2.String() != key.cloneURLStr {
			t.Errorf("got cloneURL == %q, want %q", cloneURL2, key.cloneURLStr)
		}
	}
}

package cluster

import (
	"testing"

	etcd_server "github.com/coreos/etcd/server"
	"github.com/coreos/etcd/tests"
	"github.com/coreos/go-etcd/etcd"
)

func TestDirectory(t *testing.T) {
	tests.RunServer(func(s *etcd_server.Server) {
		d := &Directory{etcd.NewClient([]string{s.URL()})}

		a := key("a")

		_, err := d.Owner(a)
		if err != ErrNoOwner {
			t.Fatal(err)
		}

		err = d.SetOwner(a, "o")
		if err != nil {
			t.Fatal(err)
		}

		owner, err := d.Owner(a)
		if err != nil {
			t.Fatal(err)
		}
		if want := "o"; owner != want {
			t.Errorf("before SetOwner, got Owner == %q, want %q", owner, want)
		}
	})
}

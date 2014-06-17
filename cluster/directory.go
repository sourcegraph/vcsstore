package cluster

import (
	"errors"

	"github.com/coreos/go-etcd/etcd"
)

type Directory struct {
	c *etcd.Client
}

var ErrNoOwner = errors.New("key has no owner")

func (d *Directory) Owner(key Keyer) (string, error) {
	resp, err := d.c.Get(key.Key(), false, false)
	if err != nil {
		if err, ok := err.(*etcd.EtcdError); ok && err.ErrorCode == 100 {
			return "", ErrNoOwner
		}
		return "", err
	}
	return resp.Node.Value, nil
}

func (d *Directory) SetOwner(key Keyer, owner string) error {
	_, err := d.c.Set(key.Key(), owner, 0)
	return err
}

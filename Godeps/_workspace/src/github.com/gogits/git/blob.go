package git

import ()

type Blob struct {
	*TreeEntry

	data   []byte
	dataed bool
}

func (b *Blob) Data() ([]byte, error) {
	if b.dataed {
		return b.data, nil
	}
	_, _, data, err := b.ptree.repo.getRawObject(b.Id)
	if err != nil {
		return nil, err
	}
	b.data = data
	b.dataed = true
	return b.data, nil
}

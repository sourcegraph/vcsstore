package git

import (
	"os"
	"path/filepath"

	"github.com/Unknwon/cae/zip"
)

func (c *Commit) CreateArchive(zipPath string) error {
	f, err := os.OpenFile(zipPath, os.O_CREATE, 0644)
	if err == nil {
		f.Close()
	}

	f, err = os.OpenFile(zipPath, os.O_WRONLY|os.O_TRUNC|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	archive := zip.NewStreamArachive(f)
	defer archive.Close()

	return createArchive(&c.Tree, archive)
}

func createArchive(tree *Tree, archive *zip.StreamArchive, relPaths ...string) error {
	var relPath string

	if len(relPaths) > 0 {
		relPath = relPaths[0]
	}

	for _, te := range tree.ListEntries() {
		if te.IsDir() {
			err := archive.StreamFile(filepath.Join(relPath, te.name), te, nil)
			if err != nil {
				return err
			}

			newTree, err := te.ptree.SubTree(te.name)
			if err != nil {
				return err
			}

			if err = createArchive(newTree, archive, filepath.Join(relPath, te.name)); err != nil {
				return err
			}
		} else {
			data, err := te.Blob().Data()
			if err != nil {
				return err
			}
			if err := archive.StreamFile(relPath, te, data); err != nil {
				return err
			}
		}
	}

	return nil
}

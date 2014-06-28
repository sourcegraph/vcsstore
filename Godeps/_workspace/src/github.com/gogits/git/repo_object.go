package git

import (
	"errors"
	"fmt"
	"os"
)

// Who am I?
type ObjectType int

const (
	ObjectCommit ObjectType = 0x10
	ObjectTree   ObjectType = 0x20
	ObjectBlob   ObjectType = 0x30
	ObjectTag    ObjectType = 0x40
)

func (t ObjectType) String() string {
	switch t {
	case ObjectCommit:
		return "Commit"
	case ObjectTree:
		return "Tree"
	case ObjectBlob:
		return "Blob"
	default:
		return ""
	}
}

type Object struct {
	Type ObjectType
	Id   sha1
}

func (repo *Repository) getRawObject(id sha1) (ObjectType, int64, []byte, error) {
	// first we need to find out where the commit is stored
	sha1 := id.String()
	objpath := filepathFromSHA1(repo.Path, sha1)
	_, err := os.Stat(objpath)
	if os.IsNotExist(err) {
		// doesn't exist, let's look if we find the object somewhere else
		for _, indexfile := range repo.indexfiles {
			if offset := indexfile.offsetValues[id]; offset != 0 {
				return readObjectBytes(indexfile.packpath, offset, false)
			}
		}
		return 0, 0, nil, errors.New(fmt.Sprintf("Object not found %s", sha1))
	}
	return readObjectFile(objpath, false)
}

// Get the type of an object.
func (repo *Repository) Type(id sha1) (ObjectType, error) {
	objtype, _, _, err := repo.getRawObject(id)
	if err != nil {
		return 0, err
	}
	return objtype, nil
}

// Get (inflated) size of an object.
func (repo *Repository) objectSize(id sha1) (int64, error) {
	sha1 := id.String()
	// todo: this is mostly the same as getRawObject -> merge
	// difference is the boolean in readObjectBytes and readObjectFile
	objpath := filepathFromSHA1(repo.Path, sha1)
	_, err := os.Stat(objpath)
	if os.IsNotExist(err) {
		// doesn't exist, let's look if we find the object somewhere else
		for _, indexfile := range repo.indexfiles {
			if offset := indexfile.offsetValues[id]; offset != 0 {
				_, length, _, err := readObjectBytes(indexfile.packpath, offset, true)
				return length, err
			}
		}

		return 0, errors.New("Object not found")
	}
	_, length, _, err := readObjectFile(objpath, true)
	return length, err
}

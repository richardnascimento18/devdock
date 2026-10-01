package fileutil

import (
	"errors"
	"os"
	"path/filepath"
)

// WriteFileAtomic syncs and closes an exclusive temporary file before rename.
// Rename is the commit point: an error means the destination was not replaced.
// Directory metadata durability across sudden power loss is filesystem-specific.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	closed := false
	committed := false
	defer func() {
		if !closed {
			err = errors.Join(err, tmp.Close())
		}
		if !committed {
			if cleanup := os.Remove(name); !os.IsNotExist(cleanup) {
				err = errors.Join(err, cleanup)
			}
		}
	}()
	if err = tmp.Chmod(perm); err != nil {
		return err
	}
	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	err = tmp.Close()
	closed = true
	if err != nil {
		return err
	}
	if err = os.Rename(name, path); err != nil {
		return err
	}
	committed = true
	return nil
}

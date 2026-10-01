package core

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// PartialMoveError means destination is complete but source cleanup failed.
// Both locations are retained in the error so users can reconcile them safely.
type PartialMoveError struct {
	Source, Destination string
	Err                 error
}

func (e *PartialMoveError) Error() string {
	return fmt.Sprintf("copied to %q but cleanup of %q failed: %v", e.Destination, e.Source, e.Err)
}
func (e *PartialMoveError) Unwrap() error { return e.Err }

func renameExclusive(src, dst string) error {
	return unix.Renameat2(unix.AT_FDCWD, src, unix.AT_FDCWD, dst, unix.RENAME_NOREPLACE)
}

func MoveProject(p Project, destPath, destRoot, destDomain string) (Project, error) {
	if err := ValidateDescendant(p.Root, p.Path); err != nil {
		return Project{}, err
	}
	if err := CheckDestination(destRoot, destPath); err != nil {
		return Project{}, err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return Project{}, err
	}
	if err := moveDir(p.Path, destPath, renameExclusive, os.RemoveAll); err != nil {
		return Project{}, err
	}
	p.Path, p.Root, p.Domain = destPath, destRoot, destDomain
	p.Group, p.Subgroup = "", ""
	rel, err := filepath.Rel(filepath.Join(destRoot, destDomain), filepath.Dir(destPath))
	if err != nil {
		return Project{}, err
	}
	if rel != "." {
		parts := strings.Split(rel, string(filepath.Separator))
		p.Group = parts[0]
		if len(parts) > 1 {
			p.Subgroup = parts[len(parts)-1]
		}
	}
	return p, nil
}

func moveDir(src, dst string, rename func(string, string) error, remove func(string) error) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("source %q must be a directory", src)
	}
	if IsDescendant(src, dst) {
		return fmt.Errorf("destination must be outside source")
	}
	if _, err := os.Lstat(dst); err == nil {
		return os.ErrExist
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := rename(src, dst); err == nil {
		return nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return fmt.Errorf("rename: %w", err)
	}
	stage, err := os.MkdirTemp(filepath.Dir(dst), ".devdock-move-*")
	if err != nil {
		return err
	}
	// Copy into an isolated destination so errors never merge or delete source data.
	copied := filepath.Join(stage, "project")
	if err := copyTree(src, copied); err != nil {
		return errors.Join(fmt.Errorf("copy: %w", err), os.RemoveAll(stage))
	}
	if err := renameExclusive(copied, dst); err != nil {
		return errors.Join(fmt.Errorf("publish copy: %w", err), os.RemoveAll(stage))
	}
	if err := os.Remove(stage); err != nil {
		return &PartialMoveError{src, dst, err}
	}
	if err := remove(src); err != nil {
		return &PartialMoveError{src, dst, err}
	}
	return nil
}

func copyTree(src, dst string) error {
	var directories []struct {
		path string
		mode os.FileMode
	}
	err := filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		if entry.IsDir() {
			directories = append(directories, struct {
				path string
				mode os.FileMode
			}{target, info.Mode().Perm()})
			return os.Mkdir(target, info.Mode().Perm()|0o700)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported file %q (%s)", path, info.Mode())
		}
		in, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
		if err != nil {
			return err
		}
		opened, err := in.Stat()
		if err != nil || !opened.Mode().IsRegular() {
			return errors.Join(fmt.Errorf("source changed during copy: %q", path), err, in.Close())
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
		if err != nil {
			return errors.Join(err, in.Close())
		}
		if err := out.Chmod(info.Mode().Perm()); err != nil {
			return errors.Join(err, in.Close(), out.Close())
		}
		_, err = io.Copy(out, in)
		if err == nil {
			err = out.Sync()
		}
		return errors.Join(err, in.Close(), out.Close())
	})
	if err != nil {
		return err
	}
	for i := len(directories) - 1; i >= 0; i-- {
		if err := os.Chmod(directories[i].path, directories[i].mode); err != nil {
			return err
		}
	}
	return nil
}

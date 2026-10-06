package core

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

type MoveStatus string

const (
	MoveValid              MoveStatus = "valid"
	MoveInvalidDestination MoveStatus = "invalid_destination"
	MoveDestinationExists  MoveStatus = "destination_exists"
	MoveSourceMissing      MoveStatus = "source_missing"
)

type MovePlan struct {
	Source        Project
	Destination   Location
	DirectoryName string
	Path          string
	Status        MoveStatus
}

func PlanMove(p Project, destination Location, directoryName string) (MovePlan, error) {
	plan := MovePlan{Source: p, Destination: destination, DirectoryName: directoryName, Status: MoveInvalidDestination}
	path, err := destination.ProjectPath(directoryName)
	if err != nil {
		return plan, err
	}
	plan.Path = path
	sourcePath, err := p.Location.ProjectPath(filepath.Base(p.Path))
	if err != nil {
		return plan, err
	}
	if sourcePath != p.Path {
		return plan, fmt.Errorf("source path disagrees with location")
	}
	info, err := os.Lstat(p.Path)
	if os.IsNotExist(err) {
		plan.Status = MoveSourceMissing
		return plan, err
	}
	if err != nil {
		return plan, err
	}
	if !info.IsDir() {
		return plan, fmt.Errorf("source must be a directory")
	}
	if IsDescendant(p.Path, path) {
		return plan, fmt.Errorf("destination must be outside source")
	}
	if err := CheckDestination(destination.Root, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			plan.Status = MoveDestinationExists
		}
		return plan, err
	}
	plan.Status = MoveValid
	return plan, nil
}

// Mover allows filesystem primitives to be supplied without global hooks. Zero
// values use the native syscalls. Both the fast path and copy publication use
// the same guarded no-replace compatibility behavior.
type Mover struct {
	NoReplace func(string, string) error
	Rename    func(string, string) error
}

func (m Mover) rename(src, dst string) error {
	exclusive := m.NoReplace
	if exclusive == nil {
		exclusive = renameExclusive
	}
	portable := m.Rename
	if portable == nil {
		portable = os.Rename
	}
	return renameCompatible(src, dst, exclusive, portable)
}

func renameCompatible(src, dst string, exclusive, portable func(string, string) error) error {
	err := exclusive(src, dst)
	if err == nil {
		return nil
	}
	// EINVAL can also mean a directory moved into itself. Validate the paths
	// before treating it as an unsupported flag; ordinary rename still reports
	// any other invalid operation. ENOTSUP aliases EOPNOTSUPP on Linux.
	if !errors.Is(err, syscall.EINVAL) && !errors.Is(err, syscall.ENOSYS) && !errors.Is(err, syscall.EOPNOTSUPP) {
		return err
	}
	info, statErr := os.Lstat(src)
	if statErr != nil {
		return statErr
	}
	if filepath.Clean(src) == filepath.Clean(dst) || info.IsDir() && IsDescendant(src, dst) {
		return syscall.EINVAL
	}
	// Lstat also rejects dangling destination symlinks. Recheck immediately
	// before publication. POSIX rename cannot atomically promise no-replace:
	// an external writer racing this check remains a Pass 4 limitation.
	if _, statErr := os.Lstat(dst); statErr == nil {
		return os.ErrExist
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	return portable(src, dst)
}

func ExecuteMove(plan MovePlan) (Project, error) { return (Mover{}).Execute(plan) }

func (m Mover) Execute(plan MovePlan) (Project, error) {
	// Plans are advisory: revalidate to catch a changed source or destination.
	checked, err := PlanMove(plan.Source, plan.Destination, plan.DirectoryName)
	if err != nil {
		return Project{}, err
	}
	if err := os.MkdirAll(filepath.Dir(checked.Path), 0o755); err != nil {
		return Project{}, err
	}
	p := checked.Source
	if p.Name == filepath.Base(p.Path) {
		p.Name = checked.DirectoryName
	}
	p.Path, p.Location = checked.Path, checked.Destination
	if err := moveDirWithPublication(checked.Source.Path, checked.Path, m.rename, m.rename, os.RemoveAll); err != nil {
		var partial *PartialMoveError
		if errors.As(err, &partial) {
			return p, err
		}
		return Project{}, err
	}
	return p, nil
}
func MoveProject(p Project, destination Location, directoryName string) (Project, error) {
	plan, err := PlanMove(p, destination, directoryName)
	if err != nil {
		return Project{}, err
	}
	return ExecuteMove(plan)
}

func moveDir(src, dst string, rename func(string, string) error, remove func(string) error) error {
	initial := func(src, dst string) error { return renameCompatible(src, dst, rename, os.Rename) }
	return moveDirWithPublication(src, dst, initial, (Mover{}).rename, remove)
}

func moveDirWithPublication(src, dst string, rename, publish func(string, string) error, remove func(string) error) error {
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
	if err := publish(copied, dst); err != nil {
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

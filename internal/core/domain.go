package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IsDescendant includes equality; destructive callers must use ValidateDescendant.
func IsDescendant(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func ValidName(name string) bool {
	return strings.TrimSpace(name) != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\\x00")
}

// ValidateDescendant rejects root equality, escapes and symlink components.
// Workspace roots themselves may be symlinks; paths below them may not be.
func ValidateDescendant(root, target string) error {
	if root == "" || !filepath.IsAbs(root) || !filepath.IsAbs(target) {
		return fmt.Errorf("workspace paths must be absolute")
	}
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == "." || !IsDescendant(root, target) {
		return fmt.Errorf("refusing path %q outside or equal to root %q", target, root)
	}
	path := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink component %q", path)
		}
	}
	return nil
}

func CheckDestination(root, target string) error {
	if err := ValidateDescendant(root, target); err != nil {
		return err
	}
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("destination %q already exists: %w", target, os.ErrExist)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect destination: %w", err)
	}
	return nil
}

func CreateDomain(root, name string) error {
	if !ValidName(name) {
		return fmt.Errorf("invalid domain name %q", name)
	}
	path := filepath.Join(root, name)
	if err := CheckDestination(root, path); err != nil {
		return err
	}
	return os.Mkdir(path, 0o755)
}

func PrepareProject(location Location, projectName string, createsFolder bool) (string, string, error) {
	projectPath, err := location.ProjectPath(projectName)
	if err != nil {
		return "", "", err
	}
	if err := CheckDestination(location.Root, projectPath); err != nil {
		return "", "", err
	}
	parent := location.Path()
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", "", err
	}
	if createsFolder {
		return projectPath, parent, nil
	}
	if err := os.Mkdir(projectPath, 0o755); err != nil {
		return "", "", err
	}
	return projectPath, projectPath, nil
}
func CreateProject(location Location, projectName string) (Project, error) {
	path, _, err := PrepareProject(location, projectName, false)
	if err != nil {
		return Project{}, err
	}
	return Project{Location: location, Name: projectName, Path: path, Kind: KindProject}, nil
}
func CreateGroup(parent Location, name string) error {
	path, _, err := PrepareProject(parent, name, false)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(path, ".ddgroup"), nil, 0o644); err != nil {
		return fmt.Errorf("group directory created but marker failed: %w", err)
	}
	return nil
}
func DeleteGroup(location Location) error {
	if len(location.GroupPath) == 0 {
		return fmt.Errorf("group path required")
	}
	path, err := location.Resolve()
	if err != nil {
		return err
	}
	return DeletePath(location.Root, path)
}

func DeletePath(root, path string) error {
	if err := ValidateDescendant(root, path); err != nil {
		return err
	}
	workspace, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("open workspace for deletion: %w", err)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return errors.Join(err, workspace.Close())
	}
	err = workspace.RemoveAll(rel)
	return errors.Join(err, workspace.Close())
}
func DeleteProject(p Project) error {
	path, err := p.Location.ProjectPath(filepath.Base(p.Path))
	if err != nil {
		return err
	}
	if path != p.Path {
		return fmt.Errorf("project path disagrees with location")
	}
	return DeletePath(p.Root, path)
}
func DeleteDomain(root, domain string) error {
	if !ValidName(domain) {
		return fmt.Errorf("invalid domain name %q", domain)
	}
	return DeletePath(root, filepath.Join(root, domain))
}

// CreateGroupPath validates the entire proposal before creating any component.
// Existing parents are allowed; the final group must not already exist.
func CreateGroupPath(parent Location, names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("group path required")
	}
	target := parent
	for _, name := range names {
		target = target.Child(name)
	}
	path, err := target.Resolve()
	if err != nil {
		return err
	}
	if err := CheckDestination(target.Root, path); err != nil {
		return err
	}
	for _, name := range names {
		child := parent.Child(name)
		if info, err := os.Lstat(child.Path()); os.IsNotExist(err) {
			if err := CreateGroup(parent, name); err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if !info.IsDir() {
			return fmt.Errorf("group parent is not a directory: %s", child.Path())
		}
		parent = child
	}
	return nil
}

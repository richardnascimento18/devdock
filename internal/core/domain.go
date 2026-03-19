package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IsDescendant returns true if target is the same as base or is nested inside it.
// Both paths are cleaned before comparison.
func IsDescendant(base, target string) bool {
	base = filepath.Clean(base)
	target = filepath.Clean(target)
	return target == base || strings.HasPrefix(target, base+string(filepath.Separator))
}

func CreateDomain(root, name string) error {
	path := filepath.Join(root, name)
	return os.MkdirAll(path, 0o755)
}

func CreateProject(root, domainName, projectName string) (Project, error) {
	domainPath := filepath.Join(root, domainName)
	if err := os.MkdirAll(domainPath, 0o755); err != nil {
		return Project{}, err
	}
	projectPath := filepath.Join(domainPath, projectName)
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		return Project{}, err
	}
	return Project{
		Name:   projectName,
		Path:   projectPath,
		Domain: domainName,
		Root:   root,
	}, nil
}

func DeleteProject(p Project) error {
	if !IsDescendant(p.Root, p.Path) {
		return fmt.Errorf("refusing to delete %q: path is outside configured root %q", p.Path, p.Root)
	}
	return os.RemoveAll(p.Path)
}

func DeleteDomain(root, domainName string) error {
	path := filepath.Join(root, domainName)
	if !IsDescendant(root, path) {
		return fmt.Errorf("refusing to delete domain %q: path is outside root %q", domainName, root)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(path, e.Name())); err != nil {
			return fmt.Errorf("failed to delete project %s: %w", e.Name(), err)
		}
	}
	return os.Remove(path)
}

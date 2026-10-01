package core

import (
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

func PrepareProject(root, domainName, projectName string, createsFolder bool) (string, string, error) {
	if !ValidName(domainName) || !ValidName(projectName) {
		return "", "", fmt.Errorf("invalid domain or project name")
	}
	domainPath := filepath.Join(root, domainName)
	projectPath := filepath.Join(domainPath, projectName)
	if err := CheckDestination(root, projectPath); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(domainPath, 0o755); err != nil {
		return "", "", err
	}
	if createsFolder {
		return projectPath, domainPath, nil
	}
	if err := os.Mkdir(projectPath, 0o755); err != nil {
		return "", "", err
	}
	return projectPath, projectPath, nil
}

func CreateProject(root, domainName, projectName string) (Project, error) {
	path, _, err := PrepareProject(root, domainName, projectName, false)
	if err != nil {
		return Project{}, err
	}
	return Project{Name: projectName, Path: path, Domain: domainName, Root: root, Kind: KindProject}, nil
}

func CreateGroup(root, domain, name string) error {
	path, _, err := PrepareProject(root, domain, name, false)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(path, ".ddgroup"), nil, 0o644); err != nil {
		return fmt.Errorf("group directory created but marker failed: %w", err)
	}
	return nil
}

func DeletePath(root, path string) error {
	if err := ValidateDescendant(root, path); err != nil {
		return err
	}
	return os.RemoveAll(path)
}
func DeleteProject(p Project) error {
	if !ValidName(p.Domain) {
		return fmt.Errorf("invalid project domain")
	}
	if err := ValidateDescendant(filepath.Join(p.Root, p.Domain), p.Path); err != nil {
		return err
	}
	return DeletePath(p.Root, p.Path)
}
func DeleteDomain(root, domain string) error {
	if !ValidName(domain) {
		return fmt.Errorf("invalid domain name %q", domain)
	}
	return DeletePath(root, filepath.Join(root, domain))
}

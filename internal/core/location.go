package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Location identifies a root, domain or full group ancestry. For operations a
// domain is required. Empty Domain is used only for root navigation keys.
type Location struct {
	Root      string   `json:"root"`
	Domain    string   `json:"domain"`
	GroupPath []string `json:"group_path,omitempty"`
}

type NodeKey string

func (l Location) Key() NodeKey {
	parts := []string{l.Root, l.Domain}
	parts = append(parts, l.GroupPath...)
	data, _ := json.Marshal(parts) // strings cannot fail JSON encoding
	return NodeKey(data)
}
func ParseNodeKey(key NodeKey) (Location, error) {
	var parts []string
	if err := json.Unmarshal([]byte(key), &parts); err != nil || len(parts) < 2 {
		return Location{}, fmt.Errorf("invalid node key %q", key)
	}
	l := Location{Root: parts[0], Domain: parts[1], GroupPath: parts[2:]}
	if l.Domain == "" && len(l.GroupPath) == 0 && filepath.IsAbs(l.Root) && l.Root == filepath.Clean(l.Root) {
		return l, nil
	}
	return l, l.Validate()
}
func (l Location) Validate() error {
	if !filepath.IsAbs(l.Root) || l.Root != filepath.Clean(l.Root) {
		return fmt.Errorf("invalid workspace root %q", l.Root)
	}
	if !ValidName(l.Domain) {
		return fmt.Errorf("invalid domain %q", l.Domain)
	}
	for _, name := range l.GroupPath {
		if !ValidName(name) {
			return fmt.Errorf("invalid group component %q", name)
		}
	}
	return nil
}
func (l Location) Path() string {
	parts := []string{l.Root, l.Domain}
	return filepath.Join(append(parts, l.GroupPath...)...)
}
func (l Location) Resolve() (string, error) {
	if err := l.Validate(); err != nil {
		return "", err
	}
	info, err := os.Stat(l.Root)
	if err != nil {
		return "", fmt.Errorf("workspace root unavailable: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspace root is not a directory")
	}
	path := l.Path()
	if err := ValidateDescendant(l.Root, path); err != nil {
		return "", err
	}
	return path, nil
}
func (l Location) ProjectPath(name string) (string, error) {
	if !ValidName(name) {
		return "", fmt.Errorf("invalid project directory name %q", name)
	}
	parent, err := l.Resolve()
	if err != nil {
		return "", err
	}
	path := filepath.Join(parent, name)
	return path, ValidateDescendant(l.Root, path)
}
func (l Location) Child(name string) Location {
	l.GroupPath = append(append([]string(nil), l.GroupPath...), name)
	return l
}
func (l Location) Breadcrumb() string {
	return strings.Join(append([]string{l.Domain}, l.GroupPath...), " › ")
}
func LocationForProject(root, domain, path string) (Location, error) {
	l := Location{Root: root, Domain: domain}
	if err := l.Validate(); err != nil {
		return l, err
	}
	base := l.Path()
	if !filepath.IsAbs(path) || path != filepath.Clean(path) || path == base || !IsDescendant(base, path) {
		return l, fmt.Errorf("project path %q must be inside domain %q", path, base)
	}
	rel, err := filepath.Rel(base, filepath.Dir(path))
	if err != nil {
		return l, err
	}
	if rel != "." {
		l.GroupPath = strings.Split(rel, string(filepath.Separator))
	}
	return l, l.Validate()
}

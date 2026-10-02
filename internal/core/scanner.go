package core

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func loadIgnoreList(root string) (map[string]struct{}, error) {
	ignored := make(map[string]struct{})
	f, err := os.Open(filepath.Join(root, ".ddignore"))
	if os.IsNotExist(err) {
		return ignored, nil
	}
	if err != nil {
		return ignored, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if _, err := filepath.Match(line, ""); err != nil {
			return nil, fmt.Errorf("invalid ignore pattern %q: %w", line, err)
		}
		ignored[line] = struct{}{}
	}
	return ignored, sc.Err()
}

func isDomainIgnored(domainName string, ignored map[string]struct{}) bool {
	for pattern := range ignored {
		// Patterns are root-relative; never interpret root path characters as glob syntax.
		// Patterns were validated while loading .ddignore.
		if matched, _ := filepath.Match(filepath.Clean(pattern), domainName); matched {
			return true
		}
	}
	return false
}

// NodeKind describes navigation, distinct from project discovery markers.
type NodeKind string

const (
	NodeRoot   NodeKind = "root"
	NodeDomain NodeKind = "domain"
	NodeGroup  NodeKind = "group"
)

type Node struct {
	Location
	Kind         NodeKind
	Name         string
	Children     []*Node
	Projects     []Project
	ProjectCount int
}
type Workspace struct {
	Roots    []*Node
	Projects []Project
}
type Discovery struct {
	Explicit  bool
	Kind      ProjectKind
	Name      string
	Languages []string
}
type InspectFunc func(string, []os.DirEntry) (Discovery, error)

// ScanWorkspace performs a single postorder traversal with local inspection.
// Symlink entries are never followed. Projects terminate hierarchy traversal.
func ScanWorkspace(roots []string, inspect InspectFunc) (Workspace, error) {
	var w Workspace
	var failures []error
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			failures = append(failures, fmt.Errorf("scan root %q: %w", root, err))
			continue
		}
		r := &Node{Location: Location{Root: root}, Kind: NodeRoot, Name: filepath.Base(root)}
		w.Roots = append(w.Roots, r)
		ignored, err := loadIgnoreList(root)
		if err != nil {
			failures = append(failures, fmt.Errorf("read .ddignore in %q: %w", root, err))
			continue
		}
		for _, entry := range entries {
			if !workspaceDirectory(entry) || strings.HasPrefix(entry.Name(), ".") || isDomainIgnored(entry.Name(), ignored) {
				continue
			}
			domain := &Node{Location: Location{Root: root, Domain: entry.Name()}, Kind: NodeDomain, Name: entry.Name()}
			r.Children = append(r.Children, domain)
			err := scanChildren(domain, inspect, ignored, &w)
			if err != nil {
				failures = append(failures, err)
			}
			r.ProjectCount += domain.ProjectCount
		}
	}
	return w, errors.Join(failures...)
}
func workspaceDirectory(entry os.DirEntry) bool {
	return entry.IsDir() && entry.Type()&os.ModeSymlink == 0
}
func excludedContainer(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "node_modules", "vendor", "src", "build", "dist", "target", "__pycache__", "venv":
		return true
	}
	return false
}
func ignoredPath(root, path string, ignored map[string]struct{}) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	for pattern := range ignored {
		if matched, _ := filepath.Match(filepath.Clean(pattern), rel); matched {
			return true
		}
	}
	return false
}
func scanChildren(parent *Node, inspect InspectFunc, ignored map[string]struct{}, w *Workspace) error {
	entries, err := os.ReadDir(parent.Location.Path())
	if err != nil {
		return fmt.Errorf("read %q: %w", parent.Location.Path(), err)
	}
	return scanEntries(parent, entries, inspect, ignored, w)
}
func scanEntries(parent *Node, entries []os.DirEntry, inspect InspectFunc, ignored map[string]struct{}, w *Workspace) error {
	var failures []error
	for _, entry := range entries {
		if !workspaceDirectory(entry) {
			continue
		}
		loc := parent.Location.Child(entry.Name())
		path := loc.Path()
		if ignoredPath(loc.Root, path, ignored) {
			continue
		}
		children, err := os.ReadDir(path)
		if err != nil {
			failures = append(failures, fmt.Errorf("read %q: %w", path, err))
			continue
		}
		found, err := inspect(path, children)
		if err != nil {
			failures = append(failures, fmt.Errorf("inspect %q: %w", path, err))
			continue
		}
		if excludedContainer(entry.Name()) && !found.Explicit {
			continue
		}
		if found.Kind == KindProject {
			name := entry.Name()
			if found.Name != "" {
				name = found.Name
			}
			p := Project{Location: parent.Location, Name: name, Path: path, Kind: KindProject, Languages: found.Languages}
			// Own the ancestry so consumers cannot mutate another project's location.
			p.GroupPath = append([]string(nil), p.GroupPath...)
			parent.Projects = append(parent.Projects, p)
			parent.ProjectCount++
			w.Projects = append(w.Projects, p)
			continue
		}
		group := &Node{Location: loc, Name: entry.Name(), Kind: NodeGroup}
		if err := scanEntries(group, children, inspect, ignored, w); err != nil {
			failures = append(failures, err)
		}
		if found.Kind == KindGroup || len(group.Children) > 0 || len(group.Projects) > 0 {
			parent.Children = append(parent.Children, group)
			parent.ProjectCount += group.ProjectCount
		}
	}
	return errors.Join(failures...)
}

// Row has stable identity and logical depth independent of rendering width.
type Row struct {
	Node    *Node
	Project *Project
	Key     string
	Depth   int
}

// Flatten visits only in-memory nodes. includeContainers adds root/domain rows.
func (w Workspace) Flatten(collapsed map[NodeKey]bool, root string, includeContainers bool) []Row {
	var rows []Row
	var visit func(*Node, int)
	visit = func(n *Node, depth int) {
		visible := includeContainers || n.Kind == NodeGroup
		if visible {
			rows = append(rows, Row{Node: n, Key: string(n.Key()), Depth: depth})
		}
		if collapsed[n.Key()] {
			return
		}
		next := depth
		if visible {
			next++
		}
		for i := range n.Projects {
			p := &n.Projects[i]
			rows = append(rows, Row{Project: p, Key: string(p.Key()), Depth: next})
		}
		for _, child := range n.Children {
			visit(child, next)
		}
	}
	for _, r := range w.Roots {
		if root == "" || r.Root == root {
			visit(r, 0)
		}
	}
	return rows
}

// Locations returns the domain and all nested groups for placement pickers.
func (w Workspace) Locations(root, domain string) []Location {
	var locations []Location
	var visit func(*Node)
	visit = func(n *Node) {
		if n.Domain == domain {
			locations = append(locations, n.Location)
		}
		for _, c := range n.Children {
			visit(c)
		}
	}
	for _, r := range w.Roots {
		if r.Root == root {
			for _, d := range r.Children {
				if d.Domain == domain {
					visit(d)
				}
			}
		}
	}
	return locations
}
func (w Workspace) Domains(root string) []string {
	var names []string
	for _, r := range w.Roots {
		if r.Root == root {
			for _, d := range r.Children {
				names = append(names, d.Name)
			}
		}
	}
	return names
}

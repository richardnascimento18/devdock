package core

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScanFunc collects projects from a domain directory.
// Injected by the caller (using detect.CollectProjects) to avoid import cycles.
type ScanFunc func(root, domain, domainPath string) ([]Project, error)

// ClassifyDirFunc classifies a directory as project/group/subgroup.
// Injected by the caller (using detect.ClassifyDir) to avoid import cycles.
type ClassifyDirFunc func(path string, depth int) ProjectKind

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

func ScanRoot(root string, collect ScanFunc) ([]Project, error) {
	ignored, err := loadIgnoreList(root)
	if err != nil {
		return nil, fmt.Errorf("read .ddignore: %w", err)
	}
	var projects []Project
	var failures []error
	domains, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, domain := range domains {
		if !domain.IsDir() {
			continue
		}
		domainName := domain.Name()
		if isDomainIgnored(domainName, ignored) {
			continue
		}
		domainPath := filepath.Join(root, domainName)
		ps, err := collect(root, domainName, domainPath)
		projects = append(projects, ps...)
		if err != nil {
			failures = append(failures, fmt.Errorf("domain %q: %w", domainPath, err))
		}
	}
	return projects, errors.Join(failures...)
}

func ScanRoots(roots []string, collect ScanFunc) ([]Project, error) {
	var all []Project
	var failures []error
	for _, root := range roots {
		ps, err := ScanRoot(root, collect)
		if err != nil {
			failures = append(failures, fmt.Errorf("scan root %q: %w", root, err))
		}
		all = append(all, ps...)
	}
	return all, errors.Join(failures...)
}

func ScanDomainsInRoot(root string) ([]string, error) {
	ignored, err := loadIgnoreList(root)
	if err != nil {
		return nil, fmt.Errorf("read .ddignore: %w", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var domains []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if isDomainIgnored(e.Name(), ignored) {
			continue
		}
		domains = append(domains, e.Name())
	}
	return domains, nil
}

// GroupInfo describes a group found inside a domain directory.
type GroupInfo struct {
	Name      string
	Path      string
	Subgroups []SubgroupInfo
}

// SubgroupInfo describes a subgroup.
type SubgroupInfo struct {
	Name      string
	Path      string
	Subgroups []SubgroupInfo
}

func ScanGroupsInDomain(domainPath string, classify ClassifyDirFunc) ([]GroupInfo, error) {
	entries, err := os.ReadDir(domainPath)
	if err != nil {
		return nil, err
	}
	var groups []GroupInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		childPath := filepath.Join(domainPath, e.Name())
		if classify(childPath, 0) == KindGroup {
			gi := GroupInfo{Name: e.Name(), Path: childPath}
			gi.Subgroups, err = scanSubgroups(childPath, 0, classify)
			if err != nil {
				return groups, err
			}
			groups = append(groups, gi)
		}
	}
	return groups, nil
}

func scanSubgroups(groupPath string, depth int, classify ClassifyDirFunc) ([]SubgroupInfo, error) {
	if depth > 3 {
		return nil, nil
	}
	entries, err := os.ReadDir(groupPath)
	if err != nil {
		return nil, err
	}
	var subs []SubgroupInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		childPath := filepath.Join(groupPath, e.Name())
		if classify(childPath, 0) == KindGroup {
			si := SubgroupInfo{Name: e.Name(), Path: childPath}
			si.Subgroups, err = scanSubgroups(childPath, depth+1, classify)
			if err != nil {
				return subs, err
			}
			subs = append(subs, si)
		}
	}
	return subs, nil
}

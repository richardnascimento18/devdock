package core

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ScanFunc collects projects from a domain directory.
// Injected by the caller (using detect.CollectProjects) to avoid import cycles.
type ScanFunc func(root, domain, domainPath string) []Project

// ClassifyDirFunc classifies a directory as project/group/subgroup.
// Injected by the caller (using detect.ClassifyDir) to avoid import cycles.
type ClassifyDirFunc func(path string, depth int) ProjectKind

func loadIgnoreList(root string) map[string]struct{} {
	ignored := make(map[string]struct{})
	f, err := os.Open(filepath.Join(root, ".ddignore"))
	if err != nil {
		return ignored
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ignored[line] = struct{}{}
	}
	return ignored
}

func isDomainIgnored(root, domainName string, ignored map[string]struct{}) bool {
	for pattern := range ignored {
		if strings.Contains(pattern, "/") {
			full := filepath.Join(root, domainName)
			if matched, _ := filepath.Match(filepath.Join(root, pattern), full); matched {
				return true
			}
		} else {
			if matched, _ := filepath.Match(pattern, domainName); matched {
				return true
			}
		}
	}
	return false
}

func ScanRoot(root string, collect ScanFunc) ([]Project, error) {
	ignored := loadIgnoreList(root)
	var projects []Project
	domains, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, domain := range domains {
		if !domain.IsDir() {
			continue
		}
		domainName := domain.Name()
		if isDomainIgnored(root, domainName, ignored) {
			continue
		}
		domainPath := filepath.Join(root, domainName)
		projects = append(projects, collect(root, domainName, domainPath)...)
	}
	return projects, nil
}

func ScanRoots(configDir string, roots []string, collect ScanFunc) ([]Project, error) {
	var all []Project
	for _, root := range roots {
		ps, err := ScanRoot(root, collect)
		if err != nil {
			continue
		}
		all = append(all, ps...)
	}
	SaveIndex(configDir, all)
	return all, nil
}

func ScanDomainsInRoot(root string) ([]string, error) {
	ignored := loadIgnoreList(root)
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var domains []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if isDomainIgnored(root, e.Name(), ignored) {
			continue
		}
		domains = append(domains, e.Name())
	}
	return domains, nil
}

func BuildGitHubIndex(projects []Project) map[string]Project {
	idx := make(map[string]Project)
	for _, p := range projects {
		if p.GitHubRepo != "" {
			idx[p.GitHubRepo] = p
		}
	}
	return idx
}

func BuildNameIndex(projects []Project) map[string]Project {
	idx := make(map[string]Project)
	for _, p := range projects {
		idx[p.Name] = p
	}
	return idx
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

func ScanGroupsInDomain(domainPath string, classify ClassifyDirFunc) []GroupInfo {
	entries, err := os.ReadDir(domainPath)
	if err != nil {
		return nil
	}
	var groups []GroupInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		childPath := filepath.Join(domainPath, e.Name())
		if classify(childPath, 0) == KindGroup {
			gi := GroupInfo{Name: e.Name(), Path: childPath}
			gi.Subgroups = scanSubgroups(childPath, 0, classify)
			groups = append(groups, gi)
		}
	}
	return groups
}

func scanSubgroups(groupPath string, depth int, classify ClassifyDirFunc) []SubgroupInfo {
	if depth > 3 {
		return nil
	}
	entries, err := os.ReadDir(groupPath)
	if err != nil {
		return nil
	}
	var subs []SubgroupInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		childPath := filepath.Join(groupPath, e.Name())
		if classify(childPath, 0) == KindGroup {
			si := SubgroupInfo{Name: e.Name(), Path: childPath}
			si.Subgroups = scanSubgroups(childPath, depth+1, classify)
			subs = append(subs, si)
		}
	}
	return subs
}

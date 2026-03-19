package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/pelletier/go-toml/v2"
	"github.com/richardnascimento18/devdock/internal/core"
)

// ---------------------------------------------------------------------------
// Scan cache — avoids repeated filesystem reads on tab/space
// ---------------------------------------------------------------------------

var (
	classifyCache  sync.Map // map[string]core.ProjectKind
	languagesCache sync.Map // map[string][]string
)

// ClearCache invalidates all cached classify and language results.
// Call this before an explicit rescan (e.g. the "r" key).
func ClearCache() {
	classifyCache.Range(func(k, _ any) bool { classifyCache.Delete(k); return true })
	languagesCache.Range(func(k, _ any) bool { languagesCache.Delete(k); return true })
}

func sniffPackageJSON(projectPath string) []string {
	data, err := os.ReadFile(filepath.Join(projectPath, "package.json"))
	if err != nil {
		return nil
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	allDeps := make(map[string]bool)
	for k := range pkg.Dependencies {
		allDeps[k] = true
	}
	for k := range pkg.DevDependencies {
		allDeps[k] = true
	}
	seen := map[string]bool{}
	var found []string
	for _, fd := range frameworkDeps {
		if allDeps[fd.dep] && !seen[fd.label] {
			seen[fd.label] = true
			found = append(found, fd.label)
		}
	}
	return found
}

func sniffRequirementsTxt(projectPath string) []string {
	data, err := os.ReadFile(filepath.Join(projectPath, "requirements.txt"))
	if err != nil {
		return nil
	}
	content := strings.ToLower(string(data))
	seen := map[string]bool{}
	var found []string
	for _, pm := range pythonFrameworkMarkers {
		if strings.Contains(content, pm.name) && !seen[pm.label] {
			seen[pm.label] = true
			found = append(found, pm.label)
		}
	}
	return found
}

func sniffPyproject(projectPath string) []string {
	data, err := os.ReadFile(filepath.Join(projectPath, "pyproject.toml"))
	if err != nil {
		return nil
	}
	content := strings.ToLower(string(data))
	seen := map[string]bool{}
	var found []string
	for _, pm := range pythonFrameworkMarkers {
		if strings.Contains(content, pm.name) && !seen[pm.label] {
			seen[pm.label] = true
			found = append(found, pm.label)
		}
	}
	return found
}

// Languages performs broad detection and returns a prioritised, deduplicated list of tech labels.
func Languages(projectPath string) []string {
	if v, ok := languagesCache.Load(projectPath); ok {
		return v.([]string)
	}

	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return nil
	}
	fileSet := make(map[string]bool)
	dirSet := make(map[string]bool)
	extCount := make(map[string]int)
	totalFiles := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			dirSet[name] = true
		} else {
			fileSet[name] = true
			ext := strings.ToLower(filepath.Ext(name))
			if ext != "" {
				extCount[ext]++
				totalFiles++
			}
		}
	}
	seen := make(map[string]bool)
	var results []string
	add := func(label string) {
		if !seen[label] {
			seen[label] = true
			results = append(results, label)
		}
	}
	for _, m := range primaryMarkers {
		if strings.HasPrefix(m.file, "*") {
			suffix := m.file[1:]
			for f := range fileSet {
				if strings.HasSuffix(f, suffix) {
					add(m.label)
					break
				}
			}
		} else if strings.HasPrefix(m.file, ".") && !strings.Contains(m.file, ".") {
			// skip — handled via suffix match above
		} else {
			if fileSet[m.file] {
				add(m.label)
			}
		}
	}
	if fileSet["package.json"] {
		for _, label := range sniffPackageJSON(projectPath) {
			add(label)
		}
		hasJSFramework := seen["Next.js"] || seen["Nuxt"] || seen["Vue"] || seen["Svelte"] ||
			seen["SvelteKit"] || seen["Angular"] || seen["Astro"] || seen["Remix"] ||
			seen["Solid"] || seen["React"] || seen["React Native"] || seen["Electron"] ||
			seen["Express"] || seen["Nest.js"]
		if !hasJSFramework {
			add("Node")
		}
	}
	if seen["Python"] {
		for _, label := range sniffRequirementsTxt(projectPath) {
			add(label)
		}
		for _, label := range sniffPyproject(projectPath) {
			add(label)
		}
	}
	for _, m := range secondaryMarkers {
		if strings.HasPrefix(m.file, "*") {
			suffix := m.file[1:]
			for f := range fileSet {
				if strings.HasSuffix(f, suffix) {
					add(m.label)
					break
				}
			}
		} else if fileSet[m.file] {
			add(m.label)
		}
	}
	for _, dm := range directoryMarkers {
		if dm.label != "" && dirSet[dm.dir] {
			add(dm.label)
		}
	}
	if dirSet["android"] {
		if seen["Dart"] || fileSet["pubspec.yaml"] {
			add("Flutter")
		} else if seen["React"] || seen["React Native"] {
			add("React Native")
		}
	}
	if len(results) == 0 && totalFiles > 0 {
		type extEntry struct {
			ext   string
			count int
		}
		var sorted []extEntry
		for ext, cnt := range extCount {
			sorted = append(sorted, extEntry{ext, cnt})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].count > sorted[j].count
		})
		for _, ee := range sorted {
			if label, ok := extensionMap[ee.ext]; ok {
				if float64(ee.count)/float64(totalFiles) >= 0.2 {
					add(label)
				}
			}
		}
	}
	hasFramework := seen["React"] || seen["Next.js"] || seen["Vue"] || seen["Nuxt"] ||
		seen["Svelte"] || seen["SvelteKit"] || seen["Angular"] || seen["Astro"] ||
		seen["Remix"] || seen["Solid"] || seen["Express"] || seen["Nest.js"] || seen["Electron"]
	if hasFramework {
		filtered := results[:0]
		for _, r := range results {
			if r != "JavaScript" && r != "TypeScript" && r != "Node" {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}
	if hasFramework && (seen["HTML"] || seen["CSS"]) {
		filtered := results[:0]
		for _, r := range results {
			if r != "HTML" && r != "CSS" {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	languagesCache.Store(projectPath, results)
	return results
}

// DevDockMarker is the parsed .devdock marker file.
type DevDockMarker struct {
	Type string `toml:"type"`
	Name string `toml:"name"`
}

func ReadDevDockMarker(dir string) *DevDockMarker {
	data, err := os.ReadFile(filepath.Join(dir, ".devdock"))
	if err != nil {
		return nil
	}
	var m DevDockMarker
	if err := toml.Unmarshal(data, &m); err != nil {
		return &DevDockMarker{Type: "project"}
	}
	return &m
}

const maxScanDepth = 5

func ClassifyDir(path string, depth int) core.ProjectKind {
	if depth > maxScanDepth {
		return ""
	}

	// Only cache at depth 0 — recursive calls are internal and short-lived.
	if depth == 0 {
		if v, ok := classifyCache.Load(path); ok {
			return v.(core.ProjectKind)
		}
	}

	var result core.ProjectKind

	if _, err := os.Stat(filepath.Join(path, ".ddgroup")); err == nil {
		result = core.KindGroup
	} else {
		marker := ReadDevDockMarker(path)
		if marker != nil {
			if marker.Type == "group" {
				result = core.KindGroup
			} else {
				result = core.KindProject
			}
		} else if langs := Languages(path); len(langs) > 0 {
			result = core.KindProject
		} else {
			entries, err := os.ReadDir(path)
			if err != nil {
				result = ""
			} else {
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					sub := filepath.Join(path, e.Name())
					if k := ClassifyDir(sub, depth+1); k == core.KindProject {
						result = core.KindGroup
						break
					}
				}
			}
		}
	}

	if depth == 0 {
		classifyCache.Store(path, result)
	}
	return result
}

func CollectProjects(root, domain, domainPath string) []core.Project {
	return collectAt(root, domain, domainPath, "", "", 0)
}

func collectAt(root, domain, dir, group, subgroup string, depth int) []core.Project {
	if depth > maxScanDepth {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var projects []core.Project
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		childPath := filepath.Join(dir, e.Name())
		kind := ClassifyDir(childPath, 0)
		switch kind {
		case core.KindProject:
			langs := Languages(childPath)
			name := e.Name()
			if m := ReadDevDockMarker(childPath); m != nil && m.Name != "" {
				name = m.Name
			}
			projects = append(projects, core.Project{
				Name:      name,
				Path:      childPath,
				Domain:    domain,
				Root:      root,
				Languages: langs,
				Group:     group,
				Subgroup:  subgroup,
				Kind:      core.KindProject,
			})
		case core.KindGroup:
			newGroup := group
			newSub := subgroup
			if group == "" {
				newGroup = e.Name()
			} else {
				newSub = e.Name()
			}
			children := collectAt(root, domain, childPath, newGroup, newSub, depth+1)
			projects = append(projects, children...)
		}
	}
	return projects
}

package detect

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/richardnascimento18/devdock/internal/core"
)

// Detector owns local marker inspection; recursive traversal belongs to core.
type Detector struct{}

func New() *Detector                 { return &Detector{} }
func Languages(path string) []string { return New().Languages(path) }

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

func dependencyName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "#") {
		return ""
	}
	end := strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.')
	})
	if end >= 0 {
		value = value[:end]
	}
	return strings.ToLower(strings.ReplaceAll(value, "_", "-"))
}
func pythonLabels(names []string) []string {
	seen := map[string]bool{}
	var found []string
	for _, value := range names {
		name := dependencyName(value)
		for _, marker := range pythonFrameworkMarkers {
			if name == marker.name && !seen[marker.label] {
				seen[marker.label] = true
				found = append(found, marker.label)
			}
		}
	}
	return found
}
func sniffRequirementsTxt(path string) []string {
	data, err := os.ReadFile(filepath.Join(path, "requirements.txt"))
	if err != nil {
		return nil
	}
	return pythonLabels(strings.Split(string(data), "\n"))
}
func sniffPyproject(path string) []string {
	data, err := os.ReadFile(filepath.Join(path, "pyproject.toml"))
	if err != nil {
		return nil
	}
	var project struct {
		Project struct {
			Dependencies         []string
			OptionalDependencies map[string][]string `toml:"optional-dependencies"`
		}
		Tool struct {
			Poetry struct{ Dependencies map[string]any }
		}
	}
	if err := toml.Unmarshal(data, &project); err != nil {
		return nil
	}
	names := append([]string(nil), project.Project.Dependencies...)
	var optional []string
	for key := range project.Project.OptionalDependencies {
		optional = append(optional, key)
	}
	sort.Strings(optional)
	for _, key := range optional {
		names = append(names, project.Project.OptionalDependencies[key]...)
	}
	var poetry []string
	for name := range project.Tool.Poetry.Dependencies {
		poetry = append(poetry, name)
	}
	sort.Strings(poetry)
	names = append(names, poetry...)
	return pythonLabels(names)
}

// Languages performs broad detection and returns a prioritised, deduplicated list of tech labels.
func (d *Detector) Languages(projectPath string) []string {
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return nil
	}
	return d.languagesFromEntries(projectPath, entries)
}
func (d *Detector) languagesFromEntries(projectPath string, entries []os.DirEntry) []string {
	fileSet := make(map[string]bool)
	dirSet := make(map[string]bool)
	extCount := make(map[string]int)
	totalFiles := 0
	for _, e := range entries {
		if e.Type()&os.ModeSymlink != 0 {
			continue
		}
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
			seen["Express"] || seen["Fastify"] || seen["Nest.js"]
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
			if sorted[i].count == sorted[j].count {
				return sorted[i].ext < sorted[j].ext
			}
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
		seen["Remix"] || seen["Solid"] || seen["Express"] || seen["Fastify"] || seen["Nest.js"] || seen["Electron"]
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

	return results
}

// DevDockMarker is the parsed .devdock marker file.
type DevDockMarker struct {
	Type string `toml:"type"`
	Name string `toml:"name"`
}

func ReadDevDockMarker(dir string) (*DevDockMarker, error) {
	data, err := os.ReadFile(filepath.Join(dir, ".devdock"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var m DevDockMarker
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid .devdock marker: %w", err)
	}
	if m.Type != "project" && m.Type != "group" && m.Type != "subgroup" {
		return nil, fmt.Errorf("unknown .devdock type %q", m.Type)
	}
	return &m, nil
}

// Inspect classifies only this directory. A blank kind means traversal may
// discover an implicit group; it never scans descendants to classify a parent.
func (d *Detector) Inspect(path string, entries []os.DirEntry) (core.Discovery, error) {
	for _, entry := range entries {
		if entry.Name() == ".ddgroup" && entry.Type()&os.ModeSymlink == 0 {
			return core.Discovery{Kind: core.KindGroup, Explicit: true}, nil
		}
	}
	var marker *DevDockMarker
	for _, entry := range entries {
		if entry.Name() == ".devdock" {
			if entry.Type()&os.ModeSymlink != 0 {
				return core.Discovery{}, fmt.Errorf("symlinked .devdock marker in %q", path)
			}
			var err error
			marker, err = ReadDevDockMarker(path)
			if err != nil {
				return core.Discovery{}, err
			}
		}
	}
	if marker != nil && marker.Type != "project" {
		return core.Discovery{Kind: core.KindGroup, Explicit: true}, nil
	}
	languages := d.languagesFromEntries(path, entries)
	if marker != nil || len(languages) > 0 {
		result := core.Discovery{Kind: core.KindProject, Languages: languages, Explicit: marker != nil}
		if marker != nil {
			result.Name = marker.Name
		}
		return result, nil
	}
	for _, entry := range entries {
		if entry.Name() == ".git" && entry.Type()&os.ModeSymlink == 0 {
			return core.Discovery{Kind: core.KindProject}, nil
		}
	}
	return core.Discovery{}, nil
}

package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/richardnascimento18/devdock/internal/core"
)

func writeFixture(t testing.TB, path, name, content string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
func TestDeepDiscoverySingleTraversal(t *testing.T) {
	for _, depth := range []int{25, 50, 200} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			root := t.TempDir()
			loc := core.Location{Root: root, Domain: "apps"}
			for i := 0; i < depth; i++ {
				loc = loc.Child("g")
			}
			writeFixture(t, filepath.Join(loc.Path(), "api"), "go.mod", "module api")
			visits := map[string]int{}
			d := New()
			w, err := core.ScanWorkspace([]string{root}, func(path string, entries []os.DirEntry) (core.Discovery, error) {
				visits[path]++
				return d.Inspect(path, entries)
			})
			if err != nil || len(w.Projects) != 1 {
				t.Fatalf("scan: %d %v", len(w.Projects), err)
			}
			if !reflect.DeepEqual(w.Projects[0].Location, loc) {
				t.Fatalf("ancestry: %+v", w.Projects[0])
			}
			if len(visits) != depth+1 {
				t.Fatalf("visits %d", len(visits))
			}
			for path, count := range visits {
				if count != 1 {
					t.Fatalf("revisited %s: %d", path, count)
				}
			}
			if len(w.Locations(root, "apps")) != depth+1 {
				t.Fatal("picker ancestry missing")
			}
		})
	}
}
func TestMixedHierarchyBoundariesAndMarkers(t *testing.T) {
	root := t.TempDir()
	domain := filepath.Join(root, "apps")
	for _, path := range []string{"projectA", "groupA/projectB", "groupA/groupB/projectC", "groupA/groupB/groupC/projectD", "backend/services/api", "frontend/services/api"} {
		writeFixture(t, filepath.Join(domain, path), "go.mod", "module demo")
	}
	writeFixture(t, filepath.Join(domain, "empty", "nested"), ".ddgroup", "")
	writeFixture(t, filepath.Join(domain, "legacy"), ".devdock", "type='subgroup'")
	writeFixture(t, filepath.Join(domain, "marker"), ".devdock", "type='project'\nname='Alias'")
	for _, internal := range []string{"src/module", "node_modules/module", "vendor/module", "build/module", "nestedProject"} {
		writeFixture(t, filepath.Join(domain, "projectA", internal), "go.mod", "module internal")
	}
	writeFixture(t, filepath.Join(domain, "git-only"), ".git", "gitdir: elsewhere")
	w, err := core.ScanWorkspace([]string{root}, New().Inspect)
	if err != nil || len(w.Projects) != 8 {
		t.Fatalf("projects %+v: %v", w.Projects, err)
	}
	counts := map[string]int{}
	keys := map[core.NodeKey]bool{}
	for _, row := range w.Flatten(nil, "", false) {
		if row.Node != nil {
			if keys[row.Node.Key()] {
				t.Fatal("node collision")
			}
			keys[row.Node.Key()] = true
		}
		if row.Project != nil {
			counts[row.Project.Name]++
		}
	}
	if counts["api"] != 2 || counts["Alias"] != 1 {
		t.Fatalf("names %v", counts)
	}
	if len(w.Locations(root, "apps")) != 11 {
		t.Fatalf("locations %d", len(w.Locations(root, "apps")))
	}
}
func TestDeepIgnoreAndSymlinkSafety(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	domain := filepath.Join(root, "apps")
	for _, path := range []string{"a/b/keep", "a/b/skip", "a/skip/also"} {
		writeFixture(t, filepath.Join(domain, path), "go.mod", "")
	}
	writeFixture(t, root, ".ddignore", "apps/a/b/skip\napps/a/skip/\n")
	writeFixture(t, outside, "go.mod", "")
	for path, target := range map[string]string{
		filepath.Join(domain, "escape"):                          outside,
		filepath.Join(domain, "a", "b", "cycle"):                 domain,
		filepath.Join(domain, "a", "alias"):                      filepath.Join(domain, "a", "b"),
		filepath.Join(domain, "a", "b", "keep", "internal-link"): domain,
	} {
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
	}
	w, err := core.ScanWorkspace([]string{root}, New().Inspect)
	if err != nil || len(w.Projects) != 1 || w.Projects[0].Name != "keep" {
		t.Fatalf("symlinks/ignore: %+v %v", w.Projects, err)
	}
}
func TestNestedErrorsPreserveSuccessfulDiscovery(t *testing.T) {
	root := t.TempDir()
	domain := filepath.Join(root, "apps")
	writeFixture(t, filepath.Join(domain, "a", "b", "good"), "go.mod", "")
	writeFixture(t, filepath.Join(domain, "a", "b", "bad"), ".devdock", "type='unknown'")
	w, err := core.ScanWorkspace([]string{root, filepath.Join(root, "missing")}, New().Inspect)
	if err == nil || len(w.Projects) != 1 || !strings.Contains(err.Error(), "bad") || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("partial %v %v", w.Projects, err)
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses unreadable fixture")
	}
	blocked := filepath.Join(domain, "a", "b", "blocked")
	if err := os.Mkdir(blocked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(blocked, 0o755); err != nil {
			t.Error(err)
		}
	})
	w, err = core.ScanWorkspace([]string{root}, New().Inspect)
	if err == nil || len(w.Projects) != 1 || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("unreadable: %v %v", w.Projects, err)
	}
}
func BenchmarkWorkspaceScan(b *testing.B) {
	for _, fixture := range []struct {
		name       string
		wide, deep int
	}{{"wide-1000", 1000, 1}, {"deep-200", 1, 200}, {"mixed-1000", 1000, 20}} {
		b.Run(fixture.name, func(b *testing.B) {
			root := b.TempDir()
			for i := 0; i < fixture.wide; i++ {
				loc := core.Location{Root: root, Domain: "apps"}
				for d := 0; d < fixture.deep; d++ {
					loc = loc.Child("g")
				}
				writeFixture(b, filepath.Join(loc.Path(), fmt.Sprintf("p%d", i)), "go.mod", "")
			}
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				w, err := core.ScanWorkspace([]string{root}, New().Inspect)
				if err != nil || len(w.Projects) != fixture.wide {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestExplicitMarkersOverrideContainerNames(t *testing.T) {
	root := t.TempDir()
	// Domain names are not dependency/output directory names.
	writeFixture(t, filepath.Join(root, "build", "src"), ".devdock", "type='project'")
	writeFixture(t, filepath.Join(root, "build", "vendor"), ".ddgroup", "")
	writeFixture(t, filepath.Join(root, "build", "vendor", "api"), "go.mod", "")
	writeFixture(t, filepath.Join(root, "build", "node_modules", "accidental"), "go.mod", "")
	w, err := core.ScanWorkspace([]string{root}, New().Inspect)
	if err != nil || len(w.Projects) != 2 || len(w.Locations(root, "build")) != 2 {
		t.Fatalf("explicit names: %+v %v", w, err)
	}
}

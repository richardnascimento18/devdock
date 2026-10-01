package detect

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/richardnascimento18/devdock/internal/core"
)

func TestDetectorFixtures(t *testing.T) {
	cases := []struct {
		name         string
		files        map[string]string
		want, absent []string
	}{
		{"go", map[string]string{"go.mod": "module example"}, []string{"Go"}, nil},
		{"rust", map[string]string{"Cargo.toml": "[package]"}, []string{"Rust"}, nil},
		{"csharp", map[string]string{"App.csproj": ""}, []string{"C#"}, nil},
		{"fsharp", map[string]string{"App.fsproj": ""}, []string{"F#"}, []string{"C#"}},
		{"solid", map[string]string{"package.json": `{"dependencies":{"@solidjs/start":"1"}}`}, []string{"Solid"}, []string{"SvelteKit", "Node"}},
		{"fastify", map[string]string{"package.json": `{"dependencies":{"fastify":"1"}}`}, []string{"Fastify"}, []string{"Express", "Node"}},
		{"svelte", map[string]string{"svelte.config.js": ""}, []string{"Svelte"}, []string{"SvelteKit"}},
		{"kit", map[string]string{"package.json": `{"devDependencies":{"@sveltejs/kit":"1"}}`}, []string{"SvelteKit"}, nil},
		{"requirements", map[string]string{"requirements.txt": "# django\nnot-flask==1\nfastapi>=1\n"}, []string{"Python", "FastAPI"}, []string{"Django", "Flask"}},
		{"pyproject", map[string]string{"pyproject.toml": "[project]\ndescription='django tutorial'\ndependencies=['flask>=1']\n"}, []string{"Python", "Flask"}, []string{"Django"}},
		{"poetry", map[string]string{"pyproject.toml": "[tool.poetry.dependencies]\ndjango='*'\n"}, []string{"Django"}, nil},
		{"generic-wsgi", map[string]string{"wsgi.py": ""}, []string{"Python"}, []string{"Django"}},
		{"unknown", map[string]string{"README.md": ""}, nil, []string{"Python", "Go", "Node"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			labels := Languages(dir)
			for _, want := range tc.want {
				if !slices.Contains(labels, want) {
					t.Errorf("missing %s in %v", want, labels)
				}
			}
			for _, bad := range tc.absent {
				if slices.Contains(labels, bad) {
					t.Errorf("false positive %s in %v", bad, labels)
				}
			}
		})
	}
}
func TestScanGroupsAndMarkers(t *testing.T) {
	root := t.TempDir()
	domain := filepath.Join(root, "apps")
	paths := []string{filepath.Join(domain, "direct"), filepath.Join(domain, "group", "sub", "nested"), filepath.Join(domain, "empty"), filepath.Join(domain, "ignored")}
	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range paths[:2] {
		if err := os.WriteFile(filepath.Join(p, "go.mod"), []byte("module demo"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(paths[2], ".ddgroup"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(domain, "direct", ".devdock"), []byte("type='project'\nname='alias'"), 0o644); err != nil {
		t.Fatal(err)
	}
	detector := New()
	projects, err := core.ScanRoot(root, detector.CollectProjects)
	if err != nil || len(projects) != 2 {
		t.Fatalf("scan: %v %v", projects, err)
	}
	if projects[0].Name != "alias" || projects[1].Group != "group" || projects[1].Subgroup != "sub" {
		t.Fatalf("classification: %+v", projects)
	}
	groups, err := core.ScanGroupsInDomain(domain, detector.ClassifyDir)
	if err != nil || len(groups) != 2 {
		t.Fatalf("groups: %v %v", groups, err)
	}
	if err := os.WriteFile(filepath.Join(paths[3], ".devdock"), []byte("not toml!"), 0o644); err != nil {
		t.Fatal(err)
	}
	projects, err = core.ScanRoot(root, New().CollectProjects)
	if err == nil || len(projects) != 2 {
		t.Fatalf("marker failure suppressed: %v %v", projects, err)
	}
}
func TestCacheIsScopedAndOrderingStable(t *testing.T) {
	dir := t.TempDir()
	if got := Languages(dir); len(got) != 0 {
		t.Fatal(got)
	}
	for _, name := range []string{"main.py", "main.go"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	first := Languages(dir)
	for i := 0; i < 10; i++ {
		if !reflect.DeepEqual(first, Languages(dir)) {
			t.Fatal("nondeterministic detection")
		}
	}
	if len(first) != 2 {
		t.Fatal("cached empty directory survived new scan")
	}
}

func TestAllPrimaryMarkers(t *testing.T) {
	for _, marker := range primaryMarkers {
		t.Run(marker.file, func(t *testing.T) {
			dir := t.TempDir()
			name := marker.file
			if name[0] == '*' {
				name = "Demo" + name[1:]
			}
			if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			if labels := Languages(dir); !slices.Contains(labels, marker.label) {
				t.Fatalf("marker %s: %v", marker.file, labels)
			}
			if kind := ClassifyDir(dir, 0); kind != core.KindProject {
				t.Fatalf("marker %s classified %s", marker.file, kind)
			}
		})
	}
}
func TestUnreadableDomainReturnsPartialProjects(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses filesystem permissions")
	}
	root := t.TempDir()
	good := filepath.Join(root, "apps", "demo")
	bad := filepath.Join(root, "blocked")
	if err := os.MkdirAll(good, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(good, "go.mod"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(bad, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(bad, 0o755); err != nil {
			t.Error(err)
		}
	})
	projects, err := core.ScanRoot(root, New().CollectProjects)
	if err == nil || len(projects) != 1 {
		t.Fatalf("unreadable domain: %v %v", projects, err)
	}
}

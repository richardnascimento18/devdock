package template

import (
	"github.com/richardnascimento18/devdock/internal/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseCommand(t *testing.T) {
	for _, tc := range []struct {
		in      string
		want    []string
		invalid bool
	}{
		{`command --title "Hello world"`, []string{"command", "--title", "Hello world"}, false},
		{`cmd '' 'a b' c\ d`, []string{"cmd", "", "a b", "c d"}, false},
		{`cmd "$HOME" '*.go' '$(id)'`, []string{"cmd", "$HOME", "*.go", "$(id)"}, false},
		{`cmd "unterminated`, nil, true}, {`cmd \`, nil, true}, {``, nil, true},
	} {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseCommand(tc.in)
			if (err != nil) != tc.invalid || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("%q %v", got, err)
			}
		})
	}
	args, err := CommandArgs(TemplateStep{Run: `cmd {{project_path}}`}, Vars{ProjectPath: `/a path/'quoted'`})
	if err != nil || !reflect.DeepEqual(args, []string{"cmd", `/a path/'quoted'`}) {
		t.Fatalf("vars: %q %v", args, err)
	}
}
func TestBuiltinSafety(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "keep"), []byte("safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{".", "..", "a/../..", "../keep", outside, "link/keep", "link", ""} {
		if err := ExecuteBuiltin("rm", path, root); err == nil {
			t.Errorf("accepted %q", path)
		}
	}
	for _, action := range []string{"touch", "mkdir"} {
		if err := ExecuteBuiltin(action, "link/escape", root); err == nil {
			t.Errorf("%s traversed symlink", action)
		}
	}
	if _, err := os.Stat(filepath.Join(outside, "keep")); err != nil {
		t.Fatal(err)
	}
	if err := ExecuteBuiltin("touch", "nested/file", root); err != nil {
		t.Fatal(err)
	}
	if err := ExecuteBuiltin("rm", "nested", root); err != nil {
		t.Fatal(err)
	}
}
func TestDefaultsRoundTripAndValidation(t *testing.T) {
	dir := t.TempDir()
	defaults, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil || !reflect.DeepEqual(defaults, loaded) {
		t.Fatalf("defaults: %v", err)
	}
	for _, collection := range [][]Template{
		{{Name: "same"}, {Name: "same"}},
		{{Name: "unsafe", Steps: []TemplateStep{{Type: "builtin", Action: "rm", Path: "."}}}},
		{{Name: "shell", Steps: []TemplateStep{{Type: "command", Run: "echo hi", Shell: true}}}},
		{{Name: "bad", Steps: []TemplateStep{{Type: "command", Run: `cmd "`}}}},
	} {
		if errs := ValidateFile(TemplateFile{Templates: collection}); len(errs) == 0 {
			t.Fatal("invalid collection accepted")
		}
	}
}
func TestRunStopsOnFailureAndCollision(t *testing.T) {
	root := t.TempDir()
	domain := filepath.Join(root, "apps")
	template := Template{Name: "fail", Steps: []TemplateStep{{Type: "command", Run: "false"}, {Type: "builtin", Action: "touch", Path: "after"}}, PostSteps: []TemplateStep{{Type: "builtin", Action: "touch", Path: "post"}}}
	if _, err := Run(template, core.Location{Root: root, Domain: "apps"}, "demo"); err == nil {
		t.Fatal("failure lost")
	}
	for _, name := range []string{"after", "post"} {
		if _, err := os.Stat(filepath.Join(domain, "demo", name)); !os.IsNotExist(err) {
			t.Fatalf("ran %s", name)
		}
	}
	if _, err := Run(Template{Name: "empty"}, core.Location{Root: root, Domain: "apps"}, "demo"); err == nil {
		t.Fatal("collision accepted")
	}
	if _, err := Run(Template{Name: "escape"}, core.Location{Root: root, Domain: "apps"}, "../escape"); err == nil {
		t.Fatal("path escape")
	}
}
func TestCommandOutput(t *testing.T) {
	root := t.TempDir()
	v := Vars{ProjectPath: root}
	if err := ExecuteSteps([]TemplateStep{{Type: "command", Run: `printf "Hello world"`, Output: "result"}}, root, v, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "result"))
	if err != nil || string(data) != "Hello world" {
		t.Fatalf("%q %v", data, err)
	}
	if _, err := CommandOutput(".", root); err == nil {
		t.Fatal("output accepted root")
	}
}

func TestMarkerWriteDoesNotFollowSymlink(t *testing.T) {
	project := t.TempDir()
	outside := filepath.Join(t.TempDir(), "keep")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(project, ".devdock")); err != nil {
		t.Fatal(err)
	}
	if err := WriteDevDockMarkerFile(project); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(outside)
	if err != nil || string(data) != "keep" {
		t.Fatal("marker followed symlink")
	}
}

func TestInternalParentTraversalIsRejected(t *testing.T) {
	for _, path := range []string{"a/../b", "link/a/../../b"} {
		if err := ExecuteBuiltin("rm", path, t.TempDir()); err == nil {
			t.Fatalf("accepted parent traversal %q", path)
		}
	}
}

func TestRunAtDeepLocation(t *testing.T) {
	root := t.TempDir()
	location := core.Location{Root: root, Domain: "apps"}
	for i := 0; i < 35; i++ {
		location = location.Child("g")
	}
	path, err := Run(Template{Name: "nested", Steps: []TemplateStep{{Type: "builtin", Action: "touch", Path: "created"}}}, location, "api")
	if err != nil || path != filepath.Join(location.Path(), "api") {
		t.Fatalf("template path %s: %v", path, err)
	}
	if _, err := os.Stat(filepath.Join(path, "created")); err != nil {
		t.Fatal(err)
	}
}

package core

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestCreateAndDeleteSafeguards(t *testing.T) {
	root := t.TempDir()
	p, err := CreateProject(root, "apps", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(root, "apps", "demo"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("collision: %v", err)
	}
	for _, name := range []string{"", ".", "..", "../other", "a/b", "a\\b", "a\x00b"} {
		if _, err := CreateProject(root, "apps", name); err == nil {
			t.Errorf("accepted %q", name)
		}
		if err := DeleteDomain(root, name); err == nil {
			t.Errorf("deleted %q", name)
		}
	}
	if err := DeletePath(root, root); err == nil {
		t.Fatal("allowed root removal")
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(root, "link", "demo"); err == nil {
		t.Fatal("followed symlink")
	}
	if err := DeletePath(root, filepath.Join(root, "link", "child")); err == nil {
		t.Fatal("allowed symlink traversal")
	}
	if err := DeleteProject(p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p.Path); !os.IsNotExist(err) {
		t.Fatal("project remains")
	}
}

func TestMove(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "rename", true: "cross-device"}[fallback], func(t *testing.T) {
			root := t.TempDir()
			src := filepath.Join(root, "src")
			dst := filepath.Join(root, "dst")
			if err := os.Mkdir(src, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(src, "data"), []byte("payload"), 0o640); err != nil {
				t.Fatal(err)
			}
			outside := t.TempDir()
			for link, target := range map[string]string{"relative": "data", "external": outside, "broken": "missing"} {
				if err := os.Symlink(target, filepath.Join(src, link)); err != nil {
					t.Fatal(err)
				}
			}
			rename := renameExclusive
			if fallback {
				rename = func(string, string) error { return syscall.EXDEV }
			}
			if err := moveDir(src, dst, rename, os.RemoveAll); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(dst, "data"))
			if err != nil || string(data) != "payload" {
				t.Fatalf("copy: %q %v", data, err)
			}
			for link, target := range map[string]string{"relative": "data", "external": outside, "broken": "missing"} {
				got, err := os.Readlink(filepath.Join(dst, link))
				if err != nil || got != target {
					t.Fatalf("link %s: %s %v", link, got, err)
				}
			}
			if _, err := os.Stat(src); !os.IsNotExist(err) {
				t.Fatal("source remains")
			}
		})
	}
}

func TestMoveFailurePreservesData(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := moveDir(src, dst, renameExclusive, os.RemoveAll); !errors.Is(err, os.ErrExist) {
		t.Fatalf("collision: %v", err)
	}
	if err := os.Remove(dst); err != nil {
		t.Fatal(err)
	}
	denied := errors.New("denied")
	if err := moveDir(src, dst, func(string, string) error { return denied }, os.RemoveAll); !errors.Is(err, denied) {
		t.Fatal(err)
	}
	err := moveDir(src, dst, func(string, string) error { return syscall.EXDEV }, func(string) error { return denied })
	var partial *PartialMoveError
	if !errors.As(err, &partial) || !errors.Is(err, denied) {
		t.Fatalf("cleanup failure: %v", err)
	}
	for _, path := range []string{src, dst} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCopyUnsupportedFilePreservesSource(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(src, "pipe"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := moveDir(src, dst, func(string, string) error { return syscall.EXDEV }, os.RemoveAll); err == nil {
		t.Fatal("accepted FIFO")
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("partial destination published")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("staging leak: %v %v", entries, err)
	}
}

func TestScanPartialFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "apps"), 0o755); err != nil {
		t.Fatal(err)
	}
	collect := func(root, domain, path string) ([]Project, error) {
		return []Project{{Name: "demo", Root: root, Domain: domain}}, nil
	}
	projects, err := ScanRoots([]string{filepath.Join(root, "missing"), root}, collect)
	if err == nil || len(projects) != 1 {
		t.Fatalf("partial scan: %v %v", projects, err)
	}
	if err := os.WriteFile(filepath.Join(root, ".ddignore"), []byte("apps\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	projects, err = ScanRoot(root, collect)
	if err != nil || len(projects) != 0 {
		t.Fatalf("ignore: %v %v", projects, err)
	}
	if err := os.WriteFile(filepath.Join(root, ".ddignore"), []byte("[\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ScanRoot(root, collect); err == nil {
		t.Fatal("invalid glob suppressed")
	}
}

func TestMoveProjectUpdatesCurrentHierarchyMetadata(t *testing.T) {
	source, destination := t.TempDir(), t.TempDir()
	p, err := CreateProject(source, "apps", "demo")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(destination, "tools", "group", "sub", "demo")
	moved, err := MoveProject(p, path, destination, "tools")
	if err != nil {
		t.Fatal(err)
	}
	if moved.Root != destination || moved.Domain != "tools" || moved.Group != "group" || moved.Subgroup != "sub" || moved.Path != path {
		t.Fatalf("metadata: %+v", moved)
	}
	if _, err := MoveProject(moved, filepath.Join(destination, "different", "demo"), destination, "tools"); err == nil {
		t.Fatal("move escaped selected domain")
	}
	if err := DeleteProject(Project{Root: destination, Domain: "tools", Path: filepath.Join(destination, "tools")}); err == nil {
		t.Fatal("project deletion allowed entire domain")
	}
}

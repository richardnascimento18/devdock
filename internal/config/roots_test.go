package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRootLexicalOwnership(t *testing.T) {
	base := t.TempDir()
	for _, tc := range []struct {
		name     string
		roots    []string
		relation RootRelation
	}{
		{"duplicate", []string{"work", "work"}, RootSame},
		{"parent-child", []string{"work", "work/client"}, RootInside},
		{"child-parent", []string{"work/client", "work"}, RootContains},
		{"prefix-siblings", []string{"code", "code2"}, ""},
		{"siblings", []string{"work/a", "work/b"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			roots := []string{filepath.Join(base, tc.roots[0]), filepath.Join(base, tc.roots[1])}
			identities, err := ValidateRoots(roots)
			if tc.relation == "" {
				if err != nil || len(identities) != 2 {
					t.Fatal(err)
				}
				for _, identity := range identities {
					if identity.ResolutionErr == nil || identity.CanonicalPath != "" {
						t.Fatal("missing roots falsely resolved")
					}
				}
			} else {
				var conflict *RootConflictError
				if !errors.As(err, &conflict) || conflict.Relation != tc.relation || conflict.Root != roots[1] || conflict.Existing != roots[0] || conflict.Physical {
					t.Fatalf("unexpected conflict: %v", err)
				}
			}
		})
	}
	for _, path := range []string{"relative", base + "/", base + "/./work", base + "/work/../client"} {
		if _, err := ValidateRoots([]string{path}); err == nil {
			t.Fatalf("unclean/relative path accepted: %s", path)
		}
	}
}

func TestRootPhysicalOwnership(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	child := filepath.Join(real, "client")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(base, "alias")
	childAlias := filepath.Join(base, "child-alias")
	for link, target := range map[string]string{alias: real, childAlias: child} {
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name     string
		roots    []string
		relation RootRelation
	}{
		{"same", []string{real, alias}, RootSame},
		{"alias-child", []string{alias, child}, RootInside},
		{"child-alias-parent", []string{child, alias}, RootContains},
		{"parent-child-alias", []string{real, childAlias}, RootInside},
		{"child-alias-parent-real", []string{childAlias, real}, RootContains},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateRoots(tc.roots)
			var conflict *RootConflictError
			if !errors.As(err, &conflict) || !conflict.Physical || conflict.Relation != tc.relation || !strings.Contains(err.Error(), tc.roots[0]) || !strings.Contains(err.Error(), tc.roots[1]) {
				t.Fatalf("unexpected physical conflict: %v", err)
			}
		})
	}
}

func TestUnresolvedRootIdentityRemainsUnknown(t *testing.T) {
	base := t.TempDir()
	broken := filepath.Join(base, "broken")
	if err := os.Symlink(filepath.Join(base, "absent"), broken); err != nil {
		t.Fatal(err)
	}
	loop := filepath.Join(base, "loop")
	if err := os.Symlink(loop, loop); err != nil {
		t.Fatal(err)
	}
	roots := []string{filepath.Join(base, "missing"), broken, loop}
	identities, err := ValidateRoots(roots)
	if err != nil || len(identities) != len(roots) {
		t.Fatal(err)
	}
	for _, identity := range identities {
		if identity.ResolutionErr == nil || identity.CanonicalPath != "" || !strings.Contains(identity.ResolutionErr.Error(), identity.Path) {
			t.Fatal("canonicalization failure lost")
		}
	}
	t.Setenv("HOME", t.TempDir())
	cfg := Config{Roots: roots}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load()
	if err != nil || !reflect.DeepEqual(loaded, cfg) {
		t.Fatalf("unavailable roots rejected/removed: %+v %v", loaded, err)
	}
	after, err := os.ReadFile(Path())
	if err != nil || string(before) != string(after) {
		t.Fatal("load rewrote unavailable roots")
	}
}

func TestPermissionFailureDoesNotInvalidateRootConfig(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}
	base := t.TempDir()
	blocked := filepath.Join(base, "blocked")
	root := filepath.Join(blocked, "workspace")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(blocked, 0o755); err != nil {
			t.Error(err)
		}
	})
	identities, err := ValidateRoots([]string{root})
	if err != nil || len(identities) != 1 || !errors.Is(identities[0].ResolutionErr, os.ErrPermission) {
		t.Fatalf("permission uncertainty lost: %+v %v", identities, err)
	}
	if err := Validate(Config{Roots: []string{root}}); err != nil {
		t.Fatal("temporarily inaccessible root invalidated config", err)
	}
}

func TestLegacyConflictsPreserveConfigAndProposal(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	real := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(real, alias); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, roots := range [][]string{{real, filepath.Join(real, "client")}, {real, alias}} {
		data := []byte("roots = ['" + roots[0] + "', '" + roots[1] + "']\n")
		if err := os.WriteFile(Path(), data, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Load()
		var conflict *RootConflictError
		if !errors.As(err, &conflict) || !strings.Contains(err.Error(), Path()) || !strings.Contains(err.Error(), "restart") {
			t.Fatalf("legacy conflict lacks actionable error: %v", err)
		}
		after, err := os.ReadFile(Path())
		if err != nil || string(after) != string(data) {
			t.Fatal("legacy config silently rewritten")
		}
	}
	current := Config{Roots: []string{real}}
	before := current.Clone()
	for _, root := range []string{real, filepath.Join(real, "client"), alias} {
		if err := current.AddRoot(root); err == nil {
			t.Fatal("conflicting addition accepted")
		}
		if !reflect.DeepEqual(current, before) {
			t.Fatal("failed addition changed config")
		}
	}
}

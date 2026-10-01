package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/richardnascimento18/devdock/internal/core"
)

func TestStatePersistenceAndMalformedFile(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.ToggleFavorite("/project")
	if err := Save(dir, s); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil || !got.Favorites["/project"] {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath(dir), []byte(`{"favorites":{"/partial":true},"recents":`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = Load(dir)
	if err == nil || len(got.Favorites) != 0 {
		t.Fatalf("partial malformed state: %+v %v", got, err)
	}
	if err := Save(filepath.Join(dir, "state.json", "blocked"), s); err == nil {
		t.Fatal("save error suppressed")
	}
}
func TestMoveDeleteAndReconcileState(t *testing.T) {
	s, _ := Load(t.TempDir())
	old := core.Project{Path: "/root/a", Name: "a", Root: "/root", Domain: "old"}
	s.ToggleFavorite(old.Path)
	s.AddRecent(old)
	s.AddRecent(old)
	if len(s.Recents) != 1 {
		t.Fatal("duplicate recent")
	}
	next := core.Project{Path: "/other/a", Name: "a", Root: "/other", Domain: "new"}
	s.MoveProject(old.Path, next)
	if s.Favorites[old.Path] || !s.Favorites[next.Path] || s.Recents[0].Domain != "new" {
		t.Fatal("stale move state")
	}
	s.RemovePath("/root")
	if !s.Favorites[next.Path] {
		t.Fatal("pruned another root")
	}
	s.RemovePath("/other")
	if len(s.Favorites) != 0 || len(s.Recents) != 0 {
		t.Fatal("delete state remains")
	}
	dir := t.TempDir()
	s.ToggleFavorite(dir)
	s.ToggleFavorite(filepath.Join(dir, "missing"))
	s.ReconcileMissing()
	if len(s.Favorites) != 1 || !s.Favorites[dir] {
		t.Fatal("missing reconciliation")
	}
	clone := s.Clone()
	clone.ToggleFavorite("/new")
	if s.Favorites["/new"] {
		t.Fatal("state clone aliases")
	}
}

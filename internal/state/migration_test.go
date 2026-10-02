package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/richardnascimento18/devdock/internal/core"
)

func TestLegacyMigrationAndIdempotence(t *testing.T) {
	dir := t.TempDir()
	root := t.TempDir()
	loc := core.Location{Root: root, Domain: "work", GroupPath: []string{"backend", "services"}}
	path := filepath.Join(loc.Path(), "api")
	legacy := fmt.Sprintf(`{"collapsed_groups":{%q:true},"collapsed_subgroups":{%q:true},"favorites":{%q:true},"recents":[{"path":%q,"name":"api","root":%q,"domain":"work","group":"backend","subgroup":"services","opened_at":"2026-01-01T00:00:00Z"}],"active_tab":2,"tree_mode":false}`, root+"::work::backend", root+"::work::backend::services", path, path, root)
	if err := os.WriteFile(statePath(dir), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Version != SchemaVersion || len(s.CollapsedNodes) != 2 || !s.CollapsedNodes[loc.Key()] || !s.Favorites[path] || s.TreeMode || s.ActiveTab != 2 || !reflect.DeepEqual(s.Recents[0].Location, loc) {
		t.Fatalf("migration %+v", s)
	}
	original, err := os.ReadFile(statePath(dir))
	if err != nil || string(original) != legacy {
		t.Fatal("Load mutated legacy file")
	}
	for i := 0; i < 2; i++ {
		if err := Save(dir, s); err != nil {
			t.Fatal(err)
		}
		loaded, err := Load(dir)
		if err != nil || !reflect.DeepEqual(s, loaded) {
			t.Fatalf("idempotence %v %v", loaded, err)
		}
		s = loaded
	}
	data, err := os.ReadFile(statePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	var persisted map[string]json.RawMessage
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted["collapsed_subgroups"] != nil || persisted["collapsed_groups"] != nil {
		t.Fatal("legacy fields persisted")
	}
}
func TestMigrationFailuresNeverOverwriteInput(t *testing.T) {
	for _, input := range []string{
		`{"version":999,"favorites":{"/kept":true}}`,
		`{"collapsed_groups":{"/root::work::a::ambiguous":true}}`,
		`{"collapsed_subgroups":{"/root::work::..::child":true}}`,
		`{"collapsed_groups":{"relative::work::a":true}}`,
		`{"recents":[{"path":"/escape/api","root":"/root","domain":"work"}]}`,
		`{"collapsed_nodes":{"invalid":true},"version":2}`,
		`{"favorites":{"/kept":true},"recents":`,
	} {
		t.Run(input, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(statePath(dir), []byte(input), 0o600); err != nil {
				t.Fatal(err)
			}
			s, err := Load(dir)
			if err == nil {
				t.Fatal("malformed input accepted")
			}
			s.ToggleFavorite("/new")
			if err := Save(dir, s); err == nil {
				t.Fatal("unsafe state saved")
			}
			data, err := os.ReadFile(statePath(dir))
			if err != nil || string(data) != input {
				t.Fatal("input lost")
			}
		})
	}
}
func TestUnlimitedFavoritesAndRecentMoveOrder(t *testing.T) {
	s, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	loc := core.Location{Root: root, Domain: "work", GroupPath: []string{"a", "services"}}
	for i := 0; i < 100; i++ {
		s.ToggleFavorite(filepath.Join(loc.Path(), fmt.Sprintf("p%d", i)))
	}
	if len(s.Favorites) != 100 {
		t.Fatal("favorite limit")
	}
	old := core.Project{Location: loc, Name: "api", Path: filepath.Join(loc.Path(), "api")}
	dest := old
	dest.Location = core.Location{Root: t.TempDir(), Domain: "personal", GroupPath: []string{"b", "services"}}
	dest.Path = filepath.Join(dest.Location.Path(), "api")
	s.AddRecent(dest)
	s.AddRecent(old)
	s.ToggleFavorite(old.Path)
	stamp := s.Recents[0].OpenedAt
	s.MoveProject(old.Path, dest)
	if len(s.Recents) != 1 || !s.Recents[0].OpenedAt.Equal(stamp) || s.Favorites[old.Path] || !s.Favorites[dest.Path] || !reflect.DeepEqual(s.Recents[0].Location, dest.Location) {
		t.Fatalf("move %+v", s)
	}
	// Known missing paths in one accessible root never prune another unavailable root.
	s.ReconcileMissing(root)
	if !s.Favorites[dest.Path] || len(s.Recents) != 1 {
		t.Fatal("pruned unrelated root")
	}
	s.RemovePath(dest.Root)
	if s.Favorites[dest.Path] || len(s.Recents) != 0 {
		t.Fatal("root cleanup")
	}
	s.Recents = []RecentEntry{{Location: loc, Path: old.Path, OpenedAt: time.Now()}}
	s.CollapsedNodes[loc.Key()] = true
	s.RemovePath(loc.Path())
	if len(s.Recents) != 0 || s.CollapsedNodes[loc.Key()] {
		t.Fatal("group cleanup")
	}
}

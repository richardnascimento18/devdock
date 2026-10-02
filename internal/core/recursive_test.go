package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDeepOperationsAndMovePlans(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	shallow := Location{Root: a, Domain: "apps"}
	deep := Location{Root: a, Domain: "apps"}
	for i := 0; i < 40; i++ {
		deep = deep.Child(fmt.Sprintf("g%d", i))
	}
	other := Location{Root: b, Domain: "personal", GroupPath: []string{"examples", "services"}}
	if err := CreateGroupPath(shallow, deep.GroupPath); err != nil {
		t.Fatal(err)
	}
	p, err := CreateProject(shallow, "api")
	if err != nil {
		t.Fatal(err)
	}
	p.Name = "display-alias"
	for _, destination := range []Location{deep, shallow, other, deep, deep.Child("last"), shallow} {
		old := p.Path
		plan, err := PlanMove(p, destination, "api")
		if err != nil || plan.Status != MoveValid {
			t.Fatalf("plan: %+v %v", plan, err)
		}
		p, err = ExecuteMove(plan)
		if err != nil {
			t.Fatal(err)
		}
		if p.Name != "display-alias" || !reflect.DeepEqual(p.Location, destination) {
			t.Fatalf("move: %+v", p)
		}
		if _, err := os.Stat(old); !os.IsNotExist(err) {
			t.Fatalf("source remains: %v", err)
		}
	}
	occupied, err := CreateProject(deep, "api")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanMove(p, deep, "api")
	if !errors.Is(err, os.ErrExist) || plan.Status != MoveDestinationExists {
		t.Fatalf("collision: %+v %v", plan, err)
	}
	if err := DeleteProject(occupied); err != nil {
		t.Fatal(err)
	}
	plan, err = PlanMove(p, deep, "api")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(deep, "api"); err != nil {
		t.Fatal(err)
	}
	if _, err := ExecuteMove(plan); !errors.Is(err, os.ErrExist) {
		t.Fatalf("stale plan: %v", err)
	}
	if _, err := os.Stat(p.Path); err != nil {
		t.Fatal("lost source")
	}
	if err := DeleteGroup(deep); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p.Path); err != nil {
		t.Fatal("deleted sibling")
	}
	p.Path = filepath.Join(shallow.Path(), "missing")
	plan, err = PlanMove(p, other, "missing")
	if err == nil || plan.Status != MoveSourceMissing {
		t.Fatalf("missing: %+v %v", plan, err)
	}
}
func TestLocationAndNodeKeys(t *testing.T) {
	roots := []string{t.TempDir(), t.TempDir()}
	keys := map[NodeKey]bool{}
	for _, root := range roots {
		for _, groups := range [][]string{nil, {"backend"}, {"backend", "services"}, {"frontend", "services"}, {"a::b"}} {
			loc := Location{Root: root, Domain: "work", GroupPath: groups}
			key := loc.Key()
			if keys[key] {
				t.Fatal("collision")
			}
			keys[key] = true
			got, err := ParseNodeKey(key)
			if err != nil || !reflect.DeepEqual(got.GroupPath, loc.GroupPath) && len(loc.GroupPath) > 0 {
				t.Fatalf("round trip: %+v %v", got, err)
			}
		}
	}
	parent := Location{Root: roots[0], Domain: "apps", GroupPath: []string{"safe"}}
	for _, names := range [][]string{{"good", ".."}, {"good", "/bad"}, {"good", "a\\b"}, {"good", ""}} {
		if err := CreateGroupPath(parent, names); err == nil {
			t.Fatal("invalid accepted")
		}
		if _, err := os.Stat(parent.Child("good").Path()); !os.IsNotExist(err) {
			t.Fatal("partial creation before validation")
		}
	}
	outside := t.TempDir()
	if err := os.MkdirAll(parent.Path(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, parent.Child("link").Path()); err != nil {
		t.Fatal(err)
	}
	if _, err := parent.Child("link").Child("deep").ProjectPath("api"); err == nil {
		t.Fatal("followed symlink")
	}
}
func TestFlattenDeepCollapseAndDistinctRoots(t *testing.T) {
	w := Workspace{}
	for _, root := range []string{"/a", "/b"} {
		r := &Node{Location: Location{Root: root}, Kind: NodeRoot}
		d := &Node{Location: Location{Root: root, Domain: "apps"}, Kind: NodeDomain}
		r.Children = []*Node{d}
		w.Roots = append(w.Roots, r)
		n := d
		for i := 0; i < 50; i++ {
			g := &Node{Location: n.Location.Child("g"), Kind: NodeGroup, Name: "g"}
			n.Children = []*Node{g}
			n = g
		}
		n.Projects = []Project{{Location: n.Location, Name: "api", Path: filepath.Join(n.Location.Path(), "api")}}
	}
	rows := w.Flatten(nil, "", false)
	if len(rows) != 102 || rows[50].Depth != 50 {
		t.Fatalf("rows %d", len(rows))
	}
	collapse := map[NodeKey]bool{w.Roots[0].Children[0].Children[0].Key(): true}
	rows = w.Flatten(collapse, "", false)
	if len(rows) != 52 {
		t.Fatalf("collapse root collision %d", len(rows))
	}
	if len(w.Flatten(nil, "/a", false)) != 51 {
		t.Fatal("root filtering")
	}
}

package detect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/richardnascimento18/devdock/internal/core"
)

func TestLegacyMarkerScanSemantics(t *testing.T) {
	for _, name := range []string{"legacy", "blank", "project", "group", "subgroup", "malformed", "unknown", "legacy-fields"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", "markers", name+".toml"))
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			dir := filepath.Join(root, "apps", "marker")
			writeFixture(t, dir, ".devdock", string(data))
			w, err := core.ScanWorkspace([]string{root}, New().Inspect)
			switch name {
			case "malformed", "unknown":
				if err == nil || len(w.Projects) != 0 {
					t.Fatalf("bad marker: %+v %v", w, err)
				}
				expected := "invalid .devdock marker"
				if name == "unknown" {
					expected = "unknown .devdock type"
				}
				if !strings.Contains(err.Error(), expected) {
					t.Fatal(err)
				}
			case "group", "subgroup":
				if err != nil || len(w.Projects) != 0 || len(w.Locations(root, "apps")) != 2 {
					t.Fatalf("group: %+v %v", w, err)
				}
			default:
				if err != nil || len(w.Projects) != 1 || w.Projects[0].Path != dir {
					t.Fatalf("project: %+v %v", w, err)
				}
			}
			after, readErr := os.ReadFile(filepath.Join(dir, ".devdock"))
			if readErr != nil || string(after) != string(data) {
				t.Fatal("marker rewritten")
			}
		})
	}
}

package preset

import (
	"os"
	"reflect"
	"testing"
)

func TestPresetValidationAndPersistence(t *testing.T) {
	dir := t.TempDir()
	defaults, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil || !reflect.DeepEqual(loaded, defaults) {
		t.Fatalf("round trip: %v", err)
	}
	for _, values := range [][]Preset{
		{{Name: "a"}},
		{{Name: "a", Windows: []Window{{Name: "one"}, {Name: "one"}}}},
		{{Name: "a", Windows: []Window{{Name: "a.b"}, {Name: "a:b"}}}},
		{{Name: "a", Windows: []Window{{Name: "one", Layout: &PaneLayout{Size: 100}}}}},
		{defaults[0], defaults[0]},
	} {
		if err := Save(dir, values); err == nil {
			t.Fatal("invalid collection accepted")
		}
	}
	loaded, err = Load(dir)
	if err != nil || !reflect.DeepEqual(loaded, defaults) {
		t.Fatal("invalid save changed persisted collection")
	}
	if err := os.WriteFile(Path(dir), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("malformed JSON accepted")
	}
}
func TestCloneLayoutIndependent(t *testing.T) {
	clone := Clone(DefaultPresets)
	clone[2].Windows[0].Layout.Panes[0].Command = "changed"
	if DefaultPresets[2].Windows[0].Layout.Panes[0].Command == "changed" {
		t.Fatal("clone aliases defaults")
	}
}

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigRoundTripAndValidation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := Config{Roots: []string{t.TempDir()}, DefaultPreset: "nvim"}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil || !reflect.DeepEqual(got, cfg) {
		t.Fatalf("%+v %v", got, err)
	}
	for _, invalid := range []Config{{Roots: []string{"relative"}}, {Roots: []string{cfg.Roots[0], cfg.Roots[0]}}, {GitHubToken: "token"}} {
		if err := Save(invalid); err == nil {
			t.Fatal("invalid config accepted")
		}
	}
	got, err = Load()
	if err != nil || !reflect.DeepEqual(got, cfg) {
		t.Fatal("invalid save replaced config")
	}
	if err := os.WriteFile(Path(), []byte("roots = ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("malformed config accepted")
	}
}
func TestLegacyLoadHasNoWriteSideEffect(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte("root = '" + filepath.Join(t.TempDir(), "root") + "'\n")
	if err := os.WriteFile(Path(), data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil || len(cfg.Roots) != 1 || cfg.Root != "" {
		t.Fatalf("migration: %+v %v", cfg, err)
	}
	after, err := os.ReadFile(Path())
	if err != nil || string(data) != string(after) {
		t.Fatal("load changed file")
	}
}

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

func TestCommitDetachedProposalAndNoOp(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, roots := range [][]string{nil, {}, {t.TempDir()}} {
		current := Config{Roots: roots}
		if err := Save(current); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(Path())
		if err != nil {
			t.Fatal(err)
		}
		got, changed, err := Commit(current, current.Clone())
		if err != nil || changed || !reflect.DeepEqual(got, current) {
			t.Fatalf("no-op: %+v %v %v", got, changed, err)
		}
		after, err := os.Stat(Path())
		if err != nil || !os.SameFile(info, after) {
			t.Fatal("no-op replaced file")
		}
	}
	current := Config{Roots: []string{t.TempDir()}}
	proposed := current.Clone()
	proposed.DefaultPreset = "nvim"
	got, changed, err := Commit(current, proposed)
	if err != nil || !changed || current.DefaultPreset != "" {
		t.Fatal("commit mutated current")
	}
	proposed.Roots[0] = "relative"
	if got.Roots[0] != current.Roots[0] {
		t.Fatal("committed roots alias proposal")
	}
	rejected, changed, err := Commit(got, proposed)
	if err == nil || changed || !reflect.DeepEqual(rejected, got) {
		t.Fatal("invalid proposal committed")
	}
	loaded, err := Load()
	if err != nil || !reflect.DeepEqual(loaded, got) {
		t.Fatal("invalid proposal replaced disk")
	}
	rejected.Roots[0] = "mutated"
	if got.Roots[0] != current.Roots[0] {
		t.Fatal("error result aliases live roots")
	}
}

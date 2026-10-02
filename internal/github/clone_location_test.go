package github

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/richardnascimento18/devdock/internal/core"
)

func TestLocalCloneAtDeepLocationAndCollision(t *testing.T) {
	source := t.TempDir()
	if out, err := exec.Command("git", "init", "--bare", source).CombinedOutput(); err != nil {
		t.Fatalf("init: %s %v", out, err)
	}
	location := core.Location{Root: t.TempDir(), Domain: "apps"}
	for i := 0; i < 35; i++ {
		location = location.Child("g")
	}
	msg := CmdCloneRepo(source, location, "api")().(CloneDoneMsg)
	if msg.Err != nil || msg.Project.Location.Key() != location.Key() || msg.Project.Path != filepath.Join(location.Path(), "api") {
		t.Fatalf("clone %+v", msg)
	}
	if _, err := os.Stat(filepath.Join(msg.Project.Path, ".git")); err != nil {
		t.Fatal(err)
	}
	msg = CmdCloneRepo(source, location, "api")().(CloneDoneMsg)
	if msg.Err == nil {
		t.Fatal("clone collision accepted")
	}
}

func TestLinkSameNamesAndMarkerAliasesByFullRemote(t *testing.T) {
	var projects []core.Project
	for i, remote := range []string{"https://github.com/first/api.git", "git@github.com:second/api.git"} {
		path := t.TempDir()
		if out, err := exec.Command("git", "init", path).CombinedOutput(); err != nil {
			t.Fatalf("init: %s %v", out, err)
		}
		if out, err := exec.Command("git", "-C", path, "remote", "add", "origin", remote).CombinedOutput(); err != nil {
			t.Fatalf("remote: %s %v", out, err)
		}
		name := "api"
		if i == 1 {
			name = "marker-alias"
		}
		projects = append(projects, core.Project{Name: name, Path: path})
	}
	linked := LinkProjectsToRepos(projects, []Repo{{Name: "api", FullName: "first/api"}, {Name: "api", FullName: "second/api"}})
	if linked[0].GitHubRepo != "first/api" || linked[1].GitHubRepo != "second/api" {
		t.Fatalf("identity: %+v", linked)
	}
}

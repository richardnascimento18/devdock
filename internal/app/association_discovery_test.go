package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/testutil"
)

func associationFixture(t testing.TB) ([]core.Project, []gh.Repo, []string) {
	t.Helper()
	t.Setenv("GIT_DIR", "")
	t.Setenv("GIT_WORK_TREE", "")
	// Empty Git path variables are invalid; unset them after registering restoration.
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE"} {
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	base := t.TempDir()
	var projects []core.Project
	var repos []gh.Repo
	var want []string
	add := func(path, identity string) {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
		projects = append(projects, core.Project{Name: filepath.Base(path), Path: path, GitHubRepo: "stale/association"})
		want = append(want, identity)
	}
	repo := func(path, identity string) {
		add(path, identity)
		testutil.Git(t, path, "init")
		testutil.Git(t, path, "remote", "add", "origin", "git@github.com:"+identity+".git")
		repos = append(repos, gh.Repo{FullName: identity})
	}
	for i := 0; i < 64; i++ {
		add(filepath.Join(base, fmt.Sprintf("plain-%d", i)), "")
	}
	parent := filepath.Join(base, "repository")
	repo(parent, "owner/remote-name")
	testutil.Git(t, parent, "commit", "--allow-empty", "-m", "test")
	worktree := filepath.Join(base, "worktree")
	testutil.Git(t, parent, "worktree", "add", worktree)
	add(worktree, "owner/remote-name")
	repo(filepath.Join(parent, "nested"), "owner/nested")
	add(filepath.Join(parent, "ordinary-project"), "")
	repo(filepath.Join(base, "left", "api"), "owner/left")
	repo(filepath.Join(base, "right", "api"), "owner/right")
	testutil.Git(t, parent, "remote", "add", "upstream", "https://github.com/owner/wrong.git")
	repos = append(repos, gh.Repo{FullName: "owner/wrong"})
	invalid := filepath.Join(base, "invalid")
	add(invalid, "")
	if err := os.WriteFile(filepath.Join(invalid, ".git"), []byte("invalid metadata\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bare := filepath.Join(base, "bare")
	add(bare, "")
	testutil.Git(t, bare, "init", "--bare")
	add(parent, "owner/remote-name") // Duplicate discovery within one snapshot.
	return projects, repos, want
}

func countingAssociationService() (Workspaces, *int) {
	service := NewWorkspaces()
	calls := new(int)
	output := service.Git.Output
	service.Git.Output = func(ctx context.Context, path string, args ...string) ([]byte, error) {
		*calls++
		return output(ctx, path, args...)
	}
	return service, calls
}

func TestAssociationDiscoveryIdentity(t *testing.T) {
	projects, repos, want := associationFixture(t)
	service, _ := countingAssociationService()
	linked := service.Associate(context.Background(), projects, repos)
	for i := range linked {
		if linked[i].GitHubRepo != want[i] {
			t.Fatalf("project %d: %q, want %q", i, linked[i].GitHubRepo, want[i])
		}
		if projects[i].GitHubRepo != "stale/association" {
			t.Fatal("association mutated input snapshot")
		}
	}
	// There is no cache across operations: remote changes are immediately visible.
	path := projects[64].Path
	testutil.Git(t, path, "remote", "set-url", "origin", "https://github.com/owner/new.git")
	repos = append(repos, gh.Repo{FullName: "owner/new"})
	linked = service.Associate(context.Background(), projects, repos)
	if linked[64].GitHubRepo != "owner/new" || linked[len(linked)-1].GitHubRepo != "owner/new" {
		t.Fatal("stale remote identity")
	}
	// A previously ordinary directory can become a repository between scans.
	testutil.Git(t, projects[0].Path, "init")
	testutil.Git(t, projects[0].Path, "remote", "add", "origin", "https://github.com/owner/new.git")
	if service.Associate(context.Background(), projects[:1], repos)[0].GitHubRepo != "owner/new" {
		t.Fatal("stale negative discovery")
	}
}

func TestAssociationAvoidsUnnecessaryProcesses(t *testing.T) {
	projects, repos, _ := associationFixture(t)
	service, calls := countingAssociationService()
	service.Associate(context.Background(), projects[:64], repos)
	if *calls != 0 {
		t.Fatalf("ordinary projects launched %d Git processes", *calls)
	}
	service.Associate(context.Background(), projects, nil)
	if *calls != 0 {
		t.Fatal("no association data still launched Git")
	}
	service.Associate(context.Background(), []core.Project{projects[64], projects[64]}, repos)
	if *calls != 2 {
		t.Fatalf("duplicate repository launched %d processes, want one identity/remote pair", *calls)
	}
}

func BenchmarkAssociationDiscovery(b *testing.B) {
	projects, repos, _ := associationFixture(b)
	service, calls := countingAssociationService()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		service.Associate(context.Background(), projects, repos)
	}
	b.ReportMetric(float64(*calls)/float64(b.N), "git/op")
	b.ReportMetric(float64(len(projects)), "projects/op")
}

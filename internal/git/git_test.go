package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/richardnascimento18/devdock/internal/testutil"
)

func TestParseRemote(t *testing.T) {
	for _, tc := range []struct{ input, want string }{{"git@github.com:owner/repo.git", "owner/repo"}, {"https://github.com/owner/repo.git", "owner/repo"}, {"ssh://git@github.com/owner/repo", "owner/repo"}, {"https://evilgithub.com/owner/repo", ""}, {"https://example.org/github.com/owner/repo", ""}, {"https://github.com/owner/repo/extra", ""}} {
		if got := ParseRemote(tc.input); got != tc.want {
			t.Errorf("%q: %q", tc.input, got)
		}
	}
}

func TestGitInitializationStopsAtEveryFailure(t *testing.T) {
	for fail := 0; fail < 6; fail++ {
		calls := 0
		sentinel := errors.New("failed")
		err := initRepo(context.Background(), t.TempDir(), "https://github.com/owner/repo.git", func(ctx context.Context, dir string, args ...string) error {
			calls++
			if calls == fail+1 {
				return sentinel
			}
			return nil
		})
		if !errors.Is(err, sentinel) || calls != fail+1 {
			t.Fatalf("step %d: %v calls %d", fail, err, calls)
		}
	}
}
func TestCloneCollisionPreflight(t *testing.T) {
	if err := CloneRepo("unused", t.TempDir()); err == nil {
		t.Fatal("existing directory accepted")
	}
}

func TestWorktreeAndParentRepositoryDiscovery(t *testing.T) {
	repo := t.TempDir()
	testutil.Git(t, repo, "init")
	testutil.Git(t, repo, "commit", "--allow-empty", "-m", "test")
	worktree := filepath.Join(t.TempDir(), "worktree")
	testutil.Git(t, repo, "worktree", "add", worktree)
	if !IsGitInitialized(worktree) {
		t.Fatal("worktree .git file not recognized")
	}
	nested := filepath.Join(repo, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if IsGitInitialized(nested) {
		t.Fatal("parent repository treated as project repository")
	}
}

func TestExplicitGitMetadataOutsideProject(t *testing.T) {
	repository, project := t.TempDir(), t.TempDir()
	testutil.Git(t, repository, "init")
	t.Setenv("GIT_DIR", filepath.Join(repository, ".git"))
	t.Setenv("GIT_WORK_TREE", project)
	if !IsGitInitialized(project) {
		t.Fatal("explicit Git metadata not recognized")
	}
}

func TestCancelledDiscoveryDoesNotRunGit(t *testing.T) {
	client := NewClient()
	client.Output = func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("Git started after cancellation")
		return nil, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if client.Initialized(ctx, t.TempDir()) {
		t.Fatal("cancelled discovery succeeded")
	}
}

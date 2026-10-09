package git

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/process"
)

func CloneRepo(cloneURL, destPath string) error {
	return NewClient().Clone(context.Background(), cloneURL, destPath)
}
func (c Client) Clone(ctx context.Context, cloneURL, destPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := os.Lstat(destPath); err == nil {
		return fmt.Errorf("clone destination already exists: %w", os.ErrExist)
	} else if !os.IsNotExist(err) {
		return err
	}
	return c.Run(ctx, "", "clone", "--", cloneURL, destPath)
}

// GitCommand is the subprocess boundary used by repository initialization.
type GitCommand func(ctx context.Context, dir string, args ...string) error

func runGit(ctx context.Context, dir string, args ...string) error {
	out, err := outputGit(ctx, dir, args...)
	if err != nil {
		return fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return nil
}
func InitRepoWithRemote(path, remote string) error {
	return InitRepoWithRemoteContext(context.Background(), path, remote)
}
func InitRepoWithRemoteContext(ctx context.Context, path, remote string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return NewClient().Init(ctx, path, remote)
}
func initRepo(ctx context.Context, path, remote string, run GitCommand) error {
	readme := filepath.Join(path, "README.md")
	if _, err := os.Lstat(readme); os.IsNotExist(err) {
		if err := os.WriteFile(readme, []byte("# "+filepath.Base(path)+"\n"), 0o644); err != nil {
			return fmt.Errorf("write README: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("inspect README: %w", err)
	}
	steps := [][]string{{"init"}, {"remote", "add", "origin", remote}, {"checkout", "-b", "main"}, {"add", "."}, {"commit", "-m", "initial commit"}, {"push", "-u", "origin", "main"}}
	for _, args := range steps {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := run(ctx, path, args...); err != nil {
			return fmt.Errorf("git %s: %w", args[0], err)
		}
	}
	return nil
}

func DetectRemote(projectPath string) string {
	return NewClient().Remote(context.Background(), projectPath)
}
func (c Client) Remote(parent context.Context, projectPath string) string {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	out, err := c.Output(ctx, projectPath, "remote", "get-url", "origin")
	if err != nil {
		return "" // A project without an origin is a normal unlinked project.
	}
	return ParseRemote(strings.TrimSpace(string(out)))
}

func ParseRemote(raw string) string {
	var ownerRepo string
	if strings.HasPrefix(raw, "git@github.com:") {
		ownerRepo = strings.TrimPrefix(raw, "git@github.com:")
	} else {
		parsed, err := url.Parse(raw)
		if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") || (parsed.Scheme != "https" && parsed.Scheme != "ssh") {
			return ""
		}
		ownerRepo = strings.TrimPrefix(parsed.Path, "/")
	}
	ownerRepo = strings.TrimSuffix(ownerRepo, ".git")
	parts := strings.Split(ownerRepo, "/")
	if len(parts) != 2 || !core.ValidName(parts[0]) || !core.ValidName(parts[1]) {
		return ""
	}
	return ownerRepo
}

// IsGitInitialized asks Git, supporting worktree .git files and rejecting a
// project nested within an unrelated parent repository.
func IsGitInitialized(projectPath string) bool {
	return NewClient().Initialized(context.Background(), projectPath)
}
func (c Client) Initialized(parent context.Context, projectPath string) bool {
	if parent.Err() != nil {
		return false
	}
	// .git files/directories identify candidates, including linked worktrees.
	// Ancestor metadata may configure core.worktree to point at this directory.
	// Git still decides the actual root; a parent repository never suffices.
	// Explicit external metadata bypasses the filesystem candidate check.
	if os.Getenv("GIT_DIR") == "" && os.Getenv("GIT_WORK_TREE") == "" && !repositoryCandidate(projectPath) {
		return false
	}

	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	out, err := c.Output(ctx, projectPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return false
	}
	root, err := filepath.EvalSymlinks(strings.TrimSpace(string(out)))
	if err != nil {
		return false
	}
	project, err := filepath.EvalSymlinks(projectPath)
	return err == nil && filepath.Clean(root) == filepath.Clean(project)
}

type Client struct {
	Run    GitCommand
	Output func(context.Context, string, ...string) ([]byte, error)
}

func NewClient() Client { return Client{Run: runGit, Output: outputGit} }
func outputGit(ctx context.Context, dir string, args ...string) ([]byte, error) {
	return process.Output(ctx, dir, "git", args...)
}
func (c Client) Init(ctx context.Context, path, remote string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return initRepo(ctx, path, remote, c.Run)
}

func repositoryCandidate(projectPath string) bool {
	path, err := filepath.Abs(projectPath)
	if err != nil {
		return true
	} // Let Git diagnose paths we cannot inspect.
	if physical, err := filepath.EvalSymlinks(path); err == nil {
		path = physical
	}
	for {
		metadata := filepath.Join(path, ".git")
		info, err := os.Stat(metadata)
		if err == nil && info.IsDir() {
			// An empty .git directory cannot be a repository. HEAD is required;
			// other corruption still goes to Git for authoritative validation.
			_, err = os.Lstat(filepath.Join(metadata, "HEAD"))
		}
		if !os.IsNotExist(err) {
			return true
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false
		}
		path = parent
	}
}

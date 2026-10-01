package github

import (
	"context"
	"fmt"
	"github.com/richardnascimento18/devdock/internal/core"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func CloneRepo(cloneURL, destPath string) error {
	if _, err := os.Lstat(destPath); err == nil {
		return fmt.Errorf("clone destination already exists: %w", os.ErrExist)
	} else if !os.IsNotExist(err) {
		return err
	}
	cmd := exec.Command("git", "clone", "--", cloneURL, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GitCommand is the subprocess boundary used by repository initialization.
type GitCommand func(ctx context.Context, dir string, args ...string) error

func runGit(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
func InitRepoWithRemote(path, remote string) error {
	return InitRepoWithRemoteContext(context.Background(), path, remote)
}
func InitRepoWithRemoteContext(ctx context.Context, path, remote string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return initRepo(ctx, path, remote, runGit)
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	cmd.Dir = projectPath
	out, err := cmd.Output()
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

func IsGitInitialized(projectPath string) bool {
	_, err := os.Stat(filepath.Join(projectPath, ".git"))
	return err == nil
}

func LinkProjectsToRepos(projects []core.Project, repos []Repo) []core.Project {
	byName := make(map[string]Repo, len(repos))
	for _, r := range repos {
		byName[strings.ToLower(r.Name)] = r
	}
	linked := make([]core.Project, len(projects))
	for i, p := range projects {
		p.GitHubRepo = ""
		ghRepo, nameMatch := byName[strings.ToLower(p.Name)]
		if nameMatch && IsGitInitialized(p.Path) {
			detected := DetectRemote(p.Path)
			if strings.EqualFold(detected, ghRepo.FullName) {
				p.GitHubRepo = ghRepo.FullName
			}
		}
		linked[i] = p
	}
	return linked
}

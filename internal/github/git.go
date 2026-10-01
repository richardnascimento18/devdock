package github

import (
	"fmt"
	"github.com/richardnascimento18/devdock/internal/core"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func InitRepoWithRemote(projectPath, remoteURL string) error {
	run := func(args ...string) error {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = projectPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	steps := [][]string{
		{"git", "init"},
		{"git", "remote", "add", "origin", remoteURL},
		{"git", "checkout", "-b", "main"},
	}
	readmePath := filepath.Join(projectPath, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		name := filepath.Base(projectPath)
		if err := os.WriteFile(readmePath, []byte("# "+name+"\n"), 0o644); err != nil {
			return fmt.Errorf("write README: %w", err)
		}
	}
	steps = append(steps,
		[]string{"git", "add", "."},
		[]string{"git", "commit", "-m", "initial commit"},
		[]string{"git", "push", "-u", "origin", "main"},
	)
	for _, args := range steps {
		if err := run(args...); err != nil {
			return fmt.Errorf("git %s: %w", args[1], err)
		}
	}
	return nil
}

func DetectRemote(projectPath string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = projectPath
	out, err := cmd.Output()
	if err != nil {
		return ""
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

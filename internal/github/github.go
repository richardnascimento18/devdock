package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/core"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// ClientID is injected at build time via:
//
//	go build -ldflags "-X 'github.com/YOURUSERNAME/devdock/internal/github.ClientID=YOUR_ID'"
//
// Falls back to the DEVDOCK_GITHUB_CLIENT_ID environment variable.
var ClientID string

func clientID() string {
	if ClientID != "" {
		return ClientID
	}
	return os.Getenv("DEVDOCK_GITHUB_CLIENT_ID")
}

type Repo struct {
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	CloneURL    string `json:"clone_url"`
	SSHURL      string `json:"ssh_url"`
	Private     bool   `json:"private"`
	Description string `json:"description"`
}

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

func StartDeviceFlow() (DeviceCodeResponse, error) {
	body := url.Values{
		"client_id": {clientID()},
		"scope":     {"repo,read:user"},
	}.Encode()
	req, err := http.NewRequest("POST", "https://github.com/login/device/code", strings.NewReader(body))
	if err != nil {
		return DeviceCodeResponse{}, fmt.Errorf("device flow request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return DeviceCodeResponse{}, fmt.Errorf("device flow request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return DeviceCodeResponse{}, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return DeviceCodeResponse{}, fmt.Errorf("GitHub returned %d: %s", resp.StatusCode, string(respBody))
	}
	var dc DeviceCodeResponse
	if jsonErr := json.Unmarshal(respBody, &dc); jsonErr != nil || dc.DeviceCode == "" {
		vals, _ := url.ParseQuery(string(respBody))
		dc.DeviceCode = vals.Get("device_code")
		dc.UserCode = vals.Get("user_code")
		dc.VerificationURI = vals.Get("verification_uri")
		if n := vals.Get("expires_in"); n != "" {
			fmt.Sscanf(n, "%d", &dc.ExpiresIn)
		}
		if n := vals.Get("interval"); n != "" {
			fmt.Sscanf(n, "%d", &dc.Interval)
		}
	}
	if dc.DeviceCode == "" {
		return dc, fmt.Errorf("could not get device code from GitHub -- check your Client ID is correct and the OAuth app has device flow enabled")
	}
	if dc.Interval == 0 {
		dc.Interval = 5
	}
	return dc, nil
}

func PollForToken(dc DeviceCodeResponse) (string, error) {
	deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)
	interval := time.Duration(dc.Interval) * time.Second
	for time.Now().Before(deadline) {
		time.Sleep(interval)
		pollBody := url.Values{
			"client_id":   {clientID()},
			"device_code": {dc.DeviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		}.Encode()
		pollReq, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(pollBody))
		pollReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		pollReq.Header.Set("Accept", "application/json")
		pollResp, err := httpClient.Do(pollReq)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(pollResp.Body)
		if err != nil {
			pollResp.Body.Close()
			continue
		}

		pollResp.Body.Close()
		var tr tokenResponse
		if jsonErr := json.Unmarshal(body, &tr); jsonErr != nil || (tr.AccessToken == "" && tr.Error == "") {
			vals, _ := url.ParseQuery(string(body))
			tr.AccessToken = vals.Get("access_token")
			tr.Error = vals.Get("error")
		}
		switch tr.Error {
		case "":
			if tr.AccessToken != "" {
				return tr.AccessToken, nil
			}
		case "authorization_pending":
		case "slow_down":
			interval += 5 * time.Second
		case "expired_token":
			return "", fmt.Errorf("authorization code expired, please try again")
		case "access_denied":
			return "", fmt.Errorf("access denied by user")
		default:
			return "", fmt.Errorf("unexpected error from GitHub: %s", tr.Error)
		}
	}
	return "", fmt.Errorf("timed out waiting for GitHub authorization")
}

func get(token, path string, out interface{}) error {
	req, _ := http.NewRequest("GET", "https://api.github.com"+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API %s returned %d: %s", path, resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func post(token, path string, payload interface{}, out interface{}) error {
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.github.com"+path, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API POST %s returned %d: %s", path, resp.StatusCode, string(body))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func FetchUsername(token string) (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := get(token, "/user", &user); err != nil {
		return "", err
	}
	return user.Login, nil
}

func FetchRepos(token string) ([]Repo, error) {
	var all []Repo
	page := 1
	for {
		var repos []Repo
		path := fmt.Sprintf("/user/repos?affiliation=owner&per_page=100&page=%d", page)
		if err := get(token, path, &repos); err != nil {
			return all, err
		}
		if len(repos) == 0 {
			break
		}
		all = append(all, repos...)
		if len(repos) < 100 {
			break
		}
		page++
	}
	return all, nil
}

func CreateRepo(token, name string, private bool) (Repo, error) {
	payload := map[string]interface{}{"name": name, "private": private}
	var repo Repo
	if err := post(token, "/user/repos", payload, &repo); err != nil {
		return Repo{}, err
	}
	return repo, nil
}

func CloneRepo(cloneURL, destPath string) error {
	cmd := exec.Command("git", "clone", cloneURL, destPath)
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
		_ = os.WriteFile(readmePath, []byte("# "+name+"\n"), 0o644)
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
	raw := strings.TrimSpace(string(out))
	var ownerRepo string
	if strings.HasPrefix(raw, "git@github.com:") {
		ownerRepo = strings.TrimSuffix(strings.TrimPrefix(raw, "git@github.com:"), ".git")
	} else if idx := strings.Index(raw, "github.com/"); idx >= 0 {
		ownerRepo = strings.TrimSuffix(raw[idx+len("github.com/"):], ".git")
	}
	if strings.Count(ownerRepo, "/") == 1 {
		return ownerRepo
	}
	return ""
}

func IsGitInitialized(projectPath string) bool {
	info, err := os.Stat(filepath.Join(projectPath, ".git"))
	return err == nil && info.IsDir()
}

func LinkProjectsToRepos(projects []core.Project, repos []Repo) []core.Project {
	byName := make(map[string]Repo, len(repos))
	for _, r := range repos {
		byName[strings.ToLower(r.Name)] = r
	}
	linked := make([]core.Project, len(projects))
	for i, p := range projects {
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

// tea.Cmd messages

type ReposLoadedMsg struct {
	Repos []Repo
	Err   error
}

type AuthDoneMsg struct {
	Token    string
	Username string
	Err      error
}

type RepoCreatedMsg struct {
	Repo Repo
	Err  error
}

type CloneDoneMsg struct {
	Project core.Project
	Err     error
}

func CmdFetchRepos(token string) tea.Cmd {
	return func() tea.Msg {
		repos, err := FetchRepos(token)
		return ReposLoadedMsg{Repos: repos, Err: err}
	}
}

func CmdPollForToken(dc DeviceCodeResponse) tea.Cmd {
	return func() tea.Msg {
		token, err := PollForToken(dc)
		if err != nil {
			return AuthDoneMsg{Err: err}
		}
		username, err := FetchUsername(token)
		return AuthDoneMsg{Token: token, Username: username, Err: err}
	}
}

func CmdCreateRepo(token, name string, private bool) tea.Cmd {
	return func() tea.Msg {
		repo, err := CreateRepo(token, name, private)
		return RepoCreatedMsg{Repo: repo, Err: err}
	}
}

func CmdCloneRepo(cloneURL, root, domain, repoName string) tea.Cmd {
	return func() tea.Msg {
		domainPath := filepath.Join(root, domain)
		if err := os.MkdirAll(domainPath, 0o755); err != nil {
			return CloneDoneMsg{Err: err}
		}
		destPath := filepath.Join(domainPath, repoName)
		if err := CloneRepo(cloneURL, destPath); err != nil {
			return CloneDoneMsg{Err: err}
		}
		ghRepo := DetectRemote(destPath)
		p := core.Project{
			Name:       repoName,
			Path:       destPath,
			Domain:     domain,
			Root:       root,
			GitHubRepo: ghRepo,
		}
		return CloneDoneMsg{Project: p}
	}
}

package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ClientID is public, injected using -ldflags -X. No client secret is needed.
var ClientID string

func clientID() string {
	if value := os.Getenv("DEVDOCK_GITHUB_CLIENT_ID"); value != "" {
		return value
	}
	return ClientID
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}
type Client struct {
	HTTP                   HTTPDoer
	APIBase, OAuthBase, ID string
}

func NewClient() *Client {
	return &Client{HTTP: &http.Client{Timeout: 30 * time.Second}, APIBase: "https://api.github.com", OAuthBase: "https://github.com", ID: clientID()}
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

func (c *Client) request(ctx context.Context, method, endpoint, token, contentType string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("create GitHub request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("GitHub request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub returned HTTP %d", resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(out); err != nil {
		return fmt.Errorf("decode GitHub response: %w", err)
	}
	return nil
}
func (c *Client) StartDeviceFlow(ctx context.Context) (DeviceCodeResponse, error) {
	if c.ID == "" {
		return DeviceCodeResponse{}, fmt.Errorf("GitHub OAuth Client ID is not configured; set DEVDOCK_GITHUB_CLIENT_ID or supply a build-time Client ID")
	}
	body := url.Values{"client_id": {c.ID}, "scope": {"repo read:user"}}.Encode()
	var dc DeviceCodeResponse
	if err := c.request(ctx, "POST", c.OAuthBase+"/login/device/code", "", "application/x-www-form-urlencoded", strings.NewReader(body), &dc); err != nil {
		return dc, err
	}
	if dc.DeviceCode == "" || dc.UserCode == "" || dc.VerificationURI == "" || dc.ExpiresIn <= 0 || dc.Interval < 0 {
		return dc, fmt.Errorf("invalid GitHub device response; check Client ID and enable device flow")
	}
	if dc.Interval == 0 {
		dc.Interval = 5
	}
	return dc, nil
}
func (c *Client) PollForToken(ctx context.Context, dc DeviceCodeResponse) (string, error) {
	if dc.ExpiresIn <= 0 || dc.Interval <= 0 {
		return "", fmt.Errorf("invalid device expiry or polling interval")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(dc.ExpiresIn)*time.Second)
	defer cancel()
	interval := time.Duration(dc.Interval) * time.Second
	for {
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
		body := url.Values{"client_id": {c.ID}, "device_code": {dc.DeviceCode}, "grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}}.Encode()
		var tr tokenResponse
		if err := c.request(ctx, "POST", c.OAuthBase+"/login/oauth/access_token", "", "application/x-www-form-urlencoded", strings.NewReader(body), &tr); err != nil {
			return "", err
		}
		switch tr.Error {
		case "":
			if tr.AccessToken != "" {
				return tr.AccessToken, nil
			}
			return "", fmt.Errorf("GitHub returned an empty token response")
		case "authorization_pending":
		case "slow_down":
			interval += 5 * time.Second
		case "expired_token":
			return "", fmt.Errorf("authorization code expired")
		case "access_denied":
			return "", fmt.Errorf("access denied by user")
		default:
			return "", fmt.Errorf("GitHub rejected authorization")
		}
	}
}
func (c *Client) FetchUsername(ctx context.Context, token string) (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := c.request(ctx, "GET", c.APIBase+"/user", token, "", nil, &user); err != nil {
		return "", err
	}
	if user.Login == "" {
		return "", fmt.Errorf("GitHub returned an empty username")
	}
	return user.Login, nil
}
func (c *Client) FetchRepos(ctx context.Context, token string) ([]Repo, error) {
	var all []Repo
	for page := 1; ; page++ {
		var repos []Repo
		endpoint := fmt.Sprintf("%s/user/repos?affiliation=owner&per_page=100&page=%d", c.APIBase, page)
		if err := c.request(ctx, "GET", endpoint, token, "", nil, &repos); err != nil {
			return all, err
		}
		all = append(all, repos...)
		if len(repos) < 100 {
			return all, nil
		}
	}
}
func (c *Client) CreateRepo(ctx context.Context, token, name string, private bool) (Repo, error) {
	payload, err := json.Marshal(map[string]any{"name": name, "private": private})
	if err != nil {
		return Repo{}, err
	}
	var repo Repo
	err = c.request(ctx, "POST", c.APIBase+"/user/repos", token, "application/json", bytes.NewReader(payload), &repo)
	return repo, err
}

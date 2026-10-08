package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}
func TestDeviceRequestAndCancellation(t *testing.T) {
	client := &Client{ID: "public-id", OAuthBase: "https://example.invalid", HTTP: doerFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != "POST" || r.Header.Get("Accept") != "application/json" {
			t.Error("wrong request")
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("client_id") != "public-id" || r.Form.Get("client_secret") != "" {
			t.Error("credentials")
		}
		return response(200, `{"device_code":"device","user_code":"ABCD","verification_uri":"https://github.com/login/device","expires_in":60,"interval":1}`), nil
	})}
	dc, err := client.StartDeviceFlow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if _, err := client.PollForToken(ctx, dc); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("cancel blocked on poll interval")
	}
}
func TestAuthHTTPFailuresAndToken(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		wantErr    bool
	}{{"token", `{"access_token":"token"}`, 200, false}, {"denied", `{"error":"access_denied"}`, 200, true}, {"empty", `{}`, 200, true}, {"malformed", `{`, 200, true}, {"http", `sensitive-token`, 500, true}} {
		t.Run(tc.name, func(t *testing.T) {
			client := &Client{ID: "id", OAuthBase: "https://example.invalid", HTTP: doerFunc(func(*http.Request) (*http.Response, error) { return response(tc.status, tc.body), nil })}
			token, err := client.PollForToken(context.Background(), DeviceCodeResponse{ExpiresIn: 10, Interval: 1})
			if (err != nil) != tc.wantErr {
				t.Fatalf("%q %v", token, err)
			}
			if err != nil && strings.Contains(err.Error(), "sensitive-token") {
				t.Fatal("response body leaked")
			}
		})
	}
}
func TestRequestContextAndErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{APIBase: "https://example.invalid", HTTP: doerFunc(func(r *http.Request) (*http.Response, error) {
		cancel()
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}
	if _, err := c.FetchUsername(ctx, "token"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	c = NewClient()
	c.ID = ""
	if _, err := c.StartDeviceFlow(context.Background()); err == nil {
		t.Fatal("missing ID not reported")
	}
}
func TestAPIPaginationAndPartialFailure(t *testing.T) {
	calls := 0
	client := &Client{APIBase: "https://example.invalid", HTTP: doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Error("missing authorization")
		}
		if r.URL.Query().Get("page") != fmt.Sprint(calls) {
			t.Error("pagination")
		}
		if calls == 2 {
			return response(500, "token must never appear in errors"), nil
		}
		repos := make([]Repo, 100)
		data, err := json.Marshal(repos)
		if err != nil {
			t.Fatal(err)
		}
		return response(200, string(data)), nil
	})}
	repos, err := client.FetchRepos(context.Background(), "token")
	if err == nil || len(repos) != 100 || calls != 2 {
		t.Fatalf("partial API failure: %d %v calls %d", len(repos), err, calls)
	}
}
func TestCreateRepoAndUsername(t *testing.T) {
	client := &Client{APIBase: "https://example.invalid", HTTP: doerFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/user" {
			return response(200, `{"login":"owner"}`), nil
		}
		var payload struct {
			Name    string
			Private bool
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if r.Method != "POST" || payload.Name != "demo" || !payload.Private {
			t.Error("wrong create request")
		}
		return response(201, `{"name":"demo","full_name":"owner/demo"}`), nil
	})}
	user, err := client.FetchUsername(context.Background(), "token")
	if err != nil || user != "owner" {
		t.Fatal(err)
	}
	repo, err := client.CreateRepo(context.Background(), "token", "demo", true)
	if err != nil || repo.FullName != "owner/demo" {
		t.Fatal(err)
	}
}

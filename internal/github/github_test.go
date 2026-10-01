package github

import (
	"context"
	"errors"
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
func TestParseRemote(t *testing.T) {
	for _, tc := range []struct{ input, want string }{{"git@github.com:owner/repo.git", "owner/repo"}, {"https://github.com/owner/repo.git", "owner/repo"}, {"ssh://git@github.com/owner/repo", "owner/repo"}, {"https://evilgithub.com/owner/repo", ""}, {"https://example.org/github.com/owner/repo", ""}, {"https://github.com/owner/repo/extra", ""}} {
		if got := ParseRemote(tc.input); got != tc.want {
			t.Errorf("%q: %q", tc.input, got)
		}
	}
}

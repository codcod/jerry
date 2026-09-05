package forge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// newTestClient points a GitHubClient at an httptest.Server via
// GITHUB_API_URL, the same seam NewGitHubFromEnv uses for GitHub
// Enterprise — no live network calls in this package's tests.
func newTestClient(t *testing.T, server *httptest.Server) *GitHubClient {
	t.Helper()
	eventPath := writeEventFile(t, 42)
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)
	t.Setenv("GITHUB_API_URL", server.URL)

	client, ok := NewGitHubFromEnv()
	if !ok {
		t.Fatalf("NewGitHubFromEnv: ok = false, want true")
	}
	return client
}

func writeEventFile(t *testing.T, prNumber int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "event.json")
	body := map[string]any{}
	if prNumber != 0 {
		body["pull_request"] = map[string]any{"number": prNumber}
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshaling event body: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing event file: %v", err)
	}
	return path
}

func TestPostOrUpdate_NoExistingComment_Posts(t *testing.T) {
	var posted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/issues/42/comments":
			_ = json.NewEncoder(w).Encode([]issueComment{})
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/issues/42/comments":
			posted = true
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)
	if err := client.PostOrUpdate("hello"); err != nil {
		t.Fatalf("PostOrUpdate: %v", err)
	}
	if !posted {
		t.Fatal("expected a POST, got none")
	}
}

func TestPostOrUpdate_ExistingMarkedComment_Patches(t *testing.T) {
	var patchedID int64
	var posted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/issues/42/comments":
			_ = json.NewEncoder(w).Encode([]issueComment{
				{ID: 7, Body: "some other comment"},
				{ID: 9, Body: "jerry says hi\n" + CommentMarker},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/owner/repo/issues/comments/9":
			patchedID = 9
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost:
			posted = true
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)
	if err := client.PostOrUpdate("hello again"); err != nil {
		t.Fatalf("PostOrUpdate: %v", err)
	}
	if patchedID != 9 {
		t.Fatalf("expected PATCH of comment 9, got patchedID=%d", patchedID)
	}
	if posted {
		t.Fatal("expected no POST when an existing comment carries the marker")
	}
}

func TestPostOrUpdate_ExistingCommentsWithoutMarker_Posts(t *testing.T) {
	var posted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/issues/42/comments":
			_ = json.NewEncoder(w).Encode([]issueComment{
				{ID: 1, Body: "unrelated comment"},
				{ID: 2, Body: "another unrelated comment"},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/issues/42/comments":
			posted = true
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPatch:
			t.Errorf("unexpected PATCH: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)
	if err := client.PostOrUpdate("hello"); err != nil {
		t.Fatalf("PostOrUpdate: %v", err)
	}
	if !posted {
		t.Fatal("expected a POST when no existing comment carries the marker")
	}
}

func TestNewGitHubFromEnv_NotOK(t *testing.T) {
	validEventPath := writeEventFile(t, 42)
	malformedEventPath := filepath.Join(t.TempDir(), "malformed.json")
	if err := os.WriteFile(malformedEventPath, []byte("not json"), 0o644); err != nil {
		t.Fatalf("writing malformed event file: %v", err)
	}
	noPRPath := writeEventFile(t, 0)

	cases := map[string]struct {
		token, repo, eventPath string
	}{
		"missing GITHUB_TOKEN":               {token: "", repo: "owner/repo", eventPath: validEventPath},
		"missing GITHUB_REPOSITORY":          {token: "t", repo: "", eventPath: validEventPath},
		"missing GITHUB_EVENT_PATH":          {token: "t", repo: "owner/repo", eventPath: ""},
		"malformed GITHUB_EVENT_PATH":        {token: "t", repo: "owner/repo", eventPath: malformedEventPath},
		"event payload with no pull_request": {token: "t", repo: "owner/repo", eventPath: noPRPath},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("GITHUB_TOKEN", tc.token)
			t.Setenv("GITHUB_REPOSITORY", tc.repo)
			t.Setenv("GITHUB_EVENT_PATH", tc.eventPath)

			client, ok := NewGitHubFromEnv()
			if ok {
				t.Fatalf("NewGitHubFromEnv: ok = true, want false (client=%+v)", client)
			}
			if client != nil {
				t.Fatalf("NewGitHubFromEnv: client = %+v, want nil when ok is false", client)
			}
		})
	}
}

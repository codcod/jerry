// Package forge posts jerry's own comments to a forge's merge requests.
//
// Scoped tight per PLAN.md's warning against building a general client no
// consumer has exercised yet: one interface, one forge (GitHub), and a
// single create-or-update call per invocation. Pagination and rate-limit
// handling are deferred to `crawl`, the first thing that actually exercises
// them (DESIGN.md §7.2, §7.4).
package forge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// Commenter posts or updates the one comment jerry owns on a pull request.
// One implementation today (GitHub), called directly by its one consumer
// (bot, JRY-015 — decision 1 keeps it off this interface on purpose, so
// there is no seam to inject in tests); a second forge implements Commenter
// once one is actually needed (DESIGN.md §7).
type Commenter interface {
	PostOrUpdate(body string) error
}

// CommentMarker tags every comment jerry posts, so PostOrUpdate can find
// its own comment among a PR's others instead of guessing by content.
const CommentMarker = "<!-- jerry:bot-comment -->"

var _ Commenter = (*GitHubClient)(nil)

// GitHubClient talks to the GitHub REST API directly (net/http +
// encoding/json, no SDK dependency) on behalf of one pull request.
type GitHubClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
	repository string
	prNumber   int
}

// NewGitHubFromEnv builds a client from the GitHub Actions environment.
// ok is false — never an error — when the environment isn't a pull-request
// CI run (no token, no repository, or the event isn't a pull_request):
// the caller (bot) treats that as "nothing to do," per DESIGN.md §7.2's
// no-token no-op rule.
func NewGitHubFromEnv() (client *GitHubClient, ok bool) {
	token := os.Getenv("GITHUB_TOKEN")
	repository := os.Getenv("GITHUB_REPOSITORY")
	if token == "" || repository == "" {
		return nil, false
	}
	prNumber, ok := pullRequestNumber(os.Getenv("GITHUB_EVENT_PATH"))
	if !ok {
		return nil, false
	}
	baseURL := os.Getenv("GITHUB_API_URL")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	return &GitHubClient{
		httpClient: http.DefaultClient,
		baseURL:    baseURL,
		token:      token,
		repository: repository,
		prNumber:   prNumber,
	}, true
}

func pullRequestNumber(eventPath string) (int, bool) {
	if eventPath == "" {
		return 0, false
	}
	data, err := os.ReadFile(eventPath)
	if err != nil {
		return 0, false
	}
	var event struct {
		PullRequest *struct {
			Number int `json:"number"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(data, &event); err != nil || event.PullRequest == nil {
		return 0, false
	}
	return event.PullRequest.Number, true
}

// Repository returns "owner/repo", as NewGitHubFromEnv already parsed it, so
// a caller logging adoption (bot, JRY-015) doesn't re-parse the environment
// a second time.
func (c *GitHubClient) Repository() string { return c.repository }

// PRNumber returns the pull request this client posts to.
func (c *GitHubClient) PRNumber() int { return c.prNumber }

type issueComment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

// PostOrUpdate lists the PR's issue comments (first page only — a PR with
// over 100 other comments could push jerry's own comment past page 1,
// causing a duplicate post rather than an update; accepted per the ticket's
// scope, not silently absorbed) and either PATCHes the one carrying
// CommentMarker or POSTs a new comment when none does.
//
// It returns ordinary Go errors on request/API failure. It does not itself
// implement "never fail the pipeline" — that swallowing is the caller's
// (bot, JRY-015) call to make for a comment tool as a whole.
func (c *GitHubClient) PostOrUpdate(body string) error {
	if !strings.Contains(body, CommentMarker) {
		body += "\n" + CommentMarker
	}

	comments, err := c.listComments()
	if err != nil {
		return err
	}
	for _, comment := range comments {
		if strings.Contains(comment.Body, CommentMarker) {
			return c.sendComment(http.MethodPatch,
				fmt.Sprintf("%s/repos/%s/issues/comments/%d", c.baseURL, c.repository, comment.ID), body)
		}
	}
	return c.sendComment(http.MethodPost,
		fmt.Sprintf("%s/repos/%s/issues/%d/comments", c.baseURL, c.repository, c.prNumber), body)
}

func (c *GitHubClient) listComments() ([]issueComment, error) {
	url := fmt.Sprintf("%s/repos/%s/issues/%d/comments", c.baseURL, c.repository, c.prNumber)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list comments: unexpected status %d", resp.StatusCode)
	}

	var comments []issueComment
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return nil, fmt.Errorf("list comments: decoding response: %w", err)
	}
	return comments, nil
}

func (c *GitHubClient) sendComment(method, url, body string) error {
	payload, err := json.Marshal(struct {
		Body string `json:"body"`
	}{Body: body})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: unexpected status %d", method, url, resp.StatusCode)
	}
	return nil
}

func (c *GitHubClient) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

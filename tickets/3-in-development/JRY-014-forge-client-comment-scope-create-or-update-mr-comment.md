---
id: JRY-014
title: Forge client, comment scope: create-or-update MR comment
project: jerry
depends-on: [JRY-001]
spawned-by: []
impact: high
complexity: medium
cost: M
---

# JRY-014 — Forge client, comment scope: create-or-update MR comment

## Outcome

jerry can post a comment to a merge request on one forge (the current forge, chosen for the
one-at-a-time rule) and, on a later push to the same MR, update that same comment instead of
adding a second one. Nothing calls this yet — it is the primitive `bot` (JRY-015) needs.

## Description

DESIGN.md §7.2/§7.4: the comment bot is the design's whole point of leverage — decisions arrive
where the work happens instead of requiring anyone to go looking. This ticket builds only the
forge-side primitive that makes that possible, scoped tight per PLAN.md's own warning against
building a general client no consumer has exercised yet: **one interface, one forge, token read
only from the CI environment** (never from a config file or flag — it must not be persisted),
**comment create-or-update** (never a second comment on the same MR — PLAN.md calls this a
correctness requirement, not a nicety, because a bot that posts a fresh comment per push is a
bot people mute).

Explicitly out of scope, deferred to `crawl`: pagination and rate-limit handling. This client's
one call per push never lists more than the single comment it owns, so there is nothing to
paginate yet — `crawl` is "the first thing that exercises" those concerns (PLAN.md), and adding
them here would be exactly the "speculative general client built against no consumer" PLAN.md
warns is the other failure mode.

The token is the first thing in this design needing write scope on a merge request — flagging
it here since it is the first ticket a security review can block on scope/handling, though
`bot-scaffold` (not yet filed) is where the requirement gets documented for scaffolded repos.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/JRY-014-forge-comment
```

WIP commits locally as you go. Publish only per the project's commit policy (no push/MR
without explicit user approval); tidy WIP into atomic commits before presenting (root-path
child, rules §0).

### Prerequisite gate (hard)

None — JRY-001 is already in `6-done/`, merged to `main`.

### Confirmed design decisions (do not deviate without asking)

1. **The one forge is GitHub.** This repo is hosted on GitHub (`git remote -v`), `jerry init
   --forge`'s default is `github` (`internal/cli/init.go`), and GitHub was the first forge
   proven end-to-end (JRY-004, before GitLab's JRY-009). Building the comment client against
   the forge jerry itself already runs on is the cheapest place to dogfood it.
2. **`net/http` + `encoding/json` against the GitHub REST API directly — no SDK dependency.**
   `go.mod` carries exactly two direct dependencies (`cobra`, `yaml.v3`) today; one
   create/list/update call each does not justify a third. Base URL is `GITHUB_API_URL` if set
   (GitHub Enterprise), else `https://api.github.com`. Every request sends
   `Authorization: Bearer <token>`, `Accept: application/vnd.github+json`, and
   `X-GitHub-Api-Version: 2022-11-28`.
3. **Package `internal/forge`.** Public surface:
   ```go
   // Commenter posts or updates the one comment jerry owns on a pull request.
   // One implementation today (GitHub); a second forge ports against this
   // interface once bot (JRY-015) is a proven consumer (DESIGN.md §7).
   type Commenter interface {
       PostOrUpdate(body string) error
   }

   // CommentMarker tags every comment jerry posts, so PostOrUpdate can find
   // its own comment among a PR's others instead of guessing by content.
   const CommentMarker = "<!-- jerry:bot-comment -->"

   // NewGitHubFromEnv builds a client from the GitHub Actions environment.
   // ok is false — never an error — when the environment isn't a pull-request
   // CI run (no token, or not running against a PR): the caller (bot) treats
   // that as "nothing to do," per DESIGN.md §7.2's no-token no-op rule.
   func NewGitHubFromEnv() (client *GitHubClient, ok bool)

   // Repository and PRNumber expose what NewGitHubFromEnv already parsed, so
   // a caller logging adoption (bot, JRY-015) doesn't re-parse the same
   // environment a second time.
   func (c *GitHubClient) Repository() string // "owner/repo"
   func (c *GitHubClient) PRNumber() int
   ```
4. **Token, repo and PR number all come from the environment, never a flag or config file.**
   `GITHUB_TOKEN` (the Actions job token), `GITHUB_REPOSITORY` (`owner/repo`), and the PR
   number parsed from `GITHUB_EVENT_PATH`'s JSON body (`.pull_request.number`) — the
   documented way to get a PR number in a `pull_request` workflow, robust to merge-queue and
   fork-PR refs that a `GITHUB_REF` regex is not. Any of the three missing or unparseable →
   `NewGitHubFromEnv` returns `ok = false`.
5. **Create-or-update, first page of comments only.** `PostOrUpdate` lists the PR's issue
   comments (`GET .../issues/{n}/comments`, default page size), and: a comment whose body
   contains `CommentMarker` → `PATCH` it; none found → `POST` a new one. Pagination is
   explicitly out of scope (ticket Description, `crawl` owns it later) — documented as a
   stated limitation in the function's doc comment: a PR with over 100 other comments could
   push jerry's own comment past page 1, causing a duplicate post rather than an update. That
   trade-off is accepted here, not silently absorbed.
6. **`PostOrUpdate` returns ordinary Go errors on request/API failure** (network error,
   non-2xx response). It does not itself implement "never fail the pipeline" — that swallowing
   is `bot`'s call (JRY-015), which must decide it for a comment tool as a whole, not just the
   forge client's slice of it.

### Tasks

#### Task 1 — `internal/forge/github.go`
`GitHubClient` struct (http.Client, base URL, token, owner, repo, PR number). `NewGitHubFromEnv`
reads and validates the three environment inputs (decision 4). `PostOrUpdate(body string)
error` implements decision 5: list → find by marker → PATCH or POST. Every posted body has
`CommentMarker` appended if the caller's body doesn't already carry it (defensive: `bot`
should not have to remember to add it).

#### Task 2 — `internal/forge/github_test.go`
`httptest.Server` fakes the three GitHub endpoints (list, create, update) — no live network
calls. Cases: no existing comment → POST; existing comment with the marker → PATCH, not a
second POST; existing comments present but none carrying the marker → POST (not mistaken for
a match); `NewGitHubFromEnv` returns `ok = false` for each of: missing `GITHUB_TOKEN`, missing
`GITHUB_REPOSITORY`, missing/malformed `GITHUB_EVENT_PATH`, an event payload with no
`pull_request` key (e.g. a push-to-main trigger).

### Acceptance test

```
go test ./internal/forge/... -v
just test
just lint
```
All green; every case in Task 2 present and passing.

### Docs update (mandatory when user-facing)

No user-facing surface yet — this ticket ships a library with no CLI command
(`internal/cli` gains nothing). `bot` (JRY-015) is what exposes this to a user and to
`CONTRIBUTING.md`'s token-scope documentation (`bot-scaffold`, unfiled, per PLAN.md); this
ticket's own docs step is "none."

### Finish (mandatory)

1. `go test ./...`, `just test`, `just lint` all clean.
2. No docs to update (see above).
3. Write the summary (files touched, decisions made, anything deferred) and suggest a
   Conventional Commit message, e.g. `feat(forge): GitHub create-or-update MR comment (JRY-014)`.
4. Tidy WIP commits into atomic ones (root-path child). Commit locally; do not push or open an
   MR without user approval. Verify `git diff --name-only origin/main...HEAD | grep '^tickets/'`
   prints nothing before pushing. Hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's build-step-2 row
  `forge-comment`, the first ticket in the design's core "decisions arrive where the work
  happens" step.
- 2026-09-04 — TO DO → READY: plan complete
- 2026-09-04 — READY → IN DEVELOPMENT: picked up

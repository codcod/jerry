---
id: JRY-015
title: jerry comment: post governing decisions to the merge request
project: jerry
depends-on: [JRY-012, JRY-014]
spawned-by: []
impact: critical
complexity: medium
cost: M
---

# JRY-015 — jerry comment: post governing decisions to the merge request

## Outcome

An engineer changing a governed path sees the decisions that govern it, in the merge request,
without going looking. DESIGN.md §1/§7.2 calls this the only argument for writing decisions
down that has ever worked; everything filed before this ticket is groundwork, everything after
it is scale.

## Description

`jerry comment` runs `related` (JRY-012) over the merge request's changed files and, when it
matches at least one decision, posts (or updates, via JRY-014's create-or-update primitive) a
comment listing the governing decisions. No output at all when nothing matches.

Two things PLAN.md flags as easy to drop in refinement and explicitly says must not be dropped
here, not deferred to a later ticket:

1. **The adoption counter.** A few lines that append one JSONL line per post (repo, MR, decision
   ids, timestamp) so §9's adoption question — is the read side actually used — is answerable
   from day one. `adoption-report` (unfiled) reads this log; deferring the counter to its own
   ticket means the measurement arrives after everything it was meant to inform.
2. **The no-token no-op.** A docs tool must never be the reason a merge request cannot merge.
   An absent or insufficiently-scoped CI token degrades this command to silence (exit 0, no
   comment, no error), never to a failed pipeline.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/JRY-015-bot
```

WIP commits locally as you go. Publish only per the project's commit policy (no push/MR
without explicit user approval); tidy WIP into atomic commits before presenting (root-path
child, rules §0).

### Prerequisite gate (hard)

JRY-012 (`internal/match` + `jerry related`) and JRY-014 (`internal/forge`) must both be in
`6-done/` and merged to `main` — this ticket has nothing to call otherwise. Both are currently
`2-ready/`, not yet picked up: **stop and confirm both shipped and merged before starting.**

### Confirmed design decisions (do not deviate without asking)

1. **In-process, not a subprocess.** "`jerry comment` runs `related`" (Description) means the
   same query, called as a library: `openCorpus(g)` then `match.Resolve` per changed path
   (JRY-011/012's own pipeline), not shelling out to the `related` binary. Same for the forge
   client: `internal/forge.NewGitHubFromEnv` / `PostOrUpdate` (JRY-014), called directly.
2. **Changed files reuse `changedFiles(cfg.Root, base)`** (`internal/cli/validate.go`, already
   powers `validate --diff`). `comment` adds its own `--base` flag, default `"origin/main"`,
   same default as `validate`'s.
3. **Aggregation across changed files, deduped by document.** A decision matching three
   changed files is posted once, not three times: collect `match.Resolve` results across every
   changed path into a set keyed by `Doc.Path`, keeping each document's highest specificity
   score seen. Sort the aggregate most-specific-first (ties broken by `Doc.Path`, same rule as
   JRY-011 decision 3) — deterministic comment content run-to-run when nothing changed.
4. **Comment body:** a Markdown bullet list, one line per governing decision — title and
   repo-relative path (no forge URL: this design has no host/link-resolution scope yet, and a
   raw path is enough for someone to open the file). `forge.PostOrUpdate` appends
   `forge.CommentMarker` itself (JRY-014 decision 3) — `comment` does not need to.
5. **Two silent no-ops, both exit 0, both skip the adoption log (nothing was posted):**
   (a) zero governing decisions across every changed path — Description's "no output at all
   when nothing matches"; (b) `forge.NewGitHubFromEnv` returns `ok = false` — the no-token
   case. Checked in that order, before any network call.
6. **Every other runtime failure degrades to logged-but-silent, exit 0** — corpus load error,
   `git diff` failure, `PostOrUpdate` returning an error (rate limit, network, non-2xx). Print
   the error to stderr for CI-log visibility, but do not fail the step: DESIGN.md §7.2 states
   the rule at the level of the whole command, not just the missing-token case — "a docs tool
   must never be the reason a merge request cannot merge." A **flag-usage error** (e.g. an
   unparseable `--base`) is the one exception: it fails before any of this runs, the same way
   every other command's flag validation does, since that is a CI-wiring mistake worth
   surfacing loudly the first time (`bot-scaffold`, unfiled, is where that wiring happens).
7. **Adoption log:** `jerry-adoption.jsonl` at `cfg.Root` (a plain committed file, per
   DESIGN.md's "a file in the repository, not a metrics backend" — mirrors `index.DefaultPath`
   being a hardcoded, not configurable, constant). Overridable via a `--adoption-log` flag for
   tests, mirroring `index`'s `--output` flag. One line appended **only on a successful post**
   (create or update): `{"repo":"<owner/repo>","pr":<number>,"decisions":["<doc
   path>",...],"timestamp":"<RFC3339>"}`, using `client.Repository()` / `client.PRNumber()`
   (JRY-014) rather than re-parsing the environment.

### Tasks

#### Task 1 — `internal/cli/comment.go`
`commentCmd(g *globals) *cobra.Command`: `Use: "comment"`, `Args: cobra.NoArgs`,
`Annotations: map[string]string{kindKey: kindWrite}` (it can write to the forge and to the
adoption log — not a read-only command like `related`). Flags: `--base` (default
`"origin/main"`), `--adoption-log` (default `adoptionLogPath = "jerry-adoption.jsonl"`).
`RunE`: `openCorpus` → `changedFiles` → aggregate via `match.Resolve` (decisions 2–3) → decision
5's first no-op check → render body (decision 4) → `forge.NewGitHubFromEnv` → decision 5's
second no-op check → `PostOrUpdate` → on success, append the adoption line (decision 7); wrap
the whole body after flag parsing so any returned error is logged to `cmd.ErrOrStderr()` and
swallowed (decision 6), always returning `nil` from `RunE` itself past that point.

#### Task 2 — Wire the command in
Add `commentCmd(g)` to `internal/cli/cli.go`'s `root.AddCommand(...)` list.

#### Task 3 — Fixtures + golden cases
`internal/cli/fixtures_test.go`: reuse `relatedFixture` (JRY-012) for the "has a match" case.
`internal/cli/golden_test.go`: because `comment` talks to a real forge API and reads CI
environment, its golden cases must stub both — inject a `forge.Commenter` (the interface from
JRY-014 decision 3) and an `httptest.Server` the same way `internal/forge/github_test.go`
already does, rather than exercising `NewGitHubFromEnv` against real env vars in the golden
harness. Cases: no matches (no comment attempted, exit 0, empty adoption log); a match with no
`GITHUB_TOKEN` set (no-op, exit 0, empty adoption log); a match with a fake token/server
(comment posted, one adoption-log line, correct JSON shape).

### Acceptance test

```
go test ./internal/cli/... -run TestGolden -v
go test ./internal/cli/... -run TestGoldenCoversEveryLeaf -v
just test
just lint
```
All green, `just test-update` run once first to record the new golden fixtures (same
convention as every other golden case).

### Docs update (mandatory when user-facing)

`CHANGELOG.md` `[Unreleased] / Added`: `jerry comment` — what it does, the no-token/no-match
no-op behaviour, and the `jerry-adoption.jsonl` file it starts writing (a new file appearing in
a user's repo the first time this runs is worth calling out explicitly, not left for them to
discover). `bot-scaffold` (unfiled) is what wires this into a scaffolded repo's CI and
documents the token requirement in `CONTRIBUTING.md` — out of scope here.

### Finish (mandatory)

1. `go test ./...`, `just test`, `just lint` all clean.
2. CHANGELOG.md updated.
3. Write the summary (files touched, decisions made, anything deferred) and suggest a
   Conventional Commit message, e.g. `feat(cli): add jerry comment, the MR decision bot (JRY-015)`.
4. Tidy WIP commits into atomic ones (root-path child). Commit locally; do not push or open an
   MR without user approval. Verify `git diff --name-only origin/main...HEAD | grep '^tickets/'`
   prints nothing before pushing. Hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's build-step-2 row `bot`,
  depending on `related` (JRY-012) and `forge-comment` (JRY-014) per PLAN.md's stated
  sequencing.
- 2026-09-04 — TO DO → READY: plan complete

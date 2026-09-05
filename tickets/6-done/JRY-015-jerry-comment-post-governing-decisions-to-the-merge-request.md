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
`"origin/main"`), `--adoption-log` (default `adoptionLogPath = "jerry-adoption.jsonl"`),
`--dry-run` (default `false`) — `write_safety_test.go` requires every `kindWrite` leaf to carry
this flag; when set, `RunE` runs match/aggregation/render as normal but prints the would-be
comment body to `cmd.OutOrStdout()` instead of calling `PostOrUpdate`, and skips the adoption-log
append (nothing was actually posted).
`RunE`: `openCorpus` → `changedFiles` → aggregate via `match.Resolve` (decisions 2–3) → decision
5's first no-op check → render body (decision 4) → `--dry-run` check (above) → `forge.NewGitHubFromEnv`
→ decision 5's second no-op check → `PostOrUpdate` → on success, append the adoption line
(decision 7); wrap the whole body after flag parsing so any returned error is logged to
`cmd.ErrOrStderr()` and swallowed (decision 6), always returning `nil` from `RunE` itself past
that point.

#### Task 2 — Wire the command in
Add `commentCmd(g)` to `internal/cli/cli.go`'s `root.AddCommand(...)` list.

#### Task 3 — Fixtures + golden cases
`internal/cli/fixtures_test.go`: reuse `relatedFixture` (JRY-012) for the "has a match" case,
but `relatedFixture`'s `gitInit` has no commit and no `origin/main` ref, and `changedFiles`
shells to `git diff base...HEAD` which errors without one — add a base commit and an
`origin/main` branch to the fixture (or a small variant of it) so `comment`'s `--base` resolves.
`internal/cli/golden_test.go`: because `comment` talks to a real forge API and reads CI
environment, its golden cases must stub both, the same way `internal/forge/github_test.go`
already does: point the real `forge.NewGitHubFromEnv` / `GitHubClient` at an `httptest.Server`
via the `GITHUB_TOKEN` / `GITHUB_REPOSITORY` / `GITHUB_EVENT_PATH` / `GITHUB_API_URL` env vars
(decision 1 — no `Commenter` seam, no injected interface). Cases: no matches (no comment
attempted, exit 0, empty adoption log); a match with no `GITHUB_TOKEN` set (no-op, exit 0, empty
adoption log); a match with a fake token/server (comment posted, one adoption-log line, correct
JSON shape).

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

- [x] Reviewer independence settled (step 0): **delegated** — the orchestrating reviewer
  authored the branch this same session, so steps 2–4a's audits ran in a fresh, independent
  sub-agent briefed adversarially (ticket, branch, `AGENTS.md`, `review-addendum.md`,
  `DESIGN.md` §1/§7.2/§10, `PLAN.md`'s `bot` row). Every finding below was re-verified by hand
  before being recorded (file:line / command output checked directly), per step 0's "delegation
  buys independence, not accuracy."
- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (steps 1, 2):
  `go test ./internal/cli/... -run TestGolden -v` (18/18 pass, including the three new
  `comment-*` cases), `TestGoldenCoversEveryLeaf` pass, `just test` all packages `ok`, `just
  lint` clean (`gofmt -l` empty, `go vet` clean), `just build` succeeds, `go.mod`/`go.sum` diff
  empty (dependency policy honoured). All seven confirmed design decisions and Tasks 1–3
  implemented exactly as specified, including Task 1's precise `RunE` ordering.
- [x] Quality audit (step 3) — idiomatic, matches sibling `write` commands' `--dry-run`
  convention; no unnecessary dependency added; error handling matches decision 6 (logged to
  stderr via `cmd.ErrOrStderr()`, swallowed). Coverage gaps noted below (F4, F5).
- [x] Consistency audit (step 4) — output-stream rule respected (`--dry-run` preview uses
  `cmd.Printf`, correctly a human-facing preview, not machine-consumed output); write-safety
  classification (`kindWrite` + `--dry-run`) correct; `match.go` diff minimal, reuses rather
  than reimplements. Governing-document staleness found and reconciled below (F2, F3).
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a): `just
  docs-check` exits 0, but that only checks include/xref syntax, not command coverage — see F1
  (blocking).
- [ ] Docs-readability pass (step 4b, optional): **conscious skip** — no docs-readability
  reviewer configured in this host session.
- [x] Findings recorded below with severity, class, and disposition (step 5).

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | docs-gap | — | `jerry comment` (new command, `--base`/`--adoption-log`/`--dry-run`) has no entry in `docs/user-manual/introduction.adoc`; review-addendum step 4a makes missing coverage for a new command blocking. | `docs/user-manual/introduction.adoc` section list (`Install` … `Resolve a path to its governing decisions` … `Global flags`) has no `comment` section; `grep -rni "comment\|adoption\|GITHUB_TOKEN" docs/` returns nothing relevant. | Add a section documenting `jerry comment`: what it does, `--base`/`--adoption-log`/`--dry-run`, the no-token/no-match no-op, and the `jerry-adoption.jsonl` file. |
| F2 | non-blocking | stale-xref | fixed inline | `DESIGN.md`'s line-3 stamp still read "the rest of the roadmap remains intent, not code" after this ticket shipped §7.2's second bullet (the comment bot). | `DESIGN.md:3` (pre-fix). | Fixed: stamp now reads "…§7.2's `related` command and merge-request comment bot are implemented…", version bumped to 2.7, and a §11 revision-history line added (commit `fadb283` on `feat/JRY-015-bot`). |
| F3 | non-blocking | stale-xref | fixed inline | `internal/forge/github.go`'s `Commenter` doc comment implied bot (JRY-015) would prove the interface as a consumer, but decision 1 deliberately keeps `comment` off the interface (concrete `*GitHubClient` called directly) — the comment's premise no longer held. | `internal/forge/github.go:19-24` (pre-fix) vs. `internal/cli/comment.go` (calls `forge.NewGitHubFromEnv()` / `client.PostOrUpdate(body)` directly, no `Commenter` reference). | Fixed: comment reworded to state bot calls the concrete type directly by design, and a second forge implements `Commenter` once one is actually needed (commit `fadb283`). |
| F4 | non-blocking | test-gap | noted | `--dry-run` is untested — the only `write` leaf whose dry-run path has no golden case (every sibling write command does: `init`, `new adr`/`sd`, `supersede`, `hooks install`/`uninstall`). | `internal/cli/golden_test.go` `goldenCases` — three `comment-*` entries, none passing `--dry-run`; `internal/cli/comment.go`'s dry-run branch never exercised by a test. | Add a `comment-dry-run` golden case if this command's test suite is revisited; not scheduled on its own. |
| F5 | non-blocking | test-gap | noted | Decision 3's cross-file dedup/highest-specificity/sort-by-path logic and decision 4's exact rendered body are exercised only by a trivial single-document, single-file fixture; the one case that posts never inspects the request body it sent. No direct unit test exists for `renderCommentBody`, `aggregateMatches`, or `appendAdoptionLog`. | `internal/cli/fixtures_test.go` (`commentMatchFixture`/`relatedFixture`: one `applies_to` entry, one changed path); `commentPostsCase`'s `check` only asserts `posted == true`, never the POST body. | Add a multi-decision fixture and assert the posted body / helper functions directly if this area sees more work; not scheduled on its own. |
| F6 | non-blocking | spec-unclear | fixed inline | `CHANGELOG.md`'s new entry (authored on this branch) read as if "absent" and "insufficiently-scoped" token both hit the same explicit no-op check; only the absent-token case does (decision 5b) — an insufficiently-scoped token instead falls through to `PostOrUpdate`'s error and the generic catch-all (decision 6). End behaviour is identical (silent, exit 0), so this was prose imprecision, not a functional bug. | `CHANGELOG.md` (pre-fix) vs. `internal/cli/comment.go`'s explicit no-op check vs. its generic `RunE` error swallow. | Fixed: reworded to name the absent-token no-op and the generic-failure no-op as the two distinct paths that both degrade the same way (commit `fadb283`). |

**Disposition summary:** 1 blocking (F1, unresolved — routes to rework), 3 `fixed inline` (F2, F3, F6), 2 `noted` (F4, F5). No `folded` or `new ticket` dispositions this round.

cost: estimated M, actual M

### Rework fix record — round 1 (commits 3d8bf10..9d05ec9)

F1 fixed: added a `== Post governing decisions to a merge request` section to
`docs/user-manual/introduction.adoc`, between "Resolve a path to its governing decisions" and
"Formatting" (mirroring `related`'s section style) — covers `jerry comment`'s behavior,
`--base`/`--dry-run`/`--adoption-log`, the no-token/no-match no-op, and the new
`jerry-adoption.jsonl` file. Re-ran `go test ./internal/cli/... -run TestGolden -v`,
`TestGoldenCoversEveryLeaf`, `just test`, `just lint`, `just docs-check` — all green.
No other findings touched this round; F2–F6 were already dispositioned above.

### Scoped re-review — round 1 (commits 3d8bf10..b8fa740)

Reviewer independence: **delegated** (same reason as the first pass — the orchestrating
reviewer authored the round-1 fix this session). Scope per protocol §1: F1's fix plus the diff
that closed it (`docs/user-manual/introduction.adoc`), not a re-audit of F2–F6 or the original
implementation.

F1 confirmed resolved: the new section covers `--base`/`--dry-run`/`--adoption-log`, the
no-match/no-token no-op, and `jerry-adoption.jsonl`; style/placement matches neighbouring
sections; `just docs-check` clean.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| G1 | non-blocking | correctness | fixed inline | The new section's own prose overclaimed: it said a missing token degrades "the same way" as any other failure — "a message on stderr, exit 0" — but a missing token (`NewGitHubFromEnv` returns `ok=false`) returns `nil` with no error, so `RunE`'s stderr `Fprintf` never runs; that path is purely silent, not logged. Only an *insufficiently-scoped* token (which passes `NewGitHubFromEnv`'s checks, then fails at `PostOrUpdate`) actually logs to stderr. | `internal/cli/comment.go:53-58` (`RunE` prints only on non-nil `err`) and `:89-92` (`if !ok { return nil }`, no error); `internal/forge/github.go:47-56` (`NewGitHubFromEnv` returns `ok=false` on empty token, never an error). | Fixed: reworded to separate the missing-token silent no-op from the logged-but-silent path any other failure (insufficiently-scoped token, network error, non-2xx) takes (commit `b8fa740`). |

**Disposition summary:** 0 blocking, 1 `fixed inline` (G1). No `noted`, `folded` or `new ticket`
dispositions this round.

**Verdict: no blocking findings — proceeds to `6-done/`.**

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's build-step-2 row `bot`,
  depending on `related` (JRY-012) and `forge-comment` (JRY-014) per PLAN.md's stated
  sequencing.
- 2026-09-04 — TO DO → READY: plan complete
- 2026-09-05 — plan amended inline: applicability audit found Task 1 missing the `--dry-run`
  flag `write_safety_test.go` requires of every `kindWrite` leaf (blocking) — added to Task 1's
  flags and `RunE` order. Also folded in two non-blocking findings (note-and-close): Task 3's
  fixture needs a base commit + `origin/main` ref for `changedFiles` to resolve, and Task 3's
  "inject a forge.Commenter" wording corrected to the actual env-var/`httptest.Server` pattern
  (decision 1 — no injected seam).
- 2026-09-05 — READY → IN DEVELOPMENT: picked up
- 2026-09-05 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-09-05 — IN REVIEW → REWORK: F1 blocking: no user-manual coverage for jerry comment
- 2026-09-05 — REWORK → IN REVIEW: findings fixed
- 2026-09-05 — IN REVIEW → DONE: scoped re-review clean, F1 resolved

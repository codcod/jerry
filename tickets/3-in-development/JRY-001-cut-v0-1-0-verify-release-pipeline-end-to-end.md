---
id: JRY-001
title: Cut v0.1.0: verify release pipeline end to end
project: jerry
depends-on: []
spawned-by: []
impact: high
complexity: low
cost: S
---

# JRY-001 — Cut v0.1.0: verify release pipeline end to end

## Outcome

jerry is installable at a pinned version: `go install github.com/codcod/jerry@v0.1.0` and
`brew install codcod/tap/jerry` both resolve, and a repository `jerry init` scaffolds pins
its CI to that real version instead of falling back to `@latest`.

## Description

Cut the first tagged release, `v0.1.0`, and verify every step of the release pipeline
actually works end to end rather than only passing `goreleaser check`/local snapshots.
Phase 1 (`jerry init`, the Tier 0 command set) is already built and shipped without a
ticket (see `PLAN.md`); this ticket is the release step, not new feature work.

One-time setup is already done: the `codcod/homebrew-tap` repository exists, and the
`HOMEBREW_TAP_GITHUB_TOKEN` repo secret is set. The GitHub remote (`git@github.com:codcod/jerry.git`)
matches `.goreleaser.yaml`'s `release.github` and `brews.repository` owner/name
(`codcod`), so that TODO comment in `.goreleaser.yaml` is already resolved and needs no
edit.

One real gap remains: `.goreleaser.yaml`'s `brews:` block still has a placeholder
`description: "TODO: one-line description of jerry"`. Left as-is, that literal string
ships into the Homebrew formula and `brew info jerry` output. Replace it with a real
one-line description before tagging — README.md's opening line is a good source: "A
single binary that scaffolds a repository of Architecture Decision Records and Solution
Designs, then owns every rule that governs it."

No code changes beyond that one config edit and `CHANGELOG.md`'s retitle — this ticket is
almost entirely a verification exercise against the real GitHub remote and the real tap
repo, not new capability.

## Implementation Plan

### Feature branch

`feat/JRY-001-cut-v0-1-0-release` in the jerry repo (root-path child; base `main`).

### Prerequisites

No `depends-on:`. One-time setup (tap repo, `HOMEBREW_TAP_GITHUB_TOKEN`) already
confirmed present.

### Confirmed decisions

- Fix the `brews:` description placeholder in `.goreleaser.yaml` as part of this ticket
  (user-confirmed 2026-09-01) — use README.md's opening line.
- The `release.github`/`brews.repository` owner/name TODO comment needs no code change:
  the real remote already matches `codcod/jerry`; only the comment is stale and may be
  deleted for cleanliness but is not load-bearing.

### Tasks

1. In `.goreleaser.yaml`, replace `description: "TODO: one-line description of jerry"`
   under `brews:` with `description: "A single binary that scaffolds a repository of
   Architecture Decision Records and Solution Designs, then owns every rule that governs
   it."`. Optionally delete the now-stale "TODO: confirm codcod/jerry is this repo's
   actual GitHub owner/name" comment block above `release:`, since the remote is already
   confirmed correct.
2. In `CHANGELOG.md`, retitle `## [Unreleased]` to `## [0.1.0] - 2026-09-01`, add a fresh
   empty `## [Unreleased]` above it, and update the link references at the bottom of the
   file per `RELEASING.md` step 1. There are no `tickets/6-done/` tickets yet to reconcile
   against (this is the first ticket ever filed), so no additional changelog entries are
   needed beyond the retitle.
3. Run `just dist-check` and `just test` locally; fix anything that fails before tagging.
4. Commit the `.goreleaser.yaml` and `CHANGELOG.md` changes on the feature branch.
5. Tag and push per `RELEASING.md`: `git tag v0.1.0 && git push origin v0.1.0` (requires
   user approval before pushing a tag — this is a publish action, not local WIP).
6. Watch the `release` GitHub Actions workflow run to completion; confirm the goreleaser
   job publishes binaries + `checksums.txt` for darwin/linux/windows on amd64/arm64, and
   creates the GitHub Release.
7. Confirm the `homebrew-tap` repository received an updated `jerry.rb` formula with the
   new description (not the TODO placeholder) and the correct version/checksums.
8. Confirm `docs-release.yml` ran (`continue-on-error`, so its failure must not block the
   release) and check whether the PDF/EPUB manual attached.

### Acceptance test

From a clean environment (or `unset` any cached `GOFLAGS`/module cache pin first):

```sh
go install github.com/codcod/jerry@v0.1.0
jerry version   # prints v0.1.0, not "dev"

brew install codcod/tap/jerry
jerry --help    # installs and runs
```

Both installs must resolve against the real, published `v0.1.0` — not a local build, not
`@latest`. `jerry init` in a scratch directory must emit CI that pins
`go install github.com/codcod/jerry@v0.1.0` (not `@latest`).

### Docs

`RELEASING.md` already documents the process this ticket follows — no doc changes needed
unless a step in it turns out to be wrong or out of date during execution, in which case
fix `RELEASING.md` inline as part of this ticket.

### Finish

Move to `4-in-review/` once the acceptance test passes and the release artifacts are
confirmed live. Record the tag and workflow run URL in the ticket's `## History` /
`## Review` for the reviewer to check without re-running the release.

## Review

### Applicability audit (pickup, 2026-09-01)

No blocking findings; every load-bearing plan assumption held against current repo state.

| finding | severity | disposition |
|---|---|---|
| Task 2's "update the link references at the bottom" of `CHANGELOG.md` refers to a section that does not exist in the current file (no-op) | non-blocking | note-and-close |
| `AGENTS.md` carries a slightly different one-line jerry description than the README line the plan chose for the Homebrew formula | non-blocking | note-and-close (README's wording stands) |

## History

- 2026-09-01 — created (TO DO). source: pickle ticket new
- 2026-09-01 — TO DO → READY: plan complete
- 2026-09-01 — READY → IN DEVELOPMENT: picked up

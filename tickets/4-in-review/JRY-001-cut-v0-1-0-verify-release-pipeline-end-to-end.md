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
go install github.com/codcod/jerry/cmd/jerry@v0.1.1
jerry version   # prints "dev" — go install never sets main.Version; only a
                # goreleaser build injects it via -ldflags. Pre-existing Go
                # ecosystem limitation, not a defect in this release.

brew install codcod/tap/jerry
jerry version   # prints 0.1.1
jerry --help    # installs and runs
```

**Amended inline** (rules §1) from the original text, which assumed `go install
github.com/codcod/jerry@v0.1.0` was the right module path and that `jerry version` would
print the tag regardless of install method — both wrong; see `## Review` for why. Final
verified tag is `v0.1.1`, not `v0.1.0` (see History). Both installs must resolve against
the real, published tag — not a local build, not `@latest`. `jerry init` in a scratch
directory must emit CI that pins `go install github.com/codcod/jerry/cmd/jerry@v0.1.1`
(not `@latest`), for both `--forge github` and `--forge gitlab`.

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

### Execution findings (acceptance testing, 2026-09-01)

The plan's own acceptance test surfaced three real defects while cutting the release. All
three were fixed inline, on the same ticket branch, per user approval at each step (rules
§1 — "plan amended inline").

| finding | severity | class | disposition |
|---|---|---|---|
| `HOMEBREW_TAP_GITHUB_TOKEN` returned `401 Bad credentials` against `codcod/homebrew-tap` on the first `v0.1.0` release run — expired/invalid PAT, not a config bug | blocking (for the acceptance test) | environment | fixed — user rotated the PAT; re-ran the same workflow run and it succeeded |
| `go install github.com/codcod/jerry@vX.Y.Z` never resolves: the module root has no `main` package, only `cmd/jerry` does. Broke the documented install command in `README.md`, `RELEASING.md`, `docs/user-manual/introduction.adoc`, both scaffold `README.md`/`CONTRIBUTING.md` templates, and — most seriously — both scaffold CI templates (`docs.yml`, `.gitlab-ci.yml`) that `jerry init` emits into every scaffolded repository | blocking | correctness | fixed — [PR #2](https://github.com/codcod/jerry/pull/2), released as `v0.1.1` since `v0.1.0`'s tag is immutable |
| `docs-release.yml`'s `on: release: types: [published]` trigger never fires: `release.yml` publishes via goreleaser using the default `GITHUB_TOKEN`, and GitHub Actions does not cascade `GITHUB_TOKEN`-created events to other workflows. Confirmed by zero runs of `docs-release.yml` across both `v0.1.0` and `v0.1.1` | non-blocking (project already treats the manual as best-effort, `continue-on-error`) | correctness | fixed — [PR #3](https://github.com/codcod/jerry/pull/3), switched to `workflow_run`; verification deferred to the next real release per user decision (not worth cutting `v0.1.2` solely to prove it) |

`jerry version` printing `"dev"` for `go install`-based installs (vs. the real tag for
`brew install`) is a pre-existing Go-ecosystem limitation (`main.Version` is only set via
`-ldflags` at goreleaser build time), not something this release introduced — disposition:
**note-and-close**. A `debug.ReadBuildInfo()` fallback would fix it but is new feature
work, out of scope for a release-verification ticket.

## History

- 2026-09-01 — created (TO DO). source: pickle ticket new
- 2026-09-01 — TO DO → READY: plan complete
- 2026-09-01 — READY → IN DEVELOPMENT: picked up
- 2026-09-01 — [PR #1](https://github.com/codcod/jerry/pull/1) merged (`0c8a303`): goreleaser formula description + CHANGELOG retitle
- 2026-09-01 — tagged and pushed `v0.1.0`; release workflow run [33549010917](https://github.com/codcod/jerry/actions/runs/33549010917) failed at the Homebrew formula step: `401 Bad credentials` on `HOMEBREW_TAP_GITHUB_TOKEN`. GitHub Release + binaries + checksums still published successfully
- 2026-09-01 — user rotated `HOMEBREW_TAP_GITHUB_TOKEN`; re-ran run 33549010917, succeeded — [v0.1.0 release](https://github.com/codcod/jerry/releases/tag/v0.1.0), tap formula confirmed correct
- 2026-09-01 — acceptance test found `go install github.com/codcod/jerry@v0.1.0` does not resolve (blocking; see `## Review`)
- 2026-09-01 — plan amended inline: fix the install path in 8 files, updated the affected test assertion; user approved
- 2026-09-01 — [PR #2](https://github.com/codcod/jerry/pull/2) merged (`22bdaed`); tagged and pushed `v0.1.1`; release workflow run [33550559396](https://github.com/codcod/jerry/actions/runs/33550559396) succeeded — [v0.1.1 release](https://github.com/codcod/jerry/releases/tag/v0.1.1)
- 2026-09-01 — re-ran acceptance test against `v0.1.1`: `go install`, `brew install`/`brew upgrade`, and `jerry init --forge github|gitlab` all confirmed correct (see amended Acceptance test)
- 2026-09-01 — found `docs-release.yml` never ran for either release (blocking finding, non-blocking disposition; see `## Review`)
- 2026-09-01 — plan amended inline: switch `docs-release.yml` to a `workflow_run` trigger; user approved, deferred re-verification to the next real release
- 2026-09-01 — [PR #3](https://github.com/codcod/jerry/pull/3) merged (`7163509`)
- 2026-09-01 — IN DEVELOPMENT → IN REVIEW: acceptance test green against `v0.1.1`; all findings fixed and merged

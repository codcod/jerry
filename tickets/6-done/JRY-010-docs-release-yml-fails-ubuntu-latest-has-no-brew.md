---
id: JRY-010
title: docs-release.yml fails: ubuntu-latest has no brew
project: jerry
depends-on: []
spawned-by: []
impact: low
complexity: low
cost: S
---

# JRY-010 — docs-release.yml fails: ubuntu-latest has no brew

## Outcome

Every future release actually gets the AsciiDoc user manual (PDF/EPUB) attached to its GitHub
release, instead of `docs-release.yml` silently failing at the first step every time.

## Description

`docs-release.yml`'s `attach-manual` job runs on `runs-on: ubuntu-latest`. Its "Install
snowball" step does `brew update --quiet && brew install codcod/tap/snowball && snowball setup`
— Homebrew is not installed on GitHub's Linux runner image at all, so this fails immediately
with `brew: command not found` (exit 127). Confirmed live during JRY-009's `v0.2.0` release cut
(2026-09-02): [run 33678453909](https://github.com/codcod/jerry/actions/runs/33678453909)
failed at exactly this step. The job is `continue-on-error`-guarded at the workflow level
(JRY-001 F4/decision), so the failure never blocked or unpublished the `v0.2.0` release itself —
but the manual has still never actually been attached to a jerry release, across `v0.1.0`,
`v0.1.1`, and now `v0.2.0`.

This is a known-solved problem: the sibling project `morty`
(`~/Projects/private/unity/projects/morty`, a separate repo using the same
`snowball`-based docs-release scaffold) hit the identical failure and fixed it in
`fix(ci): run docs-release on macos-latest, not ubuntu-latest` (commit `0b299b6`) — GitHub's
macOS runner images ship Homebrew preinstalled, Linux ones don't. Its current
`.github/workflows/docs-release.yml` job header reads:

```yaml
  attach-manual:
    if: github.event_name == 'release' || github.event.workflow_run.conclusion == 'success'
    # macOS, not ubuntu-latest: this step's own "brew update first" comment always assumed a
    # preinstalled Homebrew, which is true only on GitHub's macOS runners. Confirmed live:
    # ubuntu-latest has no `brew` at all ("command not found"), so this job has never actually
    # completed before now.
    runs-on: macos-latest
```

jerry's `.github/workflows/docs-release.yml` job is otherwise structurally identical (checkout,
install snowball, `snowball build -o dist/docs` with `continue-on-error: true`, `gh release
upload`) — the fix is the same one-line `runs-on` change, carried by the same explanatory
comment morty already wrote.

**A second, confirmed bug in the same job.** jerry's `docs-release.yml` has only the
`workflow_run` trigger (no `release: published`), and its "Attach user manual to release" step
uses `TAG: ${{ github.event.workflow_run.head_branch }}` directly. Checked live against the
actual triggering event for `v0.2.0`'s release cut: `gh api
repos/codcod/jerry/actions/runs/33678453909 --jq '.event, .head_branch'` returns `workflow_run` /
**`main`** — not `v0.2.0`. So even with the runner fixed, `gh release upload "main" ...` would
target a release that doesn't exist and fail (or worse, silently no-op) every time. morty's
version never hits this because it resolves the tag with `gh release list --limit 1 --json
tagName --jq '.[0].tagName'` for exactly the `workflow_run` case, with a comment noting that
event carries no `release` payload to read a tag from directly. jerry has no `release:
published` branch to preserve, so the fix here is simpler than morty's: always resolve the tag
via that `gh release list` lookup, no `if` branching needed.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd /Users/nicos.karagieorgopulus/Projects/private/jerry
git checkout main
git checkout -b feat/JRY-010-docs-release-brew-fix
```

Root-path child (`path = "."`): tidy WIP commits into atomic ones before presenting, per the
project's commit policy. The only file this ticket touches is
`.github/workflows/docs-release.yml`.

### Prerequisite gate (hard)

`depends-on: []` — none. `gh auth status` must show an authenticated `codcod` GitHub account
(already confirmed active in this environment) — needed to manually dispatch the workflow and
inspect run/release state in Task 4.

### Confirmed design decisions (do not deviate without asking)

1. **Fix the runner OS: `runs-on: macos-latest`, not `ubuntu-latest`.** GitHub's macOS runner
   images ship Homebrew preinstalled; the Linux ones don't, which is the entire cause of the
   `brew: command not found` failure. Adopt the sibling project `morty`'s already-proven fix
   (`~/Projects/private/unity/projects/morty`, commit `0b299b6`) and carry an equivalent
   explanatory comment so a future reader doesn't revert it back to `ubuntu-latest` by habit.
2. **Fix tag resolution: replace `TAG: ${{ github.event.workflow_run.head_branch }}` with a
   dedicated "Determine the release tag" step that resolves the tag via `gh release list --limit
   1 --json tagName --jq '.[0].tagName'`.** Confirmed live (Description) that `head_branch` is
   `main`, not the tag, for the `workflow_run` event this job actually receives. jerry has no
   `release: published` trigger to preserve (unlike morty), so no `if`/branching is needed here —
   always resolve via the `gh release list` lookup, unless overridden by decision 3's manual
   input.
3. **Add a `workflow_dispatch` trigger with an optional `tag` input** (string, no default) so
   this job can be run on-demand against an existing release, without waiting for or cutting a
   new one. In the "Determine the release tag" step: if `github.event_name ==
   'workflow_dispatch'` and the `tag` input is non-empty, use it; otherwise fall back to
   decision 2's `gh release list` lookup. This is what makes the fix verifiable today, against
   the already-published `v0.2.0` release, and stays useful afterward for any future manual
   re-run (e.g. if a `snowball build` transient failure needs a retry without a new tag).
4. **No other change.** `continue-on-error` on the build step, the `release.yml`/CI pipeline,
   and the `attach-manual` job's existing `if: github.event.workflow_run.conclusion == 'success'`
   guard (extended to also allow `workflow_dispatch`) all stay as they are otherwise.

### Tasks

#### Task 1 — Fix the runner OS

`.github/workflows/docs-release.yml`: change `runs-on: ubuntu-latest` to `runs-on:
macos-latest` under the `attach-manual` job, with a comment adapted from morty's (decision 1).

#### Task 2 — Add the `workflow_dispatch` trigger

Add a `workflow_dispatch:` block alongside the existing `workflow_run:` trigger under `on:`,
with one optional string input `tag` (decision 3). Extend the job's `if:` guard to also run on
`github.event_name == 'workflow_dispatch'`.

#### Task 3 — Fix tag resolution

Add a "Determine the release tag" step (`id: tag`) before "Install snowball", implementing
decision 2's fallback logic and writing `tag` to `$GITHUB_OUTPUT`. Update the "Attach user
manual to release" step's `TAG` env var to `${{ steps.tag.outputs.tag }}` instead of
`github.event.workflow_run.head_branch`.

#### Task 4 — Verify against the live `v0.2.0` release

```
git push -u origin feat/JRY-010-docs-release-brew-fix
gh workflow run docs-release.yml --ref feat/JRY-010-docs-release-brew-fix -f tag=v0.2.0
gh run watch                              # or: gh run list --workflow=docs-release.yml
gh release view v0.2.0 --json assets --jq '.assets[].name'
```

Confirm the run completes on `macos-latest`, the "Determine the release tag" step resolves to
`v0.2.0` (from the manual input), `snowball build` succeeds (or, if it fails, the job still
completes per `continue-on-error` — either is acceptable, but the *upload* step must actually
run and find files if the build succeeded), and the `assets` list gains a `.pdf`/`.epub` pair
that wasn't there before.

#### Task 5 — Record the evidence

In this ticket's `## Review` (added at `4-in-review/`, not now): the manual-dispatch run URL,
whether `snowball build` succeeded, and the confirmed asset list on `v0.2.0` before/after.

### Acceptance test

1. A `workflow_dispatch` run of `docs-release.yml` (input `tag=v0.2.0`) completes without the
   `brew: command not found` failure, on `runs-on: macos-latest`.
2. The "Determine the release tag" step resolves the correct tag in both the manual-input case
   (Task 4) and, by code inspection, the `gh release list` fallback case (not independently
   triggerable without a fresh `workflow_run` event, per the Description).
3. The `v0.2.0` GitHub release's asset list gains the user manual (PDF and/or EPUB, whichever
   `snowball build` produces) after the run, confirmed via `gh release view v0.2.0 --json
   assets`.
4. `just build`/`test`/`lint`/`docs-check` stay clean (only a workflow YAML file changes; run
   for completeness per the project's standard gate).

### Docs update (mandatory when user-facing)

No user-facing surface — this is a CI/tooling-only fix to a GitHub Actions workflow file. No
`CHANGELOG.md` entry (Keep a Changelog tracks jerry's own behaviour, not its own release
pipeline's internals).

### Finish (mandatory)

1. Acceptance test green (including the live verification against `v0.2.0`); `just
   build`/`test`/`lint`/`docs-check` clean.
2. Write a summary (manual-dispatch run URL, confirmed asset list, whether `snowball build`
   itself succeeded or only the upload/skip path was exercised).
3. Suggested commit message: `fix(ci): run docs-release on macos-latest, fix tag resolution
   (JRY-010)`.
4. Tidy WIP commits into atomic ones (root-path child) before presenting.
5. Commit locally; publish only per the project's commit policy (no push/MR without approval).
   `pickle ticket move JRY-010 in-review --reason "acceptance green"` and hand back.

## Review

**Applicability gate (pickup):** independent sub-agent audit, no findings — all plan
assumptions still held true against current repo/board state (`runs-on: ubuntu-latest` and the
`head_branch` tag bug both still present, unchanged since 2026-09-02; morty's cited fix commit
`0b299b6` confirmed as described; `v0.2.0` still lacked manual assets). Proceeded without
amendment.

**Live verification (Task 4):** manual-dispatch run
[33724479793](https://github.com/codcod/jerry/actions/runs/33724479793) on
`feat/JRY-010-docs-release-brew-fix` (`workflow_dispatch`, input `tag=v0.2.0`), completed
successfully in 2m29s on `macos-latest`. "Determine the release tag" resolved `v0.2.0` from the
manual input. `snowball build` itself succeeded (the upload step ran and found files, not just
the skip path) — one Homebrew tap-trust annotation (`aws/tap`) appeared on the run but did not
fail it. `v0.2.0` release assets before: 7 items (`checksums.txt` + 6 platform tarballs), no
manual. After: same 7 plus `jerry-user-manual.pdf` and `jerry-user-manual.epub` — confirmed via
`gh release view v0.2.0 --json assets`.

The `gh release list` fallback path (non-`workflow_dispatch` case) was verified by code
inspection only, per the ticket's own acceptance-test note — not independently triggerable
without a fresh `workflow_run` event.

**Reviewer independence (step 0):** the reviewing agent authored the branch in this same
session, so steps 2–4a were delegated to an independent sub-agent, spawned fresh with no
memory of writing the code and briefed adversarially. Its findings were re-verified by hand
before recording below (`git diff main...feat/JRY-010-docs-release-brew-fix`, `echo '[]' | jq
-r '.[0].tagName'` → confirmed `null`). It also independently re-derived the live-run claim
(`gh run view 33724479793`, `gh release view v0.2.0`) and confirmed it accurate.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | other | noted | Checkout step's `ref` changed to `${{ github.event.workflow_run.head_sha \|\| github.sha }}` — a fourth code change needed to make checkout work under `workflow_dispatch` (where `workflow_run` is null), not enumerated in any Task or decision | `.github/workflows/docs-release.yml:39`; confirmed correct and necessary via the green live-dispatch run | none — correct as shipped; noting the plan under-specified this line for future ticket-writing quality |
| F2 | non-blocking | correctness | noted | If `gh release list` ever returns zero releases, `jq -r '.[0].tagName'` yields the literal string `null`, so `TAG` becomes `"null"` and `gh release upload "null" ...` fails with a confusing error instead of a clean one | `.github/workflows/docs-release.yml:54`; reproduced `echo '[]' \| jq -r '.[0].tagName'` → `null`. Same latent pattern as morty's original fix; unreachable on the actual golden path since this job only runs after a release already exists (job fires post-`release.yml`, or manual dispatch against a repo that already has releases) | guard with `if [ -z "$tag" ] \|\| [ "$tag" = "null" ]; then echo "::error::no releases found"; exit 1; fi` if ever revisited — not worth its own ticket for a repo that always has ≥1 release |
| F3 | non-blocking | docs-gap | fixed inline | Top-of-file header comment documented the `workflow_run` trigger's rationale but not the newly added `workflow_dispatch` trigger | `.github/workflows/docs-release.yml:1-11` (before fix) | fixed inline: added a two-line note on the new trigger, commit `44c8665` |

Disposition summary: 2 noted (F1, F2), 1 fixed inline (F3). No blocking findings, no follow-up
tickets spawned.

cost: estimated S, actual S

## History

- 2026-09-02 — created (TO DO). source: self-host: `docs-release.yml` failed with `brew:
  command not found` during the `v0.2.0` release cut (JRY-009's follow-on release step); the
  sibling project `morty` already carries the fix (`runs-on: macos-latest`)
- 2026-09-03 — TO DO → READY: plan complete
- 2026-09-03 — READY → IN DEVELOPMENT: picked up
- 2026-09-03 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-09-03 — IN REVIEW → DONE: reviewed: 2 noted, 1 fixed inline, no blocking findings
- 2026-09-03 — pushed `feat/JRY-010-docs-release-brew-fix`, opened PR #11
  (https://github.com/codcod/jerry/pull/11) against `main`; awaiting human merge

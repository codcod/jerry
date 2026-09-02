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

**A second, independent difference worth checking while in this file.** morty's version
resolves the release tag via a dedicated "Determine the release tag" step: for the `release`
event it reads `github.event.release.tag_name`, but for the `workflow_run` case (the one that
actually fires here, per JRY-001) it looks the tag up with `gh release list --limit 1 --json
tagName --jq '.[0].tagName'`, with a comment explaining `workflow_run` carries no `release`
payload. jerry's version instead uses `TAG: ${{ github.event.workflow_run.head_branch }}`
directly in the "Attach user manual to release" step — `head_branch` is not documented to equal
the tag name for a tag-push-triggered run, and this was never verified because the job has
never gotten past the `brew install` step to reach it. Whether this is also broken, and whether
to adopt morty's `gh release list` lookup, is for refinement to confirm once the runner fix lets
the job actually run far enough to test it.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: self-host: `docs-release.yml` failed with `brew:
  command not found` during the `v0.2.0` release cut (JRY-009's follow-on release step); the
  sibling project `morty` already carries the fix (`runs-on: macos-latest`)

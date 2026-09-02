---
id: JRY-003
title: Emitted CI installs a checksum-verified release binary instead of go install
project: jerry
depends-on: []
spawned-by: []
impact: high
complexity: medium
cost: M
---

# JRY-003 — Emitted CI installs a checksum-verified release binary instead of go install

## Outcome

A repository scaffolded by `jerry init` validates its documents in CI without a Go toolchain
and without reaching the module proxy: the pipeline downloads the pinned jerry release binary
for the runner's platform, verifies it against the published checksums file, and runs it. A
runner image with nothing but `curl` and `sh` is enough.

## Description

DESIGN.md §3.3 states jerry "must run in any CI image and in a pre-commit hook without a
toolchain behind it". The CI `jerry init` actually emits contradicts that: both templates
(`internal/scaffold/templates/github/.github/workflows/docs.yml`,
`internal/scaffold/templates/gitlab/.gitlab-ci.yml`) run

    go install github.com/codcod/jerry/cmd/jerry@__JERRY_VERSION__

inside a `golang:1.26` image, on every pipeline run. That needs a Go toolchain present and the
module proxy reachable, compiles jerry from source on each run, and discards the single-binary
property at the one place it was supposed to pay for itself. Recorded as divergence 2 in
DESIGN.md §10.

JRY-001 already verified that goreleaser publishes per-platform archives **and** a checksums
file, so the artifacts this needs exist and are pinned by tag. The change is to the two emitted
CI templates plus whatever token substitution `internal/scaffold/scaffold.go` needs (it already
substitutes `__JERRY_VERSION__` in `replaceTokens`); the `go install` line stays in the emitted
`CONTRIBUTING.md` as a documented fallback for anyone who wants it.

Two things to settle in refinement: how the runner's platform is detected in a shell one-liner
that works on both forges, and what the failure message says when the pinned tag has no release
asset — a scaffolded repo must fail loudly there, not silently fall back to `@latest`.

**Sequencing.** Should land before JRY-004 (real-forge proof), because it deletes two of the
failure modes JRY-004 exists to detect rather than proving them. Not recorded in `depends-on:`
pending approval.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new

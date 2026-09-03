---
id: JRY-013
title: Harden validate --diff: fix corpus/git path mismatch and base-ref handling
project: jerry
depends-on: [JRY-007]
spawned-by: []
impact: high
complexity: medium
cost: M
---

# JRY-013 — Harden validate --diff: fix corpus/git path mismatch and base-ref handling

## Outcome

`jerry validate --diff` reports every finding on the changed paths it actually has, on any
repository layout, base ref, and clone depth CI actually produces — instead of silently
discarding all of them and exiting 0 whenever `jerry.yaml` is not at the git root.

## Description

**The live defect, confirmed in the code (DESIGN.md §10 item 6, and the worst-ranked
divergence there: "a validator that passes silently is worse than one that is absent"):**
`internal/cli/validate.go`'s `changedFiles` runs `git -C <root> diff --name-only
<base>...HEAD`, which always emits paths relative to the git top-level regardless of `-C`.
`onlyIn` then filters findings by `changed[finding.Path]`, where `finding.Path` is
corpus-root-relative (`internal/doc/corpus.go`'s `Load` computes it via `filepath.Rel` against
`cfg.Root`, the `jerry.yaml` directory — `internal/config/config.go`). When the corpus root is
not the git root, every finding's path fails to match every changed-file path, every finding is
filtered out, and `validate --diff` exits 0 having checked nothing. Nothing in the current code
path guards against this; it is real, not hypothetical.

Fix: make the two path spaces agree before comparing — either rewrite `finding.Path` to be
git-root-relative before filtering, or rewrite the changed-files list to be corpus-root-relative
using the corpus root's offset from the git top-level (`git rev-parse --show-toplevel`).

**Then, per PLAN.md's scope for this row:** base-ref autodetection (so `--diff` doesn't require
the caller to already know the right ref), shallow-clone behaviour (CI checkouts are commonly
`--depth 1`; a missing base ref must fail clearly, not silently produce an empty diff), and
detached-HEAD behaviour (the state every CI job runs in). A clear, non-zero-exit failure when
the base ref is absent replaces today's silent "0 findings."

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's cross-cutting row
  `diff-hardening`, next in the cross-cutting queue since its dependency (`golden-tests`,
  JRY-007) is done and the underlying defect is already live and silent.

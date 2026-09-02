---
id: JRY-007
title: Command-level golden tests for the cobra tree
project: jerry
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# JRY-007 — Command-level golden tests for the cobra tree

## Outcome

Every command's actual user-visible behaviour is pinned: the cobra tree is executed in-process
against fixture repositories, and stdout, stderr and the exit code are byte-compared against
golden files. A change to what a user sees becomes a test failure rather than a review comment.

## Description

Phase 1 has structural tests (`internal/cli/write_safety_test.go` enforcing the `jerry/kind`
annotation and `--dry-run` on every write leaf) and package-level tests for rules, index,
frontmatter, config, scaffold and hooks. No test asserts what a user actually sees. Two of the
three defects found during the Phase 1 pass were exactly that class: cobra's `Print*` writing
to stderr and breaking the git hook, and the scaffold shipping a repo that failed its own CI.

`internal/cli/cli.go` is already shaped for this — commands are constructors taking `*globals`,
with no package-level root and no `init()` registration, so two trees in one process never share
flag state and a test can build a fresh tree per case. `cmd.OutOrStdout()` is used throughout.

Scope: fixture repositories (one clean, one with a representative finding of each severity),
in-process execution of each leaf command, and golden files for the three observable channels.
Exit codes matter as much as output — `validate` returning `errFailed` with `SilenceErrors` is
behaviour a golden test should hold, since a validator's exit code is what CI reads.

Follow `AGENTS.md` on fixtures: they are rendered from the production path, never hand-edited
(`just test-update`), and each golden test is paired with a `…FixtureContract` test asserting
the properties the fixture exists to demonstrate, so regenerating cannot quietly drop one.

**Why it is scheduled early.** `PLAN.md` v1 parked this in a cross-cutting section with no
scheduling rule, which is where infrastructure tickets go to be permanently outranked by
features. It is buildable today, every later ticket benefits, and nothing about it gets easier
by waiting. It is also the prerequisite for JRY-006 and for the `validate --diff` hardening,
both of which are assertions about observable output.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new

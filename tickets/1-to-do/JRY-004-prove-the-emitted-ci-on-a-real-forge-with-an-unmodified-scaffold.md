---
id: JRY-004
title: Prove the emitted CI on a real forge with an unmodified scaffold
project: jerry
depends-on: []
spawned-by: []
impact: high
complexity: medium
cost: M
---

# JRY-004 — Prove the emitted CI on a real forge with an unmodified scaffold

## Outcome

We know — rather than believe — that `jerry init`'s output works: a throwaway repository
scaffolded on GitHub and one on GitLab both pass their emitted pipelines unmodified, and both
fail when a deliberately broken document is added. Until that has happened once, DESIGN.md §6's
scaffold contract is untested.

## Description

This was `PLAN.md`'s original second row and went missing: the id it expected was taken by
JRY-002 (an unplanned defect spawned by JRY-001's review) and the row was never refiled. It is
the only thing that can tell us the scaffolded CI works.

The existing tests assert the workflow file's *content* and that `actionlint` accepts it.
Neither executes it. Everything that only fails at runtime is therefore unverified: a runner
image that cannot reach the network, a GitLab runner without the tooling the pipeline assumes,
a `rules:`/`on:` block that never triggers, a permissions default that blocks a step,
`jerry index --check` disagreeing with the index `init` just generated on a real checkout.

Scope:

1. Scaffold a throwaway repo for each forge with an unmodified `jerry init`, push, confirm both
   pipelines pass with no hand edits.
2. Add a deliberately broken document — the useful case is a copied, unfilled template, since
   that is the defect DESIGN.md §5 calls the most common — and confirm each pipeline fails, with
   the finding legible in the job output.
3. Record what was actually run and observed in the ticket's Review section. The deliverable is
   evidence, not code.

**Sequencing.** Best run after JRY-003, which replaces the `go install` step with a
checksum-verified binary download and so removes the toolchain and module-proxy failure modes
before this ticket goes looking for them. If JRY-003 slips, this can still run against the
current templates — the result is just less useful. Not recorded in `depends-on:` pending
approval.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new

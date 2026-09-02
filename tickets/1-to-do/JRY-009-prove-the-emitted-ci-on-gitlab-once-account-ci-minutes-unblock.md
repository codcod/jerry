---
id: JRY-009
title: Prove the emitted CI on GitLab once account CI minutes unblock
project: jerry
depends-on: []
spawned-by: [JRY-004]
impact: medium
complexity: low
cost: S
---

# JRY-009 — Prove the emitted CI on GitLab once account CI minutes unblock

## Outcome

We know, rather than believe, that `jerry init --forge gitlab`'s emitted pipeline actually runs
and passes on a real GitLab.com runner, and fails legibly on a deliberately broken document —
closing the half of JRY-004's real-forge proof that its own execution couldn't complete.

## Description

JRY-004 proved the GitHub half of DESIGN.md §6's scaffold contract end to end: an unmodified
`jerry init --forge github` scaffold passed its Actions run, and a copied/unfilled
`templates/adr-template.md` failed it with the exact defects named
(`id-format`/`placeholder`/`team-mismatch`/`date`). The GitLab half was attempted with the same
throwaway-repo approach (`nicos.ka/jerry-forge-proof-gitlab`, scaffolded, pushed, unmodified)
but its pipeline never left `pending` — no runner ever picked up either job, despite shared
runners showing enabled and online on the project. The pipeline page showed GitLab's identity-
verification banner; after the user completed identity verification, the pipeline (and a
follow-up push-triggered retry) still stayed `pending`, and `glab api .../pipelines` showed
`status: "pending"` with no assigned runner even after the `data-identity-verification-required`
flag flipped to `false`. This is an account-level CI-minutes/quota or review-hold issue, not a
defect in the emitted `.gitlab-ci.yml` template — the template's shape is identical in kind to
the GitHub one JRY-004 already proved (same install script, same `jerry validate`/`jerry index
--check` steps), so this ticket is a re-run of JRY-004's GitLab tasks once the account-level
blocker is actually gone, not new design work.

Scope: repeat JRY-004's Task 3/4 shape — scaffold a fresh throwaway GitLab repo (the old one,
`nicos.ka/jerry-forge-proof-gitlab`, is already marked for deletion), push unmodified, confirm
the pipeline passes; add the same unfilled `adr-template.md` copy, push, confirm it fails
legibly; record the evidence in this ticket's Review; delete the repo. Before starting, confirm
in the GitLab web UI (logged in) that the account no longer shows a CI-minutes/verification
hold — `glab`/`curl` against the API could not surface that banner reliably during JRY-004's own
attempt, so don't trust a bare "verification: false" flag alone; watch a real pipeline run to
completion before concluding the blocker is gone.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: review: JRY-004's own execution hit an account-level
  GitLab CI-minutes/identity-verification blocker that never resolved within that ticket's
  session; scoped down to GitHub-only proof and this ticket carries the deferred GitLab half

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

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's build-step-2 row `bot`,
  depending on `related` (JRY-012) and `forge-comment` (JRY-014) per PLAN.md's stated
  sequencing.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: pickle ticket new

---
id: JRY-019
title: jerry show: render one decision and its supersession chain
project: jerry
depends-on: [JRY-001]
spawned-by: []
impact: medium
complexity: low
cost: S
---

# JRY-019 — jerry show: render one decision and its supersession chain

## Outcome

`jerry show <ref>` renders one decision in the terminal and resolves and displays what it
supersedes and what superseded it, so reading a decision never requires manually chasing
pointers across files.

## Description

DESIGN.md §4.3/§7.2: render one decision, resolve and display its supersession chain. Small
and self-contained — the chain-walking logic (`Supersedes`/`SupersededBy` frontmatter, per
§4.3) belongs here first; `graph` (unfiled, step 7) later reuses it for the estate-wide view
rather than re-implementing it, the same reuse discipline PLAN.md applies to `site-graph`
wanting `graph`'s traversal logic.

No hard dependency beyond the corpus jerry already parses (`release`, done); soft coupling to
`search` (JRY-018) only in that both are terminal read commands sharing conventions, not in
data or code.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's build-step-3 row `show`.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: pickle ticket new

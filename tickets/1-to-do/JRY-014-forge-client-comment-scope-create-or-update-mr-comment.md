---
id: JRY-014
title: Forge client, comment scope: create-or-update MR comment
project: jerry
depends-on: [JRY-001]
spawned-by: []
impact: high
complexity: medium
cost: M
---

# JRY-014 — Forge client, comment scope: create-or-update MR comment

## Outcome

jerry can post a comment to a merge request on one forge (the current forge, chosen for the
one-at-a-time rule) and, on a later push to the same MR, update that same comment instead of
adding a second one. Nothing calls this yet — it is the primitive `bot` (JRY-015) needs.

## Description

DESIGN.md §7.2/§7.4: the comment bot is the design's whole point of leverage — decisions arrive
where the work happens instead of requiring anyone to go looking. This ticket builds only the
forge-side primitive that makes that possible, scoped tight per PLAN.md's own warning against
building a general client no consumer has exercised yet: **one interface, one forge, token read
only from the CI environment** (never from a config file or flag — it must not be persisted),
**comment create-or-update** (never a second comment on the same MR — PLAN.md calls this a
correctness requirement, not a nicety, because a bot that posts a fresh comment per push is a
bot people mute).

Explicitly out of scope, deferred to `crawl`: pagination and rate-limit handling. This client's
one call per push never lists more than the single comment it owns, so there is nothing to
paginate yet — `crawl` is "the first thing that exercises" those concerns (PLAN.md), and adding
them here would be exactly the "speculative general client built against no consumer" PLAN.md
warns is the other failure mode.

The token is the first thing in this design needing write scope on a merge request — flagging
it here since it is the first ticket a security review can block on scope/handling, though
`bot-scaffold` (not yet filed) is where the requirement gets documented for scaffolded repos.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's build-step-2 row
  `forge-comment`, the first ticket in the design's core "decisions arrive where the work
  happens" step.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: pickle ticket new

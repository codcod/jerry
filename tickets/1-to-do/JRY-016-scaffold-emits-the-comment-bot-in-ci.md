---
id: JRY-016
title: Scaffold emits the comment bot in CI
project: jerry
depends-on: [JRY-015]
spawned-by: []
impact: high
complexity: medium
cost: M
---

# JRY-016 — Scaffold emits the comment bot in CI

## Outcome

A repository scaffolded by `jerry init` gets the comment bot for free: its CI variant already
runs `jerry comment` on merge requests, and its `CONTRIBUTING.md` already states the token and
scope it needs. Nobody adopting jerry from today forward has to wire this up by hand.

## Description

`bot` (JRY-015) makes `jerry comment` work when invoked; this ticket is distribution, not new
capability — `jerry init` adds the comment job to the CI variant for the chosen forge (the one
forge in play, per the one-at-a-time rule) and documents, in the emitted `CONTRIBUTING.md`,
the token and the exact scope it must carry (merge-request write, nothing broader).

This is scaffold-only: it does not provision a token, create the CI variable, or grant any
permission — it documents the requirement so the human setting up the repository knows what to
create. Provisioning stays outside jerry's surface, consistent with "custom identity or auth"
being cut from the design (§8).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's build-step-2 row
  `bot-scaffold`, depending on `bot` (JRY-015) per PLAN.md's stated sequencing.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: pickle ticket new

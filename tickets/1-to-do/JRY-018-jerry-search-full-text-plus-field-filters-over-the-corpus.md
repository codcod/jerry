---
id: JRY-018
title: jerry search: full-text plus field filters over the corpus
project: jerry
depends-on: [JRY-001]
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# JRY-018 — jerry search: full-text plus field filters over the corpus

## Outcome

Someone onboarding onto a system, or writing a new design, can search the accepted decisions
in this repository from a terminal — full text plus field filters — offline, with nothing to
operate. This is the review trigger the scaffolded README already promises: check for a
contradicting decision before writing one.

## Description

DESIGN.md §7.2: `jerry search <query>` — full-text plus field filters (`--status`, `--scope`,
`--team`, `--since`), ranked, offline. Distinct from `related` (JRY-011/012): `search` answers
"is there a decision about X," `related` answers "which decisions govern this path." Both read
the corpus; neither depends on the other, so this ticket has no hard dependency on the
`applies_to` family (JRY-011 family) — it only needs the corpus jerry already parses (`release`,
done).

Ranking, tokenization, and the exact query syntax are open implementation questions for
refinement — DESIGN.md does not specify them beyond "ranked." No server, no database: the index
this builds (if any) is a file or an in-process structure, never a hosted component (§3.1, §8).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's build-step-3 row `search`.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: pickle ticket new

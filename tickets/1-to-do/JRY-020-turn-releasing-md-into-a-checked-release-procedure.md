---
id: JRY-020
title: Turn RELEASING.md into a checked release procedure
project: jerry
depends-on: [JRY-001]
spawned-by: []
impact: medium
complexity: low
cost: S
---

# JRY-020 — Turn RELEASING.md into a checked release procedure

## Outcome

Cutting a jerry release stops being a manual verification exercise repeated by hand every time
— the steps `JRY-001` performed once by hand (reconcile `CHANGELOG.md`, tag, verify goreleaser
publishes binaries + checksums, verify the tap formula installs, verify `go install` resolves)
are checked by a script or CI job, so a future release cannot skip a step by omission.

## Description

`RELEASING.md` currently documents the procedure `JRY-001` executed manually; nothing enforces
it. Every release since then re-derives the checklist from memory or from re-reading the
document. This ticket turns the document's steps into something checked — a script that runs
the verification steps `JRY-001`'s own history recorded (binaries + checksums published, tap
formula installs, `go install` resolves) and fails loudly if one is skipped, rather than a
markdown file trusted to be followed.

Scope is verification tooling around the existing manual release trigger, not automating the
decision to release — nobody wants a release cut without a human choosing to cut it.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's cross-cutting row
  `release-automation`.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: pickle ticket new

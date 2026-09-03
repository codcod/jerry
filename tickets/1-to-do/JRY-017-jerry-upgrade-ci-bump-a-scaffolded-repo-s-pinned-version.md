---
id: JRY-017
title: jerry upgrade-ci: bump a scaffolded repo's pinned version
project: jerry
depends-on: [JRY-003]
spawned-by: []
impact: medium
complexity: low
cost: S
---

# JRY-017 — jerry upgrade-ci: bump a scaffolded repo's pinned version

## Outcome

A repository scaffolded by an earlier jerry release can pick up a later one: `jerry
upgrade-ci` bumps that one repository's pinned version in its emitted CI, as an ordinary
reviewed merge request the repository's own owners run.

## Description

Without this, a bugfix can never reach a repository already scaffolded: `ci-binary-install`
(JRY-003) makes the emitted CI fetch a checksum-verified release binary at a **pinned**
version, which means no repository scaffolded by e.g. `0.1.0` ever picks up `0.1.1`'s fixes on
its own — nothing currently bumps that pin. `jerry upgrade-ci` is the one command that does,
run **per repository, by that repository's owners**, producing an ordinary reviewed merge
request — never an estate-wide sweep. PLAN.md is explicit that an automatic sweep bumping every
repository's pinned version at once is deliberately cut (§6, "Deliberately not tickets"): that
would be retroactive rule enforcement wearing a version number, not this ticket's scope.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's cross-cutting row
  `upgrade-ci`, next in the cross-cutting queue since its dependency (`ci-binary-install`,
  JRY-003) is done.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: pickle ticket new

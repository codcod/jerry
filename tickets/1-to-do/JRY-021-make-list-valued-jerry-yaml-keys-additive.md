---
id: JRY-021
title: Make list-valued jerry.yaml keys additive
project: jerry
depends-on: [JRY-001]
spawned-by: []
impact: high
complexity: low
cost: S
---

# JRY-021 — Make list-valued jerry.yaml keys additive

## Outcome

A repository's `jerry.yaml` can extend the built-in `placeholders` list and the required-section
lists, but cannot replace them — so no repository can switch a governance rule off by
overriding a list-valued key to empty or to a narrower set.

## Description

DESIGN v2 §3.2: list-valued `jerry.yaml` keys (`placeholders`, required-section lists) must be
additive — a repository's config extends the built-in defaults rather than replacing them.
Today, per the current config-loading behaviour, a repository can set one of these keys and
silently drop coverage the tool is supposed to guarantee everywhere.

This is a policy-integrity guarantee, not a feature: without it, `placeholder-escapes`
(JRY-006, done) added a scoped opt-out (`<!-- jerry:allow placeholder -->`, per-document) on
the explicit premise that the cheapest fix for a false positive should never be "delete the
phrase from `jerry.yaml`" — a wholesale key override does exactly that, at repository scope
rather than per-document, and would undercut JRY-006's own reasoning. No conflict between the
two: JRY-006's opt-out stays scoped and additive; this ticket closes the coarser bypass.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's cross-cutting row
  `config-additive`.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: pickle ticket new

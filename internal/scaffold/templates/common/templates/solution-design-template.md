---
schema_version: 1
title: Short title of the design
status: Draft              # Draft | In Review | Approved | Implemented | Superseded | Archived
team: your-team            # must match the folder name under teams/
date: YYYY-MM-DD           # the YYYY-MM must match the filename prefix
authors: [alice, bob]
related_adrs: []           # e.g. [ADR-0007, cross-cutting/ADR-0003]
# applies_to: [services/ledger]
---

# Short title of the design

## Problem Statement

What problem are we solving? Who is affected, and why now?

## Goals / Non-Goals

- **Goals:** what this design commits to achieving
- **Non-Goals:** explicitly out of scope, to prevent scope creep

## Proposed Design

Describe the design. Diagrams are welcome (link or embed).

## Alternatives Considered

What else was considered, and why was it not chosen?

## Risks & Trade-offs

Be honest about what could go wrong and what's being traded away.

## Rollout / Migration Plan

How does this get deployed? Any migration steps, feature flags, or phased
rollout plan?

## Related ADRs

List ADRs that came out of this design, or that this design depends on. Keep
`related_adrs` in the frontmatter in sync — every reference is checked.

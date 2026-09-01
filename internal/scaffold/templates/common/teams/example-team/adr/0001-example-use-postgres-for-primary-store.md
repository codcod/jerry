---
schema_version: 1
id: ADR-0001
title: Use PostgreSQL as the primary data store
status: Accepted
team: example-team
date: 2026-01-15
deciders: [example-author]
---

# ADR-0001: Use PostgreSQL as the primary data store

## Context

This is a placeholder ADR demonstrating the convention. Delete this file and the
`example-team` folder once your real teams are set up.

## Decision

Use PostgreSQL as the primary store for new services in this team, unless a
specific workload requires otherwise.

## Consequences

- Consistent tooling and operational knowledge across services.
- Some workloads (for example heavy time-series) may still need a specialized
  store, so this is a default rather than a prohibition.

## Alternatives Considered

- **MySQL** — comparable, but the team has deeper Postgres operational experience.
- **DynamoDB** — rejected for now: designing around its access patterns is a
  skill the team does not yet have.

## Related

- Nothing yet. Link solution designs or other ADRs here as they appear.

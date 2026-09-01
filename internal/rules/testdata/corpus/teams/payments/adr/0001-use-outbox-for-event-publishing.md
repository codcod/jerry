---
schema_version: 1
id: ADR-0001
title: "Use the outbox pattern: at-least-once publishing for the ledger"
status: Accepted
team: payments
date: 2026-01-05
deciders: [ada]
---

# ADR-0001: Use the outbox pattern

## Context

Dual writes to Postgres and Kafka lose events when the second write fails.

## Decision

Write events to an outbox table in the same transaction, then relay them.

## Consequences

At-least-once delivery means consumers must be idempotent.

## Alternatives Considered

- **Dual writes** — rejected, it is the problem.

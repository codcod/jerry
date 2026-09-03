---
schema_version: 99
id: ADR-0002
title: "Schema version from the future"
status: Accepted
team: payments
date: 2026-01-06
deciders: [ada]
---

# ADR-0002: Schema version from the future

## Context

This fixture pins the tolerance direction: a document claiming a schema_version far ahead of
what this binary knows must warn, never error, and must otherwise validate cleanly.

## Decision

Claim `schema_version: 99` while being structurally clean in every other respect.

## Consequences

`jerry validate` reports exactly one warning for this document, and no errors.

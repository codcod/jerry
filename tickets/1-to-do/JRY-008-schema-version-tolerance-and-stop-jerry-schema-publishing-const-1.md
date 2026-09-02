---
id: JRY-008
title: schema_version tolerance, and stop jerry schema publishing const 1
project: jerry
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# JRY-008 — schema_version tolerance, and stop jerry schema publishing const 1

## Outcome

`schema_version` starts doing the job DESIGN.md §3.6 gives it: an older jerry reading a newer
document warns instead of failing, and the JSON Schema `jerry schema` publishes stops rejecting
any version but 1. Upgrading jerry stops being a prerequisite for merging.

## Description

DESIGN.md §3.6 makes `schema_version` a charter item — "the schema is the durable asset; the
binary is replaceable" — and the field is written into every document `jerry new` and
`jerry supersede` create. Nothing reads it. There is no tolerance behaviour of any kind, and
DESIGN.md §3.6 (v1) never said what a bump would mean. Divergence 3 in DESIGN.md §10.

The active harm is in `internal/cli/schema.go`, which emits

    "schema_version": {"type": "integer", "const": 1}

for both kinds. `jerry schema` exists so any editor with a YAML language server validates
frontmatter for free — so the day v2 exists, every v2 document is flagged invalid in every
author's editor by jerry's own published schema. That is the exact inverse of the field's
stated purpose. `const: 1` becomes a floor (`minimum: 1`, or the newest version the binary
knows) rather than an equality.

DESIGN.md §3.6 now defines the rest, so this ticket implements a written rule rather than
inventing one:

- A bump means a change no `v(n-1)` reader can be trusted to interpret — a field whose meaning
  changed, or one that became load-bearing. Adding an optional field is not a bump; adding a
  rule is not a bump.
- At or below the known version: read with that version's rules.
- Above it: **warn, never error**, and check with the newest rules the binary has. An old binary
  must not turn a repository red for being newer than itself.

There is no v2 to migrate to yet, so the deliverable is the mechanism plus the tests that pin
its direction — including a document claiming a version from the future, asserting it warns and
still validates. Getting the direction wrong is the whole risk: erroring on the future is the
failure mode that makes a pinned-CI estate (DESIGN.md §6) unable to adopt anything.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new

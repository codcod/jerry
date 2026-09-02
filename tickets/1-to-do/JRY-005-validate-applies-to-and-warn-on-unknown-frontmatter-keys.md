---
id: JRY-005
title: Validate applies_to and warn on unknown frontmatter keys
project: jerry
depends-on: []
spawned-by: []
impact: high
complexity: low
cost: S
---

# JRY-005 — Validate applies_to and warn on unknown frontmatter keys

## Outcome

`jerry validate` reports a malformed `applies_to` entry as an error, and reports any
frontmatter key jerry does not recognise as a warning. A misspelled `applies-to:` stops being
invisible, so a decision cannot silently govern nothing.

## Description

DESIGN.md §4.1 (v1) claimed `applies_to` was "accepted and validated now". It is not: nothing
in `internal/rules/rules.go` reads the field. It is decoded into `doc.Front`
(`internal/doc/frontmatter.go`) and typed in `jerry schema` (`internal/cli/schema.go`), and
that is the whole of it. Corrected as divergence 1 in DESIGN.md §10.

Two halves, and the second is the load-bearing one.

**Validate the field.** Reject entries that cannot be a path — empty strings, absolute paths,
`..` traversal, whitespace-only. This is deliberately *not* glob-semantics work; matching is a
separate, larger piece and DESIGN.md §4.1 now scopes the field to paths only (service ids were
removed, since nothing resolves one until a catalogue exists).

**Warn on unknown keys.** `jerry fmt` preserves keys jerry does not know, in authored order,
and that must not change — a tool that silently deletes what it does not understand cannot be
trusted to write files at all. But nothing reports them either, so `applies-to:` (hyphen) or
`applies_too:` parses clean, round-trips through `fmt` intact, and governs nothing —
permanently and silently, on the one field the whole of DESIGN.md §7.2 depends on. A warning,
never an error: a repository may legitimately carry fields jerry has no opinion about.
`doc.KeyOrder` is already the list of known keys, so the check has its input.

Why this is urgent despite being small: every day of Phase 1 use accumulates documents whose
`applies_to` was never checked, and the read side will under-report on all of them without ever
saying so. Cheap now, a data-cleanup exercise later.

Note the interaction with DESIGN.md §5: warnings currently have nowhere to go in a green
pipeline. That is a separate problem (a warning count on success is the interim), and it does
not justify making this an error.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new

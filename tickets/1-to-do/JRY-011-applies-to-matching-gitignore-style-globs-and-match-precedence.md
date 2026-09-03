---
id: JRY-011
title: Applies_to matching: gitignore-style globs and match precedence
project: jerry
depends-on: [JRY-005]
spawned-by: []
impact: high
complexity: medium
cost: M
---

# JRY-011 — Applies_to matching: gitignore-style globs and match precedence

## Outcome

Given a changed path, jerry can answer which decisions' `applies_to` entries match it, and
when several decisions match the same path, a stated precedence rule — not matching order —
decides which one wins. Nothing surfaces this yet; it is the index `related` (JRY-012) queries.

## Description

JRY-005 validates that an `applies_to` entry *can* be a path (rejects empty, absolute, `..`
entries) but implements no matching — `internal/rules/rules.go:287-289`'s own comment says so:
"Matching semantics (globs, existence) is separate, larger work." No glob or path-matching
logic exists anywhere in the repo today (confirmed: no `glob`/`filepath.Match`/`path.Match`
hits outside that comment).

**Decide the glob dialect first, in this ticket, and write it into DESIGN.md before coding it**
(PLAN.md's own instruction). The recommendation is gitignore syntax: `.gitignore` and
CODEOWNERS are the two other path-matching files already present in every one of these
repositories, so the audience is pre-trained on the dialect. "Whatever the library did" is not
an acceptable outcome — a year from now someone must be able to answer why a path did or didn't
match without reading the implementation.

Scope: directory-prefix rules (a decision governing `internal/rules/` matches everything under
it), glob matching per-entry, and an explicit precedence rule for the case where two decisions'
`applies_to` both match one path (DESIGN.md §4.1 currently has no answer to this — it needs
one). Out of scope: service ids (DESIGN v2 narrowed `applies_to` to paths only; service ids
return, if ever, with `owners`) and anything that resolves matches into a command's output —
that consumption is `related` (JRY-012), a soft coupling, not built here.

Specify the semantics in DESIGN.md §4.1 first, then implement with table-driven tests, per
PLAN.md's explicit instruction for this row.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's build-step-1 row
  `applies-to-match`, following `applies-to-validate` (JRY-005, done) per PLAN.md's stated
  sequencing rule (matching lands after validation, before the read side).

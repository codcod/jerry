---
id: JRY-012
title: jerry related: resolve changed paths to governing decisions
project: jerry
depends-on: [JRY-011]
spawned-by: []
impact: high
complexity: medium
cost: M
---

# JRY-012 — jerry related: resolve changed paths to governing decisions

## Outcome

`jerry related --paths <files>` answers "which decisions govern the code I am about to
touch," offline, from the terminal. This is the first command that reads the `applies_to`
index rather than just validating it — the landing point DESIGN.md §1/§7.2 calls the actual
point of the project, and it is what `bot` (step 2, unfiled) will call in a merge request.

## Description

DESIGN.md §7.2: "`jerry related --paths <changed files>` — which decisions govern the code I
am about to touch. Requires `applies_to`, which Phase 1 already accepts." Phase 1 accepts and
(as of JRY-005) validates the field's shape; JRY-011 builds the matching engine. This ticket is
the first consumer: given a set of changed paths, resolve them against every decision's
`applies_to` (via JRY-011's matcher) and return the governing decisions, applying JRY-011's
precedence rule when several decisions match one path.

No command named `related` exists yet (confirmed against the full `internal/cli/*.go` command
list: `version`, `fmt`, `hooks`, `index`, `schema`, `init`, `validate`, `status`, `new`,
`supersede` — no `related.go`). Follow the conventions the existing commands already set:
`--format text|json` (mirroring `validate.go`'s `--format text,json,sarif,junit` and its
`--json` shorthand), and a versioned JSON envelope on the model of
`internal/cli/output.go`'s `FindingsEnvelopeSchema = "jerry.findings/1"` — this command's
envelope is `jerry.related/1`.

**Hard dependency on JRY-011** (applies_to matching): this command has nothing to query until
the matcher exists. PLAN.md states this as a sequencing rule for build step 1 ("`related` lands
after applies-to-match"), which is the prior sign-off for the `depends-on:` below; flagging it
here rather than treating it as silently pre-approved.

Out of scope: posting anywhere (that's `bot`, step 2, unfiled — a soft coupling, not a
dependency) and service-id resolution (still deferred to `owners`).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's build-step-1 row `related`,
  the read-side landing command that queries JRY-011's matcher.

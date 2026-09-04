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

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/JRY-011-applies-to-match
```

WIP commits locally as you go. Publish only per the project's commit policy (no push/MR
without explicit user approval); tidy WIP into atomic commits before presenting (root-path
child, rules §0).

### Prerequisite gate (hard)

None outstanding — JRY-005 (`applies_to` validation) is already in `6-done/`, merged to
`main` (commit `5914ad0`).

### Confirmed design decisions (do not deviate without asking)

1. **The dialect is hand-implemented, not a vendored gitignore library.** The ticket itself
   rejects "whatever the library did" as an outcome: a third-party parser's edge cases
   (negation ordering, `**` interpretation, character-class quirks) would become jerry's
   semantics by accident, unstated anywhere. `path.Match` (stdlib) does the per-segment glob
   work; no new `go.mod` dependency.
2. **Supported syntax.** Entries are already validated relative paths (JRY-005: non-empty,
   relative, no `..`). An entry ending in `/` is a **directory-prefix**: matches every path
   nested under it, recursively, no glob expansion inside it. An entry without a trailing `/`
   is matched **segment-by-segment** against the `/`-split candidate path: each pattern
   segment matches the candidate's corresponding segment via `path.Match` (`*`, `?`, `[...]`),
   except the literal segment `**`, which matches zero or more whole path segments (gitignore's
   double-star, restricted to positive matching). **No negation (`!`)** — `applies_to` is a flat
   allow-list per decision, not a layered ignore file, so there is nothing for a later negated
   entry to override within one list.
3. **Precedence: most-specific pattern wins.** CODEOWNERS' own "last-match-wins" (DESIGN.md
   §6) needs one ordered file; independent ADRs have no such order, so it does not transfer.
   Specificity, highest wins, computed per matching `(document, pattern)` pair:
   1. literal segment count in the pattern (a segment containing none of `*`, `?`, `[` counts;
      `**` and a directory-prefix's implicit remainder count 0) — more literal segments wins;
   2. tie-break: longer pattern string (more characters) wins;
   3. final tie-break: document `Path`, lexicographic — for determinism only.
   `Resolve` sorts every match most-specific first; it does **not** collapse to a single
   winner itself — `related` (JRY-012) decides whether to surface all governing decisions or
   just the head of the slice.
4. **New package `internal/match`**, parallel to `internal/index` and `internal/rules`.
   Public surface: `Resolve(corpus *doc.Corpus, changedPath string) []Match`, where
   `Match{Doc *doc.Document, Pattern string}` is one matching (document, pattern) pair.

### Tasks

#### Task 1 — Write the dialect and precedence rule into DESIGN.md §4.1
Replace the `applies_to` table row's "Nothing yet reads the field..." note (currently line 123)
and add a subsection documenting decisions 2 and 3 above, in the same declarative style as the
existing CODEOWNERS bullet (§6, ~line 235). This must land before or alongside the code, per
the ticket's own instruction — not as an afterthought.

#### Task 2 — `internal/match/match.go`
- `type Match struct { Doc *doc.Document; Pattern string }`
- `func Resolve(corpus *doc.Corpus, changedPath string) []Match` — iterate `corpus.Docs`, then
  each `doc.Front.AppliesTo` entry, keep every `(doc, pattern)` pair that matches, sort per
  decision 3.
- `func matchPattern(pattern, candidate string) bool` — directory-prefix short-circuit for a
  trailing `/`; otherwise split both on `/` and match segment-by-segment, with `**` handled by
  trying every split point (candidate paths are short — a handful of segments — so the
  recursive backtrack is O(1) in practice, not a scaling concern).
- `func specificity(pattern string) (literalSegments, length int)` — decision 3's score.

#### Task 3 — Table-driven tests, `internal/match/match_test.go`
Cases: directory-prefix match and non-match; `*` within a segment; `**` across segments
(zero, one, several); two decisions matching the same path at different specificity (assert
order); two decisions tied on literal-segment count but different pattern length (assert the
longer wins); an `applies_to` entry that matches nothing.

### Acceptance test

```
go test ./internal/match/... -v
just test
just lint
```
All green; the new table-driven cases in Task 3 all present and passing.

### Docs update (mandatory when user-facing)

DESIGN.md §4.1 (Task 1) — the dialect and precedence rule are the user-facing contract for
anyone writing `applies_to`. No CHANGELOG entry: this ships a library with no CLI surface yet;
`jerry related` (JRY-012) is what makes it observable to a user, and that ticket's own docs
step is where the CHANGELOG line belongs.

### Finish (mandatory)

1. `go test ./internal/match/... -v`, `just test`, `just lint` all clean.
2. DESIGN.md §4.1 updated.
3. Write the summary (files touched, decisions made, anything deferred) and suggest a
   Conventional Commit message, e.g. `feat(match): gitignore-style applies_to matching (JRY-011)`.
4. Tidy WIP commits into atomic ones (root-path child). Commit locally; do not push or open an
   MR without user approval. Verify `git diff --name-only origin/main...HEAD | grep '^tickets/'`
   prints nothing before pushing. Hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's build-step-1 row
  `applies-to-match`, following `applies-to-validate` (JRY-005, done) per PLAN.md's stated
  sequencing rule (matching lands after validation, before the read side).
- 2026-09-04 — TO DO → READY: plan complete
- 2026-09-04 — READY → IN DEVELOPMENT: picked up

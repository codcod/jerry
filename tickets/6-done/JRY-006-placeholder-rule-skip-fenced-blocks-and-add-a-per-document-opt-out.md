---
id: JRY-006
title: Placeholder rule: skip fenced blocks and add a per-document opt-out
project: jerry
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# JRY-006 — Placeholder rule: skip fenced blocks and add a per-document opt-out

## Outcome

An author whose decision legitimately contains a placeholder phrase — because it documents the
id convention, or quotes a template in a fenced block — can silence that one finding on that
one document, instead of deleting the phrase from `jerry.yaml` and switching the check off for
the whole repository.

## Description

`checkPlaceholders` (`internal/rules/rules.go`) is `strings.Contains` over `document.Raw`:
frontmatter, prose and fenced code blocks alike. DESIGN.md §5 calls this the most valuable check
in the catalogue — a copied, half-filled template is the single most common real defect — but
its false-positive path is unmanaged. A decision that discusses the `ADR-NNNN` form, or shows an
example frontmatter block, fails on a phrase it is legitimately talking about, and today the only
escape is `placeholders:` in `jerry.yaml`, which is repository-wide. The cheapest fix for a false
positive is therefore disabling the rule, which is the wrong incentive for the check that catches
the most. Recorded as divergence 5 in DESIGN.md §10 (the ticket's original text said "6"; the
table row is numbered 5 — corrected here during refinement).

Two changes, both now specified in DESIGN.md §5:

1. **Exclude fenced code blocks from the scan.** Quoting a template is the common legitimate
   case and never indicates an unfilled document.
2. **Add an inline opt-out**, `<!-- jerry:allow placeholder -->`. **Decided in refinement:
   per-document**, not per-phrase — the marker anywhere in the document's `Raw` disables
   `checkPlaceholders` entirely for that document. Simpler (no marker-argument parsing, no
   multi-marker handling) and matches this ticket's `complexity: low` / `cost: S` grade;
   DESIGN.md §5's current wording ("opts out of one phrase") is updated to match in the docs
   task below.

Watch the interaction with the empty-section rule: HTML comments are stripped before that check
so that `jerry new`'s guidance prompts do not count as content (`stripComments`). An opt-out
marker is also an HTML comment, so it must not accidentally become the only content in a
section and it must not make an empty section look filled.

Also confirm the existing test that asserts every default placeholder appears in some shipped
template still holds — it is what stops the list rotting into decoration, and templates are in
`skip-dirs` so they are not themselves scanned.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/JRY-006-placeholder-fenced-and-opt-out
```

Do all work on this branch, committing locally. `jerry` is a root-path child (`path = "."`),
so tidy WIP commits into atomic ones before presenting them; do not push or open a merge
request without user approval.

### Prerequisite gate (hard)

None. No `depends-on:`, clean tree, no unmerged branch to build on.

### Confirmed design decisions (do not deviate without asking)

1. **The opt-out marker is per-document, not per-phrase.** `<!-- jerry:allow placeholder -->`
   appearing anywhere in `document.Raw` disables `checkPlaceholders` entirely for that
   document — it does not target one phrase. Chosen in refinement for lower complexity; update
   DESIGN.md §5's wording ("opts out of one phrase") to match (see Docs update).
2. **Fenced-block exclusion covers triple-backtick fences only** (```` ``` ````…```` ``` ````),
   matched non-greedily and possibly spanning multiple fences per document — mirroring the
   existing `commentPattern` regex style already in `rules.go`. Tilde fences (`~~~`) and inline
   single-backtick spans are out of scope: the motivating case in the Description is a quoted
   frontmatter/template block, which is always triple-backtick fenced in this repo's own ADRs
   and templates.
3. **Line numbers for a placeholder finding are still computed against the unmodified
   `document.Raw`** via the existing `document.LineOf(placeholder)`. Fenced-block exclusion only
   changes whether `checkPlaceholders` fires for a given phrase, never how the reported line is
   computed. (Edge case accepted as-is: if the same phrase appears both inside and outside a
   fenced block, `LineOf` may point at the in-block occurrence — pre-existing behaviour for any
   repeated phrase, not something this ticket changes.)
4. **The opt-out marker is itself an HTML comment**, so `stripComments` (used by the
   empty-section rule) already strips it like any other comment. No extra interaction code is
   needed for "the marker must not make an empty section look filled" (Description) — write a
   test that confirms this holds rather than adding logic for it.

### Tasks

#### Task 1 — Exclude fenced code blocks from the placeholder scan

`internal/rules/rules.go`. Add, next to the existing `commentPattern` (around line 331):

```go
// fencedBlockPattern matches a fenced code block. Quoting a template is the
// common legitimate reason a real placeholder phrase appears in a document,
// and never indicates an unfilled one.
var fencedBlockPattern = regexp.MustCompile("(?s)```.*?```")
```

In `checkPlaceholders` (currently lines 263–270), scan a fenced-stripped copy instead of
`document.Raw` directly, but keep `document.LineOf` on the original `Raw` (decision 3):

```go
func checkPlaceholders(findings *Findings, document *doc.Document, options Options) {
	if strings.Contains(document.Raw, placeholderAllowMarker) {
		return
	}
	scanText := fencedBlockPattern.ReplaceAllString(document.Raw, "")
	for _, placeholder := range options.Placeholders {
		if strings.Contains(scanText, placeholder) {
			findings.errorf(document.Path, document.LineOf(placeholder), "placeholder",
				"template placeholder %q was never filled in", placeholder)
		}
	}
}
```

#### Task 2 — Add the per-document opt-out marker

Same file. Add the constant next to `commentPattern`/`fencedBlockPattern`:

```go
// placeholderAllowMarker silences checkPlaceholders for the whole document it
// appears in — the escape for a decision that legitimately discusses a
// placeholder phrase instead of leaving it unfilled.
const placeholderAllowMarker = "<!-- jerry:allow placeholder -->"
```

(Already wired into `checkPlaceholders` above — the early return.)

#### Task 3 — Unit tests

`internal/rules/rules_test.go`. Add a new `TestCheckPlaceholders` (direct unit test against
`checkPlaceholders`, not the golden corpus — no fixture regen needed for this ticket) covering:

1. A placeholder phrase inside a fenced code block only → no finding.
2. The same phrase also present outside a fenced block in the same document → one finding
   (proves exclusion is per-occurrence-context, not "any fence anywhere suppresses the doc").
3. `placeholderAllowMarker` present anywhere in the document, with an unfenced placeholder
   phrase elsewhere → no finding (per-document opt-out).
4. A document with the opt-out marker inside what would otherwise be an empty required
   section → confirm (via `stripComments`, or by running `checkSections` alongside) that the
   section still reads as empty — i.e. the marker does not "fill" it (decision 4).

Construct documents directly (`&doc.Document{Path: "...", Raw: "..."}`); no need to route
through `doc.LoadFile` or the `testdata/corpus` fixtures.

### Acceptance test

```
just test    # includes the new TestCheckPlaceholders cases and the existing
             # TestPlaceholdersMatchShippedTemplates (internal/scaffold/scaffold_test.go:92),
             # which must still pass unmodified
just lint
just docs-check
```

`go test ./internal/rules/... -run TestCheckPlaceholders -v` shows all four new cases passing.

### Docs update (mandatory when user-facing)

- `DESIGN.md` §5: reword "a document opts out of one phrase with an inline
  `<!-- jerry:allow placeholder -->` marker" to "a document opts out of the whole rule with an
  inline `<!-- jerry:allow placeholder -->` marker", and drop "Neither is built yet (§10)".
- `DESIGN.md` §10: remove the now-resolved row 5 from the divergence table (matching how JRY-003
  and JRY-005 closed their rows — see §11).
- `DESIGN.md` §11: add a `Version 2.3` bullet recording that JRY-006 closed divergence 5 —
  fenced blocks are excluded and the per-document `<!-- jerry:allow placeholder -->` marker
  ships — following the `Version 2.1`/`2.2` bullets' phrasing.
- `docs/user-manual/introduction.adoc`, `== Check it` section (around line 68): add one short
  paragraph documenting the `<!-- jerry:allow placeholder -->` marker for end users, next to the
  existing note about HTML-comment prompts not counting as content.

### Finish (mandatory)

1. Acceptance test green; `just build`/`just lint`/`just docs-check` clean.
2. Docs updated per above and registered (no new doc file, so no index change).
3. Write a summary (files touched, decisions made, anything deferred).
4. Suggest commit message, e.g. `feat(rules): skip fenced blocks and add placeholder opt-out (JRY-006)`.
5. Tidy WIP commits into atomic ones before presenting (root-path child).
6. Commit locally on the ticket branch. Do not push or open a merge request without user
   approval. Present the commit message; only after approval finalize and push, verifying
   `origin/main...HEAD` carries no `tickets/` path first (in-tree layout). Hand back.

## Review

- [x] Reviewer independence settled (step 0): the implementing agent authored the branch this
  session, so audits (steps 2–4a) were delegated to an independent sub-agent, briefed
  adversarially and instructed to find defects; its findings were re-verified by hand before
  being recorded below (per step 0's "delegation buys independence, not accuracy").
- [x] Implementation audit — acceptance test re-run (`just test`, `just lint`, `just
  docs-check`, plus `go test ./internal/rules/... -run TestCheckPlaceholders -v`), all green;
  every task and confirmed design decision verified against the diff (steps 1, 2).
- [x] Quality audit (step 3) — no dependency added (`go.mod` untouched), regex has no
  pathological behaviour beyond an accepted, ticket-documented limitation (nested/odd fence
  counts), test coverage adequate.
- [x] Consistency audit (step 4) — whole-repo grep for stale `DESIGN.md §` citations and prior
  placeholder-rule descriptions found nothing else to correct.
- [x] Documentation audit (step 4a) — `docs/user-manual/introduction.adoc` edit verified
  accurate and placed sensibly; `just docs-check` passes; `CHANGELOG.md` coverage was found
  missing (F1, blocking).
- [x] Docs-readability pass — no docs-readability reviewer available in this host; conscious
  skip (step 4b).
- [x] Findings recorded below, severity + class + disposition; disposition summary and cost
  line present (step 5).
- [x] Ticket moved per step 6 (see History).
- [x] Governing documents reconciled: `DESIGN.md` §5/§10/§11 (ticket's own docs task) and the
  version stamp (F2); `PLAN.md`'s own `JRY-006` row updated to reflect this review's outcome as
  part of concluding the ticket (step 7 addendum). `PLAN.md`'s `JRY-005` row is separately stale
  (still "to do" though done/merged) — pre-existing, not caused by this branch; noted (F5) rather
  than fixed here.
- [x] Remaining-tickets impact sweep done (step 8) — no ticket in `1-to-do/`/`2-ready/` depends
  on or references JRY-006; no impact.
- [x] Summary + commit message & MR attributes presented for approval (step 9) — presented to
  the user; publishing (push/MR) awaits their approval.

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | docs-gap | — | `CHANGELOG.md`'s `## [Unreleased]` section has no entry for this ticket's user-facing change (fenced-block exclusion + the `<!-- jerry:allow placeholder -->` opt-out marker), though JRY-003 and JRY-005 both added one for their own user-facing changes | `CHANGELOG.md:9-18` (`## [Unreleased]` / `### Added`) | Add an `### Added` bullet describing the marker and the fenced-block exclusion; fixed via scoped rework (round 1) |
| F2 | non-blocking | stale-xref | fixed inline | `DESIGN.md` line 3's version stamp was left at "Version 2.2" although this ticket's own §11 entry (added in the same commit) claims "Version 2.3" | `DESIGN.md:3` | Bumped to "Version 2.3" on the feature branch (commit `8037030`) |
| F3 | non-blocking | test-gap | fixed inline | `TestCheckPlaceholders/AllowMarkerDoesNotFillAnEmptySection` only asserted `stripComments` in isolation, not `checkSections` end-to-end — a real (if narrow) gap versus what decision 4 claims to prove | `internal/rules/rules_test.go` (`AllowMarkerDoesNotFillAnEmptySection`) | Added an end-to-end case constructing a document and calling `checkSections` directly; fixed on the feature branch (commit `8037030`) |
| F4 | non-blocking | other | noted | The opt-out marker is matched by exact `strings.Contains`; a near-miss (extra space, wrong casing, "placeholders") silently fails to suppress the finding with no diagnostic telling the author their marker didn't take | `internal/rules/rules.go:264` (`placeholderAllowMarker`) | Would need a fuzzy-match warning to fix; not worth scheduling on its own — noted for a future placeholder-rule ticket to pick up if one is ever filed |
| F5 | non-blocking | stale-xref | noted | `PLAN.md`'s `applies-to-validate` / `JRY-005` row still reads "to do" although JRY-005 is done and merged — a gap in JRY-005's own review (step 7), not something this branch caused | `PLAN.md:35` | Out of scope for this ticket; a future review or audit pass can correct it |

Disposition summary: 1 blocking (F1, fixed via scoped rework round 1), 2 fixed inline (F2, F3),
2 noted (F4, F5).

cost: estimated S, actual S

### Rework fix record — round 1 (commit c284739)

Added a `### Added` bullet to `CHANGELOG.md`'s `## [Unreleased]` section describing the fenced-
block exclusion and the `<!-- jerry:allow placeholder -->` opt-out marker (F1). No other change.

**Scoped re-review of round 1:** delegated to an independent reviewer (same rationale as step
0). Confirmed the commit touches only `CHANGELOG.md` (no scope creep), the bullet accurately
describes the shipped behaviour (whole-document opt-out, fenced-block exclusion — checked
against the actual `internal/rules/rules.go` diff, not just the prose), is placed correctly
under `## [Unreleased]` → `### Added`, matches the file's existing tone, and `just docs-check`
still passes. Verdict: clean, F1 closed. No new findings from the fix itself.

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new
- 2026-09-02 — TO DO → READY: plan complete
- 2026-09-02 — READY → IN DEVELOPMENT: picked up
- 2026-09-02 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-09-02 — IN REVIEW → REWORK: F1: CHANGELOG.md missing coverage entry (blocking docs-gap)
- 2026-09-02 — REWORK → IN REVIEW: findings fixed
- 2026-09-02 — IN REVIEW → DONE: review clean; F1 fixed via scoped rework, F2/F3 fixed inline, F4/F5 noted

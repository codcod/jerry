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

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new
- 2026-09-02 — TO DO → READY: plan complete

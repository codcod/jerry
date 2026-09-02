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

### 0. Feature branch (mandatory)

```
git checkout main
git checkout -b feat/JRY-005-validate-applies-to
```

Root-path child (`path = "."`) — tidy WIP commits into atomic ones before presenting, per
`tickets/README.md` §0.

### Prerequisite gate (hard)

None. `depends-on: []`; working tree is clean (`pickle board audit` green).

### Confirmed design decisions (do not deviate without asking)

1. **`applies_to` path validation lives in `internal/rules/rules.go`, not `internal/doc`.**
   `doc` stays a pure decode/encode layer with no judgement calls; `rules` is where every other
   judgement about field content already lives (`checkDate`, `checkRefs`, `checkSections`).
2. **An entry is invalid iff:** it is empty after trimming whitespace (covers both the
   empty-string and whitespace-only cases — trim first, then check length), OR
   `path.IsAbs(entry)` is true, OR any `/`-separated segment of the untrimmed entry equals
   `..`. Anything else is accepted; matching semantics (globs, existence) is explicitly
   out of scope (Description). Use the `path` package (posix semantics — repo-relative
   forward-slash paths, matching `path.Dir` already used in this file for `duplicate-id`).
3. **Malformed `applies_to` entries are errors** (`findings.errorf`, rule id `applies-to`),
   one finding per bad entry, on the `applies_to` line. Runs once per document (both ADR and
   SD carry the field), from the top-level `Check` loop next to `checkPlaceholders` — not
   nested inside `checkADR`/`checkSD` — since the rule and its line lookup are kind-agnostic.
4. **Unknown frontmatter keys are warnings** (`findings.warnf`, rule id `unknown-key`), one
   per unknown key, reported at that key's own line via `doc.FieldLine`. "Unknown" means:
   present in the document's frontmatter mapping (`document.Mapping`) but not in
   `doc.KeyOrder`. Also runs once per document from the top-level `Check` loop, guarded on
   `document.Mapping != nil` (nil on a `ParseErr` document, already reported separately).
5. **Close DESIGN.md §10 divergence row 1** the same way JRY-003 closed row 2: rewrite the
   `applies_to` row in §4.1's table to state it is now validated, delete row 1 from the §10
   table, and add a `## 11. Revision history` entry. Do this because leaving the divergence
   row in place after this ticket ships would make DESIGN.md wrong in the opposite direction.
6. **Fix the stale "or service ids" wording in `internal/cli/schema.go`'s `applies_to`
   descriptions** (both `adrSchema` and `sdSchema`) to say "Paths", matching the DESIGN.md
   §4.1 correction already on `main` (the ticket's Description references it). In scope
   because `jerry schema` is the one other place this field is described to a user, and
   leaving it saying "or service ids" while `validate` rejects non-path forms is the same
   kind of silent lie the rest of this ticket exists to close.

### Tasks

#### Task 1 — validate `applies_to` entries
In `internal/rules/rules.go`, add:
```go
func checkAppliesTo(findings *Findings, document *doc.Document) {
	line := doc.FieldLine(document.Mapping, "applies_to")
	for _, entry := range document.Front.AppliesTo {
		if reason, bad := badAppliesToPath(entry); bad {
			findings.errorf(document.Path, line, "applies-to",
				"applies_to entry %q is not a valid path: %s", entry, reason)
		}
	}
}

func badAppliesToPath(entry string) (string, bool) {
	if strings.TrimSpace(entry) == "" {
		return "empty or whitespace-only", true
	}
	if path.IsAbs(entry) {
		return "must be relative", true
	}
	for _, segment := range strings.Split(entry, "/") {
		if segment == ".." {
			return "must not contain `..`", true
		}
	}
	return "", false
}
```
Call `checkAppliesTo(&findings, document)` from `Check`, alongside the existing
`checkPlaceholders(&findings, document, options)` line (same loop, both kinds).

#### Task 2 — warn on unknown frontmatter keys
In `internal/rules/rules.go`, add a package-level `knownKeys` set built once from
`doc.KeyOrder` (`var knownKeys = func() map[string]bool { ... }()` or an `init()` — match
whichever style reads more like the rest of the file), and:
```go
func checkUnknownKeys(findings *Findings, document *doc.Document) {
	mapping := document.Mapping
	if mapping == nil {
		return
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i].Value
		if !knownKeys[key] {
			findings.warnf(document.Path, mapping.Content[i].Line, "unknown-key",
				"frontmatter key %q is not one jerry knows — check for a typo (preserved as-is by `jerry fmt`)", key)
		}
	}
}
```
Call `checkUnknownKeys(&findings, document)` from `Check`, next to `checkAppliesTo`.

#### Task 3 — test fixtures
Add two fixtures under `internal/rules/testdata/corpus/teams/broken/adr/`, following the
existing single-issue-per-file convention (`0003-bad-references.md`, `0004-empty-section.md`):
- `0006-bad-applies-to.md` — otherwise-clean ADR whose `applies_to:` list has four entries,
  one per bad case: empty string, whitespace-only, an absolute path, and a `..`-traversal
  path — expect four `applies-to` errors.
- `0007-unknown-key.md` — otherwise-clean ADR with a valid `applies_to:` plus a sibling
  `applies-to:` (hyphen) key holding some value — the exact motivating typo from the
  Description — expect one `unknown-key` warning and no `applies-to` error (the correctly
  spelled field still validates clean).

Regenerate: `go test ./internal/rules -update`, then read the resulting diff to
`testdata/findings.golden.json` and confirm it is exactly the four `applies-to` errors and
the one `unknown-key` warning, at the expected lines, with no unrelated change.

#### Task 4 — fix stale schema wording
In `internal/cli/schema.go`, change both `applies_to` `stringList(...)` calls: `"Paths or
service ids this decision governs."` → `"Paths this decision governs."` (ADR) and `"Paths or
service ids this design covers."` → `"Paths this design covers."` (SD).

#### Task 5 — DESIGN.md
- §4.1 table, `applies_to` row: replace "Accepted and preserved now, **not yet validated**
  (§10); nothing reads it until Phase 2." with "Validated: rejects empty/whitespace-only,
  absolute, and `..`-traversal entries. Nothing yet *reads* the field for drift or ownership
  — that's still Phase 2."
- §10 table: delete row 1 (`applies_to` is validated). Renumber nothing else — the existing
  rows already survived JRY-003's row-2 removal without renumbering; follow that precedent.
- §11 Revision history: add `**Version 2.2** (2026-09-02) — JRY-005 closed divergence 1
  (§4.1/§10 vs. `applies_to` validation): the field is now validated for path shape, unknown
  frontmatter keys warn, and the resolved row was removed from §10's table.` (mirrors the
  Version 2.1 entry's wording pattern).

### Acceptance test

```
go test ./internal/rules -update   # regenerate golden fixtures, then inspect the diff
just test                          # go test ./... — full suite green, golden fixture stable
just lint                          # gofmt clean
just docs-check                    # snowball check — DESIGN.md and the user manual still resolve
jerry validate ./internal/rules/testdata/corpus   # spot-check: 0006 reports 4 applies-to errors,
                                                    # 0007 reports 1 unknown-key warning, exit
                                                    # code reflects errors present
```

### Docs update (mandatory when user-facing)

- `DESIGN.md` — §4.1 table row, §10 divergence table, §11 revision history (Task 5, above).
- `docs/user-manual/introduction.adoc` — in the `== References between decisions` section
  (after the existing paragraph on `superseded_by`/`supersedes`/`related_adrs` resolution,
  around line 131), add a short paragraph: `applies_to` entries are validated as paths
  (rejecting empty, whitespace-only, absolute, and `..`-traversal entries), and any
  frontmatter key {product} does not recognise is reported as a warning rather than silently
  preserved. Cross-reference the existing `fmt` paragraph's "preserves keys {product} has
  never heard of" sentence (~line 143) so the two don't read as contradictory.

### Finish (mandatory)

1. Acceptance test green (above).
2. Docs updated and registered (Task 5 + user manual paragraph).
3. Write a summary: files touched, decisions made, anything deferred.
4. Suggest commit message:
   ```
   feat(rules): validate applies_to and warn on unknown frontmatter keys (JRY-005)
   ```
5. Tidy WIP commits into atomic ones before presenting (root-path child).
6. Commit locally on the ticket branch. Do not push or open a merge request without user
   approval. Present the commit message; after approval, verify the remote base is not
   behind (`git fetch origin main && git diff --name-only origin/main...HEAD | grep
   '^tickets/'` prints nothing), then push and open the MR. Hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new
- 2026-09-02 — TO DO → READY: plan complete
- 2026-09-02 — READY → IN DEVELOPMENT: picked up

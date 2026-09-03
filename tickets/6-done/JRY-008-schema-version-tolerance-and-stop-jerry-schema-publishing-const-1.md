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
DESIGN.md §3.6 (v1) never said what a bump would mean. Divergence 2 in DESIGN.md §10.

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

### 0. Feature branch (mandatory)

`jerry` is a root-path child (`path = "."` in `pickle.toml`), so the branch is cut in this same
repo:

```
git checkout main
git checkout -b feat/JRY-008-schema-version-tolerance
```

Commit locally as you go (WIP commits encouraged). Do not push or open a merge request without
explicit user approval — tidy the WIP commits into a small number of atomic ones first (this is
a root-path child, so tidy-then-keep-history is the default over squash), then present the
commit message and merge-request attributes per Finish.

### Prerequisite gate (hard)

None. `depends-on: []`, and PLAN.md's informal `release` coupling (`JRY-001`) is already done
and merged.

### Confirmed design decisions (do not deviate without asking)

1. **The shared constant lives in `internal/doc` as `CurrentSchemaVersion = 1`.** Both
   `internal/cli` (`schema.go`, `new.go`, `supersede.go`) and `internal/rules` already import
   `doc`; declaring it anywhere else would let a writer drift from the tolerance check the day
   v2 exists.
2. **`jerry schema` publishes a floor (`minimum`), not an equality, and no `maximum`.** An upper
   bound would recreate exactly the bug this ticket fixes the next time a binary lags behind a
   document.
3. **The tolerance rule is kind-independent and unconditional.** It is wired into `Check()`
   alongside `checkPlaceholders`/`checkAppliesTo`/`checkUnknownKeys` — not inside `checkADR` or
   `checkSD` — because DESIGN.md §3.6 states one rule for both kinds.
4. **Above-current is a warning, never an error, and never skips the rest of validation.** The
   new rule only appends a warning finding for that document; every other rule still runs on it
   at full strength.
5. **At-or-below current, including absent, produces no finding.** There is no v2 yet, so "read
   with that version's rules" has nothing to differentiate today — the deliverable is the
   direction (silent at-or-below, warn above), pinned by a document that claims a version from
   the future.
6. **DESIGN.md §10's divergence row `#2` is what this ticket closes** (the ticket Description
   originally cited row `#3`; corrected during refinement — row `#2` is the one naming
   `schema_version`/`jerry schema`).

### Tasks

#### Task 1 — Shared current-version constant
In `internal/doc/frontmatter.go`, add, near the `Front` struct:

```go
// CurrentSchemaVersion is the newest schema_version this binary understands.
// jerry new and jerry supersede write it into every document; jerry schema
// publishes it as a floor, not an equality (DESIGN.md §3.6): a document above
// it must warn, never error, so upgrading jerry is never a merge prerequisite.
const CurrentSchemaVersion = 1
```

#### Task 2 — Stop `jerry schema` publishing an equality
In `internal/cli/schema.go`, in both `adrSchema()` (line 75) and `sdSchema()` (line 98), replace

```go
"schema_version": map[string]any{"type": "integer", "const": 1},
```

with

```go
"schema_version": map[string]any{
    "type":        "integer",
    "minimum":     doc.CurrentSchemaVersion,
    "description": "A floor, not an equality: a document above this binary's newest known version still validates, with a warning, never a hard failure (DESIGN.md §3.6).",
},
```

Regenerate the pinned golden fixture: `go test ./internal/cli -update`, then inspect
`git diff internal/cli/testdata/golden/schema.json` and confirm it only touches the two
`schema_version` blocks.

#### Task 3 — Point the writers at the constant
Replace the literal in `internal/cli/new.go:132`, `internal/cli/new.go:195`, and
`internal/cli/supersede.go:66` — each currently `SchemaVersion: 1` — with
`SchemaVersion: doc.CurrentSchemaVersion`. This is what makes the day v2 ships a one-constant
change instead of a four-site grep.

#### Task 4 — The tolerance rule
In `internal/rules/rules.go`, add (following `checkUnknownKeys`'s shape, rules.go:328-340):

```go
// checkSchemaVersion warns when a document claims a schema_version newer than
// this binary knows, and never blocks: DESIGN.md §3.6 makes "the binary is
// replaceable" a charter item, so an old jerry must not turn a newer
// repository red merely for existing.
func checkSchemaVersion(findings *Findings, document *doc.Document) {
    if document.Front.SchemaVersion <= doc.CurrentSchemaVersion {
        return
    }
    findings.warnf(document.Path, doc.FieldLine(document.Mapping, "schema_version"), "schema-version-ahead",
        "schema_version %d is newer than this jerry binary knows (%d) — checked with %[2]d's rules; upgrade jerry to read %[1]d's own rules",
        document.Front.SchemaVersion, doc.CurrentSchemaVersion)
}
```

Wire it into `Check()` next to the other kind-independent calls:
`checkSchemaVersion(&findings, document)`.

#### Task 5 — Pin the direction with a fixture and tests
- Add `internal/rules/testdata/corpus/teams/payments/adr/0002-schema-version-from-the-future.md`:
  a structurally clean ADR (`id: ADR-0002`, `team: payments`, `status: Accepted` — not
  `Proposed`, so it cannot also trip `stale-proposal` — full `## Context` / `## Decision` /
  `## Consequences` sections, `deciders:` set, no placeholder text) with `schema_version: 99` in
  its frontmatter — deliberately far from a plausible off-by-one.
- Regenerate `internal/rules/testdata/findings.golden.json` (`go test ./internal/rules -update`)
  and check the diff adds exactly one new finding for that file: `severity: warning`,
  `rule: schema-version-ahead`.
- Add `"schema-version-ahead"` to the `wanted` slice in `TestCheckFixtureContract`
  (`rules_test.go:75-92`).
- Add a subtest next to `StaleProposalIsAWarningNotAnError` (`rules_test.go:109-115`) —
  `SchemaVersionAheadIsAWarningNotAnError` — asserting every `schema-version-ahead` finding has
  `Severity == SeverityWarning`.
- Add a subtest asserting the new fixture document produces **no error-severity finding at
  all** for its path (i.e. it still validates): iterate `findings`, and fail if any finding
  whose `Path` is the new fixture has `Severity == SeverityError`.

### Acceptance test

```
just build
just lint
just test
```

Plus, specifically:

- `go test ./internal/rules -run TestCheckFixtureContract -v` passes, exercising
  `schema-version-ahead` and both new subtests.
- `go test ./internal/rules -run TestCheckGolden` passes against the committed (regenerated)
  `findings.golden.json` — no `-update` needed once it is checked in.
- `go test ./internal/cli -run TestGolden` passes against the committed (regenerated)
  `testdata/golden/schema.json`.
- `go build -o jerry ./cmd/jerry && ./jerry schema --kind adr | grep -A3 '"schema_version"'`
  shows `"minimum": 1` and no `"const"` key (repeat with `--kind sd`).

### Docs update (mandatory when user-facing)

- `DESIGN.md` §10: remove divergence row `#2` (`schema_version` / `jerry schema`) from the
  table, and add a `## 11. Revision history` entry following the exact style of Version
  2.1/2.2/2.3 — e.g. **Version 2.4** (today's date) — JRY-008 closed divergence 2 (§3.6/§10 vs.
  `schema_version` tolerance): documents above the newest known `schema_version` now warn
  instead of silently mismatching `jerry schema`'s published constraint, and `jerry schema`
  publishes a floor (`minimum`) rather than an equality; the resolved row was removed from
  §10's table.
- `CHANGELOG.md`: add a bullet under `## [Unreleased]` → `### Fixed` describing both halves —
  `jerry validate` now warns (never errors) when a document's `schema_version` is newer than
  the binary knows, and `jerry schema` no longer publishes `schema_version` as `const: 1` (which
  would have rejected every future-versioned document in any editor's YAML language server).

### Finish (mandatory)

1. Acceptance test green; `just build`, `just lint`, `just test` all clean.
2. Docs updated and registered (DESIGN.md §10 + revision history, CHANGELOG.md).
3. Write a summary: files touched, decisions made, anything deferred (there is no v2 to migrate
   to yet, so no versioned rule dispatch is added — only the constant, the floor, and the
   warn-above-current direction).
4. Suggest a Conventional Commit message, ticket id in brackets at the end of the subject, e.g.:

   ```
   fix(schema): tolerate newer schema_version instead of rejecting it (JRY-008)

   jerry schema published `schema_version` as `const: 1`, which would reject every
   v2 document in any editor's YAML language server the day v2 exists — the exact
   inverse of DESIGN.md §3.6's stated purpose. jerry schema now publishes it as a
   floor (`minimum`), and jerry validate warns, never errors, when a document's
   schema_version is newer than the binary knows.
   ```

5. Tidy the branch's WIP commits into a small number of atomic ones (root-path child default:
   keep the tidied history rather than squashing).
6. Commit locally on `feat/JRY-008-schema-version-tolerance`. Do not push or open a merge
   request without user approval. Present the commit message and, once approved, verify the
   remote base is not behind (`git fetch origin main && git diff --name-only
   origin/main...HEAD | grep '^tickets/'` must print nothing), then push and open the merge
   request. Merging is always the human's. Hand back to the user.

## Review

- [x] Reviewer independence settled (step 0): **delegated** — the reviewing agent authored the
  branch in this same session, so steps 2–4a's audits were run by a fresh, independent
  subagent (`general-purpose`, briefed adversarially), checked out on
  `feat/JRY-008-schema-version-tolerance`. Its two findings were re-verified by hand before
  being recorded below; one severity was corrected (see F1).
- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 1, 2):
  `just build`, `just lint`, `just test`, `just docs-check` all pass; `go test
  ./internal/rules -run TestCheckFixtureContract -v` passes including both new subtests;
  `./jerry schema --kind adr|sd` confirmed `"minimum": 1`, no `"const"` key. All 5 tasks and
  all 6 confirmed design decisions in the Implementation Plan verified against the diff
  (`git diff main...feat/JRY-008-schema-version-tolerance`) — no gaps.
- [x] Quality audit (step 3): idiomatic, matches `checkUnknownKeys`'s shape; boundary condition
  (`<= doc.CurrentSchemaVersion`) correctly matches DESIGN.md §3.6's "at or below → silent,
  above → warn" including the absent/zero-value case; no security concerns.
- [x] Consistency audit (step 4): no remaining bare `SchemaVersion: 1` / `"const": 1` literals
  anywhere in the tree; every new `DESIGN.md §3.6` citation matches the post-edit text; §10
  divergence row `#2` fully removed, remaining rows self-consistent; `go.mod` dependency
  policy untouched; no new `cmd.Print*` misuse; findings-accumulate rule honoured.
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a):
  `just docs-check` passes; DESIGN.md §3.6/§4.1/§10 and CHANGELOG.md accurately describe the
  shipped behaviour in the right tense. **Coverage gap found — see F2.**
- [ ] Docs-readability pass (step 4b): **conscious skip** — no docs-readability reviewer
  configured in this session.
- [x] Findings recorded below with severity, class, and disposition (step 5); disposition
  summary and cost line present.
- [x] Ticket moved to `tickets/5-rework/` then, after the scoped re-review below confirmed
  F2 closed with no new findings, to `tickets/6-done/`; `## History` appended on each move
  (step 6).
- [x] Governing documents reconciled (step 7): F1 fixed inline (commit `4df5f25`); §10 row
  removed and §11/manual updated as part of the original implementation; `PLAN.md`'s
  `schema-tolerance` row updated to `done, publish-gated`. Board regenerated by each move.
- [x] Remaining-tickets impact sweep (step 8): `tickets/1-to-do/` and `tickets/2-ready/` are
  both empty — no dependent ticket to patch.
- [x] Summary presented below; commit message & MR attributes below, pending user approval
  per the commit policy.

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | stale-xref | fixed inline | DESIGN.md's header version stamp (line 3) still read "Version 2.3" after this branch added a "Version 2.4" entry to §11 — the document contradicted its own revision history. Governing-document staleness this branch caused (review protocol §7) is non-blocking by the protocol's own text, not "blocking" as the delegated auditor first reported — corrected on re-verification. | `DESIGN.md:3` vs. `DESIGN.md:466` (pre-fix) | Bump the header stamp to match §11's newest entry. |
| F2 | blocking | docs-gap | — | The user manual (`docs/user-manual/introduction.adoc`) was not updated for either user-facing behaviour change this ticket ships: `jerry schema` now publishes `schema_version` as a floor (`minimum`) instead of `const: 1`, and `jerry validate` gained a new `schema-version-ahead` warning. AGENTS.md names this exact convention ("a change to a command's behaviour or flags has two doc surfaces to reconcile: the manual is the one reviews keep missing"), and JRY-005 set direct precedent by documenting its own new warning (`unknown-key`) in this same file. The Implementation Plan's own "Docs update" step only named DESIGN.md and CHANGELOG.md, so the gap originates in the plan, not just the execution. | `docs/user-manual/introduction.adoc` (no `schema_version`/`jerry schema` floor/warning coverage); precedent at `introduction.adoc:143` (the `unknown-key` sentence); convention at `AGENTS.md:53` | Add a sentence near `introduction.adoc`'s existing frontmatter/unknown-key paragraph (or the "Editor support" section, next to the existing `jerry schema` example) stating: `jerry schema` publishes `schema_version` as a floor, and `jerry validate` warns — never errors — when a document's `schema_version` is newer than the binary knows. |

**Disposition summary:** 1 blocking (F2 → `5-rework/`), 1 non-blocking (F1, fixed inline).

cost: estimated M, actual M

### Verdict

Blocking finding present (F2). Ticket moves to `5-rework/` for a scoped fix: add the missing
manual coverage on the same branch. No other changes in scope for the rework pass.

### Rework fix record — round 1 (commit 01dece6)

F2: added two sentences to `docs/user-manual/introduction.adoc` — one beside the existing
`unknown-key` paragraph, covering `validate`'s new `schema-version-ahead` warning
(at-or-below silent, above warns, never errors); one in `== Editor support`, covering
`jerry schema`'s `schema_version` floor (`minimum`, not `const`). No other changes; scope was
F2 only. Acceptance test (`just build`, `just lint`, `just test`, `just docs-check`) re-run
green after the fix.

### Scoped re-review (round 1)

Delegated (I authored the round-1 fix in this session): independent subagent read the fix
diff (`git show 01dece6`) plus the ticket's F1/F2 rows, checked out the branch, re-ran the
full acceptance test (`just build`/`lint`/`test`/`docs-check`, all pass), confirmed F1 has
not regressed (`DESIGN.md:3` still reads "Version 2.4"), and confirmed the commit touches
only `docs/user-manual/introduction.adoc` — F2's scope, nothing else.

**F2 verified closed.** Both required disclosures are present, correctly placed, and match
the shipped code: the `schema-version-ahead` warning description matches
`internal/rules/rules.go`'s `checkSchemaVersion`, and the floor description matches
`./jerry schema --kind adr|sd`'s actual output (`"minimum": 1`, no `"const"`).

The delegated auditor raised one more finding — that "`schema_version` defaults to `1`" is
imprecise, since the Go zero-value isn't an active default (`internal/doc/frontmatter.go`'s
own comment: "nothing here is defaulted"). **Re-verified and dismissed, not recorded as a
finding**: `DESIGN.md`'s own §4.1 frontmatter table has stated "Defaults to 1" for
`schema_version` since before this ticket (`git show main:DESIGN.md` line 112, unedited by
this branch) — the manual's phrasing matches the project's own established, pre-existing
terminology, and is behaviourally accurate (an absent `schema_version` is treated identically
to `1` by every rule that reads it today). Not a defect this branch introduced.

No new findings. Ticket proceeds to `tickets/6-done/`.

**Disposition summary (final):** 1 blocking (F2, fixed in round 1, verified closed), 1
non-blocking (F1, fixed inline in the original review).

cost: estimated M, actual M (one rework round; no cost overrun)

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new
- 2026-09-03 — TO DO → READY: plan complete
- 2026-09-03 — READY → IN DEVELOPMENT: picked up
- 2026-09-03 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-09-03 — IN REVIEW → REWORK: F2 blocking: user manual not updated
- 2026-09-03 — REWORK → IN REVIEW: findings fixed
- 2026-09-03 — IN REVIEW → DONE: scoped re-review clean, F2 verified closed

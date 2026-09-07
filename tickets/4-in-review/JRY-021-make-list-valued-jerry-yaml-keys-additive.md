---
id: JRY-021
title: Make list-valued jerry.yaml keys additive
project: jerry
depends-on: [JRY-001]
spawned-by: []
impact: high
complexity: low
cost: S
---

# JRY-021 — Make list-valued jerry.yaml keys additive

## Outcome

A repository's `jerry.yaml` can extend the built-in `placeholders` list and the required-section
lists, but cannot replace them — so no repository can switch a governance rule off by
overriding a list-valued key to empty or to a narrower set.

## Description

DESIGN.md §3 (charter item 2) and §10 divergence #3: list-valued `jerry.yaml` keys
(`placeholders`, `required-adr-sections`, `required-sd-sections`) must be additive — a
repository's config extends the built-in defaults rather than replacing them. Today,
`Config.applyDefaults` in `internal/config/config.go` only fills a list-valued key from
`rules.DefaultOptions()` when the repo left it empty (`if len(c.Placeholders) == 0 { ... }`,
same pattern for both required-section fields); any non-empty repo value replaces the
built-in list wholesale, so a repository can silently drop coverage the tool is supposed to
guarantee everywhere.

Scope is exactly these three keys. `adr-dir`, `solution-design-dir`, `skip-dirs`,
`index-path`, and `proposed-stale-days` are layout/threshold settings, not governance rules,
and DESIGN.md's divergence table names only the placeholder and required-section lists —
they keep today's replace-if-empty behaviour.

This is a policy-integrity guarantee, not a feature: without it, `placeholder-escapes`
(JRY-006, done) added a scoped opt-out (`<!-- jerry:allow placeholder -->`, per-document) on
the explicit premise that the cheapest fix for a false positive should never be "delete the
phrase from `jerry.yaml`" — a wholesale key override does exactly that, at repository scope
rather than per-document, and would undercut JRY-006's own reasoning. No conflict between the
two: JRY-006's opt-out stays scoped and additive; this ticket closes the coarser bypass.

## Implementation Plan

### 0. Feature branch (mandatory)

Before any change, create a feature branch inside the `jerry` repo (`project: jerry`, path `.`
in `pickle.toml`):

```
git checkout main
git checkout -b feat/JRY-021-additive-list-config
```

Do all work on this branch, committing locally as you go. This is a root-path child
(`path = "."`): tidy WIP commits into atomic ones before presenting, and keep that tidied
history by default rather than squashing (rules §0). Publish only per the project's commit
policy — never push or open a merge request without explicit user approval. Under
`layout = "in-tree"`, before pushing verify the remote base is not behind:
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must
print nothing.

### Prerequisite gate (hard)

JRY-001 (`depends-on:`) is done and merged to `main` (commit `08ef282`). Working tree clean.
No other precondition.

### Confirmed design decisions (do not deviate without asking)

1. **Additive scope is exactly three keys: `placeholders`, `required-adr-sections`,
   `required-sd-sections`.** `adr-dir`, `solution-design-dir`, `skip-dirs`, `index-path`, and
   `proposed-stale-days` stay replace-if-empty — they configure layout/thresholds, not a rule a
   repo could switch off, and DESIGN.md §10's divergence #3 names only the placeholder and
   required-section lists.
2. **Merge is a deduplicated union, built-ins first, repo's own entries appended in the order
   given.** A repo re-listing a built-in entry (e.g. re-declaring `## Context`) must not produce
   a duplicate in the merged list, since `checkSections`/placeholder matching would otherwise
   report or scan the same entry twice.
3. **An explicit empty list (`placeholders: []`) is equivalent to omitting the key.** Additive
   means a repo can only extend, never narrow — including to nothing — so `[]` merges to just
   the built-ins, same as an absent key. This replaces the current `len(...) == 0` gate; the
   merge now runs unconditionally for these three fields.

### Tasks

#### Task 1 — additive merge in `internal/config/config.go`
Add an unexported helper:

```go
// mergeUnique returns defaults followed by every entry in extra not already
// present, preserving order and dropping duplicates.
func mergeUnique(defaults, extra []string) []string {
	merged := make([]string, 0, len(defaults)+len(extra))
	seen := make(map[string]bool, len(defaults)+len(extra))
	for _, v := range append(append([]string{}, defaults...), extra...) {
		if !seen[v] {
			seen[v] = true
			merged = append(merged, v)
		}
	}
	return merged
}
```

In `applyDefaults`, replace the three `if len(c.X) == 0 { c.X = defaults.X }` blocks for
`RequiredADRSections`, `RequiredSDSections`, and `Placeholders` with an unconditional
`c.X = mergeUnique(defaults.X, c.X)` for each. Leave `ADRDir`, `SDDir`, `SkipDirs`,
`IndexPath`, `ProposedStaleDays` untouched.

#### Task 2 — tests in `internal/config/config_test.go`
Add table-style or individual tests covering: a repo `jerry.yaml` that sets
`required-adr-sections` to a single custom entry ends up with the built-in three plus that
entry, in that order, no duplicates; a repo that re-lists a built-in placeholder gets no
duplicate; `placeholders: []` in the file yields exactly the built-in defaults (proving the
key cannot be used to switch the rule off). Confirm `TestPartialConfigKeepsOtherDefaults` and
`TestStarterParsesAsItsOwnConfig` (`internal/config/config_test.go:71`, `:124`) still pass
unmodified — the starter's `required-*-sections` values already equal the defaults, so the
merge is idempotent there.

#### Task 3 — DESIGN.md
In §3 charter item 2, delete the trailing `(Today they replace — see §10.)` aside (line 69).
In §10's divergence table, remove the row-3 entry (`Repositories own none of the rules …`) the
same way JRY-005/JRY-006/JRY-008 closed their divergence rows (§11 revision history), and add a
dated revision-history entry recording that JRY-021 closed divergence 3.

### Acceptance test

- `just test` green, including the new `internal/config` cases from Task 2.
- `just lint` clean.
- `just build` succeeds.
- Manual check: in a scratch repo with a `jerry.yaml` containing
  `required-adr-sections: ["## Extra"]`, run the ADR-validating command and confirm findings
  fire for a missing `## Context`/`## Decision`/`## Consequences` as well as a missing
  `## Extra` — proving the repo's key extended rather than replaced the defaults.

### Docs update (mandatory when user-facing)

Update DESIGN.md per Task 3 (removes a documented divergence; no separate user-facing doc
describes `jerry.yaml`'s list-key semantics beyond the `Starter` comment in `config.go`, which
already states defaults rather than replace/merge behaviour and needs no wording change).

### Finish (mandatory)

1. Acceptance test green; `just build`/`just test`/`just lint`/`just docs-check` clean.
2. DESIGN.md updated per Task 3.
3. Write a summary (files touched, decisions made, anything deferred).
4. Suggest commit message, e.g. `fix(config): make list-valued jerry.yaml keys additive (JRY-021)`.
5. Tidy WIP commits into atomic ones (root-path child).
6. Commit locally; publish only per approval, per the branch/publish steps in step 0 above.

## Review

**Reviewer independence (step 0):** the reviewing agent authored this branch in this same
session, so steps 2–4a were delegated to an independent sub-agent, briefed adversarially with
the ticket (read from `main`), `AGENTS.md`, `review-addendum.md`, and instructed to check out
`feat/JRY-021-additive-list-config` and re-run the configured commands. Every delegated finding
was re-verified by hand before entering this table, per step 0's "delegation buys independence,
not accuracy."

**Implementation audit (step 2):** all three tasks done as specified. `mergeUnique` in
`internal/config/config.go` is correct — fresh backing array (no aliasing of `defaults`/`extra`),
nil-safe, dedup preserves order. `applyDefaults` runs it unconditionally for
`RequiredADRSections`, `RequiredSDSections`, `Placeholders`; the five layout/threshold fields are
untouched, matching confirmed decision 1. `just build`/`test`/`lint`/`docs-check` all green. The
plan's manual acceptance check was independently re-run against a scratch repo and confirmed:
`required-adr-sections: ["## Extra"]` produces findings for the built-in three sections *and*
the added one.

**Quality audit (step 3):** idiomatic; the dependency policy (`cobra`+`pflag`+`yaml.v3`+stdlib)
is untouched; findings-accumulate and output-stream conventions are not implicated (no CLI-facing
change). Test coverage was initially incomplete — see F4.

**Consistency audit (step 4)** and **documentation audit (step 4a):** no stale cross-reference
introduced by the code change itself; `docs/user-manual.adoc` never documented list-key
semantics, so no doc there goes stale. But reconciling this branch with `main` (below) surfaced
that the branch was cut before origin's JRY-015 merge and its own `DESIGN.md` reconciliation
commit — the branch's `DESIGN.md` edit collided with an already-used version number. Resolved by
rebasing `main` onto `origin/main` (fast, no conflicts — the 3 local board-bookkeeping commits
were unpushed) and then rebasing the feature branch onto the reconciled `main`, renumbering this
ticket's revision-history entry from the colliding "Version 2.7" to **"Version 2.8"** and
bumping the line-3 stamp to match (this *is* finding F1, discovered and fixed together with the
rebase — see its row).

**Governing-documents reconciliation (step 7):** `DESIGN.md`, `PLAN.md`, and `CHANGELOG.md` are
this project's governing documents (`review-addendum.md`). `DESIGN.md`'s §10 divergence-3 row and
§3 aside were correctly removed by the ticket's own Task 3. `PLAN.md`'s "Filed so far" table and
its `config-additive` cross-cutting row, and `CHANGELOG.md`'s `[Unreleased]` section, were not
updated — F2 and F3 below, both within this review's reach (same repository, root-path child).

**Impact sweep (step 8):** no ticket in `tickets/1-to-do/` or `tickets/2-ready/` references
JRY-021 in `depends-on:` or Description. Nothing to patch.

**Docs-readability pass (step 4b):** conscious skip — no docs-readability reviewer configured in
this session.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | stale-xref | — | `DESIGN.md`'s version stamp (line 3) was not bumped when a new §11 revision-history entry was added, and the entry itself used a version number ("2.7") already claimed by origin's JRY-015 reconciliation commit (`2a89cce`), since the branch was cut before that merge | `DESIGN.md:3` read "Version 2.6" pre-rebase; origin/main's `DESIGN.md` already reads "Version 2.7" as of JRY-015 | Already resolved: fixed while reconciling the branch with `origin/main` (rebase, commit `469ab3e`) — line 3 now reads "Version 2.8", and the JRY-021 revision-history entry is renumbered to match, sitting after JRY-015's 2.7 entry |
| F2 | blocking | docs-gap | — | `PLAN.md`'s "Filed so far" table has no row for `config-additive`/JRY-021, and the cross-cutting table's `config-additive` row (line 339) still describes it as an un-filed future task | `PLAN.md:339` unchanged; no `JRY-021` row added to the "Filed so far" table (`review-addendum.md` step 7: "a review that concludes a ticket updates that row") | Add `\| config-additive \| JRY-021 \| done, publish-gated \|` to "Filed so far"; mark the cross-cutting row 339 resolved |
| F3 | blocking | docs-gap | — | `CHANGELOG.md`'s `## [Unreleased]` section has no entry for this fix, despite it being a real user-facing behavior change (list-valued `jerry.yaml` keys go from replace-if-empty to additive) | `CHANGELOG.md` `[Unreleased]` (lines 9–23) has entries for JRY-012 and JRY-008 but none for JRY-021; `review-addendum.md` step 4a names the Unreleased section as coverage, class `docs-gap` | Add a `### Fixed` (or `### Changed`) bullet describing the additive-merge behavior change, following the JRY-008 entry's style |
| F4 | non-blocking | test-gap | fixed inline | `TestListValuedKeysDedupeRepeatedDefaults` and `TestEmptyListCannotSwitchOffTheDefaults` compared only `len(...)` against the defaults, so a coincidental length match without content equality would have passed | `internal/config/config_test.go` (pre-fix, lines ~133–172) | Fixed inline (branch-authored idiom, no behaviour change): both now use `reflect.DeepEqual` for full content comparison — commit `0ce44da` |

**Disposition summary:** 3 blocking (F1 already fixed via the main-rebase; F2, F3 open — ticket
to `5-rework/`), 1 non-blocking fixed inline (F4).

`cost: estimated S, actual S`

### Rework fix record — round 1 (commit `0c6607e`)

Scope: F2 and F3 only (F1 and F4 were already resolved before this round — see their rows
above). Branch tip before this round's fix: `0ce44da`.

- **F2** — added `| \`config-additive\` | \`JRY-021\` | in review |` to `PLAN.md`'s "Filed so
  far" table. The cross-cutting description row (`PLAN.md:339`) was deliberately **left
  untouched**: `git log -p` on `PLAN.md` shows every prior filed ticket (`JRY-006`, `JRY-008`)
  only ever edited the "Filed so far" status cell, never its cross-cutting description row —
  the original finding's suggestion to "mark it resolved" did not match that precedent, so the
  fix follows the actual convention instead. Status is `in review`, not `done`, since the
  ticket has not concluded yet; whichever review concludes it updates this row again per
  `review-addendum.md` step 7.
- **F3** — added a bullet under `CHANGELOG.md`'s existing `[Unreleased]` → `### Fixed` heading,
  in the style of the adjacent JRY-008 entry, describing the additive-merge behavior change.
- Re-ran `just build`/`test`/`lint`/`docs-check` — all green (unchanged from the prior round;
  these two files carry no executable behavior).

**Scoped re-review (step 1/6a):** reviewer independence — the reviewing agent authored round
1's fix in this same session, so the audit was delegated to an independent sub-agent, briefed
with the ticket's Review section and the fix diff `0ce44da..0c6607e`, and re-verified by hand
before recording. Scope was exactly F2 and F3 plus the round's own new text (per protocol §1,
not a re-audit of the whole feature). Findings:

- **F2** — confirmed resolved. `PLAN.md:44` carries the new row, correct columns/style. The
  claim that leaving the cross-cutting row untouched matches precedent was independently
  verified against `git log -p -- PLAN.md` (JRY-006's and JRY-008's filings each touched only
  the "Filed so far" status cell) — not just re-asserted.
- **F3** — confirmed resolved. The new `CHANGELOG.md` bullet was checked against
  `internal/config/config.go`'s current `applyDefaults`/`mergeUnique` and matches the shipped
  behavior exactly; correctly placed under the existing `[Unreleased]` → `### Fixed` heading.
- Diff hygiene: `0ce44da..0c6607e` touches only `PLAN.md` (+1) and `CHANGELOG.md` (+4), no
  scope creep. `just build`/`test`/`lint`/`docs-check` re-run clean on `0c6607e`.
- **No new findings.** Zero blocking, zero non-blocking this round.

**Verdict: clean.** No blocking findings remain (F1/F4 resolved before this round, F2/F3
resolved this round) → ticket proceeds to `6-done/` (protocol §6b). `PLAN.md`'s "Filed so far"
row for `config-additive`/JRY-021 is updated from `in review` to `done, publish-gated` as part
of concluding this review (`review-addendum.md` step 7: "a review that concludes a ticket
updates that row").

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's cross-cutting row
  `config-additive`.
- 2026-09-06 — refined: implementation plan complete.
- 2026-09-06 — TO DO → READY: plan complete
- 2026-09-06 — READY → IN DEVELOPMENT: picked up
- 2026-09-06 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-09-07 — IN REVIEW → REWORK: F1 fixed (branch rebase), F2/F3 open: PLAN.md + CHANGELOG.md governing-doc coverage
- 2026-09-07 — REWORK → IN REVIEW: findings fixed (F2/F3 governing-doc coverage)

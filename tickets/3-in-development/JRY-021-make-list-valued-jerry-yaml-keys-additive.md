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

<!-- empty until IN REVIEW -->

## History

- 2026-09-03 — created (TO DO). source: chat: filed from PLAN.md's cross-cutting row
  `config-additive`.
- 2026-09-06 — refined: implementation plan complete.
- 2026-09-06 — TO DO → READY: plan complete
- 2026-09-06 — READY → IN DEVELOPMENT: picked up
